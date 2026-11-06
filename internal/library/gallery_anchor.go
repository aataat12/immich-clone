package library

import (
	"bytes"
	bolt "go.etcd.io/bbolt"
	"sort"
)

// PreviousPage is an indexed, bounded reverse seek. The empty cursor denotes
// page one; HasPrevious distinguishes it from the first page itself.
func (r *galleryResults) previousPage(after []byte, limit int) string {
	if len(after) == 0 {
		return ""
	}
	if r.matches != nil {
		i := sort.Search(len(r.ordered), func(i int) bool { return bytes.Compare(r.ordered[i], after) >= 0 }) - limit
		if i < 0 || i >= len(r.ordered) {
			return ""
		}
		t := r.token
		t.Key = r.ordered[i]
		return encodeGalleryCursor(t)
	}
	c := r.index.Cursor()
	k, v := c.Seek(after)
	for n := 0; n < limit; {
		k, v = c.Prev()
		if k == nil {
			return ""
		}
		if r.includes(v) {
			n++
		}
	}
	t := r.token
	t.Key = append([]byte(nil), k...)
	return encodeGalleryCursor(t)
}

type GalleryAnchor struct {
	Page   GalleryPage `json:"page"`
	After  string      `json:"after"`
	Number uint64      `json:"number"`
	Asset  string      `json:"asset"`
}

// GalleryAnchorPage scans only index keys/IDs to locate a page, then reads at
// most limit assets. It never materializes preceding asset records or originals.
// Empty snapshot permits explicit refresh/re-anchor after an invalidation.
func (s *Store) GalleryAnchorPage(query GalleryQuery, id, snapshot string, limit int) (GalleryAnchor, error) {
	var out GalleryAnchor
	q, e := query.Normalize()
	if e != nil || !validID(id) || limit < 1 || limit > 100 {
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
		if snapshot != "" {
			if _, e = r.validate(snapshot, false); e != nil {
				return e
			}
		}
		if !r.includes([]byte(id)) {
			return ErrGalleryChanged
		}
		a, e := r.asset([]byte(id))
		if e != nil {
			return ErrGalleryChanged
		}
		target := galleryKeys(a)[q.Sort]
		c := r.index.Cursor()
		var before, last []byte
		var ordinal uint64
		found := false
		for k, v := c.First(); k != nil; k, v = c.Next() {
			if !r.includes(v) {
				continue
			}
			if ordinal%uint64(limit) == 0 {
				before = append(before[:0], last...)
			}
			if bytes.Equal(k, target) && string(v) == id {
				found = true
				break
			}
			last = append(last[:0], k...)
			ordinal++
		}
		if !found {
			return ErrGalleryChanged
		}
		out.Number = ordinal/uint64(limit) + 1
		out.Asset = id
		if len(before) > 0 {
			t := r.token
			t.Key = before
			out.After = encodeGalleryCursor(t)
		}
		out.Page, e = r.page(out.After, limit)
		return e
	})
	return out, e
}

func (r *galleryResults) page(after string, limit int) (GalleryPage, error) {
	out := GalleryPage{Assets: []Asset{}, Total: r.total, Hidden: r.hidden, Count: r.count, Query: r.query, Snapshot: encodeGalleryCursor(r.token)}
	c := r.index.Cursor()
	k, v := c.First()
	if after != "" {
		token, e := r.validate(after, true)
		if e != nil {
			return out, e
		}
		out.HasPrevious = true
		out.Previous = r.previousPage(token.Key, limit)
		c.Seek(token.Key)
		k, v = c.Next()
	}
	var last []byte
	for ; k != nil; k, v = c.Next() {
		if !r.includes(v) {
			continue
		}
		if len(out.Assets) == limit {
			t := r.token
			t.Key = last
			out.Next = encodeGalleryCursor(t)
			break
		}
		a, e := r.asset(v)
		if e != nil {
			return out, e
		}
		out.Assets = append(out.Assets, a)
		last = append(last[:0], k...)
	}
	return out, nil
}
