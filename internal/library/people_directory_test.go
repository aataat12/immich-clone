package library

import (
	"fmt"
	"testing"
	"time"

	bolt "go.etcd.io/bbolt"
)

func TestPeopleDirectoryCountsDistinctVisibleAssets(t *testing.T) {
	s, _ := openTest(t)
	id := func(n int) string { return fmt.Sprintf("%064x", n) }
	videoPositionA, videoPositionB := int64(0), int64(2_000_000)
	faces := []struct {
		asset     int
		name      string
		sample    *int64
		dismissed bool
	}{
		{2, "JORDAN & Co", &videoPositionA, false},
		{2, "Jordan & Co", &videoPositionB, false},
		{1, "Jordan & Co", nil, false},
		{1, "jordan & co", nil, false},
		{3, "Jordan & Co", nil, false}, // hidden
		{4, "Jordan & Co", nil, true},
		{1, "", nil, false},
		{4, "Jordan", nil, false}, // a different exact label
	}
	if err := s.db.Update(func(tx *bolt.Tx) error {
		for n := 1; n <= 4; n++ {
			mime := "image/jpeg"
			if n == 2 {
				mime = "video/mp4"
			}
			a := Asset{ID: id(n), Name: fmt.Sprintf("Asset %d", n), MIME: mime,
				Added: time.Date(2024, 1, n, 0, 0, 0, 0, time.UTC), Hidden: n == 3}
			if err := saveAsset(tx, a); err != nil {
				return err
			}
		}
		for i, fixture := range faces {
			face := StoredFace{ID: id(10 + i), AssetID: id(fixture.asset), SampleUS: fixture.sample,
				Box: []float64{.1, .1, .2, .2}, Provider: "manual"}
			label := FaceLabel{Name: fixture.name, Source: "manual", Dismissed: fixture.dismissed}
			if err := writeJSON(tx.Bucket([]byte(facesBucket)), face.ID, face); err != nil {
				return err
			}
			if err := writeJSON(tx.Bucket([]byte(labelsBucket)), face.ID, label); err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	check := func(want DirectoryPerson) {
		t.Helper()
		// Directory results are independent of the face-review filters and cursor.
		p, err := s.OrganizationFilteredPage("", id(99), "other", true, true)
		if err != nil || len(p.People) != 2 || p.People[0].Name != "Jordan" || p.People[1] != want {
			t.Fatalf("people: %+v, %v", p.People, err)
		}
		gallery, err := s.QueryGalleryPage(GalleryQuery{
			Sort: "name", Visibility: "library", People: GalleryLabels{All: []string{want.Name}},
		}, "", 60)
		if err != nil || gallery.Count != uint64(want.Count) {
			t.Fatalf("directory count %d, gallery count %d, error %v", want.Count, gallery.Count, err)
		}
	}
	check(DirectoryPerson{Name: "Jordan & Co", FaceID: id(12), Count: 2})
	if err := s.SetHidden(id(1), true); err != nil {
		t.Fatal(err)
	}
	check(DirectoryPerson{Name: "JORDAN & Co", FaceID: id(10), Count: 1})
}
