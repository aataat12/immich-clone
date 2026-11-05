package library

import (
	"encoding/json"
	"errors"
	bolt "go.etcd.io/bbolt"
	"strings"
	"unicode/utf8"
)

var errCatalogInput = errors.New("invalid catalog input")

// Validate authoritative display/input records before rebuilding derived indexes.
// Structural bbolt validation and hard resource limits remain restore-worker work.
func validateCatalogInput(tx *bolt.Tx) error {
	return tx.ForEach(func(name []byte, b *bolt.Bucket) error {
		var walk func(*bolt.Bucket, int) error
		walk = func(bucket *bolt.Bucket, depth int) error {
			if depth > 3 {
				return errCatalogInput
			}
			return bucket.ForEach(func(k, v []byte) error {
				if len(k) == 0 || len(k) > 1100 {
					return errCatalogInput
				}
				if v == nil {
					switch string(name) {
					case "assets", facesBucket, labelsBucket, tagsBucket, "dismissed_ai_tags_v1", analysisBucket, reanalysisBucket, deletionBucket:
						return errCatalogInput
					}
					return walk(bucket.Bucket(k), depth+1)
				}
				limit := 1 << 20
				if string(name) == "assets" {
					limit = 32 << 20
				}
				if len(v) > limit {
					return errCatalogInput
				}
				switch string(name) {
				case "assets", facesBucket, labelsBucket, tagsBucket, "dismissed_ai_tags_v1", analysisBucket, importBucket, reanalysisBucket, deletionBucket:
					if !utf8.Valid(v) {
						return errCatalogInput
					}
				}
				// Nested import records receive semantic checks after this size/tree walk.
				if depth != 0 {
					return nil
				}
				switch string(name) {
				case "assets":
					var a Asset
					if json.Unmarshal(v, &a) != nil || !validCatalogAsset(a, string(k)) {
						return errCatalogInput
					}
				case facesBucket:
					var f StoredFace
					if json.Unmarshal(v, &f) != nil || !validFace(f, string(k)) {
						return errCatalogInput
					}
				case labelsBucket:
					var l FaceLabel
					if !validID(string(k)) || json.Unmarshal(v, &l) != nil || !textOK(l.Name, 128) || !textOK(l.Previous, 128) {
						return errCatalogInput
					}
				case tagsBucket, "dismissed_ai_tags_v1":
					var tags []string
					max := 32
					if string(name) != tagsBucket {
						max = 80
					}
					if !validID(string(k)) || json.Unmarshal(v, &tags) != nil || !validCatalogTags(tags, max) {
						return errCatalogInput
					}
				case deletionBucket:
					var d deletionRecord
					if json.Unmarshal(v, &d) != nil || !validDeletion(d, string(k)) {
						return errCatalogInput
					}
				case reanalysisBucket:
					var j reanalysisJob
					if json.Unmarshal(v, &j) != nil || !validReanalysis(j, string(k)) || !textOK(j.Provider, 128) || !textOK(j.MediaProvider, 128) || len(j.Error) > 4096 || !validCatalogTags(j.Objects, 80) || (j.Video != nil && !validCatalogVideo(j.Video)) {
						return errCatalogInput
					}
					for _, face := range j.Faces {
						if !textOK(face.Provider, 128) {
							return errCatalogInput
						}
					}
				case analysisBucket:
					var a Analysis
					if !validID(string(k)) || json.Unmarshal(v, &a) != nil || a.AssetID != string(k) || !textOK(a.Provider, 128) || len(a.Error) > 4096 || !validCatalogTags(a.Objects, 80) {
						return errCatalogInput
					}
					switch a.Status {
					case "queued", "ready", "failed", "paused", "cancelled":
					default:
						return errCatalogInput
					}
					if a.Video != nil && !validCatalogVideo(a.Video) {
						return errCatalogInput
					}
				}
				return nil
			})
		}
		if err := walk(b, 0); err != nil {
			return err
		}
		if string(name) == importBucket {
			return validateCatalogImports(b)
		}
		return nil
	})
}
func validCatalogTags(tags []string, max int) bool {
	if len(tags) > max {
		return false
	}
	for _, tag := range tags {
		if tag == "" || !textOK(tag, 64) || strings.Contains(tag, ",") {
			return false
		}
	}
	return true
}
func validCatalogAsset(a Asset, key string) bool {
	if !validID(key) || a.ID != key || !textOK(a.Name, 1024) || !textOK(a.MIME, 128) || a.Size < 0 || a.Size > MaxUpload || a.Width < 0 || a.Height < 0 || !textOK(a.Preview, 16) || a.PreviewRevision < 0 || len(a.Error) > 4096 || len(a.Sources) > 4096 {
		return false
	}
	for _, source := range a.Sources {
		if _, e := cleanRelative(source); e != nil {
			return false
		}
	}
	p := a.Photo
	if p.Orientation < 0 || p.Orientation > 8 || !textOK(p.Make, 256) || !textOK(p.Model, 256) || !textOK(p.Captured, 64) || !textOK(p.Warning, 128) {
		return false
	}
	return true
}

func validateCatalogImports(b *bolt.Bucket) error {
	sessions := 0
	return b.ForEach(func(k, v []byte) error {
		sessions++
		if sessions > 128 || v != nil || !validID(string(k)) {
			return errCatalogInput
		}
		child := b.Bucket(k)
		var m ImportSession
		if readJSON(child, "meta", &m) != nil || m.ID != string(k) || m.TotalFiles < 1 || m.TotalFiles > MaxImportFiles || m.TotalBytes < 1 || m.TotalBytes > int64(MaxImportFiles)*MaxUpload || m.AcceptedFiles < 0 || m.AcceptedFiles > m.TotalFiles || m.AcceptedBytes < 0 || m.AcceptedBytes > m.TotalBytes || m.NewFiles < 0 || m.Duplicates < 0 || m.NewFiles+m.Duplicates != m.AcceptedFiles {
			return errCatalogInput
		}
		switch m.Status {
		case "running", "paused", "cancelled", "completed":
		default:
			return errCatalogInput
		}
		entries := 0
		var total int64
		if e := child.ForEach(func(key, value []byte) error {
			if string(key) == "meta" {
				return nil
			}
			var entry ImportEntry
			if value == nil || json.Unmarshal(value, &entry) != nil || entry.Index < 0 || entry.Index >= m.TotalFiles || string(key) != entryKey(entry.Index) || !textOK(entry.Name, 256) || entry.Name == "" || strings.ContainsAny(entry.Name, "/\\") || entry.Size <= 0 || entry.Size > MaxUpload || entry.Modified < 0 || len(entry.Error) > 4096 || (entry.AssetID != "" && !validID(entry.AssetID)) {
				return errCatalogInput
			}
			if entry.Relative != "" {
				if _, e := cleanRelative(entry.Relative); e != nil {
					return errCatalogInput
				}
			}
			total += entry.Size
			entries++
			return nil
		}); e != nil {
			return e
		}
		if entries != m.TotalFiles || total != m.TotalBytes {
			return errCatalogInput
		}
		return nil
	})
}

func validCatalogVideo(v *VideoProgress) bool {
	if !validVideo(v) {
		return false
	}
	for _, sample := range v.Samples {
		if !textOK(sample.Provider, 128) || !validCatalogTags(sample.Objects, 80) || sample.Width < 0 || sample.Height < 0 {
			return false
		}
	}
	return true
}
