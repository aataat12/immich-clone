package library

import (
	"context"
	"errors"
	"image"
	"io"
 "golang.org/x/image/tiff"
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
 return decodeParentImageLimit(ctx,r,maxParentImageBytes)
}
func decodeParentImageLimit(ctx context.Context, r io.ReadSeeker, encodedLimit int64) (image.Image, error) {
	if err := ctx.Err(); err != nil { return nil, err }
	end, err := r.Seek(0, io.SeekEnd)
	if err != nil { return nil, err }
	if end <= 0 || end > encodedLimit { return nil, errors.New("image exceeds encoded decode limit; original remains available") }
	if _, err = r.Seek(0, io.SeekStart); err != nil { return nil, err }
	var signature [4]byte
 if _,err=io.ReadFull(contextReader{ctx,r},signature[:]);err!=nil { return nil,err }
 if _,err=r.Seek(0,io.SeekStart);err!=nil {return nil,err}
 isTIFF:=string(signature[:])=="II*\x00" || string(signature[:])=="MM\x00*"
 var cfg image.Config
 var format string
 if isTIFF {
  cfg,err=tiff.DecodeConfig(contextImageReader{ctx:ctx,r:r,size:end})
  format="tiff"
 } else {
  cfg,format,err=image.DecodeConfig(contextReader{ctx,io.LimitReader(r,encodedLimit)})
 }
	if ctx.Err() != nil { return nil, ctx.Err() }
	if err != nil { return nil, errors.New("malformed image") }
	if !validImageDimensions(cfg.Width, cfg.Height) { return nil, errors.New("image exceeds 40 megapixel decode limit") }
	if err = ctx.Err(); err != nil { return nil, err }
	if _, err = r.Seek(0, io.SeekStart); err != nil { return nil, err }
	var img image.Image
 decodedFormat:=format
 if isTIFF {
  img,err=tiff.Decode(contextImageReader{ctx:ctx,r:r,size:end})
 } else {
  img,decodedFormat,err=image.Decode(contextReader{ctx,io.LimitReader(r,encodedLimit)})
 }
	if ctx.Err() != nil { return nil, ctx.Err() }
	if err != nil { return nil, errors.New("image decode failed") }
	if img == nil || decodedFormat != format || img.Bounds().Dx() != cfg.Width || img.Bounds().Dy() != cfg.Height { return nil, errors.New("inconsistent decoded image") }
	return img, nil
}

// TIFF makes random reads. Preserve cancellation at those reads, including
// repeated compressed strips, instead of allowing an internal cached reader to
// replay encoded bytes without consulting the context. The decoder is serial.
type contextImageReader struct {
 ctx context.Context
 r io.ReadSeeker
 size int64
}
func (r contextImageReader) Read(p []byte)(int,error) {
 return contextReader{r.ctx,r.r}.Read(p)
}
func (r contextImageReader) ReadAt(p []byte,offset int64)(int,error) {
 if err:=r.ctx.Err();err!=nil{return 0,err}
 if offset<0 || offset>r.size || int64(len(p))>r.size-offset {return 0,io.ErrUnexpectedEOF}
 if _,err:=r.r.Seek(offset,io.SeekStart);err!=nil{return 0,err}
 return io.ReadFull(contextReader{r.ctx,r.r},p)
}
