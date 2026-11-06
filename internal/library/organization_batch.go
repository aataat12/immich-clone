package library

import (
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

const organizationBatchesBucket = "organization_batches_v3"
const maxOrganizationBatches = 64

var ErrBatchExpired = errors.New("operation unavailable or expired; review a new selection")

type OrganizationTarget struct {
	ID       string    `json:"id"`
	Added    time.Time `json:"added"`
	Name     string    `json:"name"`
	Expected string    `json:"expected"`
	Outcome  string    `json:"outcome"`
}
type OrganizationBatch struct {
	Refreshed  bool                 `json:"refreshed,omitempty"`
	ID         string               `json:"id"`
	Created    time.Time            `json:"created"`
	Action     string               `json:"action"`
	Tag        string               `json:"tag,omitempty"`
	AlbumID    string               `json:"album_id,omitempty"`
	AlbumToken string               `json:"album_token,omitempty"`
	AlbumTitle string               `json:"album_title,omitempty"`
	Targets    []OrganizationTarget `json:"targets"`
	Query      GalleryQuery         `json:"query"`
	Status     string               `json:"status"`
}
type OrganizationBatchRequest struct {
	Action   string               `json:"action"`
	Tag      string               `json:"tag"`
	AlbumID  string               `json:"album_id"`
	Query    GalleryQuery         `json:"query"`
	Snapshot string               `json:"snapshot"`
	Targets  []OrganizationTarget `json:"targets"`
}

func batchActionOK(action, tag, album string) bool {
	switch action {
	case "favorite_on", "favorite_off", "hide":
		return tag == "" && album == ""
	case "add_tag", "remove_tag":
		return album == "" && strings.TrimSpace(tag) != "" && textOK(tag, 64) && !strings.Contains(tag, ",")
	case "add_album", "remove_album":
		return tag == "" && validID(album)
	}
	return false
}
func batchTargetHash(tx *bolt.Tx, a Asset, album string) string {
	// Stable across reopen/restore, but sensitive to the reviewed mutable state.
	// AI results and preview progress deliberately do not invalidate organization.
	h := sha256.New()
	b, _ := json.Marshal(struct {
		ID     string
		Added  time.Time
		Hidden bool
	}{a.ID, a.Added, a.Hidden})
	h.Write(b)
	for _, name := range []string{assetOrganizationBucket, tagsBucket} {
		if bucket := tx.Bucket([]byte(name)); bucket != nil {
			h.Write(bucket.Get([]byte(a.ID)))
		}
		h.Write([]byte{0})
	}
	if bucket := tx.Bucket([]byte(albumMembersBucket)); bucket != nil && album != "" {
		h.Write(bucket.Get([]byte(album + ":" + a.ID)))
	}
	return hex.EncodeToString(h.Sum(nil))
}
func batchAlbumStamp(tx *bolt.Tx, id string) (string, error) {
	if id == "" {
		return "", nil
	}
	a, e := curationAlbum(tx, id)
	if e != nil {
		return "", e
	}
	a.Token = ""
	raw, _ := json.Marshal(a)
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:]), nil
}
func (s *Store) batchWriteReady() error {
	if s.offline != nil {
		return ErrCurationInput
	}
	free, e := AvailableBytes(s.root)
	if e != nil {
		return e
	}
	if free < 16<<20 {
		return ErrCurationCapacity
	}
	return nil
}

