package library

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	bolt "go.etcd.io/bbolt"
	"os"
	"sort"
	"strings"
	"time"
)

type gallerySearchRow struct {
	sources      []string
	batches      map[string][]string
	organization string
	favorite     bool
	albums       map[string]uint64
	hidden       bool
	terms        string
	filename     string
	mime         string
	captured     string
	added        time.Time
	people       []string
	tags         []string
}

// This derived metadata cache never retains image bytes or face embeddings and
// is never exported. Reopen and every committed catalog edit invalidate it.
type gallerySearchIndex struct {
	batches     []ImportSession
	state       string
	transaction int
	assets      map[string]gallerySearchRow
	queries     map[string]map[string]bool
	queryKeys   map[string][][]byte
}

func (s *Store) gallerySearchFor(tx *bolt.Tx) (*gallerySearchIndex, error) {
	if !tx.Writable() && s.gallerySearch != nil && s.gallerySearch.transaction == tx.ID() {
		return s.gallerySearch, nil
	}
	index := &gallerySearchIndex{transaction: tx.ID(), assets: map[string]gallerySearchRow{}, queries: map[string]map[string]bool{}}
	if e := tx.Bucket([]byte("assets")).ForEach(func(k, v []byte) error {
		var a Asset
		if e := json.Unmarshal(v, &a); e != nil {
			return e
		}
		if a.ID != string(k) || !validID(a.ID) {
			return ErrGalleryInput
		}
		var tags, dismissed []string
		var analysis Analysis
		for _, r := range []struct {
			bucket string
			value  any
		}{{tagsBucket, &tags}, {analysisBucket, &analysis}, {"dismissed_ai_tags_v1", &dismissed}} {
			if e := readJSON(tx.Bucket([]byte(r.bucket)), a.ID, r.value); e != nil && !errors.Is(e, os.ErrNotExist) {
				return e
			}
		}
		terms := append([]string{a.Name}, tags...)
		activeTags := append([]string{}, tags...)
		blocked := map[string]bool{}
		for _, tag := range dismissed {
			blocked[tag] = true
		}
		for _, tag := range analysis.Objects {
			if !blocked[tag] {
				terms = append(terms, tag)
				activeTags = append(activeTags, tag)
			}
		}
		for i, tag := range activeTags {
			activeTags[i] = strings.ToLower(strings.TrimSpace(tag))
		}
		index.assets[a.ID] = gallerySearchRow{sources: append([]string{}, a.Sources...), hidden: a.Hidden, terms: strings.ToLower(strings.Join(terms, "\x00")), filename: strings.ToLower(a.Name), mime: a.MIME, captured: a.Photo.Captured, added: a.Added, tags: activeTags}
		return nil
	}); e != nil {
		return nil, e
	}
	if e := index.recordImportSources(tx); e != nil { return nil, e }
	// Only assigned names are searchable; inferred suggestions never enter terms.
	// Hidden media has its own searchable view without exposing it in the library.
	if e := tx.Bucket([]byte(facesBucket)).ForEach(func(k, v []byte) error {
		var face struct {
			ID      string `json:"id"`
			AssetID string `json:"asset_id"`
		}
		if e := json.Unmarshal(v, &face); e != nil {
			return e
		}
		if face.ID != string(k) || !validID(face.ID) || !validID(face.AssetID) {
			return ErrOrganizationInput
		}
		row, exists := index.assets[face.AssetID]
		if !exists {
			return nil
		}
		var label FaceLabel
		if e := readJSON(tx.Bucket([]byte(labelsBucket)), face.ID, &label); e != nil && !errors.Is(e, os.ErrNotExist) {
			return e
		}
		if label.Name != "" && !label.Dismissed {
			row.terms += "\x00" + strings.ToLower(label.Name)
			row.people = append(row.people, strings.ToLower(strings.TrimSpace(label.Name)))
			index.assets[face.AssetID] = row
		}
		return nil
	}); e != nil {
		return nil, e
	}
	if b := tx.Bucket([]byte(assetOrganizationBucket)); b != nil {
		if e := b.ForEach(func(k, v []byte) error {
			var o AssetOrganization
			if e := json.Unmarshal(v, &o); e != nil {
				return e
			}
			row := index.assets[string(k)]
			row.favorite = o.Favorite
			row.terms += "\x00" + strings.ToLower(strings.Join([]string{o.Title, o.Caption, o.Notes}, "\x00"))
			sum := sha256.Sum256(v)
			row.organization = hex.EncodeToString(sum[:])
			index.assets[string(k)] = row
			return nil
		}); e != nil {
			return nil, e
		}
	}
	if b := tx.Bucket([]byte(albumMembersBucket)); b != nil {
		if e := b.ForEach(func(k, v []byte) error {
			var m AlbumMember
			if e := json.Unmarshal(v, &m); e != nil {
				return e
			}
			row := index.assets[m.AssetID]
			if row.albums == nil {
				row.albums = map[string]uint64{}
			}
			row.albums[m.AlbumID] = m.Position
			index.assets[m.AssetID] = row
			return nil
		}); e != nil {
			return nil, e
		}
	}
	// Snapshot the shared matching/order inputs, not unrelated transaction IDs.
	// Preview dimensions/status and durable batch receipts do not redefine results.
	ids := make([]string, 0, len(index.assets))
	for id := range index.assets {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	digest := sha256.New()
	for _, id := range ids {
		row := index.assets[id]
		raw, e := json.Marshal(struct {
			ID           string
			Hidden       bool
			Terms        string
			Filename     string
			MIME         string
			Captured     string
			Added        string
			People       []string
			Tags         []string
			Favorite     bool
			Albums       map[string]uint64
			Organization string
			Sources      []string
			Batches      map[string][]string
		}{id, row.hidden, row.terms, row.filename, row.mime, row.captured, row.added.UTC().Format(time.RFC3339Nano), row.people, row.tags, row.favorite, row.albums, row.organization, row.sources, row.batches})
		if e != nil {
			return nil, e
		}
		digest.Write(raw)
	}
	index.state = hex.EncodeToString(digest.Sum(nil))
	// A rolled-back write transaction may reuse its ID for a later, different
	// commit. Never publish a cache built from uncommitted transaction state.
	if !tx.Writable() {
		s.gallerySearch = index
	}
	return index, nil
}

// Keep at most eight query membership sets per committed metadata snapshot.
// Count and paging reuse this set; the browser never needs the full ID list.
func (index *gallerySearchIndex) matching(q GalleryQuery) map[string]bool {
	key := q.key()
	if ids, ok := index.queries[key]; ok {
		return ids
	}
	ids := map[string]bool{}
	for id, row := range index.assets {
		if q.matches(row) {
			ids[id] = true
		}
	}
	if len(index.queries) >= 8 {
		index.queries = map[string]map[string]bool{}
		index.queryKeys = nil
	}
	index.queries[key] = ids
	return ids
}

func (index *gallerySearchIndex) orderedKeys(q GalleryQuery) [][]byte {
	key := q.key()
	if keys, ok := index.queryKeys[key]; ok {
		return keys
	}
	ids := index.matching(q)
	keys := make([][]byte, 0, len(ids))
	for id := range ids {
		row := index.assets[id]
		a := Asset{ID: id, Name: row.filename, Added: row.added, Photo: PhotoMetadata{Captured: row.captured}}
		if q.Sort == "manual" {
			keys = append(keys, []byte(id))
		} else {
			keys = append(keys, galleryKeys(a)[q.Sort])
		}
	}
	sort.Slice(keys, func(i, j int) bool {
		if q.Sort == "manual" {
			return index.assets[string(keys[i])].albums[q.Album] < index.assets[string(keys[j])].albums[q.Album]
		}
		return bytes.Compare(keys[i], keys[j]) < 0
	})
	if index.queryKeys == nil {
		index.queryKeys = map[string][][]byte{}
	}
	index.queryKeys[key] = keys
	return keys
}
