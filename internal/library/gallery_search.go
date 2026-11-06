package library

import (
	"encoding/json"
	"errors"
	"os"
	"strings"
	bolt "go.etcd.io/bbolt"
)

type gallerySearchRow struct {
	hidden bool
	terms string
}

// This derived metadata cache never retains image bytes or face embeddings and
// is never exported. Reopen and every committed catalog edit invalidate it.
type gallerySearchIndex struct {
	transaction int
	assets map[string]gallerySearchRow
}

func (s *Store) gallerySearchFor(tx *bolt.Tx) (*gallerySearchIndex, error) {
	if s.gallerySearch != nil && s.gallerySearch.transaction == tx.ID() { return s.gallerySearch, nil }
	index := &gallerySearchIndex{transaction: tx.ID(), assets: map[string]gallerySearchRow{}}
	if e := tx.Bucket([]byte("assets")).ForEach(func(k, v []byte) error {
		var a Asset
		if e := json.Unmarshal(v, &a); e != nil { return e }
		if a.ID != string(k) || !validID(a.ID) { return ErrGalleryInput }
		var tags, dismissed []string
		var analysis Analysis
		for _, r := range []struct{ bucket string; value any }{{tagsBucket, &tags}, {analysisBucket, &analysis}, {"dismissed_ai_tags_v1", &dismissed}} {
			if e := readJSON(tx.Bucket([]byte(r.bucket)), a.ID, r.value); e != nil && !errors.Is(e, os.ErrNotExist) { return e }
		}
		terms := append([]string{a.Name}, tags...)
		blocked := map[string]bool{}
		for _, tag := range dismissed { blocked[tag] = true }
		for _, tag := range analysis.Objects { if !blocked[tag] { terms = append(terms, tag) } }
		index.assets[a.ID] = gallerySearchRow{a.Hidden, strings.ToLower(strings.Join(terms, "\x00"))}
		return nil
	}); e != nil { return nil, e }
	// Only assigned names are searchable; inferred suggestions never enter terms.
	// Hidden media has its own searchable view without exposing it in the library.
	if e := tx.Bucket([]byte(facesBucket)).ForEach(func(k, v []byte) error {
		var face struct { ID string `json:"id"`; AssetID string `json:"asset_id"` }
		if e := json.Unmarshal(v, &face); e != nil { return e }
		if face.ID != string(k) || !validID(face.ID) || !validID(face.AssetID) { return ErrOrganizationInput }
		row, exists := index.assets[face.AssetID]
		if !exists { return nil }
		var label FaceLabel
		if e := readJSON(tx.Bucket([]byte(labelsBucket)), face.ID, &label); e != nil && !errors.Is(e, os.ErrNotExist) { return e }
		if label.Name != "" && !label.Dismissed {
			row.terms += "\x00" + strings.ToLower(label.Name)
			index.assets[face.AssetID] = row
		}
		return nil
	}); e != nil { return nil, e }
	s.gallerySearch = index
	return index, nil
}
