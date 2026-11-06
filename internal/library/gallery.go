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
	Organization map[string]GalleryOrganization `json:"organization"`
 Previous    string       `json:"previous,omitempty"`
	HasPrevious bool         `json:"has_previous"`
	Assets      []Asset      `json:"assets"`
	Next        string       `json:"next,omitempty"`
	Total       uint64       `json:"total"`
	Hidden      uint64       `json:"hidden"`
	Count       uint64       `json:"count"`
	Query       GalleryQuery `json:"query"`
	Snapshot    string       `json:"snapshot"`
}
type galleryCursor struct {
	Epoch          string `json:"epoch"`
	Revision       uint64 `json:"revision"`
	Sort           string `json:"sort"`
	Visibility     string `json:"visibility"`
	Key            []byte `json:"key"`
	Query          string `json:"query,omitempty"`
	SearchRevision int    `json:"search_revision,omitempty"`
 SearchState string `json:"search_state,omitempty"`
	QueryKey       string `json:"query_key,omitempty"`
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

// GalleryPage retains the 0.1.43 public entry point.
func (s *Store) GalleryPage(order, visibility, after string, limit int) (GalleryPage, error) {
	return s.SearchGalleryPage(order, visibility, after, "", limit)
}

func (s *Store) SearchGalleryPage(order, visibility, after, query string, limit int) (GalleryPage, error) {
	return s.QueryGalleryPage(GalleryQuery{Sort: order, Visibility: visibility, Text: query}, after, limit)
}

type galleryResults struct {
	ordered              [][]byte
	tx                   *bolt.Tx
	query                GalleryQuery
	index                *bolt.Bucket
	token                galleryCursor
	matches              map[string]bool
	total, hidden, count uint64
}

func (s *Store) galleryResultsFor(tx *bolt.Tx, q GalleryQuery) (*galleryResults, error) {
	if q.Album!=""{if _,e:=curationAlbum(tx,q.Album);e!=nil{return nil,ErrGalleryChanged}}
 root := tx.Bucket([]byte(galleryBucket))
	if root == nil {
		return nil, errors.New("gallery index unavailable")
	}
	r := &galleryResults{tx: tx, query: q, index: root.Bucket([]byte(q.Visibility + "_" + q.Sort)), total: galleryNumber(root, "total"), hidden: galleryNumber(root, "hidden")}
	r.count = r.total - r.hidden
	if q.Visibility == "hidden" {
		r.count = r.hidden
	}
	r.token = galleryCursor{Epoch: string(root.Get([]byte("epoch"))), Revision: galleryNumber(root, "revision"), Sort: q.Sort, Visibility: q.Visibility, Query: q.Text, QueryKey: q.key()}
	if q.filtered() {
		search, e := s.gallerySearchFor(tx)
		if e != nil {
			return nil, e
		}
		r.token.SearchState=search.state
		r.matches = search.matching(q)
		r.ordered = search.orderedKeys(q)
		r.count = uint64(len(r.matches))
	}
	return r, nil
}

func encodeGalleryCursor(c galleryCursor) string {
	b, _ := json.Marshal(c)
	return base64.RawURLEncoding.EncodeToString(b)
}

func (r *galleryResults) validate(encoded string, page bool) (galleryCursor, error) {
	var c galleryCursor
	if len(encoded) > 4096 {
		return c, ErrGalleryInput
	}
	b, e := base64.RawURLEncoding.DecodeString(encoded)
	if e != nil || json.Unmarshal(b, &c) != nil || c.QueryKey != r.token.QueryKey || c.Sort != r.query.Sort || c.Visibility != r.query.Visibility || len(c.Key) > 1100 || (page && len(c.Key) == 0) || (!page && len(c.Key) != 0) {
		return c, ErrGalleryInput
	}
	if c.Epoch != r.token.Epoch || c.Revision != r.token.Revision || c.SearchRevision != r.token.SearchRevision || c.SearchState!=r.token.SearchState {
		return c, ErrGalleryChanged
	}
	if page && r.query.Sort=="manual" {if !r.matches[string(c.Key)]{return c,ErrGalleryInput};return c,nil}
 if page && r.index.Get(c.Key) == nil {
		return c, ErrGalleryInput
	}
	return c, nil
}

func (r *galleryResults) includes(id []byte) bool { return r.matches == nil || r.matches[string(id)] }
func (r *galleryResults) asset(id []byte) (Asset, error) {
	var a Asset
	raw := r.tx.Bucket([]byte("assets")).Get(id)
	if raw == nil || json.Unmarshal(raw, &a) != nil || a.ID != string(id) || !validID(a.ID) || galleryGroup(a) != r.query.Visibility {
		return a, errors.New("gallery index inconsistent; reopen library")
	}
	return a, nil
}

// Unfiltered requests still visit only bounded indexed rows, without building
// metadata caches or taking the metadata gate. Filtered counts and rows share
// one membership set at the same read transaction.
func (s *Store) QueryGalleryPage(query GalleryQuery, after string, limit int) (GalleryPage, error) {
	out := GalleryPage{Assets: []Asset{}}
	q, e := query.Normalize()
	if e != nil || limit < 1 || limit > 100 || len(after) > 4096 {
		return out, ErrGalleryInput
	}
	if q.filtered() {
		s.gate.Lock()
		defer s.gate.Unlock()
	}
	e = s.db.View(func(tx *bolt.Tx) error {
		r, e := s.galleryResultsFor(tx, q)
		if e != nil {
			return e
		}
		out, e = r.page(after, limit)
		return e
	})
	return out, e
}

type GalleryNeighbors struct {
	Current  Asset        `json:"current"`
	Previous *Asset       `json:"previous,omitempty"`
	Next     *Asset       `json:"next,omitempty"`
	Count    uint64       `json:"count"`
	Query    GalleryQuery `json:"query"`
	Snapshot string       `json:"snapshot"`
}

// Neighbors seek the active item's key and scan only until the nearest matches.
// A token is mandatory: never silently navigate across a changed result set.
func (s *Store) GalleryNeighbors(query GalleryQuery, id, snapshot string) (GalleryNeighbors, error) {
	var out GalleryNeighbors
	q, e := query.Normalize()
	if e != nil || !validID(id) || snapshot == "" {
		return out, ErrGalleryInput
	}
	if q.filtered() {
		s.gate.Lock()
		defer s.gate.Unlock()
	}
	e = s.db.View(func(tx *bolt.Tx) error {
		r, e := s.galleryResultsFor(tx, q)
		if e != nil {
			return e
		}
		if _, e = r.validate(snapshot, false); e != nil {
			return e
		}
		if !r.includes([]byte(id)) {
			return ErrGalleryChanged
		}
		a, e := r.asset([]byte(id))
		if e != nil {
			return ErrGalleryChanged
		}
		if q.Sort=="manual" {out=GalleryNeighbors{Current:a,Count:r.count,Query:q,Snapshot:encodeGalleryCursor(r.token)};for i,k:=range r.ordered{if string(k)==id{if i>0{x,e:=r.asset(r.ordered[i-1]);if e!=nil{return e};out.Previous=&x};if i+1<len(r.ordered){x,e:=r.asset(r.ordered[i+1]);if e!=nil{return e};out.Next=&x};return nil}};return ErrGalleryChanged}
		key := galleryKeys(a)[q.Sort]
		if !bytes.Equal(r.index.Get(key), []byte(id)) {
			return ErrGalleryChanged
		}
		out.Current, out.Count, out.Query, out.Snapshot = a, r.count, q, encodeGalleryCursor(r.token)
		c := r.index.Cursor()
		c.Seek(key)
		for k, v := c.Prev(); k != nil; k, v = c.Prev() {
			if r.includes(v) {
				a, e := r.asset(v)
				if e != nil {
					return e
				}
				out.Previous = &a
				break
			}
		}
		c.Seek(key)
		for k, v := c.Next(); k != nil; k, v = c.Next() {
			if r.includes(v) {
				a, e := r.asset(v)
				if e != nil {
					return e
				}
				out.Next = &a
				break
			}
		}
		return nil
	})
	return out, e
}
