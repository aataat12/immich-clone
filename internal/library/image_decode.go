package library

import (
	"context"
 "encoding/binary"
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
 if isTIFF {
  if err=validateTIFFBlockBudget(contextImageReader{ctx:ctx,r:r,size:end});err!=nil{return nil,err}
 }
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

// TIFF allocates decompressed tile storage using padded tile dimensions, not
// only image dimensions. A very skinny image can otherwise multiply a nominal
// 40MP image by the permitted 16-pixel padding. Permit an extra 1MP for ordinary
// edge padding while keeping each block finite before calling Decode.
func validateTIFFBlockBudget(r contextImageReader) error {
 var header [8]byte
 if _,err:=r.ReadAt(header[:],0);err!=nil{return err}
 var order binary.ByteOrder=binary.LittleEndian
 if string(header[:2])=="MM" {order=binary.BigEndian}
 offset:=int64(order.Uint32(header[4:]))
 var count [2]byte
 if _,err:=r.ReadAt(count[:],offset);err!=nil{return err}
 entries:=make([]byte,int(order.Uint16(count[:]))*12)
 if _,err:=r.ReadAt(entries,offset+2);err!=nil{return err}
 var width,height uint64
 for i:=0;i<len(entries);i+=12 {
  entry:=entries[i:i+12]
  tag:=order.Uint16(entry)
  if tag!=322 && tag!=323 {continue}
  kind,n:=order.Uint16(entry[2:]),uint64(order.Uint32(entry[4:]))
  var size uint64
  switch kind {case 1:size=1;case 3:size=2;case 4:size=4;default:return errors.New("invalid TIFF tile metadata")}
  if n==0 {continue}
  value:=entry[8:12]
  if n*size>4 {
   var scalar [4]byte
   if _,err:=r.ReadAt(scalar[:size],int64(order.Uint32(value)));err!=nil{return err}
   value=scalar[:]
  }
  var v uint64
  switch size {case 1:v=uint64(value[0]);case 2:v=uint64(order.Uint16(value));case 4:v=uint64(order.Uint32(value))}
  if tag==322 {width=v}else{height=v}
 }
 const maxBlockPixels=maxParentImagePixels+(1<<20)
 if width>0 && height>0 && width>maxBlockPixels/height {return errors.New("TIFF padded tile exceeds decoded block budget")}
 return nil
}
