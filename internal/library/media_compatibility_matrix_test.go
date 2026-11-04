package library

import (
	"context"
	"encoding/json"
	"image"
	"image/color"
	_ "image/png"
	"os"
	"path/filepath"
	"testing"
)

type compatibilityFixture struct {
	Name      string         `json:"name"`
	SHA256    string         `json:"sha256"`
	Width     int            `json:"width"`
	Height    int            `json:"height"`
	Photo     *PhotoMetadata `json:"photo"`
	Pixels    [][4]int       `json:"pixels"`
	Tolerance int            `json:"tolerance"`
	Camera    bool           `json:"camera"`
}

// Independent Pillow expectations cover actual production import/preview/restore.
// Camera originals are reduced public samples with retained real camera EXIF.
func TestMediaCompatibilityImageMatrix(t *testing.T) {
	fixtures := os.Getenv("IMMICH_COMPAT_FIXTURES")
	if fixtures == "" {
		t.Skip("requires independent media compatibility fixture matrix")
	}
	raw, e := os.ReadFile(filepath.Join(fixtures, "expected.json"))
	if e != nil {
		t.Fatal(e)
	}
	var expected []compatibilityFixture
	if e = json.Unmarshal(raw, &expected); e != nil || len(expected) != 24 {
		t.Fatal("independent fixture manifest count", len(expected), e)
	}
	root := t.TempDir()
	source := filepath.Join(root, "source")
	s, e := Open(source)
	if e != nil {
		t.Fatal(e)
	}
	defer func() {
		if s != nil {
			s.Close()
		}
	}()
	ids := map[string]string{}
	cameras := 0
	for _, f := range expected {
		if filepath.Base(f.Name) != f.Name {
			t.Fatal("invalid fixture name")
		}
		file, e := os.Open(filepath.Join(fixtures, f.Name))
		if e != nil {
			t.Fatal(e)
		}
		a, dup, e := s.PutWithPath(f.Name, "Compatibility/"+f.Name, file)
		file.Close()
		if e != nil || dup || a.ID != f.SHA256 {
			t.Fatal("fixture import identity", f.Name, e)
		}
		ids[f.Name] = a.ID
		if f.Camera {
			cameras++
		}
	}
	if cameras != 3 {
		t.Fatal("real camera EXIF subset absent")
	}
	process := func(store *Store) {
		t.Helper()
		for {
			more, e := store.ProcessOneContext(context.Background())
			if e != nil {
				t.Fatal("production preview", e)
			}
			if !more {
				break
			}
		}
	}
	process(s)
	check := func(store *Store) {
		t.Helper()
		for _, f := range expected {
			a, e := store.Get(ids[f.Name])
			if e != nil {
				t.Fatal(e)
			}
			if a.Preview != "ready" || a.Width != f.Width || a.Height != f.Height || len(a.Sources) != 1 || a.Sources[0] != "Compatibility/"+f.Name {
				t.Fatal("preview geometry/source", f.Name, a.Preview, a.Width, a.Height, a.Error)
			}
			if f.Photo != nil && (a.Photo.Orientation != f.Photo.Orientation || a.Photo.Make != f.Photo.Make || a.Photo.Model != f.Photo.Model || a.Photo.Captured != f.Photo.Captured || a.Photo.Warning != "") {
				t.Fatal("camera/EXIF subset mismatch", f.Name)
			}
			file, e := os.Open(store.PreviewPath(a.ID))
			if e != nil {
				t.Fatal(e)
			}
			img, _, e := image.Decode(file)
			file.Close()
			if e != nil {
				t.Fatal(e)
			}
			b := img.Bounds()
			for i, xy := range [][2]int{{1, 1}, {3, 1}, {1, 3}, {3, 3}} {
				pixel := color.NRGBAModel.Convert(img.At(b.Min.X+b.Dx()*xy[0]/4, b.Min.Y+b.Dy()*xy[1]/4)).(color.NRGBA)
				for c, v := range []uint8{pixel.R, pixel.G, pixel.B, pixel.A} {
					delta := int(v) - f.Pixels[i][c]
					if delta < 0 {
						delta = -delta
					}
					if delta > f.Tolerance {
						t.Fatal("independent first-frame/page/orientation pixel mismatch", f.Name, i, c, delta)
					}
				}
			}
			actual, e := fileHash(store.Original(a.ID))
			if e != nil || actual != f.SHA256 {
				t.Fatal("original changed", f.Name, e)
			}
		}
		if e := store.Verify(); e != nil {
			t.Fatal(e)
		}
	}
	check(s)
	archive := filepath.Join(root, "matrix.zip")
	out, e := os.Create(archive)
	if e != nil {
		t.Fatal(e)
	}
	e = s.Backup(out)
	closeErr := out.Close()
	if e != nil || closeErr != nil {
		t.Fatal(e, closeErr)
	}
	if e = s.Close(); e != nil {
		t.Fatal(e)
	}
	s = nil
	if e = os.Rename(source, source+".offline"); e != nil {
		t.Fatal(e)
	}
	restored := filepath.Join(root, "restored")
	if e = Restore(archive, restored); e != nil {
		t.Fatal("independent matrix restore", e)
	}
	r, e := Open(restored)
	if e != nil {
		t.Fatal(e)
	}
	defer r.Close()
	process(r)
	check(r)
	t.Logf("PASS: %d independent image fixtures, three reduced camera EXIF samples, all eight JPEG transforms, DateTimeOriginal priority, progressive/CMYK JPEG, PNG RGB/alpha/palette/gray, BMP, animated GIF first frame, four compressed/multipage TIFF first pages, 12MP JPEG; original identities/aliases and independently restored regenerated previews", len(expected))
}
