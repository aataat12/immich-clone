package library

import (
	"bytes"
	"encoding/binary"
	"golang.org/x/image/bmp"
	"golang.org/x/image/tiff"
	"image"
	"image/color"
	"image/gif"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func codecFixture(t *testing.T, kind string) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 32, 24))
	for y := 0; y < 24; y++ {
		for x := 0; x < 32; x++ {
			img.Set(x, y, color.RGBA{255, 0, 0, 255})
		}
	}
	var b bytes.Buffer
	var e error
	switch kind {
	case "bmp":
		e = bmp.Encode(&b, img)
	case "tiff":
		e = tiff.Encode(&b, img, &tiff.Options{Compression: tiff.Deflate, Predictor: true})
	case "gif":
		pal := color.Palette{color.RGBA{255, 0, 0, 255}, color.RGBA{0, 0, 255, 255}}
		first := image.NewPaletted(img.Bounds(), pal)
		second := image.NewPaletted(img.Bounds(), pal)
		for i := range second.Pix {
			second.Pix[i] = 1
		}
		e = gif.EncodeAll(&b, &gif.GIF{Image: []*image.Paletted{first, second}, Delay: []int{10, 10}})
	}
	if e != nil {
		t.Fatal(e)
	}
	return b.Bytes()
}
func TestExtendedCodecsOriginalsPreviewsRecovery(t *testing.T) {
	for _, kind := range []string{"bmp", "gif", "tiff"} {
		t.Run(kind, func(t *testing.T) {
			s, root := openTest(t)
			data := codecFixture(t, kind)
			a, _, e := s.Put("synthetic."+kind, bytes.NewReader(data))
			if e != nil {
				t.Fatal(e)
			}
			// Emulate an older catalog: TIFF was not previously recognized by MIME sniffing.
			a.Preview = "failed"
			a.PreviewRevision = 1
			if kind == "tiff" {
				a.MIME = "application/octet-stream"
			}
			if e = s.save(a); e != nil {
				t.Fatal(e)
			}
			if e = s.Close(); e != nil {
				t.Fatal(e)
			}
			s, e = Open(root)
			if e != nil {
				t.Fatal(e)
			}
			defer s.Close()
			a, _ = s.Get(a.ID)
			if a.Preview != "queued" {
				t.Fatal("legacy failed preview not requeued")
			}
			if _, e = s.ProcessOne(); e != nil {
				t.Fatal(e)
			}
			a, _ = s.Get(a.ID)
			if a.Preview != "ready" || a.Width != 32 || a.Height != 24 || a.MIME != "image/"+kind {
				t.Fatalf("bad preview: %+v", a)
			}
			preview, e := os.Open(s.PreviewPath(a.ID))
			if e != nil {
				t.Fatal(e)
			}
			img, e := png.Decode(preview)
			preview.Close()
			if e != nil {
				t.Fatal(e)
			}
			r, g, b, _ := img.At(1, 1).RGBA()
			if r != 65535 || g != 0 || b != 0 {
				t.Fatal("decoded pixel or GIF first frame incorrect")
			}
			original, _ := os.ReadFile(s.Original(a.ID))
			if !bytes.Equal(original, data) {
				t.Fatal("original modified")
			}
			var archive bytes.Buffer
			if e = s.Backup(&archive); e != nil {
				t.Fatal(e)
			}
			zip := filepath.Join(t.TempDir(), "backup.zip")
			if e = os.WriteFile(zip, archive.Bytes(), 0600); e != nil {
				t.Fatal(e)
			}
			target := filepath.Join(t.TempDir(), "restore")
			if e = Restore(zip, target); e != nil {
				t.Fatal(e)
			}
			restored, e := Open(target)
			if e != nil {
				t.Fatal(e)
			}
			defer restored.Close()
			if _, e = restored.ProcessOne(); e != nil {
				t.Fatal(e)
			}
			ra, _ := restored.Get(a.ID)
			if ra.Preview != "ready" {
				t.Fatal("restored derived preview failed")
			}
			if e = restored.Verify(); e != nil {
				t.Fatal(e)
			}
		})
	}
}
func TestExtendedCodecsRejectLimitsAndMalformed(t *testing.T) {
	for _, kind := range []string{"bmp", "gif", "tiff"} {
		t.Run(kind, func(t *testing.T) {
			s, _ := openTest(t)
			data := codecFixture(t, kind)
			a, _, e := s.Put("bad."+kind, bytes.NewReader(data[:len(data)/2]))
			if e != nil {
				t.Fatal(e)
			}
			if _, e = s.ProcessOne(); e != nil {
				t.Fatal(e)
			}
			a, _ = s.Get(a.ID)
			if a.Preview != "failed" {
				t.Fatal("truncated media accepted")
			}
			if e = s.Verify(); e != nil {
				t.Fatal(e)
			}
			good, _, e := s.Put("large."+kind, bytes.NewReader(data))
			if e != nil {
				t.Fatal(e)
			}
			good.Size = 64<<20 + 1
			if e = s.makePreview(&good); e == nil || !strings.Contains(e.Error(), "64 MiB") {
				t.Fatal("encoded limit missed")
			}
		})
	}
	s, _ := openTest(t)
	data := codecFixture(t, "gif")
	binary.LittleEndian.PutUint16(data[6:], 65535)
	binary.LittleEndian.PutUint16(data[8:], 65535)
	a, _, e := s.Put("huge.gif", bytes.NewReader(data))
	if e != nil {
		t.Fatal(e)
	}
	if e = s.makePreview(&a); e == nil || !strings.Contains(e.Error(), "40 megapixel") {
		t.Fatal("pixel limit missed")
	}
}
