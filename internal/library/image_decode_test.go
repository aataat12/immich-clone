package library

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"hash/crc32"
	"image"
	"image/color"
	"image/gif"
	"image/jpeg"
	"image/png"
	"io"
	"math"
	"os"
	"runtime"
	"strings"
	"testing"
	"time"
)

func parentFixture(t *testing.T, kind string) []byte {
	t.Helper()
	if kind != "jpeg" && kind != "png" && kind != "gif" {
		return codecFixture(t, kind)
	}
	var b bytes.Buffer
	img := image.NewRGBA(image.Rect(0, 0, 32, 24))
	var err error
	if kind == "jpeg" {
		err = jpeg.Encode(&b, img, nil)
	} else if kind == "gif" {
		err = gif.Encode(&b, img, nil)
	} else {
		err = png.Encode(&b, img)
	}
	if err != nil {
		t.Fatal(err)
	}
	return b.Bytes()
}
func TestSEC06ParentDecodeCorpusAndRecovery(t *testing.T) {
	for _, kind := range []string{"jpeg", "png", "gif", "bmp", "tiff"} {
		t.Run(kind, func(t *testing.T) {
			data := parentFixture(t, kind)
			for _, n := range []int{0, 1, 8, len(data) / 2} {
				var before, after runtime.MemStats
				runtime.ReadMemStats(&before)
				started := time.Now()
				_, err := decodeParentImage(context.Background(), bytes.NewReader(data[:n]))
				runtime.ReadMemStats(&after)
				if err == nil {
					t.Fatalf("truncated %d accepted", n)
				}
				t.Logf("truncated=%d wall=%s allocated_bytes=%d", n, time.Since(started), after.TotalAlloc-before.TotalAlloc)
			}
			img, err := decodeParentImage(context.Background(), bytes.NewReader(data))
			if err != nil || img.Bounds().Dx() != 32 || img.Bounds().Dy() != 24 {
				t.Fatal("valid parity", err)
			}
			s, _ := openTest(t)
			bad := data[:len(data)/2]
			a, _, err := s.Put("malformed."+kind, bytes.NewReader(bad))
			if err != nil {
				t.Fatal(err)
			}
			if _, err = s.ProcessOne(); err != nil {
				t.Fatal(err)
			}
			got, err := s.Get(a.ID)
			if err != nil || got.Preview != "failed" {
				t.Fatal("persisted failure", err, got.Preview)
			}
			if _, err = s.analysisImage(context.Background(), a.ID); err == nil {
				t.Fatal("malformed AI accepted")
			}
			if h, err := fileHash(s.Original(a.ID)); err != nil || h != a.ID {
				t.Fatal("original altered", err)
			}
			good, _, err := s.Put("valid."+kind, bytes.NewReader(data))
			if err != nil {
				t.Fatal(err)
			}
			if _, err = s.ProcessOne(); err != nil {
				t.Fatal(err)
			}
			got, err = s.Get(good.ID)
			if err != nil || got.Preview != "ready" {
				t.Fatal("post-failure preview recovery", err, got.Preview)
			}
			if _, err = s.analysisImage(context.Background(), good.ID); err != nil {
				t.Fatal("post-failure AI recovery", err)
			}
		})
	}
}
func TestSEC06DimensionAndEncodedBombs(t *testing.T) {
	for _, d := range [][2]int{{0, 1}, {1, 0}, {-1, 1}, {math.MaxInt, math.MaxInt}, {40_000_001, 1}, {8000, 5001}} {
		if validImageDimensions(d[0], d[1]) {
			t.Fatal("unsafe dimensions admitted", d)
		}
	}
	if !validImageDimensions(8000, 5000) {
		t.Fatal("40MP parity")
	}
	data := parentFixture(t, "png")
	// Real valid-CRC PNG header: reject advertised size before any pixel allocation.
	binary.BigEndian.PutUint32(data[16:20], 0x7fffffff)
	binary.BigEndian.PutUint32(data[20:24], 0x7fffffff)
	binary.BigEndian.PutUint32(data[29:33], crc32.ChecksumIEEE(data[12:29]))
	if _, err := decodeParentImage(context.Background(), bytes.NewReader(data)); err == nil {
		t.Fatal("dimension bomb accepted")
	}
	f, err := os.CreateTemp(t.TempDir(), "large")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if err = f.Truncate(maxParentImageBytes + 1); err != nil {
		t.Fatal(err)
	}
	if _, err = decodeParentImage(context.Background(), f); err == nil {
		t.Fatal("encoded limit bypass")
	}
}

type cancelReadSeeker struct {
	*bytes.Reader
	cancel context.CancelFunc
	reads  int
}

func (r *cancelReadSeeker) Read(p []byte) (int, error) {
	r.reads++
	if r.reads == 2 {
		r.cancel()
	}
	return r.Reader.Read(p)
}

