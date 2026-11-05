package library

import (
	"bytes"
	"compress/zlib"
	"context"
	"encoding/binary"
	"errors"
	"golang.org/x/image/tiff"
	"runtime"
	"testing"
	"time"
)

// Synthetic baseline TIFF: one grayscale pixel per row, with deliberately
// repeated strip offsets. Compressed data and all metadata are public fixtures.
func sec06TIFF(t *testing.T, rows int, expanded int, tile bool) ([]byte, int) {
	t.Helper()
	var compressed bytes.Buffer
	zw := zlib.NewWriter(&compressed)
	if _, e := zw.Write(bytes.Repeat([]byte{127}, expanded)); e != nil {
		t.Fatal(e)
	}
	if e := zw.Close(); e != nil {
		t.Fatal(e)
	}
	type entry struct {
		tag, kind    uint16
		count, value uint32
	}
	entries := []entry{{256, 4, 1, 1}, {257, 4, 1, uint32(rows)}, {258, 3, 1, 8}, {259, 3, 1, 8}, {262, 3, 1, 1}}
	count := rows
	if tile {
		count = 1
		entries = append(entries, entry{322, 4, 1, 1025}, entry{323, 4, 1, 1025}, entry{324, 4, 1, 0}, entry{325, 4, 1, uint32(compressed.Len())})
	} else {
		entries = append(entries, entry{273, 4, uint32(rows), 0}, entry{278, 4, 1, 1}, entry{279, 4, uint32(rows), 0})
	}
	dataOffset := 8 + 2 + len(entries)*12 + 4
	offsetsStart, countsStart := dataOffset, dataOffset+count*4
	if count > 1 {
		dataOffset += count * 8
	}
	b := make([]byte, dataOffset)
	copy(b, "II")
	binary.LittleEndian.PutUint16(b[2:], 42)
	binary.LittleEndian.PutUint32(b[4:], 8)
	binary.LittleEndian.PutUint16(b[8:], uint16(len(entries)))
	for i, e := range entries {
		switch e.tag {
		case 273, 324:
			if count == 1 {
				e.value = uint32(dataOffset)
			} else {
				e.value = uint32(offsetsStart)
			}
		case 279:
			if count == 1 {
				e.value = uint32(compressed.Len())
			} else {
				e.value = uint32(countsStart)
			}
		}
		p := 10 + i*12
		binary.LittleEndian.PutUint16(b[p:], e.tag)
		binary.LittleEndian.PutUint16(b[p+2:], e.kind)
		binary.LittleEndian.PutUint32(b[p+4:], e.count)
		binary.LittleEndian.PutUint32(b[p+8:], e.value)
	}
	if count > 1 {
		for i := 0; i < count; i++ {
			binary.LittleEndian.PutUint32(b[offsetsStart+i*4:], uint32(dataOffset))
			binary.LittleEndian.PutUint32(b[countsStart+i*4:], uint32(compressed.Len()))
		}
	}
	return append(b, compressed.Bytes()...), dataOffset
}

func TestSEC06TIFFHostileTileAndStripMetadata(t *testing.T) {
	tile, _ := sec06TIFF(t, 1, 1, true)
	if _, e := decodeParentImage(context.Background(), bytes.NewReader(tile)); e == nil {
		t.Fatal("oversized tile accepted")
	}
	strips, _ := sec06TIFF(t, 1, 1, false)
	// StripOffsets count overflows the upstream TIFF datatype-size bound.
	binary.LittleEndian.PutUint32(strips[10+5*12+4:], 0xffffffff)
	if _, e := decodeParentImage(context.Background(), bytes.NewReader(strips)); e == nil {
		t.Fatal("hostile strip count accepted")
	}
	good, _ := sec06TIFF(t, 4, 1, false)
	img, e := decodeParentImage(context.Background(), bytes.NewReader(good))
	if e != nil || img.Bounds().Dx() != 1 || img.Bounds().Dy() != 4 {
		t.Fatal("valid strip parity", e)
	}
}

func TestSEC06TIFFCompressedExpansionIsPixelBounded(t *testing.T) {
	data, _ := sec06TIFF(t, 1, 1<<20, false)
	var before, after runtime.MemStats
	runtime.ReadMemStats(&before)
	start := time.Now()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	img, e := decodeParentImage(ctx, bytes.NewReader(data))
	runtime.ReadMemStats(&after)
	if e != nil || img.Bounds().Dx() != 1 || img.Bounds().Dy() != 1 {
		t.Fatal("bounded strip expansion", e)
	}
	allocation := after.TotalAlloc - before.TotalAlloc
	if allocation > 8<<20 {
		t.Fatalf("small-pixel compressed strip allocated %d bytes", allocation)
	}
	t.Logf("TIFF expanded_stream_bytes=%d pixel_count=1 wall=%s allocated_bytes=%d", 1<<20, time.Since(start), allocation)
}

type sec06StripCancelReader struct {
	*bytes.Reader
	cancel        context.CancelFunc
	payloadOffset int64
	payloadReads  int
}

func (r *sec06StripCancelReader) Read(p []byte) (int, error) {
	pos, _ := r.Reader.Seek(0, 1)
	if pos >= r.payloadOffset {
		r.payloadReads++
		if r.payloadReads == 8 {
			r.cancel()
		}
	}
	return r.Reader.Read(p)
}
func TestSEC06TIFFRepeatedStripCancellation(t *testing.T) {
	data, offset := sec06TIFF(t, 1024, 1, false)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	r := &sec06StripCancelReader{Reader: bytes.NewReader(data), cancel: cancel, payloadOffset: int64(offset)}
	_, e := decodeParentImage(ctx, r)
	if !errors.Is(e, context.Canceled) || r.payloadReads > 9 {
		t.Fatalf("repeated strips failed cooperative cancellation: reads=%d error=%v", r.payloadReads, e)
	}
	t.Logf("TIFF repeated_strips=1024 payload_reads_before_cancel=%d", r.payloadReads)
}

// Never build the multi-gigabyte decompressed attack body. A one-byte payload
// proves preflight refusal before pixel allocation/decompression is entered.
// Without the padded-tile guard, the upstream decoder reaches payload reads.
type sec06NoTilePayloadReader struct {
	*bytes.Reader
	payloadOffset int64
	touched       bool
}

func (r *sec06NoTilePayloadReader) Read(p []byte) (int, error) {
	pos, _ := r.Reader.Seek(0, 1)
	if pos >= r.payloadOffset {
		r.touched = true
		return 0, errors.New("tile payload reached")
	}
	return r.Reader.Read(p)
}
func TestSEC06TIFFPaddingAmplificationRejectedBeforeDecode(t *testing.T) {
	data, offset := sec06TIFF(t, 1, 1, true)
	// Output is exactly40MP; sixteen permitted padding columns advertise680MP
	// in one tile. Upstream's relative-padding check skips its1024-tile bound.
	binary.LittleEndian.PutUint32(data[10+1*12+8:], 40_000_000)
	binary.LittleEndian.PutUint32(data[10+5*12+8:], 17)
	binary.LittleEndian.PutUint32(data[10+6*12+8:], 40_000_000)
	cfg, e := tiff.DecodeConfig(bytes.NewReader(data))
	if e != nil || cfg.Width != 1 || cfg.Height != 40_000_000 {
		t.Fatal("attack header precondition", e)
	}
	r := &sec06NoTilePayloadReader{Reader: bytes.NewReader(data), payloadOffset: int64(offset)}
	if _, e = decodeParentImage(context.Background(), r); e == nil || r.touched {
		t.Fatalf("padded tile entered decode: payload_read=%t error=%v", r.touched, e)
	}
}
