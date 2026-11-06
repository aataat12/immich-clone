package library

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	bolt "go.etcd.io/bbolt"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"testing"
	"time"
)

func galleryBrowseReceipt(t *testing.T, s *Store, sources []string, ids []int) string {
	t.Helper()
	files := []ImportFile{}
	for i, source := range sources {
		files = append(files, ImportFile{Name: fmt.Sprintf("synthetic-%d.jpg", i), Relative: source, Size: 1})
	}
	m, e := s.CreateImport(files)
	if e != nil {
		t.Fatal(e)
	}
	e = s.db.Update(func(tx *bolt.Tx) error {
		child := tx.Bucket([]byte(importBucket)).Bucket([]byte(m.ID))
		for i, id := range ids {
			entry := ImportEntry{ImportFile: files[i], Index: i, AssetID: fmt.Sprintf("%064x", id), Duplicate: i > 0}
			if e := writeJSON(child, entryKey(i), entry); e != nil {
				return e
			}
		}
		m.AcceptedFiles, m.Status = len(ids), "completed"
		m.AcceptedBytes, m.NewFiles, m.Duplicates = int64(len(ids)), 1, len(ids)-1
		return writeJSON(child, "meta", m)
	})
	if e != nil {
		t.Fatal(e)
	}
	return m.ID
}

func galleryBrowseIDs(t *testing.T, s *Store, q GalleryQuery) []string {
	t.Helper()
	ids := []string{}
	after, snapshot := "", ""
	for {
		p, e := s.QueryGalleryPage(q, after, 1)
		if e != nil {
			t.Fatal(e)
		}
		if snapshot != "" && snapshot != p.Snapshot {
			t.Fatal("snapshot changed while paging")
		}
		snapshot = p.Snapshot
		for _, a := range p.Assets {
			ids = append(ids, a.ID)
			n, e := s.GalleryNeighbors(q, a.ID, p.Snapshot)
			if e != nil || n.Current.ID != a.ID || n.Count != p.Count {
				t.Fatal("neighbors", e, n)
			}
			anchor, e := s.GalleryAnchorPage(q, a.ID, p.Snapshot, 1)
			if e != nil || len(anchor.Page.Assets) != 1 || anchor.Page.Assets[0].ID != a.ID {
				t.Fatal("anchor", e)
			}
		}
		if p.Next == "" {
			if uint64(len(ids)) != p.Count {
				t.Fatal("count mismatch")
			}
			break
		}
		after = p.Next
	}
	return ids
}