// Preparing a preview freezes exact explicit targets and makes no asset edits.
// Server-issued IDs can only be continued, never recreated by a retry.
func (s *Store) PrepareOrganizationBatch(in OrganizationBatchRequest) (OrganizationBatch, error) {
	var out OrganizationBatch
	q, e := in.Query.Normalize()
	if e != nil || len(in.Targets) < 1 || len(in.Targets) > 500 || !batchActionOK(in.Action, in.Tag, in.AlbumID) {
		return out, ErrCurationInput
	}
	s.gate.Lock()
	defer s.gate.Unlock()
	if e = s.batchWriteReady(); e != nil {
		return out, e
	}
	e = s.db.Update(func(tx *bolt.Tx) error {
		results, e := s.galleryResultsFor(tx, q)
		if e != nil {
			return e
		}
		_, e = results.validate(in.Snapshot, false)
		refreshed := errors.Is(e, ErrGalleryChanged)
		if e != nil && !refreshed {
			return e
		}
		stamp, e := batchAlbumStamp(tx, in.AlbumID)
		if e != nil {
			return e
		}
		out = OrganizationBatch{Refreshed: refreshed, Created: time.Now().UTC(), Action: in.Action, Tag: strings.TrimSpace(in.Tag), AlbumID: in.AlbumID, AlbumToken: stamp, Query: q, Status: "paused", Targets: []OrganizationTarget{}}
		if in.AlbumID != "" {
			a, e := curationAlbum(tx, in.AlbumID)
			if e != nil {
				return e
			}
			out.AlbumTitle = a.Title
		}
		seen := map[string]bool{}
		for _, target := range in.Targets {
			a, e := curationAsset(tx, target.ID)
			if e != nil {
				return e
			}
			if seen[a.ID] || !a.Added.Equal(target.Added) || galleryGroup(a) != q.Visibility || !results.includes([]byte(a.ID)) {
				return ErrCurationConflict
			}
			seen[a.ID] = true
			out.Targets = append(out.Targets, OrganizationTarget{ID: a.ID, Added: a.Added, Name: a.Name, Expected: batchTargetHash(tx, a, in.AlbumID), Outcome: "pending"})
		}
		if e = ensureCuration(tx); e != nil {
			return e
		}
		b, e := tx.CreateBucketIfNotExists([]byte(organizationBatchesBucket))
		if e != nil {
			return e
		}
		// Thirty-day expiry, maximum 64 receipts. Unfinished unexpired work is never
		// evicted to make room; completed receipts retire oldest first at capacity.
		var completed []OrganizationBatch
		var expired []string
		if e = b.ForEach(func(k, v []byte) error {
			var old OrganizationBatch
			if e := json.Unmarshal(v, &old); e != nil {
				return e
			}
			if time.Since(old.Created) > 30*24*time.Hour {
				expired = append(expired, string(k))
				return nil
			}
			if old.Status == "done" {
				completed = append(completed, old)
			}
			return nil
		}); e != nil {
			return e
		}
		for _, id := range expired {
			if e = curationWrite(tx, organizationBatchesBucket, id, nil); e != nil {
				return e
			}
		}
		if b.Stats().KeyN >= maxOrganizationBatches {
			sort.Slice(completed, func(i, j int) bool { return completed[i].Created.Before(completed[j].Created) })
			if len(completed) == 0 {
				return ErrCurationLimit
			}
			if e = curationWrite(tx, organizationBatchesBucket, completed[0].ID, nil); e != nil {
				return e
			}
		}
		out.ID, e = newFaceID()
		if e != nil {
			return e
		}
		return curationWrite(tx, organizationBatchesBucket, out.ID, out)
	})
	return out, e
}
func readOrganizationBatch(tx *bolt.Tx, id string) (OrganizationBatch, error) {
	var out OrganizationBatch
	if !validID(id) {
		return out, ErrCurationInput
	}
	e := curationRead(tx, organizationBatchesBucket, id, &out)
	if errors.Is(e, os.ErrNotExist) || time.Since(out.Created) > 30*24*time.Hour {
		return out, ErrBatchExpired
	}
	return out, e
}
func (s *Store) OrganizationBatch(id string) (OrganizationBatch, error) {
	var out OrganizationBatch
	e := s.db.View(func(tx *bolt.Tx) error { var e error; out, e = readOrganizationBatch(tx, id); return e })
	return out, e
}

type OrganizationBatchSummary struct {
	ID      string    `json:"id"`
	Created time.Time `json:"created"`
	Action  string    `json:"action"`
	Status  string    `json:"status"`
	Count   int       `json:"count"`
}

func (s *Store) OrganizationBatches() ([]OrganizationBatchSummary, error) {
	out := []OrganizationBatchSummary{}
	e := s.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(organizationBatchesBucket))
		if b == nil {
			return nil
		}
		return b.ForEach(func(k, v []byte) error {
			var row OrganizationBatch
			if e := json.Unmarshal(v, &row); e != nil {
				return e
			}
			if time.Since(row.Created) <= 30*24*time.Hour {
				out = append(out, OrganizationBatchSummary{ID: row.ID, Created: row.Created, Action: row.Action, Status: row.Status, Count: len(row.Targets)})
			}
			return nil
		})
	})
	sort.Slice(out, func(i, j int) bool { return out[i].Created.After(out[j].Created) })
	return out, e
}

