package library

import (
	"fmt"
	"os"
	"reflect"
	"testing"
	"time"
)

func TestRelease44CombinedCatalogScale(t *testing.T) {
	if os.Getenv("IMMICH_TEST_RELEASE44_SCALE") != "1" {
		t.Skip("requires the dedicated 100k-row synthetic scale gate")
	}
	s, _ := openTest(t)
	const catalogSize = 100000
	seedGalleryFoundation(t, s, catalogSize)

	paths := make([]string, MaxImportFiles)
	ids := make([]int, len(paths))
	for i := range paths {
		// Every catalog ID in this batch has two distinct recorded aliases.
		paths[i] = fmt.Sprintf("Synthetic/Trip/Day%02d/item-%05d.jpg", i%28+1, i)
		ids[i] = i/2 + 1
	}
	batch := galleryBrowseReceipt(t, s, paths, ids)
	otherPaths := make([]string, 10000)
	for i := range otherPaths {
		otherPaths[i] = fmt.Sprintf("Synthetic/Other/item-%05d.jpg", i)
	}
	otherBatch := galleryBrowseReceipt(t, s, otherPaths, ids[len(ids)-len(otherPaths):])
	album := fmt.Sprintf("%064x", catalogSize+1)
	curationEdit(t, s, CurationEdit{Action: "create_album", AlbumID: album, Title: "Combined scale fixture"})
	want := make([]string, 0, 145)
	for i := catalogSize; i >= 1 && len(want) < cap(want); i-- {
		if i > 25000 || i%2 != 0 || i%3 == 0 || i%5 == 0 || i%7 == 0 || i%11 == 0 {
			continue
		}
		id := fmt.Sprintf("%064x", i)
		o, err := s.AssetOrganization(id)
		if err != nil {
			t.Fatal(err)
		}
		curationEdit(t, s, CurationEdit{Action: "set_asset", AssetID: id, Added: o.Added, Expected: o.Token, Favorite: true, Caption: "Synthetic scale marker"})
		a, err := s.AlbumPage(album, "", 1)
		if err != nil {
			t.Fatal(err)
		}
		curationEdit(t, s, CurationEdit{Action: "add_member", AlbumID: album, AssetID: id, Added: o.Added, Expected: a.Album.Token})
		want = append(want, id)
	}
	q := GalleryQuery{
		Sort: "manual", Visibility: "library", Album: album, Favorite: "true", Media: "photo",
		Text: "SCALE MARKER", SourceFolder: "Synthetic/Trip", ImportBatch: batch,
		People: GalleryLabels{All: []string{"Alice", "Bob"}},
		Tags:   GalleryLabels{All: []string{"Even", "Trip"}, Exclude: []string{"blocked"}},
	}
	start := time.Now()
	calendar, err := s.GalleryCalendar(q, "capture", "UTC", 2024, 2)
	coldCalendar := time.Since(start)
	if err != nil || calendar.Count != uint64(len(want)) || len(calendar.Days) != 1 ||
		calendar.Days[0].Day != 29 || calendar.Days[0].Count != uint64(len(want)) {
		t.Fatalf("combined calendar mismatch: %+v (%v)", calendar, err)
	}
	q.DateBasis, q.From, q.Until = "capture", calendar.Days[0].From, calendar.Days[0].Until

	start = time.Now()
	const pageSize = 60
	var got []string
	cursor, snapshot := "", ""
	for page := 0; page < 3; page++ {
		p, err := s.QueryGalleryPage(q, cursor, pageSize)
		if err != nil || p.Count != uint64(len(want)) || (page > 0 && p.Snapshot != snapshot) {
			t.Fatalf("combined page %d count %d: %v", page+1, p.Count, err)
		}
		for _, asset := range p.Assets {
			got = append(got, asset.ID)
		}
		cursor, snapshot = p.Next, p.Snapshot
	}
	pageTime := time.Since(start)
	if cursor != "" || !reflect.DeepEqual(got, want) {
		t.Fatal("combined pages differ from independent manual-order fixture")
	}

	start = time.Now()
	n, err := s.GalleryNeighbors(q, want[pageSize-1], snapshot)
	if err != nil || n.Next == nil || n.Next.ID != want[pageSize] || n.Count != uint64(len(want)) {
		t.Fatalf("viewer did not cross page boundary: %v", err)
	}
	anchor, err := s.GalleryAnchorPage(q, want[pageSize], snapshot, pageSize)
	if err != nil || anchor.Number != 2 || len(anchor.Page.Assets) != pageSize || anchor.Page.Assets[0].ID != want[pageSize] {
		t.Fatalf("viewer return anchor: page %d, %v", anchor.Number, err)
	}
	viewerTime := time.Since(start)
	// The same IDs appear under distinct folders in the two receipts. The
	// matching folder must come from the active batch's own entries.
	q.ImportBatch = otherBatch
	none, err := s.QueryGalleryPage(q, "", pageSize)
	if err != nil || none.Count != 0 {
		t.Fatalf("folder from the first batch returned %d results in the second: %v", none.Count, err)
	}
	q.SourceFolder = "Synthetic/Other"
	other, err := s.QueryGalleryPage(q, "", pageSize)
	if err != nil || other.Count != uint64(len(want)) {
		t.Fatalf("duplicate batch membership count %d: %v", other.Count, err)
	}
	t.Logf("100,000 synthetic catalog rows, 60,000 recorded entries: cold combined calendar %s, three pages %s, viewer neighbor and anchor %s", coldCalendar, pageTime, viewerTime)
}
