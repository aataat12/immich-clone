package library

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	bolt "go.etcd.io/bbolt"
)

// This opt-in test fills only the disposable volume mounted by Test-FullDisk.ps1.
func TestActualDeletionDiskFull(t *testing.T) {
	volume := os.Getenv("IMMICH_SMALL_VOLUME")
	if volume == "" {
		t.Skip("requires isolated bounded CI volume")
	}
	root, err := os.MkdirTemp(volume, "deletion-disk-full-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(root)
	s, err := Open(filepath.Join(root, "state"))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	data := fixture(t)
	a, _, err := s.Put("preserved.png", bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}
	if err = s.SetHidden(a.ID, true); err != nil {
		t.Fatal(err)
	}
	// Avoid the normal growth reserve in this fault fixture.
	s.db.AllocSize = os.Getpagesize()
	err = s.db.Update(func(tx *bolt.Tx) error {
		for i := 0; i < 4096; i++ {
			id := fmt.Sprintf("%064x", i+1)
			if e := writeJSON(tx.Bucket([]byte(facesBucket)), id, StoredFace{ID: id, AssetID: a.ID, Box: []float64{0, 0, 1, 1}, Provider: "manual"}); e != nil {
				return e
			}
			if e := writeJSON(tx.Bucket([]byte(labelsBucket)), id, FaceLabel{Name: "Preserved name"}); e != nil {
				return e
			}
		}
		b, e := tx.CreateBucket([]byte("disk_fixture_padding"))
		if e != nil {
			return e
		}
		return b.Put([]byte("retained"), make([]byte, 20<<20))
	})
	if err != nil {
		t.Fatal(err)
	}
	// Windows extends the file to the mapped capacity. Consume that existing
	// reserve too; filling the volume alone does not exhaust catalog write space.
	info, err := os.Stat(s.db.Path())
	if err != nil {
		t.Fatal(err)
	}
	var used int64
	if err = s.db.View(func(tx *bolt.Tx) error { used = tx.Size(); return nil }); err != nil {
		t.Fatal(err)
	}
	if tail := info.Size() - used - (64 << 10); tail > 0 {
		if tail > 32<<20 {
			t.Fatal("unexpected catalog reserve exceeds bounded fixture")
		}
		if err = s.db.Update(func(tx *bolt.Tx) error {
			b, e := tx.CreateBucket([]byte("disk_fixture_reserve"))
			if e != nil {
				return e
			}
			return b.Put([]byte("mapped-reserve"), make([]byte, int(tail)))
		}); err != nil {
			t.Fatal(err)
		}
	}
	info, err = os.Stat(s.db.Path())
	if err != nil {
		t.Fatal(err)
	}
	if err = s.db.View(func(tx *bolt.Tx) error { used = tx.Size(); return nil }); err != nil {
		t.Fatal(err)
	}
	stats := s.db.Stats()
	available := info.Size() - used + int64(stats.FreePageN+stats.PendingPageN)*int64(s.db.Info().PageSize)
	t.Logf("Catalog reusable reserve before fill: %d bytes", available)
	if available >= 200<<10 {
		t.Fatal("fixture retained enough reserved space for deletion journal")
	}
	before := snapshotReview(t, s, a.ID)
	filler := filepath.Join(root, "filler.bin")
	f, err := os.Create(filler)
	if err != nil {
		t.Fatal(err)
	}
	block := make([]byte, 4096)
	full := false
	for total := 0; total < 128<<20; total += len(block) {
		if _, err = f.Write(block); err != nil {
			full = true
			break
		}
	}
	if err = f.Close(); err != nil && !full {
		t.Fatal(err)
	}
	if !full {
		t.Fatal("volume did not fill within bounded fixture limit")
	}
	pending, err := deleteForTest(s, context.Background(), a.ID, a.ID)
	if err == nil || pending {
		t.Fatalf("expected uncommitted deletion failure, pending=%v err=%v", pending, err)
	}
	t.Logf("Actual full-volume deletion transaction rejected: %v", err)
	if s.deletionPending(a.ID) {
		t.Fatal("failed transaction published an intent")
	}
	current, err := s.Get(a.ID)
	if err != nil || !current.Hidden || !current.Added.Equal(a.Added) {
		t.Fatal("asset changed after rejected transaction", err)
	}
	if !reflect.DeepEqual(before, snapshotReview(t, s, a.ID)) {
		t.Fatal("review data changed after rejected transaction")
	}
	got, err := os.ReadFile(s.Original(a.ID))
	if err != nil || !bytes.Equal(got, data) {
		t.Fatal("original changed after rejected transaction", err)
	}
	if err = os.Remove(filler); err != nil {
		t.Fatal(err)
	}
	if pending, err = deleteForTest(s, context.Background(), a.ID, a.ID); err != nil || pending {
		t.Fatal("deletion did not recover after freeing space", pending, err)
	}
	if _, err = os.Stat(s.Original(a.ID)); !os.IsNotExist(err) {
		t.Fatal("original remains after successful retry", err)
	}
	if err = s.Verify(); err != nil {
		t.Fatal(err)
	}
}
