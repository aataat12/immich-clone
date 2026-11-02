package library

import (
	"bytes"
	"context"
	"crypto/rand"
	"errors"
	bolt "go.etcd.io/bbolt"
	"immichclone/internal/diagnostics"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestActualDiskFull(t *testing.T) {
	volume := os.Getenv("IMMICH_SMALL_VOLUME")
	if volume == "" {
		t.Skip("requires isolated bounded CI volume")
	}
	if e := diagnostics.Init(filepath.Join(t.TempDir(), "logs"), "0.1.3-prototype", "application"); e != nil {
		t.Fatal(e)
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
	m, e := s.CreateImport([]ImportFile{{Name: "bulk-full.bin", Size: 8 << 20}})
	if e != nil {
		t.Fatal(e)
	}
	if e = s.ControlImport(m.ID, "start"); e != nil {
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
	if _, e = s.PutImport(context.Background(), m.ID, 0, bytes.NewReader(make([]byte, 8<<20))); e == nil {
		t.Fatal("full disk import falsely succeeded")
	}
	page, e := s.ImportPage(m.ID, 0)
	if e != nil || page.Session.AcceptedFiles != 0 || page.Entries[0].AssetID != "" {
		t.Fatal("full disk created false receipt", e)
	}
	var support bytes.Buffer
	if e := diagnostics.Export(&support); e != nil || !strings.Contains(support.String(), "disk_full") {
		t.Fatal("actual full-disk failure missing from diagnostic log", e)
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
	if _, e = s.PutImport(context.Background(), m.ID, 0, bytes.NewReader(make([]byte, 8<<20))); e != nil {
		t.Fatal("import did not recover", e)
	}
	if e = s.ControlImport(m.ID, "finish"); e != nil {
		t.Fatal(e)
	}
	page, _ = s.ImportPage(m.ID, 0)
	if page.Session.AcceptedFiles != 1 || page.Session.AcceptedBytes != 8<<20 || s.ImportsRunning() {
		t.Fatal("recovered receipt incorrect")
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

func TestActualBackupDiskFull(t *testing.T) {
	volume := os.Getenv("IMMICH_SMALL_VOLUME")
	if volume == "" {
		t.Skip("requires isolated bounded CI volume")
	}
	root, e := os.MkdirTemp(volume, "backup-disk-full-")
	if e != nil {
		t.Fatal(e)
	}
	defer os.RemoveAll(root)
	s, e := Open(filepath.Join(root, "state"))
	if e != nil {
		t.Fatal(e)
	}
	defer s.Close()
	data := make([]byte, 1<<20)
	if _, e = rand.Read(data); e != nil {
		t.Fatal(e)
	}
	_, _, e = s.Put("synthetic-random.bin", bytes.NewReader(data))
	if e != nil {
		t.Fatal(e)
	}
	os.MkdirAll(filepath.Join(s.root, "backups"), 0700)
	path := filepath.Join(s.root, "backups", "fault.part")
	plan, e := s.BackupPlan(context.Background())
	if e != nil || !plan.Fits {
		t.Fatal("disposable volume lacks initial backup reserve", e)
	}
	filler := filepath.Join(root, "filler.bin")
	filled := false
	fillErr := error(nil)
	e = s.StagedBackup(context.Background(), path, func(p BackupProgress) {
		if filled || p.Phase != "copying" || p.Bytes == 0 {
			return
		}
		filled = true
		f, err := os.Create(filler)
		if err != nil {
			fillErr = err
			return
		}
		defer f.Close()
		block := make([]byte, 64<<10)
		full := false
		for total := 0; total < 256<<20; total += len(block) {
			if _, err = f.Write(block); err != nil {
				full = true
				break
			}
		}
		if !full {
			fillErr = errors.New("bounded volume did not fill")
		}
	})
	if fillErr != nil {
		t.Fatal(fillErr)
	}
	if !filled || e == nil {
		t.Fatal("space disappearing after preflight falsely succeeded")
	}
	if _, e = os.Stat(path); !os.IsNotExist(e) {
		t.Fatal("failed backup partial retained")
	}
	if e = s.Verify(); e != nil {
		t.Fatal(e)
	}
	plan, e = s.BackupPlan(context.Background())
	if e != nil || plan.Fits {
		t.Fatal("full volume preflight accepted", e)
	}
	if e = s.StagedBackup(context.Background(), path, nil); !errors.Is(e, ErrBackupCapacity) {
		t.Fatal("capacity rejection", e)
	}
	if e = os.Remove(filler); e != nil {
		t.Fatal(e)
	}
	if e = s.StagedBackup(context.Background(), path, nil); e != nil {
		t.Fatal("space recovery", e)
	}
	if e = VerifyBackup(context.Background(), path, nil); e != nil {
		t.Fatal(e)
	}
	if e = s.Verify(); e != nil {
		t.Fatal(e)
	}
	t.Log("actual post-preflight ENOSPC, partial cleanup, insufficient-space rejection, retry and archive read-back passed")
}
