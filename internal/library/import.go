package library

import (
	"context"
	"errors"
	"fmt"
	bolt "go.etcd.io/bbolt"
	"io"
	"os"
	"strings"
	"time"
)

const importBucket = "imports_v1"
const MaxImportFiles = 50000
const ImportLeaseDuration = 60 * time.Second

var ErrImportInput = errors.New("invalid import selection")
var ErrImportConflict = errors.New("import paused, busy or source changed; committed originals retained")

type importLease struct {
	ID    string
	Until time.Time
}
type ImportFile struct {
	Name     string `json:"name"`
	Relative string `json:"relative"`
	Size     int64  `json:"size"`
	Modified int64  `json:"modified"`
}
type ImportEntry struct {
	ImportFile
	Index     int    `json:"index"`
	AssetID   string `json:"asset_id,omitempty"`
	Duplicate bool   `json:"duplicate"`
	Deleted   bool   `json:"deleted,omitempty"`
	Error     string `json:"error,omitempty"`
}
type ImportSession struct {
	ID            string    `json:"id"`
	Status        string    `json:"status"`
	Created       time.Time `json:"created"`
	TotalFiles    int       `json:"total_files"`
	TotalBytes    int64     `json:"total_bytes"`
	AcceptedFiles int       `json:"accepted_files"`
	AcceptedBytes int64     `json:"accepted_bytes"`
	NewFiles      int       `json:"new_files"`
	Duplicates    int       `json:"duplicates"`
}
type ImportPage struct {
	Session          ImportSession `json:"session"`
	Entries          []ImportEntry `json:"entries"`
	Next             int           `json:"next"`
	BackgroundPaused bool          `json:"background_paused"`
}

func entryKey(index int) string { return fmt.Sprintf("%08d", index) }
func initImports(tx *bolt.Tx) error {
	b, e := tx.CreateBucketIfNotExists([]byte(importBucket))
	if e != nil {
		return e
	}
	return b.ForEach(func(k, v []byte) error {
		if v != nil {
			return ErrImportInput
		}
		child := b.Bucket(k)
		var m ImportSession
		if e := readJSON(child, "meta", &m); e != nil {
			return e
		}
		if m.Status == "running" {
			m.Status = "paused"
			return writeJSON(child, "meta", m)
		}
		return nil
	})
}
func (s *Store) ImportsRunning() bool {
	l := s.importLease.Load()
	return l != nil && time.Now().Before(l.Until)
}
func (s *Store) renewImport(id string) {
	s.importLease.Store(&importLease{ID: id, Until: time.Now().Add(ImportLeaseDuration)})
}
func (s *Store) liveImport(id string) bool {
	l := s.importLease.Load()
	return l != nil && l.ID == id && time.Now().Before(l.Until)
}
func importSession(tx *bolt.Tx, id string) (*bolt.Bucket, ImportSession, error) {
	var m ImportSession
	if !validID(id) {
		return nil, m, ErrImportInput
	}
	b := tx.Bucket([]byte(importBucket)).Bucket([]byte(id))
	if b == nil {
		return nil, m, os.ErrNotExist
	}
	e := readJSON(b, "meta", &m)
	return b, m, e
}
func (s *Store) CreateImport(files []ImportFile) (ImportSession, error) {
	var m ImportSession
	if len(files) == 0 || len(files) > MaxImportFiles {
		return m, ErrImportInput
	}
	seen := map[string]bool{}
	for i := range files {
		f := &files[i]
		if !textOK(f.Name, 256) || f.Name == "" || strings.ContainsAny(f.Name, "/\\") || f.Size <= 0 || f.Size > MaxUpload || f.Modified < 0 {
			return m, ErrImportInput
		}
		if f.Relative != "" {
			p, e := cleanRelative(f.Relative)
			if e != nil {
				return m, ErrImportInput
			}
			f.Relative = p
		}
		key := f.Relative
		if key == "" {
			key = f.Name
		}
		if seen[key] {
			return m, ErrImportInput
		}
		seen[key] = true
		m.TotalBytes += f.Size
	}
	id, e := newFaceID()
	if e != nil {
		return m, e
	}
	m.ID = id
	m.TotalFiles = len(files)
	m.Status = "paused"
	m.Created = time.Now().UTC()
	s.importMu.Lock()
	defer s.importMu.Unlock()
	s.gate.Lock()
	defer s.gate.Unlock()
	e = s.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(importBucket))
		if b.Stats().BucketN > 128 {
			return errors.New("import history limit reached")
		}
		child, e := b.CreateBucket([]byte(id))
		if e != nil {
			return e
		}
		if e = writeJSON(child, "meta", m); e != nil {
			return e
		}
		for i, f := range files {
			if e = writeJSON(child, entryKey(i), ImportEntry{ImportFile: f, Index: i}); e != nil {
				return e
			}
		}
		return nil
	})
	return m, e
}

