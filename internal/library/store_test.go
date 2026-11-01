package library

import (
	"archive/zip"
	"bytes"
	"errors"
	"image"
	"image/color"
	"image/png"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"testing"
)

func fixture(t *testing.T) []byte {
	t.Helper()
	im := image.NewRGBA(image.Rect(0, 0, 640, 480))
	im.Set(3, 3, color.RGBA{255, 120, 20, 255})
	var b bytes.Buffer
	if e := png.Encode(&b, im); e != nil {
		t.Fatal(e)
	}
	return b.Bytes()
}
func openTest(t *testing.T) (*Store, string) {
	t.Helper()
	root := filepath.Join(t.TempDir(), "state")
	s, e := Open(root)
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() { s.Close() })
	return s, root
}

type failedReader struct{}

func (failedReader) Read([]byte) (int, error) { return 0, io.ErrUnexpectedEOF }
func TestFailedUploadIsNotCatalogued(t *testing.T) {
	s, root := openTest(t)
	for _, r := range []io.Reader{failedReader{}, bytes.NewReader(nil)} {
		if _, _, e := s.Put("test", r); e == nil {
			t.Fatal("failed input accepted")
		}
	}
	assets, _ := s.List()
	staging, _ := os.ReadDir(filepath.Join(root, "staging"))
	if len(assets) != 0 || len(staging) != 0 {
		t.Fatal("failed upload leaked catalog or staging")
	}
}
func TestConcurrentDuplicateAndCorruption(t *testing.T) {
	s, _ := openTest(t)
	b := fixture(t)
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, _, e := s.Put("../photo.png", bytes.NewReader(b)); e != nil {
				t.Error(e)
			}
		}()
	}
	wg.Wait()
	assets, e := s.List()
	if e != nil || len(assets) != 1 || assets[0].Name != "photo.png" {
		t.Fatal("duplicate publication failed")
	}
	if e = s.Verify(); e != nil {
		t.Fatal(e)
	}
	if e = os.WriteFile(s.Original(assets[0].ID), []byte("corrupt"), 0600); e != nil {
		t.Fatal(e)
	}
	if _, _, e = s.Put("photo", bytes.NewReader(b)); e == nil {
		t.Fatal("corruption hidden by duplicate")
	}
	if e = s.Verify(); e == nil {
		t.Fatal("corruption missed")
	}
}
func TestJobsSurviveReopenAndUnsupportedMedia(t *testing.T) {
	s, root := openTest(t)
	a, _, e := s.Put("test.png", bytes.NewReader(fixture(t)))
	if e != nil {
		t.Fatal(e)
	}
	s.Close()
	if e = os.WriteFile(filepath.Join(root, "staging", "partial.part"), []byte("partial"), 0600); e != nil {
		t.Fatal(e)
	}
	s, e = Open(root)
	if e != nil {
		t.Fatal(e)
	}
	defer s.Close()
	staging, _ := os.ReadDir(filepath.Join(root, "staging"))
	if len(staging) != 0 {
		t.Fatal("staging not recovered")
	}
	if worked, e := s.ProcessOne(); e != nil || !worked {
		t.Fatal("job not recovered", e)
	}
	a, e = s.Get(a.ID)
	if e != nil || a.Preview != "ready" || a.Width != 640 {
		t.Fatal("preview metadata failed", e)
	}
	f, e := os.Open(s.PreviewPath(a.ID))
	if e != nil {
		t.Fatal(e)
	}
	im, e := png.Decode(f)
	f.Close()
	if e != nil || im.Bounds().Dx() != 320 || im.Bounds().Dy() != 240 {
		t.Fatal("thumbnail dimensions failed")
	}
	bad, _, e := s.Put("malformed.png", bytes.NewReader([]byte("not an image")))
	if e != nil {
		t.Fatal(e)
	}
	s.ProcessOne()
	bad, _ = s.Get(bad.ID)
	if bad.Preview != "failed" || bad.Error == "" {
		t.Fatal("unsupported file not surfaced")
	}
	if e = s.Verify(); e != nil {
		t.Fatal(e)
	}
}
func TestBackupRestorationIndependentAndTamper(t *testing.T) {
	s, root := openTest(t)
	a, _, e := s.Put("photo.png", bytes.NewReader(fixture(t)))
	if e != nil {
		t.Fatal(e)
	}
	s.ProcessOne()
	if e = os.WriteFile(filepath.Join(root, "access-code.txt"), []byte("synthetic-test-secret"), 0600); e != nil {
		t.Fatal(e)
	}
	var archive bytes.Buffer
	if e = s.Backup(&archive); e != nil {
		t.Fatal(e)
	}
	path := filepath.Join(t.TempDir(), "backup.zip")
	if e = os.WriteFile(path, archive.Bytes(), 0600); e != nil {
		t.Fatal(e)
	}
	destination := filepath.Join(t.TempDir(), "restored")
	if e = Restore(path, destination); e != nil {
		t.Fatal(e)
	}
	restored, e := Open(destination)
	if e != nil {
		t.Fatal(e)
	}
	defer restored.Close()
	if e = restored.Verify(); e != nil {
		t.Fatal(e)
	}
	copy, _ := restored.Get(a.ID)
	if copy.Preview != "queued" {
		t.Fatal("previews not queued")
	}
	secret, e := os.ReadFile(filepath.Join(destination, "access-code.txt"))
	if e != nil || string(secret) != "synthetic-test-secret" {
		t.Fatal("secret not restored")
	}
	if e = Restore(path, destination); e == nil {
		t.Fatal("restore overwrote destination")
	}
	if e = os.Remove(s.Original(a.ID)); e != nil {
		t.Fatal(e)
	}
	if e = restored.Verify(); e != nil {
		t.Fatal("restore depends on source")
	}
	var broken bytes.Buffer
	if e = s.Backup(&broken); e == nil {
		t.Fatal("backup claimed success with missing original")
	}
	// A modified file with the original manifest must be rejected without publication.
	zr, e := zip.NewReader(bytes.NewReader(archive.Bytes()), int64(archive.Len()))
	if e != nil {
		t.Fatal(e)
	}
	var tampered bytes.Buffer
	zw := zip.NewWriter(&tampered)
	for _, entry := range zr.File {
		out, _ := zw.Create(entry.Name)
		in, _ := entry.Open()
		if entry.Name == "originals/"+a.ID {
			out.Write([]byte("tampered"))
		} else {
			io.Copy(out, in)
		}
		in.Close()
	}
	zw.Close()
	if e = os.WriteFile(path, tampered.Bytes(), 0600); e != nil {
		t.Fatal(e)
	}
	target := filepath.Join(t.TempDir(), "rejected")
	if e = Restore(path, target); e == nil {
		t.Fatal("tampered backup accepted")
	}
	if _, e = os.Stat(target); !errors.Is(e, os.ErrNotExist) {
		t.Fatal("invalid restore published")
	}
}
func TestAbruptProcessRecovery(t *testing.T) {
	if os.Getenv("IMMICH_CRASH_PROBE") == "1" {
		s, e := Open(os.Getenv("IMMICH_CRASH_ROOT"))
		if e != nil {
			os.Exit(2)
		}
		if _, _, e = s.Put("durable.bin", bytes.NewReader([]byte("synthetic committed original"))); e != nil {
			os.Exit(3)
		}
		os.Exit(19)
	}
	root := filepath.Join(t.TempDir(), "state")
	exe, e := os.Executable()
	if e != nil {
		t.Fatal(e)
	}
	cmd := exec.Command(exe, "-test.run=^TestAbruptProcessRecovery$")
	cmd.Env = append(os.Environ(), "IMMICH_CRASH_PROBE=1", "IMMICH_CRASH_ROOT="+root)
	e = cmd.Run()
	var exit *exec.ExitError
	if !errors.As(e, &exit) || exit.ExitCode() != 19 {
		t.Fatal("child did not reach abrupt-exit point", e)
	}
	s, e := Open(root)
	if e != nil {
		t.Fatal(e)
	}
	defer s.Close()
	if e = s.Verify(); e != nil {
		t.Fatal(e)
	}
	assets, e := s.List()
	if e != nil || len(assets) != 1 || assets[0].Preview != "queued" {
		t.Fatal("committed job lost on abrupt exit")
	}
}
func TestTraversalAndUnknownSchema(t *testing.T) {
	s, _ := openTest(t)
	if _, e := s.Get("../../outside"); !errors.Is(e, os.ErrNotExist) {
		t.Fatal("invalid ID accepted")
	}
	s.Close()
	if _, _, e := s.Put("test", bytes.NewReader([]byte("test"))); e == nil {
		t.Fatal("closed storage accepted write")
	}
}
