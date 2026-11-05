package library

import (
	"context"
	"errors"
	"image"
	"io"
)

const maxParentImageBytes int64 = 64 << 20
const maxParentImagePixels = 40_000_000

// Division avoids overflowing on malicious dimensions before any pixel allocation.
func validImageDimensions(w, h int) bool {
	return w > 0 && h > 0 && w <= maxParentImagePixels/h
}

// Both parent decode callers use the same encoded/pixel bounds and verify that
// the decoded bounds agree with the preflight. These are cooperative limits,
// not the hard memory/CPU boundary provided to native child processes.
func decodeParentImage(ctx context.Context, r io.ReadSeeker) (image.Image, error) {
	if err := ctx.Err(); err != nil { return nil, err }
	end, err := r.Seek(0, io.SeekEnd)
	if err != nil { return nil, err }
	if end <= 0 || end > maxParentImageBytes { return nil, errors.New("image exceeds encoded decode limit; original remains available") }
	if _, err = r.Seek(0, io.SeekStart); err != nil { return nil, err }
	cfg, format, err := image.DecodeConfig(contextReader{ctx, io.LimitReader(r, maxParentImageBytes)})
	if ctx.Err() != nil { return nil, ctx.Err() }
	if err != nil { return nil, errors.New("malformed image") }
	if !validImageDimensions(cfg.Width, cfg.Height) { return nil, errors.New("image exceeds 40 megapixel decode limit") }
	if err = ctx.Err(); err != nil { return nil, err }
	if _, err = r.Seek(0, io.SeekStart); err != nil { return nil, err }
	img, decodedFormat, err := image.Decode(contextReader{ctx, io.LimitReader(r, maxParentImageBytes)})
	if ctx.Err() != nil { return nil, ctx.Err() }
	if err != nil { return nil, errors.New("image decode failed") }
	if img == nil || decodedFormat != format || img.Bounds().Dx() != cfg.Width || img.Bounds().Dy() != cfg.Height { return nil, errors.New("inconsistent decoded image") }
	return img, nil
}