// Each explicit Continue commits at most 25 targets and their outcomes together.
// There is no automatic worker: reopen and backup restore always leave work paused.
func (s *Store) ContinueOrganizationBatch(id string, expected int) (OrganizationBatch, error) {
	return s.resolveOrganizationBatch(id, expected, false)
}
func (s *Store) CancelOrganizationBatch(id string, expected int) (OrganizationBatch, error) {
	return s.resolveOrganizationBatch(id, expected, true)
}
func (s *Store) resolveOrganizationBatch(id string, expected int, cancel bool) (OrganizationBatch, error) {
	var out OrganizationBatch
	s.gate.Lock()
	defer s.gate.Unlock()
	if e := s.batchWriteReady(); e != nil {
		return out, e
	}
	e := s.db.Update(func(tx *bolt.Tx) error {
		var e error
		out, e = readOrganizationBatch(tx, id)
		if e != nil {
			return e
		}
		processed := 0
		for _, t := range out.Targets {
			if t.Outcome != "pending" {
				processed++
			}
		}
		if expected < 0 || expected > len(out.Targets) {
			return ErrCurationInput
		}
		if expected != processed || out.Status == "done" {
			return nil
		}
		if cancel {
			for i := range out.Targets {
				if out.Targets[i].Outcome == "pending" {
					out.Targets[i].Outcome = "canceled"
				}
			}
			out.Status = "done"
			return curationWrite(tx, organizationBatchesBucket, id, out)
		}
		stamp, e := batchAlbumStamp(tx, out.AlbumID)
		albumConflict := e != nil || stamp != out.AlbumToken
		n := 0
		for i := range out.Targets {
			target := &out.Targets[i]
			if target.Outcome != "pending" {
				continue
			}
			if n >= 25 {
				break
			}
			n++
			a, e := curationAsset(tx, target.ID)
			if e != nil || !a.Added.Equal(target.Added) {
				target.Outcome = "missing"
				continue
			}
			if albumConflict || batchTargetHash(tx, a, out.AlbumID) != target.Expected {
				target.Outcome = "conflict"
				continue
			}
			changed, e := applyOrganizationTarget(tx, out, a)
			if e != nil {
				return e
			}
			target.Outcome = "skipped"
			if changed {
				target.Outcome = "saved"
			}
		}
		if !albumConflict {
			out.AlbumToken, e = batchAlbumStamp(tx, out.AlbumID)
			if e != nil {
				return e
			}
		}
		out.Status = "done"
		for _, t := range out.Targets {
			if t.Outcome == "pending" {
				out.Status = "paused"
			}
		}
		return curationWrite(tx, organizationBatchesBucket, id, out)
	})
	return out, e
}
func applyOrganizationTarget(tx *bolt.Tx, b OrganizationBatch, a Asset) (bool, error) {
	switch b.Action {
	case "hide":
		if a.Hidden {
			return false, nil
		}
		a.Hidden = true
		return true, saveAsset(tx, a)
	case "favorite_on", "favorite_off":
		o, e := assetOrganization(tx, a.ID)
		if e != nil {
			return false, e
		}
		favorite := b.Action == "favorite_on"
		if o.Favorite == favorite {
			return false, nil
		}
		_, e = editCurationTx(tx, CurationEdit{Action: "set_asset", AssetID: a.ID, Added: a.Added, Expected: o.Token, Favorite: favorite, Title: o.Title, Caption: o.Caption, Notes: o.Notes})
		return true, e
	case "add_tag", "remove_tag":
		tags := []string{}
		if e := readJSON(tx.Bucket([]byte(tagsBucket)), a.ID, &tags); e != nil && !errors.Is(e, os.ErrNotExist) {
			return false, e
		}
		found := false
		clean := []string{}
		for _, t := range tags {
			if strings.EqualFold(t, b.Tag) {
				found = true
				if b.Action == "remove_tag" {
					continue
				}
			}
			clean = append(clean, t)
		}
		if b.Action == "add_tag" {
			if found {
				return false, nil
			}
			if len(clean) >= 32 {
				return false, ErrCurationLimit
			}
			clean = append(clean, b.Tag)
		} else if !found {
			return false, nil
		}
		sort.Strings(clean)
		return true, writeJSON(tx.Bucket([]byte(tagsBucket)), a.ID, clean)
	case "add_album", "remove_album":
		album, e := curationAlbum(tx, b.AlbumID)
		if e != nil {
			return false, e
		}
		exists := tx.Bucket([]byte(albumMembersBucket)).Get([]byte(b.AlbumID+":"+a.ID)) != nil
		add := b.Action == "add_album"
		if exists == add {
			return false, nil
		}
		action := "remove_member"
		if add {
			action = "add_member"
		}
		_, e = editCurationTx(tx, CurationEdit{Action: action, AlbumID: album.ID, Expected: album.Token, AssetID: a.ID, Added: a.Added})
		return true, e
	}
	return false, ErrCurationInput
}

func validateOrganizationBatches(tx *bolt.Tx) (uint64, error) {
	b := tx.Bucket([]byte(organizationBatchesBucket))
	if b == nil {
		return 0, nil
	}
	if b.Stats().KeyN > maxOrganizationBatches {
		return 0, errCatalogInput
	}
	var size uint64
	e := b.ForEach(func(k, v []byte) error {
		var row OrganizationBatch
		if len(v) > 1<<20 || json.Unmarshal(v, &row) != nil || row.ID != string(k) || !validID(row.ID) || row.Created.IsZero() || !curationText(row.AlbumTitle, 256, false) || !batchActionOK(row.Action, row.Tag, row.AlbumID) || len(row.Targets) < 1 || len(row.Targets) > 500 || (row.Status != "paused" && row.Status != "done") {
			return errCatalogInput
		}
		if _, e := row.Query.Normalize(); e != nil {
			return errCatalogInput
		}
		seen := map[string]bool{}
		pending := false
		for _, t := range row.Targets {
			if !validID(t.ID) || seen[t.ID] || t.Added.IsZero() || !textOK(t.Name, 1024) || len(t.Expected) != 64 {
				return errCatalogInput
			}
			seen[t.ID] = true
			switch t.Outcome {
			case "pending":
				pending = true
			case "saved", "skipped", "conflict", "missing", "canceled":
			default:
				return errCatalogInput
			}
		}
		if (row.Status == "paused") != pending {
			return errCatalogInput
		}
		size += uint64(len(v))
		return nil
	})
	return size, e
}
