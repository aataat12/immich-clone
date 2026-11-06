package library

import (
	"encoding/json"
	"errors"
	"immichclone/internal/recognition"
	"math"
	"os"
	"sort"
	"strings"

	bolt "go.etcd.io/bbolt"
)

type reviewAsset struct {
	Name   string
	Hidden bool
}
type reviewReference struct {
	f StoredFace
	l FaceLabel
}

// Derived, per-Store and immutable while its committed bbolt transaction ID is
// current. Every successful write transaction invalidates it automatically.
// It is never serialized, copied to backups or trusted after reopen/rollback.
type organizationIndex struct {
	transaction   int
	assets        map[string]reviewAsset
	faces         []reviewReference
	references    []reviewReference
	names         []string
	namesByAsset  map[string][]string
	searchByAsset map[string]string
}

func (s *Store) organizationIndexFor(tx *bolt.Tx) (*organizationIndex, error) {
	if s.reviewIndex != nil && s.reviewIndex.transaction == tx.ID() {
		return s.reviewIndex, nil
	}
	index := &organizationIndex{transaction: tx.ID(), assets: map[string]reviewAsset{}, namesByAsset: map[string][]string{}, searchByAsset: map[string]string{}}
	if err := tx.Bucket([]byte("assets")).ForEach(func(k, v []byte) error {
		var a Asset
		if err := json.Unmarshal(v, &a); err != nil {
			return err
		}
		index.assets[string(k)] = reviewAsset{a.Name, a.Hidden}
		return nil
	}); err != nil {
		return nil, err
	}
	names := map[string]bool{}
	if err := tx.Bucket([]byte(facesBucket)).ForEach(func(k, v []byte) error {
		var f StoredFace
		if err := json.Unmarshal(v, &f); err != nil {
			return err
		}
		if !validFace(f, string(k)) {
			return ErrOrganizationInput
		}
		a, ok := index.assets[f.AssetID]
		if !ok || a.Hidden {
			return nil
		}
		var label FaceLabel
		if err := readJSON(tx.Bucket([]byte(labelsBucket)), f.ID, &label); err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}
		if label.Name != "" && !label.Dismissed {
			names[label.Name] = true
			index.namesByAsset[f.AssetID] = append(index.namesByAsset[f.AssetID], label.Name)
			if trustedFaceLabel(label) {
				index.references = append(index.references, reviewReference{f, label})
			}
		}
		// Only named reference embeddings stay resident. Unnamed vectors are read
		// for the requested page rather than retained for the whole face library.
		f.Embedding = nil
		index.faces = append(index.faces, reviewReference{f, label})
		return nil
	}); err != nil {
		return nil, err
	}
	for name := range names {
		index.names = append(index.names, name)
	}
	sort.Strings(index.names)
	for id, a := range index.assets {
		if a.Hidden {
			continue
		}
		terms := []string{a.Name}
		terms = append(terms, index.namesByAsset[id]...)
		var tags, dismissed []string
		var analysis Analysis
		for _, record := range []struct {
			bucket string
			value  any
		}{{tagsBucket, &tags}, {analysisBucket, &analysis}, {"dismissed_ai_tags_v1", &dismissed}} {
			if err := readJSON(tx.Bucket([]byte(record.bucket)), id, record.value); err != nil && !errors.Is(err, os.ErrNotExist) {
				return nil, err
			}
		}
		terms = append(terms, tags...)
		for _, tag := range analysis.Objects {
			blocked := false
			for _, d := range dismissed {
				if tag == d {
					blocked = true
					break
				}
			}
			if !blocked {
				terms = append(terms, tag)
			}
		}
		index.searchByAsset[id] = strings.ToLower(strings.Join(terms, "\x00"))
	}
	s.reviewIndex = index
	return index, nil
}

// Same review-only threshold/margin as the original implementation. No naming
// or clustering is applied here; confirmation remains an explicit user action.
func suggestReviewFace(subject StoredFace, refs []reviewReference) (string, float64) {
	scores := map[string]float64{}
	for _, ref := range refs {
		if !trustedFaceLabel(ref.l) || !recognition.CompatibleEmbeddings(ref.f.Provider, subject.Provider) || len(ref.f.Embedding) != 128 || len(subject.Embedding) != 128 {
			continue
		}
		score := 0.
		for i, v := range subject.Embedding {
			score += v * ref.f.Embedding[i]
		}
		if score > scores[ref.l.Name] {
			scores[ref.l.Name] = score
		}
	}
	best, runner := 0., 0.
	name := ""
	for candidate, score := range scores {
		if score > best {
			runner = best
			best = score
			name = candidate
		} else if score > runner {
			runner = score
		}
	}
	if best < .363 || best-runner < .05 {
		return "", 0
	}
	return name, math.Min(1, best)
}