func TestGalleryBrowseSourcesCaptionsAndCombinedFilters(t *testing.T) {
	s, _ := openTest(t)
	seedGalleryFoundation(t, s, 12)
	batch := galleryBrowseReceipt(t, s, []string{"Trip/Leap/a.jpg", "Trip/Leap/again.jpg", "TripTwo/other.jpg", "Trip/Day/b.jpg"}, []int{2, 2, 4, 6})
	other := galleryBrowseReceipt(t, s, []string{"Other/repeated.jpg"}, []int{2})
	id := fmt.Sprintf("%064x", 2)
	o, _ := s.AssetOrganization(id)
	curationEdit(t, s, CurationEdit{Action: "set_asset", AssetID: id, Added: o.Added, Expected: o.Token, Favorite: true, Title: "RÉSUMÉ <title>", Caption: "雪 Leap memory", Notes: "Private searchable note"})
	album := fmt.Sprintf("%064x", 99001)
	curationEdit(t, s, CurationEdit{Action: "create_album", AlbumID: album, Title: "Synthetic"})
	a, _ := s.AlbumPage(album, "", 1)
	curationEdit(t, s, CurationEdit{Action: "add_member", AlbumID: album, AssetID: id, Added: o.Added, Expected: a.Album.Token})
	q := GalleryQuery{Sort: "name", Visibility: "library", SourceFolder: "Trip", ImportBatch: batch, Favorite: "true", Album: album, Media: "photo", People: GalleryLabels{All: []string{"Alice", "Bob"}}, Tags: GalleryLabels{All: []string{"Even", "Trip"}, Exclude: []string{"blocked"}}, DateBasis: "capture", From: "2024-02-29T00:00:00", Until: "2024-03-01T00:00:00"}
	for _, text := range []string{"résumé", "雪 leap", "searchable note"} {
		q.Text = text
		if ids := galleryBrowseIDs(t, s, q); !reflect.DeepEqual(ids, []string{id}) {
			t.Fatal(text, ids)
		}
	}
	q.Text = ""
	q.ImportBatch = other
	if ids := galleryBrowseIDs(t, s, q); len(ids) != 0 {
		t.Fatal("folder matched another batch provenance", ids)
	}
	q.ImportBatch = batch
	q.SourceFolder = "TripTwo"
	if ids := galleryBrowseIDs(t, s, q); len(ids) != 0 {
		t.Fatal("folder prefix boundary", ids)
	}
	base := GalleryQuery{Sort: "name", Visibility: "library", ImportBatch: batch}
	if ids := galleryBrowseIDs(t, s, base); len(ids) != 3 {
		t.Fatal("duplicate receipts counted more than once", ids)
	}
	base.SourceFolder = "Trip"
	if ids := galleryBrowseIDs(t, s, base); len(ids) != 2 {
		t.Fatal("folder subtree", ids)
	}
	root, e := s.GallerySources(GalleryQuery{Sort: "name", Visibility: "library", ImportBatch: batch}, "", 1)
	if e != nil || len(root.Folders) != 1 || root.Folders[0].Path != "Trip" || root.Folders[0].Count != 2 || root.Next != "Trip" || root.RootCount != 3 {
		t.Fatal(root, e)
	}
	next, e := s.GallerySources(GalleryQuery{Sort: "name", Visibility: "library", ImportBatch: batch}, root.Next, 1)
	if e != nil || len(next.Folders) != 1 || next.Folders[0].Path != "TripTwo" || next.Next != "" {
		t.Fatal(next, e)
	}
	children, e := s.GallerySources(base, "", 100)
	if e != nil || len(children.Folders) != 2 || children.Folders[0].Path != "Trip/Day" || children.Folders[1].Count != 1 {
		t.Fatal(children, e)
	}
	for _, b := range children.Batches {
		if b.ID == other && b.Count != 0 {
			t.Fatal("batch choice crossed source provenance")
		}
	}
	// A recorded source can exist without any original folder being available.
	base.SourceFolder = "Trip/Leap"
	if ids := galleryBrowseIDs(t, s, base); !reflect.DeepEqual(ids, []string{id}) {
		t.Fatal(ids)
	}
	if e := s.db.Update(func(tx *bolt.Tx) error {
		child := tx.Bucket([]byte(importBucket)).Bucket([]byte(batch))
		for _, i := range []int{0, 1} {
			var entry ImportEntry
			if e := readJSON(child, entryKey(i), &entry); e != nil {
				return e
			}
			entry.Deleted = true
			if e := writeJSON(child, entryKey(i), entry); e != nil {
				return e
			}
		}
		return nil
	}); e != nil {
		t.Fatal(e)
	}
	if ids := galleryBrowseIDs(t, s, base); len(ids) != 0 {
		t.Fatal("deleted receipt retained membership", ids)
	}
}

func TestGalleryBrowseSourceSnapshotAndNormalization(t *testing.T) {
	s, _ := openTest(t)
	seedGalleryFoundation(t, s, 4)
	q := GalleryQuery{Sort: "name", Visibility: "library", SourceFolder: "."}
	p, e := s.QueryGalleryPage(q, "", 1)
	if e != nil || p.Count != 1 {
		t.Fatal(p, e)
	}
	if e := s.db.Update(func(tx *bolt.Tx) error {
		var a Asset
		if e := readJSON(tx.Bucket([]byte("assets")), fmt.Sprintf("%064x", 2), &a); e != nil {
			return e
		}
		a.Sources = []string{"Trip/new.jpg"}
		return saveAsset(tx, a)
	}); e != nil {
		t.Fatal(e)
	}
	if _, e := s.GalleryNeighbors(q, p.Assets[0].ID, p.Snapshot); !errors.Is(e, ErrGalleryChanged) {
		t.Fatal("source change did not invalidate snapshot", e)
	}
	for _, folder := range []string{"/absolute", "C:/absolute", "../Trip", "Trip/../Other", "Trip//Other", "Trip/.", "Trip\x00Other"} {
		q.SourceFolder = folder
		if _, e := q.Normalize(); !errors.Is(e, ErrGalleryInput) {
			t.Fatal("invalid source", folder, e)
		}
	}
	q.SourceFolder = "Trip\\Leap"
	n, e := q.Normalize()
	if e != nil || n.SourceFolder != "Trip/Leap" {
		t.Fatal(n, e)
	}
	q.ImportBatch = "invalid"
	if _, e := q.Normalize(); !errors.Is(e, ErrGalleryInput) {
		t.Fatal("invalid import batch", e)
	}
}

