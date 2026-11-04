package library

import (
	"archive/zip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	bolt "go.etcd.io/bbolt"
	"io"
	"os"
	"time"
)

type offlineFiles interface {
	rootFile(string) (*os.File, error)
	original(string) (*os.File, error)
	validate(*os.File) error
	close() error
	physicalRoot() (string, error)
}

var ErrOfflineArchiveCleanup = errors.New("private archive cleanup incomplete")

type offlineState struct {
	catalogSize  int64
	files        offlineFiles
	marker, code *os.File
}

func (o *offlineState) close() (err error) {
	for _, f := range []*os.File{o.code, o.marker} {
		if f != nil {
			err = errors.Join(err, f.Close())
		}
	}
	return errors.Join(err, o.files.close())
}

// OpenReadOnly opens only existing state. No schema/index repair, directory
// creation, code generation, staging or diagnostic initialization is performed.
func OpenReadOnly(root string) (_ *Store, err error) {
	files, e := openOfflineFiles(root)
	if e != nil {
		return nil, e
	}
	o := &offlineState{files: files}
	owned := true
	defer func() {
		if owned {
			err = errors.Join(err, o.close())
		}
	}()
	o.marker, e = files.rootFile("state-format.txt")
	if e != nil {
		return nil, e
	}
	b, e := io.ReadAll(io.LimitReader(o.marker, 128))
	if e != nil {
		return nil, e
	}
	if string(b) != "immichclone-schema-1\n" {
		return nil, errors.New("unsupported state format")
	}
	o.code, e = files.rootFile("access-code.txt")
	if e != nil && !offlineMissing(e) {
		return nil, e
	}
	f, e := files.rootFile("catalog.db")
	if e != nil {
		return nil, e
	}
	info, e := f.Stat()
	if e != nil || info.Size() == 0 {
		f.Close()
		if e != nil {
			return nil, e
		}
		return nil, errors.New("empty catalog")
	}
	used := false
	db, e := bolt.Open("held-catalog", 0600, &bolt.Options{ReadOnly: true, Timeout: time.Second, OpenFile: func(_ string, flag int, _ os.FileMode) (*os.File, error) {
		if used || flag != os.O_RDONLY {
			return nil, errors.New("unexpected catalog open")
		}
		used = true
		return f, nil
	}})
	if e != nil {
		if !used {
			f.Close()
		}
		return nil, e
	}
	o.catalogSize = info.Size()
	s := &Store{root: root, db: db, offline: o}
	e = db.View(func(tx *bolt.Tx) error {
		meta := tx.Bucket([]byte("meta"))
		assets := tx.Bucket([]byte("assets"))
		if meta == nil || assets == nil || string(meta.Get([]byte("schema"))) != "1" {
			return errors.New("unsupported catalog schema")
		}
		return nil
	})
	if e == nil {
		_, e = s.List()
	}
	if e != nil {
		db.Close()
		return nil, e
	}
	owned = false
	return s, nil
}
func (s *Store) offlineOriginal(id string) (*os.File, error) {
	if !validID(id) {
		return nil, errors.New("invalid original identifier")
	}
	return s.offline.files.original(id)
}
func (s *Store) verifyOffline() error {
	assets, e := s.List()
	if e != nil {
		return e
	}
	for _, a := range assets {
		f, e := s.offlineOriginal(a.ID)
		if e != nil {
			return e
		}
		h := sha256.New()
		_, e = io.Copy(h, f)
		if e == nil {
			e = s.offline.files.validate(f)
		}
		ce := f.Close()
		if e != nil || ce != nil {
			return errors.Join(e, ce)
		}
		if hex.EncodeToString(h.Sum(nil)) != a.ID {
			return errors.New("original integrity failure")
		}
	}
	return nil
}

// backupOffline streams the read transaction directly, never a snapshot file in
// the source staging directory. Normal service/browser backup remains unchanged.
func (s *Store) backupOffline(ctx context.Context, w io.Writer) error {
	if e := s.backupLock(ctx); e != nil {
		return e
	}
	defer s.gate.Unlock()
	if s.deletionPending("") {
		return ErrDeletionPending
	}
	if s.offline.catalogSize > MaxUpload {
		return errors.New("catalog exceeds restore entry bound")
	}

	assets, e := s.List()
	if e != nil {
		return e
	}
	if len(assets) > 99996 {
		return errors.New("backup entry count exceeds restore bound")
	}
	for _, a := range assets {
		if a.Size <= 0 || a.Size > MaxUpload {
			return errors.New("original outside backup bound")
		}
	}
	for _, f := range []*os.File{s.offline.marker, s.offline.code} {
		if f != nil {
			info, e := f.Stat()
			if e != nil {
				return e
			}
			if info.Size() > 4096 {
				return errors.New("configuration exceeds backup bound")
			}
		}
	}
	z := zip.NewWriter(w)
	manifest := Manifest{Schema: 1, Files: map[string]string{}}
	add := func(name string, r io.Reader, expected string) error {
		out, e := z.Create(name)
		if e != nil {
			return e
		}
		h := sha256.New()
		if _, e = io.Copy(io.MultiWriter(out, h), backupReader{ctx: ctx, r: r}); e != nil {
			return e
		}
		sum := hex.EncodeToString(h.Sum(nil))
		if expected != "" && sum != expected {
			return errors.New("backup original integrity failure")
		}
		manifest.Files[name] = sum
		return nil
	}
	e = s.db.View(func(tx *bolt.Tx) error {
		out, e := z.Create("catalog.db")
		if e != nil {
			return e
		}
		h := sha256.New()
		_, e = tx.WriteTo(backupContextWriter{ctx: ctx, w: io.MultiWriter(out, h)})
		if e == nil {
			manifest.Files["catalog.db"] = hex.EncodeToString(h.Sum(nil))
		}
		return e
	})
	if e != nil {
		return e
	}
	for _, item := range []struct {
		name string
		f    *os.File
	}{{"state-format.txt", s.offline.marker}, {"access-code.txt", s.offline.code}} {
		if item.f == nil {
			continue
		}
		if _, e = item.f.Seek(0, io.SeekStart); e != nil {
			return e
		}
		if e = add(item.name, item.f, ""); e != nil {
			return e
		}
		if e = s.offline.files.validate(item.f); e != nil {
			return e
		}
	}
	for _, a := range assets {
		f, e := s.offlineOriginal(a.ID)
		if e != nil {
			return e
		}
		info, se := f.Stat()
		if se != nil {
			f.Close()
			return se
		}
		if info.Size() != a.Size || info.Size() > MaxUpload {
			f.Close()
			return errors.New("original size differs from catalog or restore bound")
		}
		e = add("originals/"+a.ID, f, a.ID)
		if e == nil {
			e = s.offline.files.validate(f)
		}
		ce := f.Close()
		if e != nil || ce != nil {
			return errors.Join(e, ce)
		}
	}
	out, e := z.Create("manifest.json")
	if e != nil {
		return e
	}
	if e = json.NewEncoder(out).Encode(manifest); e != nil {
		return e
	}
	if e = ctx.Err(); e != nil {
		return e
	}
	return z.Close()
}
