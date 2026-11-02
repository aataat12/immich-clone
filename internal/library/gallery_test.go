package library

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	bolt "go.etcd.io/bbolt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func seedGallery(t *testing.T, s *Store, n int) {
	t.Helper()
	e := s.db.Update(func(tx *bolt.Tx) error {
		for i := 0; i < n; i++ {
			a := Asset{ID: fmt.Sprintf("%064x", i+1), Name: fmt.Sprintf("Synthetic-%06d.png", n-i), MIME: "image/png", Size: 1, Added: time.Date(2024, 1, 1, 0, i%60, 0, 0, time.UTC), Preview: "failed", PreviewRevision: 3, Hidden: i%7 == 0}
			if e := saveAsset(tx, a); e != nil {
				return e
			}
		}
		return nil
	})
	if e != nil {
		t.Fatal(e)
	}
}
func TestGalleryStablePagesOrderingVisibilityAndRebuild(t *testing.T) {
	s, root := openTest(t)
	seedGallery(t, s, 247)
	for _, order := range []string{"capture", "upload", "name"} {
		for _, visibility := range []string{"library", "hidden"} {
			seen := map[string]bool{}
			after := ""
			var previous []byte
			for {
				page, e := s.GalleryPage(order, visibility, after, 17)
				if e != nil || len(page.Assets) > 17 {
					t.Fatal(e)
				}
				for _, a := range page.Assets {
					if seen[a.ID] {
						t.Fatal("duplicate page item")
					}
					seen[a.ID] = true
					key := galleryKeys(a)[order]
					if previous != nil && bytes.Compare(previous, key) >= 0 {
						t.Fatal("incorrect ordering")
					}
					previous = key
				}
				if page.Next == "" {
					if uint64(len(seen)) != page.Count || page.Total != 247 {
						t.Fatal("skipped rows/counts")
					}
					break
				}
				after = page.Next
			}
		}
	}
	page, _ := s.GalleryPage("name", "library", "", 20)
	cursor := page.Next
	s.Close()
	again, e := Open(root)
	if e != nil {
		t.Fatal(e)
	}
	defer again.Close()
	if _, e = again.GalleryPage("name", "library", cursor, 20); !errors.Is(e, ErrGalleryChanged) {
		t.Fatal("restart cursor not invalidated", e)
	}
	// Simulate an older writer changing authoritative rows without maintaining new indexes.
	e = again.db.Update(func(tx *bolt.Tx) error {
		var a Asset
		b := tx.Bucket([]byte("assets"))
		k, v := b.Cursor().First()
		json.Unmarshal(v, &a)
		a.Hidden = !a.Hidden
		a.Name = "Older writer change"
		data, _ := json.Marshal(a)
		return b.Put(k, data)
	})
	if e != nil {
		t.Fatal(e)
	}
	again.Close()
	again, e = Open(root)
	if e != nil {
		t.Fatal(e)
	}
	defer again.Close()
	all, _ := again.List()
	expected := uint64(0)
	for _, a := range all {
		if a.Hidden {
			expected++
		}
	}
	page, e = again.GalleryPage("name", "hidden", "", 100)
	if e != nil || page.Hidden != expected {
		t.Fatal("index rebuild/count recovery", e)
	}
}
func TestGalleryUpdatesAtomicityCursorAndIndependentRestore(t *testing.T) {
	s, root := openTest(t)
	data := fixture(t)
	a, _, e := s.PutWithPath("native.png", "Synthetic/native.png", bytes.NewReader(data))
	if e != nil {
		t.Fatal(e)
	}
	other, _, _ := s.Put("other.txt", bytes.NewReader([]byte("synthetic other original")))
	first, _ := s.GalleryPage("capture", "library", "", 1)
	cursor := first.Next
	// Preview-only updates retain ordering cursors.
	a.Preview = "failed"
	a.Error = "synthetic failure"
	if e = s.save(a); e != nil {
		t.Fatal(e)
	}
	if _, e = s.GalleryPage("capture", "library", cursor, 1); e != nil {
		t.Fatal("preview status invalidated order", e)
	}
	a.Photo.Captured = "2023-05-06T12:13:14"
	if e = s.save(a); e != nil {
		t.Fatal(e)
	}
	if _, e = s.GalleryPage("capture", "library", cursor, 1); !errors.Is(e, ErrGalleryChanged) {
		t.Fatal("capture change did not invalidate", e)
	}
	s.SetHidden(other.ID, true)
	p, _ := s.GalleryPage("name", "library", "", 60)
	if len(p.Assets) != 1 || p.Hidden != 1 {
		t.Fatal(p)
	}
	p, _ = s.GalleryPage("name", "hidden", "", 60)
	if len(p.Assets) != 1 || p.Assets[0].ID != other.ID {
		t.Fatal(p)
	}
	// A failed write rolls back ordering/counts alongside the asset.
	e = s.db.Update(func(tx *bolt.Tx) error {
		a.Hidden = true
		if e := saveAsset(tx, a); e != nil {
			return e
		}
		return errors.New("injected rollback")
	})
	if e == nil {
		t.Fatal("rollback missing")
	}
	p, _ = s.GalleryPage("name", "library", "", 60)
	if len(p.Assets) != 1 || p.Hidden != 1 {
		t.Fatal("index survived rollback")
	}
	var archive bytes.Buffer
	s.Backup(&archive)
	path := filepath.Join(t.TempDir(), "backup.zip")
	os.WriteFile(path, archive.Bytes(), 0600)
	target := filepath.Join(t.TempDir(), "restored")
	if e = Restore(path, target); e != nil {
		t.Fatal(e)
	}
	s.Close()
	os.Rename(root, root+".offline")
	r, e := Open(target)
	if e != nil {
		t.Fatal(e)
	}
	defer r.Close()
	p, e = r.GalleryPage("capture", "library", "", 1)
	if e != nil || len(p.Assets) != 1 || p.Hidden != 1 || p.Assets[0].Photo.Captured != "2023-05-06T12:13:14" {
		t.Fatal("restored index", e)
	}
	if e = r.Verify(); e != nil {
		t.Fatal(e)
	}
	for _, args := range [][3]string{{"bad", "library", ""}, {"name", "bad", ""}, {"name", "library", "!"}} {
		if _, e = r.GalleryPage(args[0], args[1], args[2], 60); e == nil {
			t.Fatal("bad query accepted")
		}
	}
	if _, e = r.GalleryPage("name", "library", "", 101); e == nil {
		t.Fatal("oversized page")
	}
}
func TestGalleryPageAllocationBoundedAtScale(t *testing.T) {
	s, _ := openTest(t)
	seedGallery(t, s, 5000)
	p, e := s.GalleryPage("name", "library", "", 60)
	if e != nil || len(p.Assets) != 60 || p.Next == "" {
		t.Fatal(e)
	}
	allocs := testing.AllocsPerRun(3, func() {
		if _, e := s.GalleryPage("name", "library", "", 60); e != nil {
			panic(e)
		}
	})
	if allocs > 4000 {
		t.Fatalf("page allocated catalog-sized work: %.0f", allocs)
	}
	t.Logf("60-row gallery page among 5000 metadata-only rows: %.0f allocations; not bulk media acceptance", allocs)
}
