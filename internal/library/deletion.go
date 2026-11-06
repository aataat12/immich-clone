package library

import (
	"context"
	"errors"
	bolt "go.etcd.io/bbolt"
	"os"
	"path/filepath"
	"time"
)

const deletionBucket = "managed_deletions_v1"

var ErrDeletionBusy = errors.New("deletion busy; pause imports and retry after processing or backup finishes")
var ErrDeletionOriginal = errors.New("original integrity check failed; nothing was deleted")
var ErrDeletionPending = errors.New("managed deletion cleanup is pending; retry cleanup in Hidden Media")
var ErrDeletionConflict = errors.New("cleanup stopped because a catalog item reappeared; keep the current copy to resolve")
var ErrDeletionSpace = errors.New("deletion needs at least 16 MiB free application-storage space to record recovery safely; free space and retry; nothing was deleted")

type Deletion struct {
	Version int    `json:"version"`
	ID      string `json:"id"`
	Name    string `json:"name"`
	Status  string `json:"status"`
}

// The private recovery record contains only fixed hash identifiers, never paths.
type deletionRecord struct {
	Version int      `json:"version"`
	ID      string   `json:"id"`
	Name    string   `json:"name"`
	Faces   []string `json:"faces"`
}

func validDeletion(d deletionRecord, id string) bool {
	if d.Version != 1 || d.ID != id || !validID(id) || !textOK(d.Name, 1024) || len(d.Faces) > 10000 {
		return false
	}
	seen := map[string]bool{}
	for _, f := range d.Faces {
		if !validID(f) || seen[f] {
			return false
		}
		seen[f] = true
	}
	return true
}
func initDeletions(tx *bolt.Tx) error {
	b, e := tx.CreateBucketIfNotExists([]byte(deletionBucket))
	if e != nil {
		return e
	}
	return b.ForEach(func(k, v []byte) error {
		var d deletionRecord
		if e := readJSON(b, string(k), &d); e != nil {
			return e
		}
		if !validDeletion(d, string(k)) {
			return ErrOrganizationInput
		}
		return nil
	})
}
func (s *Store) PendingDeletions() ([]Deletion, error) {
	out := []Deletion{}
	e := s.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(deletionBucket))
		return b.ForEach(func(k, v []byte) error {
			var d deletionRecord
			if e := readJSON(b, string(k), &d); e != nil {
				return e
			}
			if !validDeletion(d, string(k)) {
				return ErrOrganizationInput
			}
			status := "pending"
			if tx.Bucket([]byte("assets")).Get(k) != nil {
				status = "conflict"
			}
			out = append(out, Deletion{Version: 1, ID: d.ID, Name: d.Name, Status: status})
			return nil
		})
	})
	return out, e
}
func (s *Store) deletionPending(id string) bool {
	pending := false
	s.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(deletionBucket))
		pending = b != nil && (id == "" && b.Stats().KeyN > 0 || id != "" && b.Get([]byte(id)) != nil)
		return nil
	})
	return pending
}

// All acquisitions are nonblocking; no HTTP request waits behind an unbounded copy.
func (s *Store) deletionLock() (func(), error) {
	if !s.analysisGate.TryLock() {
		return nil, ErrDeletionBusy
	}
	if !s.importMu.TryLock() {
		s.analysisGate.Unlock()
		return nil, ErrDeletionBusy
	}
	if !s.gate.TryLock() {
		s.importMu.Unlock()
		s.analysisGate.Unlock()
		return nil, ErrDeletionBusy
	}
	release := func() { s.gate.Unlock(); s.importMu.Unlock(); s.analysisGate.Unlock() }
	if s.ImportsRunning() {
		release()
		return nil, ErrDeletionBusy
	}
	select {
	case s.mediaAnalysisGate <- struct{}{}:
		return func() { <-s.mediaAnalysisGate; release() }, nil
	default:
		release()
		return nil, ErrDeletionBusy
	}
}
func (s *Store) DeleteHidden(ctx context.Context, id, confirmation string, added time.Time) (bool, error) {
	if !validID(id) || confirmation != id {
		return false, ErrOrganizationInput
	}
	release, e := s.deletionLock()
	if e != nil {
		return false, e
	}
	defer release()
	if e = ctx.Err(); e != nil {
		return false, e
	}
	if s.deletionPending(id) {
		err := s.cleanupDeletion(ctx, id)
		return err != nil, err
	}
	a, e := s.Get(id)
	if e != nil {
		return false, e
	}
	if !a.Hidden || !a.Added.Equal(added) {
		return false, ErrOrganizationInput
	}
	// Deletion must durably journal before removing any bytes. On Windows a
	// failed database mapping growth can also invalidate the open handle, so
	// refuse already exhausted storage before beginning that transaction.
	free, e := AvailableBytes(s.root)
	if e != nil {
		return false, e
	}
	if free < 16<<20 {
		return false, ErrDeletionSpace
	}
	if e = checkDeletionOriginal(ctx, s.Original(id), id); e != nil {
		return false, ErrDeletionOriginal
	}
	d := deletionRecord{Version: 1, ID: id, Name: a.Name, Faces: []string{}}
	e = s.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(deletionBucket))
		if b.Stats().KeyN >= 128 {
			return ErrDeletionPending
		}
		fb := tx.Bucket([]byte(facesBucket))
		if e := fb.ForEach(func(k, v []byte) error {
			if e := ctx.Err(); e != nil {
				return e
			}
			var f StoredFace
			if e := readJSON(fb, string(k), &f); e != nil {
				return e
			}
			if f.AssetID == id {
				d.Faces = append(d.Faces, string(k))
			}
			return nil
		}); e != nil {
			return e
		}
		if !validDeletion(d, id) {
			return ErrOrganizationInput
		}
		// Journal and metadata removal are one durable transaction. No file changes precede it.
		if e := writeJSON(b, id, d); e != nil {
			return e
		}
		for _, face := range d.Faces {
			for _, bucket := range []string{facesBucket, labelsBucket} {
				if e := tx.Bucket([]byte(bucket)).Delete([]byte(face)); e != nil {
					return e
				}
			}
		}
		for _, bucket := range []string{analysisBucket, analysisQueue, queueBucket, tagsBucket, "dismissed_ai_tags_v1", reanalysisBucket, reanalysisQueue, batchMembers} {
			if e := tx.Bucket([]byte(bucket)).Delete([]byte(id)); e != nil {
				return e
			}
		}
		if e := deleteAssetCuration(tx, id); e != nil {
			return e
		}
		if e := markDeletedImports(ctx, tx, id); e != nil {
			return e
		}
		if e := recoverAnalysisBatch(tx); e != nil {
			return e
		}
		root := tx.Bucket([]byte(galleryBucket))
		for order, key := range galleryKeys(a) {
			if e := root.Bucket([]byte("hidden_" + order)).Delete(key); e != nil {
				return e
			}
		}
		if e := setGalleryNumber(root, "total", galleryNumber(root, "total")-1); e != nil {
			return e
		}
		if e := setGalleryNumber(root, "hidden", galleryNumber(root, "hidden")-1); e != nil {
			return e
		}
		if e := setGalleryNumber(root, "revision", galleryNumber(root, "revision")+1); e != nil {
			return e
		}
		if e := ctx.Err(); e != nil {
			return e
		}
		return tx.Bucket([]byte("assets")).Delete([]byte(id))
	})
	if e != nil {
		return false, e
	}
	e = s.cleanupDeletion(ctx, id)
	return e != nil, e
}
func markDeletedImports(ctx context.Context, tx *bolt.Tx, id string) error {
	b := tx.Bucket([]byte(importBucket))
	return b.ForEach(func(k, v []byte) error {
		child := b.Bucket(k)
		updates := map[string]ImportEntry{}
		if e := child.ForEach(func(key, value []byte) error {
			if e := ctx.Err(); e != nil {
				return e
			}
			if string(key) == "meta" {
				return nil
			}
			var entry ImportEntry
			if e := readJSON(child, string(key), &entry); e != nil {
				return e
			}
			if entry.AssetID == id {
				entry.Deleted = true
				entry.Error = "Managed copy permanently deleted; start a new import to copy this source again."
				updates[string(key)] = entry
			}
			return nil
		}); e != nil {
			return e
		}
		for key, entry := range updates {
			if e := writeJSON(child, key, entry); e != nil {
				return e
			}
		}
		return nil
	})
}