type cancelImage struct {
	image.Image
	cancel context.CancelFunc
}

func (i cancelImage) At(x, y int) color.Color { i.cancel(); return i.Image.At(x, y) }
func TestSEC06DecodeCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := decodeParentImage(ctx, bytes.NewReader(parentFixture(t, "png"))); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	ctx, cancel = context.WithCancel(context.Background())
	defer cancel()
	r := &cancelReadSeeker{Reader: bytes.NewReader(parentFixture(t, "png")), cancel: cancel}
	if _, err := decodeParentImage(ctx, r); !errors.Is(err, context.Canceled) {
		t.Fatal("decode cancellation", err)
	}
	ctx, cancel = context.WithCancel(context.Background())
	defer cancel()
	img := cancelImage{Image: image.NewRGBA(image.Rect(0, 0, 64, 64)), cancel: cancel}
	if _, err := orientedContext(ctx, img, 6); !errors.Is(err, context.Canceled) {
		t.Fatal("rotation cancellation", err)
	}
}
func TestSEC06FilenameRejectedBeforeRead(t *testing.T) {
	s, _ := openTest(t)
	for _, name := range []string{"bad\r\nheader.jpg", "nul\x00.jpg", string([]byte{255}), strings.Repeat("a", 257)} {
		if _, _, err := s.Put(name, failInputReader{}); err == nil || strings.Contains(err.Error(), "reader touched") {
			t.Fatal("invalid name read input", err)
		}
	}
	a, _, err := s.Put("C:\\fakepath\\family \u00e9 <script>.jpg", bytes.NewReader(parentFixture(t, "jpeg")))
	if err != nil || a.Name != "family \u00e9 <script>.jpg" {
		t.Fatal("display basename parity", err, a.Name)
	}
	if _, err = cleanRelative("album/" + string([]byte{255})); err == nil {
		t.Fatal("invalid UTF8 path")
	}
}

type failInputReader struct{}

func (failInputReader) Read([]byte) (int, error) { return 0, errors.New("reader touched") }

var _ io.Reader = failInputReader{}

// Exercise actual large decoded pixels and orientation, rather than only headers.
// TotalAlloc is an allocation measurement for this finite case, not process RSS.
func TestSEC06ParentLargeImageResourceEvidence(t *testing.T) {
	if testing.Short() {
		t.Skip("large finite resource probe")
	}
	var encoded bytes.Buffer
	src := image.NewNRGBA64(image.Rect(0, 0, 8000, 5000))
	src.SetNRGBA64(0, 0, color.NRGBA64{R: 65535, A: 65535})
	if err := png.Encode(&encoded, src); err != nil {
		t.Fatal(err)
	}
	src = nil
	runtime.GC()
	var before, after runtime.MemStats
	runtime.ReadMemStats(&before)
	started := time.Now()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	img, err := decodeParentImage(ctx, bytes.NewReader(encoded.Bytes()))
	if err != nil {
		t.Fatal(err)
	}
	img, err = orientedContext(ctx, img, 6)
	if err != nil {
		t.Fatal(err)
	}
	runtime.ReadMemStats(&after)
	if img.Bounds().Dx() != 5000 || img.Bounds().Dy() != 8000 {
		t.Fatal("40MP orientation parity")
	}
	if after.TotalAlloc-before.TotalAlloc > 520<<20 {
		t.Fatal("decode/rotation exceeded finite allocation budget")
	}
	r, g, b, a := img.At(4999, 0).RGBA()
	if r != 65535 || g != 0 || b != 0 || a != 65535 {
		t.Fatal("16bit orientation color parity")
	}
	t.Logf("40MP PNG16+rotation wall=%s allocated_bytes=%d encoded_bytes=%d", time.Since(started), after.TotalAlloc-before.TotalAlloc, encoded.Len())
}

func TestSEC06LargeEncodedJPEGPreviewParity(t *testing.T) {
	s, _ := openTest(t)
	f, err := os.CreateTemp(t.TempDir(), "large-jpeg")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if _, err = f.Write(parentFixture(t, "jpeg")); err != nil {
		t.Fatal(err)
	}
	if err = f.Truncate(maxParentImageBytes + 1); err != nil {
		t.Fatal(err)
	}
	if _, err = f.Seek(0, 0); err != nil {
		t.Fatal(err)
	}
	a, _, err := s.Put("large.jpg", f)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.ProcessOne(); err != nil {
		t.Fatal(err)
	}
	a, err = s.Get(a.ID)
	if err != nil || a.Preview != "ready" {
		t.Fatal("prior JPEG preview compatibility", err, a.Preview)
	}
	if _, err = s.analysisImage(context.Background(), a.ID); err == nil {
		t.Fatal("existing AI size limit lost")
	}
}