// Controls never delete accepted originals. A browser must explicitly resume after expiry/restart.
func (s *Store) ControlImport(id, action string) error {
	s.importMu.Lock()
	defer s.importMu.Unlock()
	s.gate.Lock()
	defer s.gate.Unlock()
	e := s.db.Update(func(tx *bolt.Tx) error {
		_, m, e := importSession(tx, id)
		if e != nil {
			return e
		}
		b, _, _ := importSession(tx, id)
		switch action {
		case "start":
			if m.Status == "cancelled" || m.Status == "completed" {
				return ErrImportConflict
			}
			l := s.importLease.Load()
			if s.ImportsRunning() && l.ID != id {
				return ErrImportConflict
			}
			m.Status = "running"
		case "heartbeat":
			if m.Status != "running" || !s.liveImport(id) {
				return ErrImportConflict
			}
		case "pause":
			if m.Status == "cancelled" || m.Status == "completed" {
				return ErrImportConflict
			}
			m.Status = "paused"
		case "cancel":
			if m.Status == "completed" {
				return ErrImportConflict
			}
			m.Status = "cancelled"
		case "finish":
			if m.AcceptedFiles != m.TotalFiles || m.Status == "cancelled" {
				return ErrImportConflict
			}
			m.Status = "completed"
		default:
			return ErrImportInput
		}
		return writeJSON(b, "meta", m)
	})
	if e != nil {
		return e
	}
	if action == "start" || action == "heartbeat" {
		s.renewImport(id)
	} else if l := s.importLease.Load(); l != nil && l.ID == id {
		s.importLease.Store(nil)
	}
	return nil
}
func (s *Store) ImportPage(id string, after int) (ImportPage, error) {
	out := ImportPage{Entries: []ImportEntry{}, Next: -1, BackgroundPaused: s.ImportsRunning()}
	if after < 0 {
		return out, ErrImportInput
	}
	e := s.db.View(func(tx *bolt.Tx) error {
		b, m, e := importSession(tx, id)
		if e != nil {
			return e
		}
		out.Session = m
		if m.Status == "running" && !s.liveImport(id) {
			out.Session.Status = "paused"
		}
		for i := after; i < m.TotalFiles && i < after+250; i++ {
			var f ImportEntry
			if e = readJSON(b, entryKey(i), &f); e != nil {
				return e
			}
			out.Entries = append(out.Entries, f)
		}
		if after+len(out.Entries) < m.TotalFiles {
			out.Next = after + len(out.Entries)
		}
		return nil
	})
	return out, e
}
func (s *Store) ImportHistory() ([]ImportSession, error) {
	out := []ImportSession{}
	e := s.db.View(func(tx *bolt.Tx) error {
		return tx.Bucket([]byte(importBucket)).ForEach(func(k, v []byte) error {
			var m ImportSession
			if e := readJSON(tx.Bucket([]byte(importBucket)).Bucket(k), "meta", &m); e != nil {
				return e
			}
			if m.Status == "running" && !s.liveImport(m.ID) {
				m.Status = "paused"
			}
			out = append(out, m)
			return nil
		})
	})
	return out, e
}

type importReader struct {
	ctx   context.Context
	r     io.Reader
	renew func()
}

func (r importReader) Read(p []byte) (int, error) {
	if e := r.ctx.Err(); e != nil {
		return 0, e
	}
	n, e := r.r.Read(p)
	if n > 0 {
		r.renew()
	}
	return n, e
}

// Replayed accepted entries are re-hashed against their receipt; no metadata-only skip.
func (s *Store) PutImport(ctx context.Context, id string, index int, r io.Reader) (ImportEntry, error) {
	var entry ImportEntry
	if index < 0 {
		return entry, ErrImportInput
	}
	s.importMu.Lock()
	defer s.importMu.Unlock()
	e := s.db.View(func(tx *bolt.Tx) error {
		b, m, e := importSession(tx, id)
		if e != nil {
			return e
		}
		if m.Status != "running" || !s.liveImport(id) {
			return ErrImportConflict
		}
		if e := readJSON(b, entryKey(index), &entry); e != nil {
			return e
		}
		if entry.Deleted {
			return ErrImportConflict
		}
		return nil
	})
	if e != nil {
		return entry, e
	}
	s.renewImport(id)
	originalEntry := entry
	_, _, e = s.putWithPath(entry.Name, entry.Relative, io.LimitReader(importReader{ctx, r, func() {
		if l := s.importLease.Load(); l == nil || time.Until(l.Until) < 45*time.Second {
			s.renewImport(id)
		}
	}}, entry.Size+1), entry.Size, entry.AssetID, func(tx *bolt.Tx, a Asset, duplicate bool) error {
		b, m, e := importSession(tx, id)
		if e != nil {
			return e
		}
		if entry.AssetID == "" {
			entry.AssetID = a.ID
			entry.Duplicate = duplicate
			m.AcceptedFiles++
			m.AcceptedBytes += a.Size
			if duplicate {
				m.Duplicates++
			} else {
				m.NewFiles++
			}
		}
		entry.Error = ""
		if e = writeJSON(b, entryKey(index), entry); e != nil {
			return e
		}
		return writeJSON(b, "meta", m)
	})
	if e != nil {
		entry = originalEntry
		entry.Error = "Copy failed; retry this file. Earlier accepted originals retained."
		if errors.Is(e, ErrImportConflict) {
			entry.Error = "Source size or content changed; original receipt retained. Start a new import for changed files."
		}
		_ = s.db.Update(func(tx *bolt.Tx) error {
			b, _, err := importSession(tx, id)
			if err != nil {
				return err
			}
			return writeJSON(b, entryKey(index), entry)
		})
	}
	return entry, e
}