func TestGalleryBrowseCalendarCaptureUploadAndDST(t *testing.T) {
	s, _ := openTest(t)
	seedGalleryFoundation(t, s, 4)
	stamps := []string{"2024-03-10T06:30:00Z", "2024-03-11T04:30:00Z", "2024-11-03T06:30:00Z", "2024-11-04T05:30:00Z"}
	if e := s.db.Update(func(tx *bolt.Tx) error {
		for i, stamp := range stamps {
			var a Asset
			if e := readJSON(tx.Bucket([]byte("assets")), fmt.Sprintf("%064x", i+1), &a); e != nil {
				return e
			}
			a.Added, _ = time.Parse(time.RFC3339, stamp)
			if e := saveAsset(tx, a); e != nil {
				return e
			}
		}
		return nil
	}); e != nil {
		t.Fatal(e)
	}
	q := GalleryQuery{Sort: "capture", Visibility: "library", Tags: GalleryLabels{All: []string{"trip"}}, NoCapture: true}
	c, e := s.GalleryCalendar(q, "capture", "America/Chicago", 2024, 2)
	if e != nil || c.Count != 4 || c.MissingCapture != 2 || len(c.Years) != 1 || c.Years[0].Count != 2 || len(c.Days) != 1 || c.Days[0].Day != 29 || c.Days[0].From != "2024-02-29T00:00:00" || c.Days[0].Until != "2024-03-01T00:00:00" {
		t.Fatal(c, e)
	}
	for _, tc := range []struct {
		month int
		day   int
		hours time.Duration
	}{{3, 10, 23 * time.Hour}, {11, 3, 25 * time.Hour}} {
		c, e = s.GalleryCalendar(q, "upload", "America/Chicago", 2024, tc.month)
		if e != nil || len(c.Days) != 1 || c.Days[0].Count != 2 || c.Days[0].Day != tc.day {
			t.Fatal(c, e)
		}
		from, _ := time.Parse(time.RFC3339, c.Days[0].From)
		until, _ := time.Parse(time.RFC3339, c.Days[0].Until)
		if until.Sub(from) != tc.hours {
			t.Fatal("DST duration", c.Days[0], until.Sub(from))
		}
		filter := q
		filter.NoCapture = false
		filter.DateBasis, filter.From, filter.Until = "upload", c.Days[0].From, c.Days[0].Until
		if ids := galleryBrowseIDs(t, s, filter); len(ids) != 2 {
			t.Fatal("calendar bucket/query mismatch", ids)
		}
	}
	for _, tc := range []struct {
		basis, zone string
		year, month int
	}{{"wrong", "UTC", 0, 0}, {"upload", "Invalid/Zone", 0, 0}, {"capture", "UTC", 0, 2}, {"capture", "UTC", 10000, 0}, {"capture", "UTC", 2024, 13}} {
		if _, e := s.GalleryCalendar(q, tc.basis, tc.zone, tc.year, tc.month); !errors.Is(e, ErrGalleryInput) {
			t.Fatal(tc, e)
		}
	}
}