func checkDeletionOriginal(ctx context.Context, path, id string) error {
	info, e := os.Lstat(path)
	if os.IsNotExist(e) {
		return nil
	}
	if e != nil {
		return e
	}
	if !info.Mode().IsRegular() {
		return ErrDeletionOriginal
	}
	hash, e := fileHashContext(ctx, path)
	if e != nil {
		return e
	}
	if hash != id {
		return ErrDeletionOriginal
	}
	return nil
}

// Caller holds the copy gate and analysis gate, or the unopened Store owns the DB exclusively.
func (s *Store) cleanupDeletion(ctx context.Context, id string) error {
	var d deletionRecord
	e := s.db.View(func(tx *bolt.Tx) error {
		if tx.Bucket([]byte("assets")).Get([]byte(id)) != nil {
			return ErrDeletionConflict
		}
		return readJSON(tx.Bucket([]byte(deletionBucket)), id, &d)
	})
	if e != nil {
		return e
	}
	if !validDeletion(d, id) {
		return ErrOrganizationInput
	}
	if e = ctx.Err(); e != nil {
		return e
	}
	if e = checkDeletionOriginal(ctx, s.Original(id), id); e != nil {
		return ErrDeletionPending
	}
	paths := []string{s.PreviewPath(id)}
	for _, face := range d.Faces {
		paths = append(paths, filepath.Join(s.root, "faces", face+".png"))
	}
	paths = append(paths, s.Original(id))
	for _, path := range paths {
		if e = ctx.Err(); e != nil {
			return e
		}
		info, err := os.Lstat(path)
		if os.IsNotExist(err) {
			continue
		}
		if err != nil || !info.Mode().IsRegular() {
			return ErrDeletionPending
		}
		if e = os.Remove(path); e != nil {
			return ErrDeletionPending
		}
	}
	return s.db.Update(func(tx *bolt.Tx) error { return tx.Bucket([]byte(deletionBucket)).Delete([]byte(id)) })
}
func (s *Store) RetryDeletion(ctx context.Context, id string) error {
	if !validID(id) {
		return ErrOrganizationInput
	}
	release, e := s.deletionLock()
	if e != nil {
		return e
	}
	defer release()
	return s.cleanupDeletion(ctx, id)
}
func (s *Store) KeepDeletionConflict(id string) error {
	if !validID(id) {
		return ErrOrganizationInput
	}
	release, e := s.deletionLock()
	if e != nil {
		return e
	}
	defer release()
	return s.db.Update(func(tx *bolt.Tx) error {
		if tx.Bucket([]byte("assets")).Get([]byte(id)) == nil || tx.Bucket([]byte(deletionBucket)).Get([]byte(id)) == nil {
			return ErrOrganizationInput
		}
		return tx.Bucket([]byte(deletionBucket)).Delete([]byte(id))
	})
}
func (s *Store) recoverDeletions() {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	jobs, e := s.PendingDeletions()
	if e != nil {
		return
	}
	for _, d := range jobs {
		if ctx.Err() != nil {
			return
		}
		_ = s.cleanupDeletion(ctx, d.ID)
	}
}
