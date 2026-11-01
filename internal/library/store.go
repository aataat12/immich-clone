package library

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	bolt "go.etcd.io/bbolt"
	_ "golang.org/x/image/bmp"
	_ "golang.org/x/image/tiff"
	"image"
	_ "image/gif"
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
	Photo           PhotoMetadata `json:"photo"`
	Sources         []string      `json:"sources,omitempty"`
	PreviewRevision int           `json:"preview_revision,omitempty"`
	ID              string        `json:"id"`
	Name            string        `json:"name"`
	Size            int64         `json:"size"`
	MIME            string        `json:"mime"`
	Added           time.Time     `json:"added"`
	Width           int           `json:"width,omitempty"`
	Height          int           `json:"height,omitempty"`
	Preview         string        `json:"preview"`
	Error           string        `json:"error,omitempty"`
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
		if e := b.Put([]byte("schema"), []byte("1")); e != nil {
			return e
		}
		assets := tx.Bucket([]byte("assets"))
		return assets.ForEach(func(k, v []byte) error {
			var a Asset
			if e := json.Unmarshal(v, &a); e != nil {
				return e
			}
			if ((a.MIME == "image/jpeg" || a.MIME == "image/png") && a.PreviewRevision < 1) || ((a.MIME == "image/bmp" || a.MIME == "image/gif" || a.MIME == "application/octet-stream" || a.MIME == "image/tiff") && a.PreviewRevision < 2) {
				a.Preview = "queued"
				a.Error = ""
				updated, e := json.Marshal(a)
				if e != nil {
					return e
				}
				return assets.Put(k, updated)
			}
			return nil
		})
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
func (s *Store) Put(name string, r io.Reader) (Asset, bool, error) { return s.PutWithPath(name, "", r) }
func (s *Store) PutWithPath(name, relative string, r io.Reader) (result Asset, duplicate bool, returnErr error) {
	if relative != "" {
		clean, e := cleanRelative(relative)
		if e != nil {
			return result, false, e
		}
		relative = clean
	}
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
		if relative != "" {
			found := false
			for _, p := range existing.Sources {
				if p == relative {
					found = true
					break
				}
			}
			if !found {
				if len(existing.Sources) >= 4096 {
					return a, false, errors.New("too many source aliases")
				}
				existing.Sources = append(existing.Sources, relative)
				if e = s.save(existing); e != nil {
					return a, false, e
				}
			}
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
	a = Asset{ID: id, Name: name, Size: n, MIME: detectMediaType(header[:read]), Added: time.Now().UTC(), Preview: "queued"}
	if relative != "" {
		a.Sources = []string{relative}
	}
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
		a.PreviewRevision = 2
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
	// Older catalog entries may predate TIFF signature recognition.
	if a.MIME == "application/octet-stream" {
		f, err := os.Open(s.Original(a.ID))
		if err == nil {
			var header [512]byte
			n, _ := f.Read(header[:])
			f.Close()
			a.MIME = detectMediaType(header[:n])
		}
	}
	if !supportsImagePreview(a.MIME) {
		return errors.New("preview unsupported in this build; original remains available")
	}
	if (a.MIME == "image/tiff" || a.MIME == "image/bmp" || a.MIME == "image/gif") && a.Size > 64<<20 {
		return errors.New("image exceeds 64 MiB encoded preview limit; original remains available")
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
	a.Photo = ReadPhotoMetadata(f)
	if _, e = f.Seek(0, 0); e != nil {
		return e
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
	img = oriented(img, a.Photo.Orientation)
	a.Width, a.Height = img.Bounds().Dx(), img.Bounds().Dy()
	w, hgt := a.Width, a.Height
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
			thumb.Set(x, y, img.At(bounds.Min.X+x*a.Width/w, bounds.Min.Y+y*a.Height/hgt))
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

func cleanRelative(p string) (string, error) {
	p = strings.ReplaceAll(p, "\\", "/")
	if len(p) > 1024 || strings.HasPrefix(p, "/") || strings.Contains(p, ":") {
		return "", errors.New("invalid relative import path")
	}
	for _, r := range p {
		if r < 32 || r == 127 {
			return "", errors.New("invalid relative import path")
		}
	}
	for _, c := range strings.Split(p, "/") {
		if c == "" || c == "." || c == ".." {
			return "", errors.New("invalid relative import path")
		}
	}
	return p, nil
}

func detectMediaType(header []byte) string {
	if len(header) >= 4 && (string(header[:4]) == "II*\x00" || string(header[:4]) == "MM\x00*") {
		return "image/tiff"
	}
	return http.DetectContentType(header)
}
func supportsImagePreview(mime string) bool {
	switch mime {
	case "image/jpeg", "image/png", "image/bmp", "image/gif", "image/tiff":
		return true
	}
	return false
}