func TestGalleryBrowseCalendarMidnightGapAndYearBounds(t *testing.T) {
	s, _ := openTest(t)
	seedGalleryFoundation(t, s, 1)
	setAdded := func(stamp string) {
		t.Helper()
		if e := s.db.Update(func(tx *bolt.Tx) error {
			var a Asset
			if e := readJSON(tx.Bucket([]byte("assets")), fmt.Sprintf("%064x", 1), &a); e != nil {
				return e
			}
			a.Added, _ = time.Parse(time.RFC3339, stamp)
			a.Photo.Captured = "0000-01-01T00:00:00"
			return saveAsset(tx, a)
		}); e != nil {
			t.Fatal(e)
		}
	}
	q := GalleryQuery{Sort: "upload", Visibility: "library"}
	for _, tc := range []struct {
		stamp, zone             string
		year, month, day, hours int
	}{
		{"2018-11-04T04:30:00Z", "America/Sao_Paulo", 2018, 11, 4, 23},
		{"2011-12-29T22:00:00Z", "Pacific/Apia", 2011, 12, 29, 24},
	} {
		setAdded(tc.stamp)
		c, e := s.GalleryCalendar(q, "upload", tc.zone, tc.year, tc.month)
		if e != nil || len(c.Days) != 1 || c.Days[0].Day != tc.day {
			t.Fatal(c, e)
		}
		b := c.Days[0]
		from, _ := time.Parse(time.RFC3339, b.From)
		until, _ := time.Parse(time.RFC3339, b.Until)
		if until.Sub(from) != time.Duration(tc.hours)*time.Hour {
			t.Fatal("midnight gap bucket", b, until.Sub(from))
		}
		filter := q
		filter.DateBasis, filter.From, filter.Until = "upload", b.From, b.Until
		if len(galleryBrowseIDs(t, s, filter)) != 1 {
			t.Fatal("bucket omitted its media")
		}
	}
	setAdded("1800-02-03T01:00:00Z")
	historical, e := s.GalleryCalendar(q, "upload", "Asia/Tokyo", 1800, 2)
	if e != nil || len(historical.Days) != 1 {
		t.Fatal(historical, e)
	}
	zone, e := time.LoadLocation("Asia/Tokyo")
	if e != nil {
		t.Fatal(e)
	}
	wantFrom := time.Date(1800, 2, 3, 0, 0, 0, 0, zone).UTC()
	wantUntil := time.Date(1800, 2, 4, 0, 0, 0, 0, zone).UTC()
	if historical.Days[0].From != wantFrom.Format(time.RFC3339Nano) || historical.Days[0].Until != wantUntil.Format(time.RFC3339Nano) {
		t.Fatal("historical offset seconds lost", historical.Days[0], wantFrom, wantUntil)
	}
	setAdded("0001-01-01T01:00:00Z")
	c, e := s.GalleryCalendar(q, "upload", "Asia/Tokyo", 1, 1)
	if e != nil || len(c.Years) != 1 {
		t.Fatal(c, e)
	}
	q.DateBasis, q.From, q.Until = "upload", c.Years[0].From, c.Years[0].Until
	if _, e := q.Normalize(); e != nil {
		t.Fatal("calendar emitted unusable earliest-year link", q, e)
	}
	q.DateBasis, q.From, q.Until, q.NoCapture = "", "", "", true
	if len(galleryBrowseIDs(t, s, q)) != 1 {
		t.Fatal("year zero capture not treated as missing")
	}
}

