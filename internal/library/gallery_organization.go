package library

import (
	"bytes"
	"encoding/json"
	bolt "go.etcd.io/bbolt"
)

type GalleryOrganization struct {
	Favorite bool `json:"favorite"`
}

func galleryOrganization(tx *bolt.Tx, id string) (GalleryOrganization, error) {
	var out GalleryOrganization
	if b := tx.Bucket([]byte(assetOrganizationBucket)); b != nil {
		if raw := b.Get([]byte(id)); raw != nil {
			e := json.Unmarshal(raw, &out)
			return out, e
		}
	}
	return out, nil
}

// Manual album order uses the same matching set, snapshot and page envelope.
// Only metadata keys are cached; the response still reads at most limit assets.
func (r *galleryResults) manualPage(after string, limit int) (GalleryPage, error) {
	out := GalleryPage{Assets: []Asset{}, Total: r.total, Hidden: r.hidden, Count: r.count, Query: r.query, Snapshot: encodeGalleryCursor(r.token)}
	start := 0
	if after != "" {
		t, e := r.validate(after, true)
		if e != nil {
			return out, e
		}
		for i, k := range r.ordered {
			if bytes.Equal(k, t.Key) {
				start = i + 1
				break
			}
		}
		out.HasPrevious = start > 0
		if start > limit {
			t.Key = r.ordered[start-limit-1]
			out.Previous = encodeGalleryCursor(t)
		}
	}
	end := start + limit
	if end > len(r.ordered) {
		end = len(r.ordered)
	}
	for _, id := range r.ordered[start:end] {
		a, e := r.asset(id)
		if e != nil {
			return out, e
		}
		out.Assets = append(out.Assets, a)
		if out.Organization == nil {
			out.Organization = map[string]GalleryOrganization{}
		}
		o, e := galleryOrganization(r.tx, a.ID)
		if e != nil {
			return out, e
		}
		out.Organization[a.ID] = o
	}
	if end < len(r.ordered) {
		t := r.token
		t.Key = r.ordered[end-1]
		out.Next = encodeGalleryCursor(t)
	}
	return out, nil
}
