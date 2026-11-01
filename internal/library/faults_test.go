package library

import (
	"bytes"
	bolt "go.etcd.io/bbolt"
	"os"
	"path/filepath"
	"testing"
)

func TestActualDiskFull(t *testing.T) {
	volume := os.Getenv("IMMICH_SMALL_VOLUME")
	if volume == "" {
		t.Skip("requires isolated bounded CI volume")
	}
	root, e := os.MkdirTemp(volume, "immich-disk-full-")
	if e != nil {
		t.Fatal(e)
	}
	defer os.RemoveAll(root)
	state := filepath.Join(root, "state")
	s, e := Open(state)
	if e != nil {
		t.Fatal(e)
	}
	defer s.Close()
	a, _, e := s.Put("original.bin", bytes.NewReader([]byte("preserved before full disk")))
	if e != nil {
		t.Fatal(e)
	}
	fillerPath := filepath.Join(root, "filler.bin")
	f, e := os.Create(fillerPath)
	if e != nil {
		t.Fatal(e)
	}
	block := make([]byte, 64<<10)
	full := false
	// Hard cap even if a test volume was misconfigured; never consume unbounded disk.
	for total := 0; total < 128<<20; total += len(block) {
		if _, e = f.Write(block); e != nil {
			full = true
			break
		}
	}
	f.Close()
	if !full {
		os.Remove(fillerPath)
		t.Fatal("test volume did not fill within 128MiB cap")
	}
	if _, _, e = s.Put("should-fail.bin", bytes.NewReader(make([]byte, 8<<20))); e == nil {
		t.Fatal("full disk upload falsely succeeded")
	}
	assets, e := s.List()
	if e != nil || len(assets) != 1 || assets[0].ID != a.ID {
		t.Fatal("full disk changed catalog")
	}
	if e = s.Verify(); e != nil {
		t.Fatal(e)
	}
	if e = os.Remove(fillerPath); e != nil {
		t.Fatal(e)
	}
	if _, _, e = s.Put("after-recovery.bin", bytes.NewReader(make([]byte, 8<<20))); e != nil {
		t.Fatal("storage did not recover after space was freed", e)
	}
	if e = s.Verify(); e != nil {
		t.Fatal(e)
	}
}
func TestMissingOriginalDirectoryAndRecovery(t *testing.T) {
	s, root := openTest(t)
	a, _, e := s.Put("image.png", bytes.NewReader(fixture(t)))
	if e != nil {
		t.Fatal(e)
	}
	dir := filepath.Join(root, "originals")
	offline := filepath.Join(root, "offline-originals")
	if e = os.Rename(dir, offline); e != nil {
		t.Fatal(e)
	}
	if _, e = s.ProcessOne(); e != nil {
		t.Fatal(e)
	}
	failed, e := s.Get(a.ID)
	if e != nil || failed.Preview != "failed" {
		t.Fatal("missing original not surfaced")
	}
	assets, e := s.List()
	if e != nil || len(assets) != 1 {
		t.Fatal("unavailable original was removed from catalog")
	}
	if _, _, e = s.Put("new.bin", bytes.NewReader([]byte("should not publish"))); e == nil {
		t.Fatal("unavailable originals path falsely succeeded")
	}
	if e = os.Rename(offline, dir); e != nil {
		t.Fatal(e)
	}
	if e = s.Retry(a.ID); e != nil {
		t.Fatal(e)
	}
	if _, e = s.ProcessOne(); e != nil {
		t.Fatal(e)
	}
	recovered, _ := s.Get(a.ID)
	if recovered.Preview != "ready" {
		t.Fatal("retry after storage recovery failed")
	}
}
func TestRefusesUnrelatedFolderAndUnknownSchema(t *testing.T) {
	root := t.TempDir()
	sentinel := filepath.Join(root, "unrelated.txt")
	if e := os.WriteFile(sentinel, []byte("leave untouched"), 0600); e != nil {
		t.Fatal(e)
	}
	if s, e := Open(root); e == nil {
		s.Close()
		t.Fatal("unrelated directory accepted")
	}
	b, _ := os.ReadFile(sentinel)
	if string(b) != "leave untouched" {
		t.Fatal("unrelated file changed")
	}
	s, state := openTest(t)
	e := s.db.Update(func(tx *bolt.Tx) error { return tx.Bucket([]byte("meta")).Put([]byte("schema"), []byte("99")) })
	if e != nil {
		t.Fatal(e)
	}
	s.Close()
	if reopened, e := Open(state); e == nil {
		reopened.Close()
		t.Fatal("unknown schema accepted")
	}
}
