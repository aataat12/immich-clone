package library

import (
	"context"
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	bolt "go.etcd.io/bbolt"
)

func TestSuggestionCaptureTime(t *testing.T) {
	for _, tc := range []struct{ filename, embedded, stamp, basis string }{
		{"2026-09-17 09.37.41.jpg", "", "2026-09-17T09:37:41", "filename"},
		{"VID_20261003_144200.mp4", "", "2026-10-03T14:42:00", "filename"},
		{"PXL_20261003_144200123.jpg", "", "2026-10-03T14:42:00", "filename"},
		{"IMG_20261003_144200.jpg", "2020-01-04T02:03:04", "2020-01-04T02:03:04", "embedded"},
		{"2024-02-30 12.00.00.jpg", "", "", ""},
		{"2024-02-29.jpg", "", "", ""},
		{"IMG_144200.jpg", "", "", ""},
	} {
		stamp, basis := suggestionDate(Asset{Name: tc.filename, Added: time.Now(), Photo: PhotoMetadata{Captured: tc.embedded}})
		if stamp != tc.stamp || basis != tc.basis {
			t.Errorf("%s got (%q, %q), want (%q, %q)", tc.filename, stamp, basis, tc.stamp, tc.basis)
		}
	}
}

func TestSuggestedAlbumReviewAndAtomicSave(t *testing.T) {
	s, _ := openTest(t)
	ctx := context.Background()
	id := func(n int) string { return fmt.Sprintf("%064x", n) }
	added := time.Date(2026, 11, 5, 1, 0, 0, 0, time.UTC)
	names := []string{
		"2026-10-02 08.00.00.jpg",
		"2026-10-03 14.42.00.jpg", "VID_20261003_150000.mp4", "2026-10-03 16.00.00.jpg",
		"2026-10-04 09.30.00.jpg", "2026-10-04 10.00.00.jpg", "2026-10-04 11.00.00.jpg",
		"2026-10-05 18.00.00.jpg", "2026-10-03 15.01.00.jpg", "Unknown.jpg",
	}
	if err := s.db.Update(func(tx *bolt.Tx) error {
		for i, name := range names {
			a := Asset{ID: id(i + 1), Name: name, MIME: "image/jpeg", Added: added, Hidden: i == 8}
			if i == 2 {
				a.MIME = "video/mp4"
			}
			if err := saveAsset(tx, a); err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	events, err := s.SuggestedAlbums(ctx, false, "")
	if err != nil || events.Undated != 1 || len(events.Suggestions) != 2 || events.Suggestions[1].ID != id(2) {
		t.Fatalf("events: %+v, %v", events, err)
	}
	trips, err := s.SuggestedAlbums(ctx, true, "")
	if err != nil || len(trips.Suggestions) != 1 || trips.Suggestions[0].Count != 6 {
		t.Fatalf("trips: %+v, %v", trips, err)
	}
	review, err := s.SuggestedAlbum(ctx, true, id(2))
	if err != nil || len(review.Items) != 8 || review.Start != 1 || review.End != 7 ||
		review.Items[0].ID != id(1) || review.Items[2].MIME != "video/mp4" || review.Items[7].ID != id(8) {
		t.Fatalf("review: %+v, %v", review, err)
	}
	edit := CurationEdit{Action: "create_album", AlbumID: id(100), Title: "Reviewed trip",
		Members: []CurationAlbumTarget{{id(1), added}, {id(3), added}, {id(8), added}}}
	if err := s.SetHidden(id(8), true); err != nil {
		t.Fatal(err)
	}
	if _, err := s.EditCuration(edit); !errors.Is(err, ErrCurationConflict) {
		t.Fatalf("hidden target accepted: %v", err)
	}
	if _, err := s.AlbumPage(edit.AlbumID, "", 10); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("partially saved album: %v", err)
	}
	if err := s.SetHidden(id(8), false); err != nil {
		t.Fatal(err)
	}
	first := curationEdit(t, s, edit).(Album)
	if replay := curationEdit(t, s, edit).(Album); replay != first {
		t.Fatalf("retry created a different album: %+v", replay)
	}
	page, err := s.AlbumPage(edit.AlbumID, "", 10)
	if err != nil || page.Count != len(edit.Members) {
		t.Fatalf("saved album: %+v, %v", page, err)
	}
	for i, member := range page.Members {
		if member.Position != uint64(i) || member.AssetID != edit.Members[i].AssetID {
			t.Fatalf("unexpected member %d: %+v", i, member)
		}
	}
	edit.Members[1], edit.Members[2] = edit.Members[2], edit.Members[1]
	if _, err := s.EditCuration(edit); !errors.Is(err, ErrCurationConflict) {
		t.Fatalf("nonidentical retry accepted: %v", err)
	}
}
