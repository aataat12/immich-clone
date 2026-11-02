package recognition

import (
	"bytes"
	"context"
	"encoding/binary"
	"image"
	"image/color"
	"math"
	"testing"
)

func TestFrameLayoutAndBounds(t *testing.T) {
	img := image.NewNRGBA(image.Rect(7, 9, 39, 41))
	img.SetNRGBA(7, 9, color.NRGBA{R: 255, G: 128, B: 64, A: 255})
	var b bytes.Buffer
	if e := WriteFrame(&b, img); e != nil {
		t.Fatal(e)
	}
	data := b.Bytes()
	if string(data[:4]) != "ICAI" || binary.LittleEndian.Uint32(data[4:]) != 32 || !bytes.Equal(data[12:15], []byte{64, 128, 255}) || len(data) != 12+32*32*3 {
		t.Fatal("BGR framing")
	}
	if e := WriteFrame(&b, image.NewRGBA(image.Rect(0, 0, 31, 32))); e == nil {
		t.Fatal("tiny frame")
	}
	scaled := Frame(image.NewRGBA(image.Rect(0, 0, 1920, 1080)))
	if scaled.Bounds() != image.Rect(0, 0, 960, 540) {
		t.Fatal("frame bounds")
	}
}
func TestRejectUntrustedResultAndMissingProvider(t *testing.T) {
	v := make([]float64, 128)
	v[0] = 1
	r := Result{Protocol: 1, Faces: []Face{{Box: []float64{1, 2, 10, 10}, Confidence: .95, Embedding: v}}}
	if e := Validate(r, 32, 32); e != nil {
		t.Fatal(e)
	}
	v[0] = math.NaN()
	if Validate(r, 32, 32) == nil {
		t.Fatal("NaN")
	}
	v[0] = 2
	if Validate(r, 32, 32) == nil {
		t.Fatal("bad normalization")
	}
	p := New(t.TempDir())
	if p.Available() {
		t.Fatal("missing provider available")
	}
	if _, e := p.Analyze(context.Background(), image.NewRGBA(image.Rect(0, 0, 32, 32)), t.TempDir()); e != ErrUnavailable {
		t.Fatal(e)
	}
}
