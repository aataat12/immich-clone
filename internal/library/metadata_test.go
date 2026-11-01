package library

import (
	"bytes"
	"encoding/binary"
	"image"
	"image/color"
	"image/jpeg"
	"os"
	"path/filepath"
	"testing"
)

func exifJPEG(t *testing.T, orientation uint16) []byte {
	t.Helper()
	b := make([]byte, 100)
	copy(b, "II")
	binary.LittleEndian.PutUint16(b[2:], 42)
	binary.LittleEndian.PutUint32(b[4:], 8)
	binary.LittleEndian.PutUint16(b[8:], 3)
	put := func(pos int, tag, kind uint16, n, value uint32) {
		binary.LittleEndian.PutUint16(b[pos:], tag)
		binary.LittleEndian.PutUint16(b[pos+2:], kind)
		binary.LittleEndian.PutUint32(b[pos+4:], n)
		binary.LittleEndian.PutUint32(b[pos+8:], value)
	}
	put(10, 0x112, 3, 1, uint32(orientation))
	put(22, 0x10f, 2, 6, 50)
	copy(b[50:], "Demo!\x00")
	put(34, 0x132, 2, 20, 60)
	copy(b[60:], "2024:03:02 10:11:12\x00")
	var jpg bytes.Buffer
	if e := jpeg.Encode(&jpg, image.NewRGBA(image.Rect(0, 0, 20, 10)), nil); e != nil {
		t.Fatal(e)
	}
	payload := append([]byte("Exif\x00\x00"), b...)
	out := []byte{255, 216, 255, 225, 0, byte(len(payload) + 2)}
	out = append(out, payload...)
	return append(out, jpg.Bytes()[2:]...)
}
func TestMetadataRealDecodeAndImmutableOriginal(t *testing.T) {
	s, _ := openTest(t)
	data := exifJPEG(t, 6)
	m := ReadPhotoMetadata(bytes.NewReader(data))
	if m.Orientation != 6 || m.Captured != "2024-03-02T10:11:12" || m.Make != "Demo!" || m.Warning != "" {
		t.Fatalf("unexpected synthetic metadata: %+v", m)
	}
	a, _, e := s.Put("rotated.jpg", bytes.NewReader(data))
	if e != nil {
		t.Fatal(e)
	}
	if _, e = s.ProcessOne(); e != nil {
		t.Fatal(e)
	}
	a, _ = s.Get(a.ID)
	if a.Width != 10 || a.Height != 20 || a.Photo.Orientation != 6 {
		t.Fatal("oriented preview metadata wrong")
	}
	original, _ := os.ReadFile(s.Original(a.ID))
	if !bytes.Equal(original, data) {
		t.Fatal("original modified")
	}
	if e = s.Verify(); e != nil {
		t.Fatal(e)
	}
}
func TestAllEightOrientationTransforms(t *testing.T) {
	im := image.NewRGBA(image.Rect(0, 0, 3, 2))
	for y := 0; y < 2; y++ {
		for x := 0; x < 3; x++ {
			im.Set(x, y, color.RGBA{uint8(y*3 + x + 1), 0, 0, 255})
		}
	}
	expected := [][]uint8{{1, 2, 3, 4, 5, 6}, {3, 2, 1, 6, 5, 4}, {6, 5, 4, 3, 2, 1}, {4, 5, 6, 1, 2, 3}, {1, 4, 2, 5, 3, 6}, {4, 1, 5, 2, 6, 3}, {6, 3, 5, 2, 4, 1}, {3, 6, 2, 5, 1, 4}}
	for o := 1; o <= 8; o++ {
		out := oriented(im, o)
		i := 0
		for y := 0; y < out.Bounds().Dy(); y++ {
			for x := 0; x < out.Bounds().Dx(); x++ {
				r, _, _, _ := out.At(x, y).RGBA()
				if uint8(r>>8) != expected[o-1][i] {
					t.Fatalf("orientation %d", o)
				}
				i++
			}
		}
	}
}
func TestMalformedMetadataBounded(t *testing.T) {
	for _, b := range [][]byte{nil, []byte("bad"), {255, 216, 255, 225, 255, 255}, make([]byte, 1<<20)} {
		ReadPhotoMetadata(bytes.NewReader(b))
	}
	if m := readTIFF([]byte("II\x2a\x00\xff\xff\xff\xff")); m.Warning != "invalid_exif" {
		t.Fatal("offset bounds missed")
	}
}
func TestRelativePathsAndMetadataRebuild(t *testing.T) {
	s, root := openTest(t)
	data := exifJPEG(t, 8)
	a, _, e := s.PutWithPath("photo.jpg", "Trip/Nested/photo.jpg", bytes.NewReader(data))
	if e != nil {
		t.Fatal(e)
	}
	if _, dup, e := s.PutWithPath("copy.jpg", "Other/copy.jpg", bytes.NewReader(data)); e != nil || !dup {
		t.Fatal("duplicate aliases failed", e)
	}
	for _, path := range []string{"../outside", "/absolute", "C:/absolute", "a/../../b", "a//b", "a/./b"} {
		if _, _, e = s.PutWithPath("bad", path, bytes.NewReader(data)); e == nil {
			t.Fatal("unsafe relative path accepted")
		}
	}
	s.ProcessOne()
	a, _ = s.Get(a.ID)
	if len(a.Sources) != 2 {
		t.Fatal("relative aliases lost")
	}
	a.PreviewRevision = 0
	a.Photo = PhotoMetadata{}
	s.save(a)
	s.Close()
	s, e = Open(root)
	if e != nil {
		t.Fatal(e)
	}
	defer s.Close()
	a, _ = s.Get(a.ID)
	if a.Preview != "queued" {
		t.Fatal("legacy previews not queued")
	}
	s.ProcessOne()
	a, _ = s.Get(a.ID)
	if a.Photo.Orientation != 8 || a.Photo.Captured == "" || len(a.Sources) != 2 {
		t.Fatal("metadata rebuild/paths failed")
	}
	if e = s.Verify(); e != nil {
		t.Fatal(e)
	}
}
func TestMetadataFixtureEmitter(t *testing.T) {
	if path := os.Getenv("IMMICH_METADATA_FIXTURE"); path != "" {
		if e := os.WriteFile(path, exifJPEG(t, 6), 0600); e != nil {
			t.Fatal(e)
		}
	}
}
func TestBigEndianAndCycles(t *testing.T) {
	b := make([]byte, 26)
	copy(b, "MM")
	binary.BigEndian.PutUint16(b[2:], 42)
	binary.BigEndian.PutUint32(b[4:], 8)
	binary.BigEndian.PutUint16(b[8:], 1)
	binary.BigEndian.PutUint16(b[10:], 0x112)
	binary.BigEndian.PutUint16(b[12:], 3)
	binary.BigEndian.PutUint32(b[14:], 1)
	binary.BigEndian.PutUint16(b[18:], 8)
	if m := readTIFF(b); m.Orientation != 8 || m.Warning != "" {
		t.Fatal("big endian parsing failed")
	}
	binary.BigEndian.PutUint16(b[10:], 0x8769)
	binary.BigEndian.PutUint16(b[12:], 4)
	binary.BigEndian.PutUint32(b[18:], 8)
	if m := readTIFF(b); m.Warning != "invalid_exif" {
		t.Fatal("cyclic IFD not rejected")
	}
}

func TestPhotoMetadataAndSourcesSurviveBackup(t *testing.T) {
	s, _ := openTest(t)
	a, _, e := s.PutWithPath("photo.jpg", "Trip/photo.jpg", bytes.NewReader(exifJPEG(t, 6)))
	if e != nil {
		t.Fatal(e)
	}
	s.ProcessOne()
	var zip bytes.Buffer
	if e = s.Backup(&zip); e != nil {
		t.Fatal(e)
	}
	archive := filepath.Join(t.TempDir(), "backup.zip")
	os.WriteFile(archive, zip.Bytes(), 0600)
	target := filepath.Join(t.TempDir(), "restored")
	if e = Restore(archive, target); e != nil {
		t.Fatal(e)
	}
	restored, e := Open(target)
	if e != nil {
		t.Fatal(e)
	}
	defer restored.Close()
	copy, e := restored.Get(a.ID)
	if e != nil || copy.Photo.Captured != "2024-03-02T10:11:12" || len(copy.Sources) != 1 || copy.Sources[0] != "Trip/photo.jpg" {
		t.Fatal("metadata or source alias lost during restore")
	}
	if e = restored.Verify(); e != nil {
		t.Fatal(e)
	}
}
