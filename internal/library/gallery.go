package library

import (
	"bytes"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"errors"
	bolt "go.etcd.io/bbolt"
	"sort"
	"strings"
	"time"
)

const galleryBucket = "gallery_v1"

var ErrGalleryInput = errors.New("invalid gallery page")
var ErrGalleryChanged = errors.New("gallery ordering changed; refresh from first page")

type GalleryPage struct {
	Assets []Asset `json:"assets"`
	Next   string  `json:"next,omitempty"`
	Total  uint64  `json:"total"`
	Hidden uint64  `json:"hidden"`
	Count  uint64  `json:"count"`
}
type galleryCursor struct {
	Epoch          string `json:"epoch"`
	Revision       uint64 `json:"revision"`
	Sort           string `json:"sort"`
	Visibility     string `json:"visibility"`
	Key            []byte `json:"key"`
	Query          string `json:"query,omitempty"`
	SearchRevision int    `json:"search_revision,omitempty"`
}

func galleryGroup(a Asset) string {
	if a.Hidden {
		return "hidden"
	}
	return "library"
}
func galleryKeys(a Asset) map[string][]byte {
	uploaded := a.Added.UTC().Format("2006-01-02T15:04:05")
	captured := uploaded
	if _, e := time.Parse("2006-01-02T15:04:05", a.Photo.Captured); e == nil {
		captured = a.Photo.Captured
	}
	descending := func(s string) []byte {
		b := []byte(s)
		for i := range b {
			b[i] = 255 - b[i]
		}
		return append(append(b, 0), []byte(a.ID)...)
	}
	name := strings.ToLower(a.Name)
	if len(name) > 1024 {
		name = name[:1024]
	}
	return map[string][]byte{"capture": descending(captured), "upload": descending(uploaded), "name": []byte(name + "\x00" + a.ID)}
}
func galleryNumber(b *bolt.Bucket, key string) uint64 {
	v := b.Get([]byte(key))
	if len(v) != 8 {
		return 0
	}
	return binary.LittleEndian.Uint64(v)
}
func setGalleryNumber(b *bolt.Bucket, key string, n uint64) error {
	v := make([]byte, 8)
	binary.LittleEndian.PutUint64(v, n)
	return b.Put([]byte(key), v)
}

// Derived indexes always rebuild from authoritative assets, including after older-writer rollback.
func initGallery(tx *bolt.Tx) error {
	if tx.Bucket([]byte(galleryBucket)) != nil {
		if e := tx.DeleteBucket([]byte(galleryBucket)); e != nil {
			return e
		}
	}
	root, e := tx.CreateBucket([]byte(galleryBucket))
	if e != nil {
		return e
	}
	epoch, e := newFaceID()
	if e != nil {
		return e
	}
	if e = root.Put([]byte("epoch"), []byte(epoch)); e != nil {
		return e
	}
	for _, visibility := range []string{"library", "hidden"} {
		for _, order := range []string{"capture", "upload", "name"} {
			if _, e = root.CreateBucket([]byte(visibility + "_" + order)); e != nil {
				return e
			}
		}
	}
	type indexEntry struct {
		key []byte
		id  string
	}
	entries := map[string][]indexEntry{}
	var total, hidden uint64
	e = tx.Bucket([]byte("assets")).ForEach(func(k, v []byte) error {
		var a Asset
		if e := json.Unmarshal(v, &a); e != nil {
			return e
		}
		if !validID(a.ID) || a.ID != string(k) {
			return ErrGalleryInput
		}
		for order, key := range galleryKeys(a) {
			name := galleryGroup(a) + "_" + order
			entries[name] = append(entries[name], indexEntry{key, a.ID})
		}
		total++
		if a.Hidden {
			hidden++
		}
		return nil
	})
	if e != nil {
		return e
	}
	// Sorted bulk insertion avoids quadratic insertion into large unspilled nodes.
	for name, items := range entries {
		sort.Slice(items, func(i, j int) bool { return bytes.Compare(items[i].key, items[j].key) < 0 })
		b := root.Bucket([]byte(name))
		for _, item := range items {
			if e = b.Put(item.key, []byte(item.id)); e != nil {
				return e
			}
		}
	}
	if e = setGalleryNumber(root, "total", total); e != nil {
		return e
	}
	return setGalleryNumber(root, "hidden", hidden)
}
func updateGallery(tx *bolt.Tx, a Asset) error {
	root := tx.Bucket([]byte(galleryBucket))
	if root == nil {
		return errors.New("gallery index unavailable; reopen library")
	}
	var old Asset
	data := tx.Bucket([]byte("assets")).Get([]byte(a.ID))
	exists := data != nil
	if exists {
		if e := json.Unmarshal(data, &old); e != nil {
			return e
		}
	}
	keys := galleryKeys(a)
	oldKeys := galleryKeys(old)
	changed := !exists || a.Hidden != old.Hidden
	for order, key := range keys {
		if !exists || a.Hidden != old.Hidden || !bytes.Equal(oldKeys[order], key) {
			changed = true
			if exists {
				if e := root.Bucket([]byte(galleryGroup(old) + "_" + order)).Delete(oldKeys[order]); e != nil {
					return e
				}
			}
			if e := root.Bucket([]byte(galleryGroup(a)+"_"+order)).Put(key, []byte(a.ID)); e != nil {
				return e
			}
		}
	}
	if !exists {
		if e := setGalleryNumber(root, "total", galleryNumber(root, "total")+1); e != nil {
			return e
		}
	}
	hidden := galleryNumber(root, "hidden")
	if exists && old.Hidden && !a.Hidden {
		hidden--
	}
	if a.Hidden && (!exists || !old.Hidden) {
		hidden++
	}
	if e := setGalleryNumber(root, "hidden", hidden); e != nil {
		return e
	}
	if changed {
		return setGalleryNumber(root, "revision", galleryNumber(root, "revision")+1)
	}
	return nil
}

