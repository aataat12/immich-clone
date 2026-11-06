package library

// Bounded JPEG APP1/TIFF reader for a deliberately narrow EXIF subset. No
// external/native dependency, arbitrary directory traversal or maker-note decode.
import (
	"bytes"
	"context"
	"encoding/binary"
	"image"
	"image/color"
	"io"
	"strings"
	"time"
)

type PhotoMetadata struct {
	Orientation int    `json:"orientation,omitempty"`
	Captured    string `json:"captured,omitempty"`
	Make        string `json:"make,omitempty"`
	Model       string `json:"model,omitempty"`
	Warning     string `json:"warning,omitempty"`
}

func ReadPhotoMetadata(r io.Reader) PhotoMetadata {
	m := PhotoMetadata{Orientation: 1}
	b, e := io.ReadAll(io.LimitReader(r, 1<<20))
	if e != nil {
		m.Warning = "metadata_read_failed"
		return m
	}
	if len(b) < 2 || b[0] != 255 || b[1] != 216 {
		return m
	}
	for p := 2; p+4 <= len(b); {
		if b[p] != 255 {
			break
		}
		tag := b[p+1]
		p += 2
		if tag == 0xda || tag == 0xd9 {
			break
		}
		if tag == 0xff {
			p--
			continue
		}
		if tag == 0x01 || (tag >= 0xd0 && tag <= 0xd7) {
			continue
		}
		n := int(binary.BigEndian.Uint16(b[p : p+2]))
		if n < 2 || p+n > len(b) {
			m.Warning = "metadata_truncated"
			return m
		}
		if tag == 0xe1 && n >= 8 && bytes.Equal(b[p+2:p+8], []byte("Exif\x00\x00")) {
			return readTIFF(b[p+8 : p+n])
		}
		p += n
	}
	return m
}
func readTIFF(b []byte) PhotoMetadata {
	m := PhotoMetadata{Orientation: 1}
	bad := func() PhotoMetadata { return PhotoMetadata{Orientation: 1, Warning: "invalid_exif"} }
	if len(b) < 8 {
		return bad()
	}
	var order binary.ByteOrder
	switch string(b[:2]) {
	case "II":
		order = binary.LittleEndian
	case "MM":
		order = binary.BigEndian
	default:
		return bad()
	}
	if order.Uint16(b[2:4]) != 42 {
		return bad()
	}
	seen := map[uint32]bool{}
	var original, fallback string
	var directory func(uint32, int) bool
	directory = func(offset uint32, depth int) bool {
		if depth > 1 || seen[offset] {
			return false
		}
		seen[offset] = true
		start := uint64(offset)
		if start+2 > uint64(len(b)) {
			return false
		}
		count := uint64(order.Uint16(b[start : start+2]))
		if count > 256 || start+2+count*12 > uint64(len(b)) {
			return false
		}
		for i := uint64(0); i < count; i++ {
			entry := b[start+2+i*12 : start+14+i*12]
			tag := order.Uint16(entry)
			kind := order.Uint16(entry[2:])
			n := uint64(order.Uint32(entry[4:]))
			value := entry[8:12]
			if tag == 0x8769 && kind == 4 && n == 1 {
				if !directory(order.Uint32(value), depth+1) {
					return false
				}
				continue
			}
			if tag == 0x112 && kind == 3 && n == 1 {
				v := int(order.Uint16(value))
				if v >= 1 && v <= 8 {
					m.Orientation = v
				} else {
					m.Warning = "invalid_orientation"
				}
				continue
			}
			if kind != 2 || n == 0 || n > 256 {
				continue
			}
			var data []byte
			if n <= 4 {
				data = value[:n]
			} else {
				off := uint64(order.Uint32(value))
				if off+n > uint64(len(b)) {
					return false
				}
				data = b[off : off+n]
			}
			text := strings.TrimRight(string(data), "\x00 ")
			for _, c := range text {
				if c < 32 || c > 126 {
					text = ""
					break
				}
			}
			switch tag {
			case 0x10f:
				m.Make = text
			case 0x110:
				m.Model = text
			case 0x132:
				fallback = text
			case 0x9003:
				original = text
			}
		}
		return true
	}
	if !directory(order.Uint32(b[4:]), 0) {
		return bad()
	}
	if original == "" {
		original = fallback
	}
	if original != "" {
		if t, e := time.Parse("2006:01:02 15:04:05", original); e == nil {
			m.Captured = t.Format("2006-01-02T15:04:05")
		} else {
			m.Warning = "invalid_capture_date"
		}
	}
	// EXIF wall-clock time is kept without inventing a timezone or UTC offset.
	return m
}
func oriented(img image.Image, orientation int) image.Image {
	out, _ := orientedContext(context.Background(), img, orientation)
	return out
}
func orientedContext(ctx context.Context, img image.Image, orientation int) (image.Image, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if orientation <= 1 || orientation > 8 {
		return img, nil
	}
	bounds := img.Bounds()
	w, h := bounds.Dx(), bounds.Dy()
	ow, oh := w, h
	if orientation >= 5 {
		ow, oh = h, w
	}
	out := image.NewRGBA(image.Rect(0, 0, ow, oh))
	rgba64, fast := img.(image.RGBA64Image)
	for y := 0; y < h; y++ {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		for x := 0; x < w; x++ {
			dx, dy := x, y
			switch orientation {
			case 2:
				dx = w - 1 - x
			case 3:
				dx, dy = w-1-x, h-1-y
			case 4:
				dy = h - 1 - y
			case 5:
				dx, dy = y, x
			case 6:
				dx, dy = h-1-y, x
			case 7:
				dx, dy = h-1-y, w-1-x
			case 8:
				dx, dy = y, w-1-x
			}
			if fast {
				c := rgba64.RGBA64At(bounds.Min.X+x, bounds.Min.Y+y)
				out.SetRGBA(dx, dy, color.RGBA{R: uint8(c.R >> 8), G: uint8(c.G >> 8), B: uint8(c.B >> 8), A: uint8(c.A >> 8)})
			} else {
				r, g, b, a := img.At(bounds.Min.X+x, bounds.Min.Y+y).RGBA()
				out.SetRGBA(dx, dy, color.RGBA{R: uint8(r >> 8), G: uint8(g >> 8), B: uint8(b >> 8), A: uint8(a >> 8)})
			}
		}
	}
	return out, nil
}
