package main

import (
	"golang.org/x/image/bmp"
	"golang.org/x/image/tiff"
	"image"
	"image/color"
	"image/gif"
	"os"
	"path/filepath"
	"testing"
)

func TestActualCodecs(t *testing.T) {
	root := t.TempDir()
	img := image.NewRGBA(image.Rect(0, 0, 32, 24))
	for y := 0; y < 24; y++ {
		for x := 0; x < 32; x++ {
			img.Set(x, y, color.RGBA{255, 0, 0, 255})
		}
	}
	for _, ext := range []string{"bmp", "tiff", "gif"} {
		f, e := os.Create(filepath.Join(root, "synthetic."+ext))
		if e != nil {
			t.Fatal(e)
		}
		switch ext {
		case "bmp":
			e = bmp.Encode(f, img)
		case "tiff":
			e = tiff.Encode(f, img, &tiff.Options{Compression: tiff.Deflate, Predictor: true})
		case "gif":
			e = gif.Encode(f, img, nil)
		}
		f.Close()
		if e != nil {
			t.Fatal(e)
		}
	}
	if e := run(root); e != nil {
		t.Fatal(e)
	}
}