// GalleryPage visits only requested indexed rows; legacy List remains for explicit export/backup callers.
func (s *Store) GalleryPage(order, visibility, after string, limit int) (GalleryPage, error) {
	return s.SearchGalleryPage(order, visibility, after, "", limit)
}

// SearchGalleryPage filters the entire eligible catalog before bounded paging.
// The unfiltered path retains its index-only traversal and ordering cursors.
func (s *Store) SearchGalleryPage(order, visibility, after, query string, limit int) (GalleryPage, error) {
	out := GalleryPage{Assets: []Asset{}}
	if !textOK(query, 128) {
		return out, ErrGalleryInput
	}
	query = strings.ToLower(strings.TrimSpace(query))
	if (order != "capture" && order != "upload" && order != "name") || (visibility != "library" && visibility != "hidden") || limit < 1 || limit > 100 || len(after) > 4096 {
		return out, ErrGalleryInput
	}
	var cursor galleryCursor
	if after != "" {
		b, e := base64.RawURLEncoding.DecodeString(after)
		if e != nil || json.Unmarshal(b, &cursor) != nil || cursor.Sort != order || cursor.Visibility != visibility || cursor.Query != query || len(cursor.Key) == 0 || len(cursor.Key) > 1100 {
			return out, ErrGalleryInput
		}
	}
	if query != "" {
		s.gate.Lock()
		defer s.gate.Unlock()
	}
	e := s.db.View(func(tx *bolt.Tx) error {
		root := tx.Bucket([]byte(galleryBucket))
		if root == nil {
			return errors.New("gallery index unavailable")
		}
		epoch := string(root.Get([]byte("epoch")))
		revision := galleryNumber(root, "revision")
		if after != "" && (cursor.Epoch != epoch || cursor.Revision != revision) {
			return ErrGalleryChanged
		}
		out.Total = galleryNumber(root, "total")
		out.Hidden = galleryNumber(root, "hidden")
		out.Count = out.Total - out.Hidden
		if visibility == "hidden" {
			out.Count = out.Hidden
		}
		var search *gallerySearchIndex
		if query != "" {
			if after != "" && cursor.SearchRevision != tx.ID() {
				return ErrGalleryChanged
			}
			var e error
			search, e = s.gallerySearchFor(tx)
			if e != nil {
				return e
			}
			out.Count = 0
			for _, row := range search.assets {
				if row.hidden == (visibility == "hidden") && strings.Contains(row.terms, query) {
					out.Count++
				}
			}
		}
		index := root.Bucket([]byte(visibility + "_" + order))
		c := index.Cursor()
		k, v := c.First()
		if after != "" {
			if index.Get(cursor.Key) == nil {
				return ErrGalleryInput
			}
			k, v = c.Seek(cursor.Key)
			k, v = c.Next()
		}
		var last []byte
		for ; k != nil; k, v = c.Next() {
			if search != nil && !strings.Contains(search.assets[string(v)].terms, query) {
				continue
			}
			if len(out.Assets) == limit {
				next := galleryCursor{Epoch: epoch, Revision: revision, Sort: order, Visibility: visibility, Key: last, Query: query}
				if query != "" {
					next.SearchRevision = tx.ID()
				}
				b, e := json.Marshal(next)
				if e != nil {
					return e
				}
				out.Next = base64.RawURLEncoding.EncodeToString(b)
				break
			}
			var a Asset
			raw := tx.Bucket([]byte("assets")).Get(v)
			if raw == nil || json.Unmarshal(raw, &a) != nil || a.ID != string(v) || !validID(a.ID) || galleryGroup(a) != visibility {
				return errors.New("gallery index inconsistent; reopen library")
			}
			out.Assets = append(out.Assets, a)
			last = append(last[:0], k...)
		}
		return nil
	})
	return out, e
}