func TestGalleryBrowseRealImportRestartAndIndependentRestore(t *testing.T) {
	s, root := openTest(t)
	data := fixture(t)
	m, e := s.CreateImport([]ImportFile{{Name: "one.png", Relative: "Recorded/Leap/one.png", Size: int64(len(data))}, {Name: "copy.png", Relative: "Recorded/Leap/copy.png", Size: int64(len(data))}})
	if e != nil {
		t.Fatal(e)
	}
	if e := s.ControlImport(m.ID, "start"); e != nil {
		t.Fatal(e)
	}
	var id string
	for i := 0; i < 2; i++ {
		entry, e := s.PutImport(context.Background(), m.ID, i, bytes.NewReader(data))
		if e != nil || entry.Duplicate != (i == 1) {
			t.Fatal(entry, e)
		}
		id = entry.AssetID
	}
	o, _ := s.AssetOrganization(id)
	curationEdit(t, s, CurationEdit{Action: "set_asset", AssetID: id, Added: o.Added, Expected: o.Token, Title: "Surviving title", Caption: "Restored caption", Notes: "Restored notes"})
	q := GalleryQuery{Sort: "name", Visibility: "library", SourceFolder: "Recorded/Leap", ImportBatch: m.ID, Text: "restored notes"}
	if ids := galleryBrowseIDs(t, s, q); !reflect.DeepEqual(ids, []string{id}) {
		t.Fatal(ids)
	}
	s.Close()
	s, e = Open(root)
	if e != nil {
		t.Fatal(e)
	}
	defer s.Close()
	if ids := galleryBrowseIDs(t, s, q); !reflect.DeepEqual(ids, []string{id}) {
		t.Fatal("restart", ids)
	}
	archive := filepath.Join(t.TempDir(), "recorded.zip")
	var backup bytes.Buffer
	if e := s.Backup(&backup); e != nil {
		t.Fatal(e)
	}
	if e := os.WriteFile(archive, backup.Bytes(), 0600); e != nil {
		t.Fatal(e)
	}
	s.Close()
	if e := os.Rename(root, root+".unavailable"); e != nil {
		t.Fatal(e)
	}
	restored := filepath.Join(t.TempDir(), "restored")
	if e := Restore(archive, restored); e != nil {
		t.Fatal(e)
	}
	r, e := Open(restored)
	if e != nil {
		t.Fatal(e)
	}
	defer r.Close()
	if ids := galleryBrowseIDs(t, r, q); !reflect.DeepEqual(ids, []string{id}) {
		t.Fatal("restore", ids)
	}
	metadata, e := r.AssetOrganization(id)
	if e != nil || metadata.Title != "Surviving title" || metadata.Caption != "Restored caption" || metadata.Notes != "Restored notes" {
		t.Fatal(metadata, e)
	}
	if hash, e := fileHash(r.Original(id)); e != nil || hash != id {
		t.Fatal("original", hash, e)
	}
}

// This fixture is also run after HTTP backup -> independent CLI restore, with
// the source state unavailable. The JSON manifest contains synthetic data only.
func TestGalleryBrowse78RestoredFixture(t *testing.T) {
	root, manifest := os.Getenv("IMMICH_BROWSE78_ROOT"), os.Getenv("IMMICH_BROWSE78_EXPECTED")
	if root == "" || manifest == "" {
		t.Skip("independent restored browse fixture not configured")
	}
	raw, e := os.ReadFile(manifest)
	if e != nil {
		t.Fatal(e)
	}
	var expected struct {
		Query   GalleryQuery `json:"query"`
		IDs     []string     `json:"ids"`
		Batch   string       `json:"batch"`
		AssetID string       `json:"asset_id"`
		Title   string       `json:"title"`
		Caption string       `json:"caption"`
		Notes   string       `json:"notes"`
	}
	if e := json.Unmarshal(raw, &expected); e != nil {
		t.Fatal(e)
	}
	s, e := Open(root)
	if e != nil {
		t.Fatal(e)
	}
	defer s.Close()
	actual := galleryBrowseIDs(t, s, expected.Query)
	sort.Strings(actual)
	sort.Strings(expected.IDs)
	if len(expected.IDs) == 0 || !reflect.DeepEqual(actual, expected.IDs) {
		t.Fatal("restored browse membership", actual, expected.IDs)
	}
	o, e := s.AssetOrganization(expected.AssetID)
	if e != nil || o.Title != expected.Title || o.Caption != expected.Caption || o.Notes != expected.Notes {
		t.Fatal("restored text", e)
	}
	q := expected.Query
	q.ImportBatch = expected.Batch
	sources, e := s.GallerySources(q, "", 100)
	if e != nil || sources.Count != uint64(len(actual)) || sources.RootCount < uint64(len(actual)) {
		t.Fatal("restored sources", sources, e)
	}
	found := false
	for _, b := range sources.Batches {
		if b.ID == expected.Batch && b.Count >= uint64(len(actual)) {
			found = true
		}
	}
	if !found {
		t.Fatal("restored recorded batch")
	}
	c, e := s.GalleryCalendar(q, "capture", "UTC", 0, 0)
	if e != nil || c.Count < uint64(len(actual)) {
		t.Fatal("restored calendar", c, e)
	}
}
