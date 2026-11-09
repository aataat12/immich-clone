package library

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	bolt "go.etcd.io/bbolt"
	"reflect"
	"testing"
)

func TestGalleryBrowseDirectMissingAndReceiptOutcomes(t *testing.T) {
	s, _ := openTest(t)
	seedGalleryFoundation(t, s, 6)
	paths := []string{"Trip/direct.jpg", "Trip/Only/Deeper/deep.jpg", "root.jpg", "", "Trip/deleted.jpg", "Trip/failed.jpg", "Trip/pending.jpg", "Trip/Only/direct.jpg", "Trip/Zero/Deep/other.jpg"}
	files := []ImportFile{}
	for i, p := range paths {
		files = append(files, ImportFile{Name: fmt.Sprintf("synthetic-%d.jpg", i), Relative: p, Size: 1})
	}
	m, e := s.CreateImport(files)
	if e != nil {
		t.Fatal(e)
	}
	ids := map[int]int{0: 2, 1: 3, 2: 4, 3: 5, 4: 6, 7: 3, 8: 4}
	if e := s.db.Update(func(tx *bolt.Tx) error {
		child := tx.Bucket([]byte(importBucket)).Bucket([]byte(m.ID))
		for i, f := range files {
			entry := ImportEntry{ImportFile: f, Index: i, Duplicate: i != 0 && i != 4}
			if id, accepted := ids[i]; accepted {
				entry.AssetID = fmt.Sprintf("%064x", id)
			}
			if i == 4 {
				entry.Deleted = true
				entry.Error = "Managed copy deleted"
			}
			if i == 5 {
				entry.Error = "Copy failed; retry"
			}
			if i == 0 {
				entry.Error = "A later source retry changed; accepted bytes retained"
			}
			if e := writeJSON(child, entryKey(i), entry); e != nil {
				return e
			}
		}
		m.AcceptedFiles, m.AcceptedBytes, m.NewFiles, m.Duplicates = 7, 7, 2, 5
		if e := writeJSON(child, "meta", m); e != nil {
			return e
		}
		var a Asset
		if e := readJSON(tx.Bucket([]byte("assets")), fmt.Sprintf("%064x", 5), &a); e != nil {
			return e
		}
		a.Sources = []string{"Elsewhere/recorded.jpg"}
		return saveAsset(tx, a)
	}); e != nil {
		t.Fatal(e)
	}
	q := GalleryQuery{Sort: "name", Visibility: "library", ImportBatch: m.ID, SourceFolder: "Trip"}
	if actual := galleryBrowseIDs(t, s, q); len(actual) != 3 {
		t.Fatal("subtree unique count", actual)
	}
	q.SourceMode = "direct"
	if actual := galleryBrowseIDs(t, s, q); !reflect.DeepEqual(actual, []string{fmt.Sprintf("%064x", 2)}) {
		t.Fatal("direct scope", actual)
	}
	choices, e := s.GallerySources(q, "", 100)
	if e != nil || len(choices.Folders) != 2 || choices.Folders[0].Path != "Trip/Only" || choices.Folders[0].Count != 1 || choices.Folders[1].Path != "Trip/Zero" || choices.Folders[1].Count != 0 {
		t.Fatal("direct intermediate navigation", choices, e)
	}
	if len(choices.Batches) != 1 || choices.Batches[0].Count != 1 || choices.Batches[0].GalleryImportOutcomes != (GalleryImportOutcomes{Total: 9, Accepted: 7, New: 2, Duplicates: 5, Failed: 1, Deleted: 1, Pending: 1}) {
		t.Fatal("historical receipt outcomes", choices.Batches)
	}
	q.SourceFolder = "."
	if actual := galleryBrowseIDs(t, s, q); !reflect.DeepEqual(actual, []string{fmt.Sprintf("%064x", 4)}) {
		t.Fatal("direct root files", actual)
	}
	q.SourceFolder, q.SourceMode, q.SourceMissing = "", "", true
	if actual := galleryBrowseIDs(t, s, q); !reflect.DeepEqual(actual, []string{fmt.Sprintf("%064x", 5)}) {
		t.Fatal("batch specific missing aliases", actual)
	}
	choices, e = s.GallerySources(q, "", 100)
	if e != nil || choices.Count != 1 || choices.RootCount != 3 || choices.MissingCount != 1 || len(choices.Folders) != 1 || choices.Folders[0].Path != "Trip" || choices.Batches[0].Count != 1 {
		t.Fatal("missing facet still allows navigation", choices, e)
	}
	q.ImportBatch = ""
	for _, actual := range galleryBrowseIDs(t, s, q) {
		if actual == fmt.Sprintf("%064x", 5) {
			t.Fatal("global missing matched asset with another recorded alias")
		}
	}
	q.ImportBatch = m.ID
	o, _ := s.AssetOrganization(fmt.Sprintf("%064x", 5))
	curationEdit(t, s, CurationEdit{Action: "set_asset", AssetID: o.AssetID, Added: o.Added, Expected: o.Token, Favorite: true})
	q.Favorite, q.People, q.Tags = "true", GalleryLabels{All: []string{"Alice"}}, GalleryLabels{All: []string{"Trip"}}
	if actual := galleryBrowseIDs(t, s, q); !reflect.DeepEqual(actual, []string{o.AssetID}) {
		t.Fatal("combined missing scope", actual)
	}
	c, e := s.GalleryCalendar(q, "capture", "UTC", 0, 0)
	if e != nil || c.Count != 1 || c.MissingCapture != 1 {
		t.Fatal("missing scope calendar", c, e)
	}
	q.SourceFolder = "Trip"
	if _, e := q.Normalize(); !errors.Is(e, ErrGalleryInput) {
		t.Fatal("contradictory missing/folder allowed", e)
	}
	q.SourceFolder, q.SourceMode = "", "invalid"
	if _, e := q.Normalize(); !errors.Is(e, ErrGalleryInput) {
		t.Fatal("invalid source mode", e)
	}
	q.SourceMode = "subtree"
	if normalized, e := q.Normalize(); e != nil || normalized.SourceMode != "" {
		t.Fatal("default mode canonicalization", normalized, e)
	}
}

