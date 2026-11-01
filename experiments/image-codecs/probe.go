package main

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	_ "golang.org/x/image/bmp"
	_ "golang.org/x/image/tiff"
	"image"
	_ "image/gif"
	"image/png"
	"os"
	"path/filepath"
)

func decode(b []byte) (image.Image, error) {
	c, _, err := image.DecodeConfig(bytes.NewReader(b))
	if err != nil {
		return nil, err
	}
	if c.Width <= 0 || c.Height <= 0 || int64(c.Width)*int64(c.Height) > 40_000_000 {
		return nil, fmt.Errorf("dimensions rejected")
	}
	returnImage, _, err := image.Decode(bytes.NewReader(b))
	return returnImage, err
}
func run(root string) error {
	for _, ext := range []string{"bmp", "tiff", "gif"} {
		p := filepath.Join(root, "synthetic."+ext)
		b, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		before := sha256.Sum256(b)
		img, err := decode(b)
		if err != nil {
			return fmt.Errorf("%s decode: %w", ext, err)
		}
		if img.Bounds().Dx() != 32 || img.Bounds().Dy() != 24 {
			return fmt.Errorf("%s dimensions", ext)
		}
		for _, point := range []image.Point{{1, 1}, {30, 22}} {
			r, g, bl, _ := img.At(point.X, point.Y).RGBA()
			if r < 60000 || g > 4000 || bl > 4000 {
				return fmt.Errorf("%s decoded pixel", ext)
			}
		}
		var out bytes.Buffer
		if err = png.Encode(&out, img); err != nil {
			return err
		}
		roundtrip, err := png.Decode(&out)
		if err != nil || roundtrip.Bounds() != img.Bounds() {
			return fmt.Errorf("%s PNG roundtrip", ext)
		}
		if _, err = decode(b[:len(b)/2]); err == nil {
			return fmt.Errorf("%s truncated input accepted", ext)
		}
		after, err := os.ReadFile(p)
		if err != nil || before != sha256.Sum256(after) {
			return fmt.Errorf("%s original modified", ext)
		}
	}
	if _, err := decode([]byte("malformed")); err == nil {
		return fmt.Errorf("malformed accepted")
	}
	return nil
}
