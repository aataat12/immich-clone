package library

import (
	"bytes"
	"encoding/binary"
	"testing"
)

// OffsetTimeOriginal is deliberately outside the current extraction subset.
// Its presence must not silently reinterpret the camera's wall-clock date.
func TestCaptureDateDoesNotInventTimezone(t *testing.T) {
	for _, tc := range []struct {
		name, captured, offset, want, warning string
	}{
		{"no offset", "2024:03:02 10:11:12", "", "2024-03-02T10:11:12", ""},
		{"embedded unread offset", "2024:03:02 10:11:12", "+05:30", "2024-03-02T10:11:12", ""},
		{"missing capture date", "", "+05:30", "", ""},
		{"invalid capture date", "2024:02:30 10:11:12", "-06:00", "", "invalid_capture_date"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			b := make([]byte, 128)
			copy(b, "II")
			binary.LittleEndian.PutUint16(b[2:], 42)
			binary.LittleEndian.PutUint32(b[4:], 8)
			binary.LittleEndian.PutUint16(b[8:], 2)
			put := func(pos int, tag uint16, value string, offset uint32) {
				binary.LittleEndian.PutUint16(b[pos:], tag)
				binary.LittleEndian.PutUint16(b[pos+2:], 2)
				binary.LittleEndian.PutUint32(b[pos+4:], uint32(len(value)+1))
				if len(value)+1 > 4 {
					binary.LittleEndian.PutUint32(b[pos+8:], offset)
					copy(b[offset:], value)
				} else {
					copy(b[pos+8:], value)
				}
			}
			put(10, 0x9003, tc.captured, 50)
			put(22, 0x9011, tc.offset, 80)
			payload := append([]byte("Exif\x00\x00"), b...)
			jpeg := append([]byte{255, 216, 255, 225, 0, byte(len(payload) + 2)}, payload...)
			before := append([]byte(nil), jpeg...)
			got := ReadPhotoMetadata(bytes.NewReader(jpeg))
			if got.Captured != tc.want || got.Warning != tc.warning {
				t.Fatalf("capture date or warning: %+v", got)
			}
			if !bytes.Equal(before, jpeg) {
				t.Fatal("metadata extraction changed original bytes")
			}
		})
	}
}
