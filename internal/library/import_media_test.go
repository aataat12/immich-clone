package library

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"golang.org/x/image/bmp"
	"golang.org/x/image/tiff"
	"image"
	"image/color"
	"image/gif"
	"image/jpeg"
	"image/png"
	"os"
	"path/filepath"
	"testing"
)

func TestActualMixedMediaImportRestartDrainAndRestore(t *testing.T) {
	s, root := openTest(t)
	var files []ImportFile
	var payloads [][]byte
	for i := 0; i < 600; i++ {
		img := image.NewRGBA(image.Rect(0, 0, 64, 48))
		for y := 0; y < 48; y++ {
			for x := 0; x < 64; x++ {
				img.Set(x, y, color.RGBA{uint8(i * 2), uint8(x * 3), uint8(y*4 + i), 255})
			}
		}
		var b bytes.Buffer
		ext := ""
		var e error
		switch i % 5 {
		case 0:
			ext = "png"
			e = png.Encode(&b, img)
		case 1:
			ext = "jpg"
			e = jpeg.Encode(&b, img, nil)
		case 2:
			ext = "gif"
			e = gif.Encode(&b, img, nil)
		case 3:
			ext = "bmp"
			e = bmp.Encode(&b, img)
		case 4:
			ext = "tiff"
			e = tiff.Encode(&b, img, nil)
		}
		if e != nil {
			t.Fatal(e)
		}
		data := append([]byte(nil), b.Bytes()...)
		payloads = append(payloads, data)
		name := fmt.Sprintf("synthetic-%03d.%s", i, ext)
		files = append(files, ImportFile{Name: name, Relative: "Synthetic/" + name, Size: int64(len(data))})
	}
	m, e := s.CreateImport(files)
	if e != nil {
		t.Fatal(e)
	}
	s.ControlImport(m.ID, "start")
	for i := 0; i < 37; i++ {
		if _, e = s.PutImport(context.Background(), m.ID, i, bytes.NewReader(payloads[i])); e != nil {
			t.Fatal(i, e)
		}
	}
	s.Close()
	s, e = Open(root)
	if e != nil {
		t.Fatal(e)
	}
	defer s.Close()
	p, _ := s.ImportPage(m.ID, 0)
	if p.Session.Status != "paused" || p.Session.AcceptedFiles != 37 || s.ImportsRunning() {
		t.Fatal("restart receipt recovery")
	}
	s.ControlImport(m.ID, "start")
	// Verify replay without double counting, then copy the remaining actual encoded media.
	for i, data := range payloads {
		entry, e := s.PutImport(context.Background(), m.ID, i, bytes.NewReader(data))
		if e != nil {
			t.Fatal(i, e)
		}
		h := sha256.Sum256(data)
		if entry.AssetID != hex.EncodeToString(h[:]) {
			t.Fatal("receipt hash mismatch")
		}
	}
	if e = s.ControlImport(m.ID, "finish"); e != nil {
		t.Fatal(e)
	}
	p, _ = s.ImportPage(m.ID, 0)
	if p.Session.AcceptedFiles != 600 || p.Session.NewFiles != 600 || p.Session.Duplicates != 0 {
		t.Fatal(p.Session)
	}
	jobs := 0
	for {
		ok, e := s.ProcessOne()
		if e != nil {
			t.Fatal(e)
		}
		if !ok {
			break
		}
		jobs++
	}
	if jobs != 600 {
		t.Fatal("missing preview jobs", jobs)
	}
	assets, _ := s.List()
	for _, a := range assets {
		if a.Preview != "ready" || a.Width != 64 || a.Height != 48 {
			t.Fatal("actual decode failed", a)
		}
	}
	var archive bytes.Buffer
	if e = s.Backup(&archive); e != nil {
		t.Fatal(e)
	}
	zipPath := filepath.Join(t.TempDir(), "mixed.zip")
	os.WriteFile(zipPath, archive.Bytes(), 0600)
	target := filepath.Join(t.TempDir(), "restored")
	if e = Restore(zipPath, target); e != nil {
		t.Fatal(e)
	}
	s.Close()
	if e = os.Rename(root, root+".offline"); e != nil {
		t.Fatal(e)
	}
	restored, e := Open(target)
	if e != nil {
		t.Fatal(e)
	}
	defer restored.Close()
	p, e = restored.ImportPage(m.ID, 0)
	if e != nil || p.Session.AcceptedFiles != 600 || p.Session.Status != "completed" {
		t.Fatal("restored import receipts", e)
	}
	if e = restored.Verify(); e != nil {
		t.Fatal(e)
	}
	t.Log("600 actual JPEG/PNG/BMP/GIF/TIFF originals accepted, interruption/replay reconciled, 600 native previews drained, receipts and original hashes independently restored")
}
