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
	// Avoid the normal growth reserve in this fault fixture. A live reader keeps
	// deleted pages unavailable for reuse until the attempted write finishes.
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
	before := snapshotReview(t, s, a.ID)
	reader, err := s.db.Begin(false)
	if err != nil {
		t.Fatal(err)
	}
	defer reader.Rollback()
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
	if err = reader.Rollback(); err != nil {
		t.Fatal(err)
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