func TestGalleryBrowseImportInterchangeSnapshotAndBudget(t *testing.T) {
	s, _ := openTest(t)
	seedGalleryFoundation(t, s, 2)
	id := galleryBrowseReceipt(t, s, []string{"Trip/one.jpg", "Trip/duplicate.jpg"}, []int{1, 1})
	exported, e := s.OrganizationExport()
	if e != nil {
		t.Fatal(e)
	}
	history, ok := exported["import_history"].(ImportHistoryExport)
	if !ok || history.Version != 1 || len(history.Sessions) != 1 || len(history.Sessions[id].Entries) != 2 {
		t.Fatal("missing receipt interchange", exported["import_history"])
	}
	var expectedMeta, expectedEntry []byte
	if e := s.db.View(func(tx *bolt.Tx) error {
		child := tx.Bucket([]byte(importBucket)).Bucket([]byte(id))
		expectedMeta = append([]byte(nil), child.Get([]byte("meta"))...)
		expectedEntry = append([]byte(nil), child.Get([]byte(entryKey(1)))...)
		if !bytes.Equal(history.Sessions[id].Session, expectedMeta) || !bytes.Equal(history.Sessions[id].Entries[entryKey(1)], expectedEntry) {
			t.Fatal("export altered saved receipts")
		}
		bounded, e := exportImportHistory(tx, 1)
		if !errors.Is(e, ErrImportHistoryExportLimit) || bounded.Sessions != nil {
			t.Fatal("partial result after aggregate budget refusal", bounded, e)
		}
		return nil
	}); e != nil {
		t.Fatal(e)
	}
	// Returned JSON must survive transaction closure and subsequent writes.
	if e := s.ControlImport(id, "cancel"); e == nil {
		t.Fatal("completed fixture session changed")
	}
	if e := s.SetTags(fmt.Sprintf("%064x", 1), []string{"later committed write"}); e != nil {
		t.Fatal(e)
	}
	if !bytes.Equal(history.Sessions[id].Session, expectedMeta) || !bytes.Equal(history.Sessions[id].Entries[entryKey(1)], expectedEntry) {
		t.Fatal("export retained transaction-owned bytes")
	}
	// Exceed the actual per-record production guard without growing a huge
	// fixture. A full export must fail, never return the already-built fields.
	if e := s.db.Update(func(tx *bolt.Tx) error {
		oversize := append(append([]byte{}, expectedMeta...), bytes.Repeat([]byte(" "), (1<<20)+1-len(expectedMeta))...)
		if !json.Valid(oversize) {
			t.Fatal("invalid budget fixture")
		}
		return tx.Bucket([]byte(importBucket)).Bucket([]byte(id)).Put([]byte("meta"), oversize)
	}); e != nil {
		t.Fatal(e)
	}
	if result, e := s.OrganizationExport(); !errors.Is(e, ErrImportHistoryExportLimit) || result != nil {
		t.Fatal("partial full export", result, e)
	}
}
