package library

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	bolt "go.etcd.io/bbolt"
	"image"
	_ "image/jpeg"
	"image/png"
	"immichclone/internal/diagnostics"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

const MaxUpload int64 = 4 << 30

var ErrTooLarge = errors.New("file exceeds 4 GiB upload limit")

type Asset struct {
	ID      string    `json:"id"`
	Name    string    `json:"name"`
	Size    int64     `json:"size"`
	MIME    string    `json:"mime"`
	Added   time.Time `json:"added"`
	Width   int       `json:"width,omitempty"`
	Height  int       `json:"height,omitempty"`
	Preview string    `json:"preview"`
	Error   string    `json:"error,omitempty"`
}
type Store struct {
	root string
	db   *bolt.DB
	gate sync.Mutex
}

func Open(root string) (*Store, error) {
	if root == "" {
		return nil, errors.New("data directory required")
	}
	if e := os.MkdirAll(root, 0700); e != nil {
		return nil, e
	}
	marker := filepath.Join(root, "state-format.txt")
	if b, e := os.ReadFile(marker); e == nil {
		if string(b) != "immichclone-schema-1\n" {
			return nil, errors.New("unknown state format")
		}
	} else if errors.Is(e, os.ErrNotExist) {
		entries, e := os.ReadDir(root)
		if e != nil {
			return nil, e
		}
		if len(entries) != 0 {
			return nil, errors.New("refusing an unrelated nonempty data directory")
		}
		f, e := os.OpenFile(marker, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
		if e != nil {
			return nil, e
		}
		_, e = f.WriteString("immichclone-schema-1\n")
		if e == nil {
			e = f.Sync()
		}
		ce := f.Close()
		if e == nil {
			e = ce
		}
		if e != nil {
			return nil, e
		}
	} else {
		return nil, e
	}
	for _, d := range []string{"", "originals", "staging", "previews"} {
		if e := os.MkdirAll(filepath.Join(root, d), 0700); e != nil {
			return nil, e
		}
	}
	db, e := bolt.Open(filepath.Join(root, "catalog.db"), 0600, &bolt.Options{Timeout: time.Second})
	if e != nil {
		return nil, e
	}
	s := &Store{root: root, db: db}
	e = db.Update(func(tx *bolt.Tx) error {
		for _, n := range []string{"meta", "assets"} {
			if _, e := tx.CreateBucketIfNotExists([]byte(n)); e != nil {
				return e
			}
		}
		b := tx.Bucket([]byte("meta"))
		v := b.Get([]byte("schema"))
		if v != nil && string(v) != "1" {
			return errors.New("unsupported schema; refusing migration")
		}
		return b.Put([]byte("schema"), []byte("1"))
	})
	if e != nil {
		db.Close()
		return nil, e
	}
	// Staging is never authoritative. The database lock excludes another instance.
	entries, e := os.ReadDir(filepath.Join(root, "staging"))
	if e != nil {
		db.Close()
		return nil, e
	}
	for _, f := range entries {
		if !f.IsDir() {
			if e = os.Remove(filepath.Join(root, "staging", f.Name())); e != nil {
				db.Close()
				return nil, e
			}
		}
	}
	return s, nil
}
func (s *Store) Close() error                 { return s.db.Close() }
func (s *Store) Original(id string) string    { return filepath.Join(s.root, "originals", id) }
func (s *Store) PreviewPath(id string) string { return filepath.Join(s.root, "previews", id+".png") }
func validID(id string) bool {
	b, e := hex.DecodeString(id)
	return e == nil && len(b) == 32 && id == strings.ToLower(id)
}
func (s *Store) Get(id string) (a Asset, e error) {
	if !validID(id) {
		return a, os.ErrNotExist
	}
	e = s.db.View(func(tx *bolt.Tx) error {
		v := tx.Bucket([]byte("assets")).Get([]byte(id))
		if v == nil {
			return os.ErrNotExist
		}
		return json.Unmarshal(v, &a)
	})
	return
}
func (s *Store) List() ([]Asset, error) {
	assets := []Asset{}
	e := s.db.View(func(tx *bolt.Tx) error {
		return tx.Bucket([]byte("assets")).ForEach(func(k, v []byte) error {
			var a Asset
			if e := json.Unmarshal(v, &a); e != nil {
				return e
			}
			if !validID(a.ID) || a.ID != string(k) {
				return errors.New("invalid catalog record")
			}
			assets = append(assets, a)
			return nil
		})
	})
	return assets, e
}
func (s *Store) save(a Asset) error {
	b, e := json.Marshal(a)
	if e != nil {
		return e
	}
	return s.db.Update(func(tx *bolt.Tx) error { return tx.Bucket([]byte("assets")).Put([]byte(a.ID), b) })
}
func fileHash(path string) (string, error) {
	f, e := os.Open(path)
	if e != nil {
		return "", e
	}
	defer f.Close()
	h := sha256.New()
	if _, e = io.Copy(h, f); e != nil {
		return "", e
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}
func (s *Store) Put(name string, r io.Reader) (result Asset, duplicate bool, returnErr error) {
	started := time.Now()
	defer func() { diagnostics.Record("upload", "original_write", returnErr, 0, result.Size, time.Since(started)) }()
	s.gate.Lock()
	defer s.gate.Unlock()
	var a Asset
	f, e := os.CreateTemp(filepath.Join(s.root, "staging"), "upload-*.part")
	if e != nil {
		return a, false, e
	}
	tmp := f.Name()
	defer os.Remove(tmp)
	h := sha256.New()
	n, e := io.Copy(io.MultiWriter(f, h), io.LimitReader(r, MaxUpload+1))
	if e == nil && n > MaxUpload {
		e = ErrTooLarge
	}
	if e == nil {
		e = f.Sync()
	}
	ce := f.Close()
	if e == nil {
		e = ce
	}
	if e != nil {
		return a, false, e
	}
	if n == 0 {
		return a, false, errors.New("empty files are not accepted")
	}
	id := hex.EncodeToString(h.Sum(nil))
	if existing, e := s.Get(id); e == nil {
		actual, e := fileHash(s.Original(id))
		if e != nil || actual != id {
			return a, false, errors.New("existing original is missing or corrupted")
		}
		return existing, true, nil
	} else if !errors.Is(e, os.ErrNotExist) {
		return a, false, e
	}
	// Hard-link publication is exclusive: an existing blob is never overwritten.
	dest := s.Original(id)
	if e = os.Link(tmp, dest); e != nil {
		if !errors.Is(e, os.ErrExist) {
			return a, false, e
		}
		actual, e := fileHash(dest)
		if e != nil || actual != id {
			return a, false, errors.New("orphan original failed integrity check")
		}
	}
	source, e := os.Open(dest)
	if e != nil {
		return a, false, e
	}
	header := make([]byte, 512)
	read, _ := source.Read(header)
	source.Close()
	name = filepath.Base(strings.ReplaceAll(name, "\\", "/"))
	if name == "." || name == "/" || name == "" {
		name = "original"
	}
	a = Asset{ID: id, Name: name, Size: n, MIME: http.DetectContentType(header[:read]), Added: time.Now().UTC(), Preview: "queued"}
	if e = s.save(a); e != nil {
		return a, false, e
	}
	return a, false, nil
}

// ProcessOne persists the result; queued jobs survive process termination.
// Processing is serialized with ingest/backup to keep this initial slice simple.
func (s *Store) ProcessOne() (bool, error) {
	s.gate.Lock()
	defer s.gate.Unlock()
	assets, e := s.List()
	if e != nil {
		return false, e
	}
	for _, a := range assets {
		if a.Preview != "queued" {
			continue
		}
		e = s.makePreview(&a)
		diagnostics.Record("preview", "preview_generate", e, 0, 0, 0)
		if e != nil {
			a.Preview = "failed"
			a.Error = e.Error()
		} else {
			a.Preview = "ready"
			a.Error = ""
		}
		if e = s.save(a); e != nil {
			return true, e
		}
		return true, nil
	}
	return false, nil
}
func (s *Store) makePreview(a *Asset) error {
	if a.MIME != "image/png" && a.MIME != "image/jpeg" {
		return errors.New("preview unsupported in this build; original remains available")
	}
	f, e := os.Open(s.Original(a.ID))
	if e != nil {
		return errors.New("original unavailable")
	}
	defer f.Close()
	h, e := fileHash(s.Original(a.ID))
	if e != nil || h != a.ID {
		return errors.New("original checksum mismatch")
	}
	config, _, e := image.DecodeConfig(f)
	if e != nil {
		return errors.New("malformed image")
	}
	if config.Width <= 0 || config.Height <= 0 || int64(config.Width)*int64(config.Height) > 40_000_000 {
		return errors.New("image exceeds 40 megapixel preview limit")
	}
	a.Width, a.Height = config.Width, config.Height
	if _, e = f.Seek(0, 0); e != nil {
		return e
	}
	img, _, e := image.Decode(f)
	if e != nil {
		return errors.New("image decode failed")
	}
	w, hgt := config.Width, config.Height
	if w > 320 || hgt > 320 {
		if w >= hgt {
			hgt = hgt * 320 / w
			w = 320
		} else {
			w = w * 320 / hgt
			hgt = 320
		}
	}
	if w < 1 {
		w = 1
	}
	if hgt < 1 {
		hgt = 1
	}
	thumb := image.NewRGBA(image.Rect(0, 0, w, hgt))
	bounds := img.Bounds()
	for y := 0; y < hgt; y++ {
		for x := 0; x < w; x++ {
			thumb.Set(x, y, img.At(bounds.Min.X+x*config.Width/w, bounds.Min.Y+y*config.Height/hgt))
		}
	}
	out, e := os.CreateTemp(filepath.Join(s.root, "staging"), "preview-*.part")
	if e != nil {
		return e
	}
	defer os.Remove(out.Name())
	e = png.Encode(out, thumb)
	if e == nil {
		e = out.Sync()
	}
	ce := out.Close()
	if e == nil {
		e = ce
	}
	if e != nil {
		return e
	}
	// Preview is disposable; remove a leftover from an interrupted queued job.
	if e = os.Remove(s.PreviewPath(a.ID)); e != nil && !errors.Is(e, os.ErrNotExist) {
		return e
	}
	return os.Rename(out.Name(), s.PreviewPath(a.ID))
}
func (s *Store) Retry(id string) error {
	s.gate.Lock()
	defer s.gate.Unlock()
	a, e := s.Get(id)
	if e != nil {
		return e
	}
	a.Preview = "queued"
	a.Error = ""
	return s.save(a)
}
func (s *Store) Verify() (result error) {
	started := time.Now()
	defer func() { diagnostics.Record("verify", "integrity", result, 0, 0, time.Since(started)) }()
	s.gate.Lock()
	defer s.gate.Unlock()
	assets, e := s.List()
	if e != nil {
		return e
	}
	for _, a := range assets {
		hash, e := fileHash(s.Original(a.ID))
		if e != nil || hash != a.ID {
			return fmt.Errorf("original integrity failure: %s", a.ID)
		}
	}
	return nil
}
