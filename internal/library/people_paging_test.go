package library

import (
	"bytes"
	"fmt"
	bolt "go.etcd.io/bbolt"
	"testing"
)

func TestPeopleFiltersBeforePagingAndAccurateContinuation(t *testing.T) {
	s, _ := openTest(t)
	a, _, e := s.Put("review-fixture.png", bytes.NewReader(fixture(t)))
	if e != nil {
		t.Fatal(e)
	}
	id := func(n int) string { return fmt.Sprintf("%064x", n) }
	// Metadata-only rows prove filter/paging semantics, not native inference/scale.
	e = s.db.Update(func(tx *bolt.Tx) error {
		for n := 1; n <= 207; n++ {
			f := StoredFace{ID: id(n), AssetID: a.ID, Box: []float64{.1, .1, .2, .2}, Provider: "manual"}
			if e := writeJSON(tx.Bucket([]byte(facesBucket)), f.ID, f); e != nil {
				return e
			}
			l := FaceLabel{}
			if n <= 105 {
				l.Name = "Named"
			}
			if n == 205 {
				l.Dismissed = true
			}
			if e := writeJSON(tx.Bucket([]byte(labelsBucket)), f.ID, l); e != nil {
				return e
			}
		}
		return nil
	})
	if e != nil {
		t.Fatal(e)
	}
	first, e := s.OrganizationFilteredPage("", "", "", false, true)
	if e != nil || len(first.Faces) != 100 || first.Faces[0].ID != id(106) || first.FaceAfter == "" {
		t.Fatal("unnamed first page skipped matches", e)
	}
	second, e := s.OrganizationFilteredPage("", first.FaceAfter, "", false, true)
	if e != nil || len(second.Faces) != 1 || second.Faces[0].ID != id(207) || second.FaceAfter != "" {
		t.Fatal("unnamed continuation incomplete", e)
	}
	seen := map[string]bool{}
	for _, f := range append(first.Faces, second.Faces...) {
		if seen[f.ID] || f.Name != "" || f.Dismissed {
			t.Fatal("duplicate or ineligible result")
		}
		seen[f.ID] = true
	}
	if len(first.Names) != 1 || first.Names[0] != "Named" {
		t.Fatal("filter removed named-reference choices")
	}
	// Exactly 100 named matches, followed only by nonmatches: no phantom Next.
	// Use name search independently of filename, then rename five excess matches.
	if e = s.LabelFaces([]string{id(101), id(102), id(103), id(104), id(105)}, "Other"); e != nil {
		t.Fatal(e)
	}
	matched, e := s.OrganizationFilteredPage("", "", "Named", false, false)
	if e != nil || len(matched.Faces) != 100 || matched.FaceAfter != "" {
		t.Fatal("search advertised empty continuation", e)
	}
	dismissed, e := s.OrganizationFilteredPage("", "", "", true, true)
	if e != nil || len(dismissed.Faces) != 1 || dismissed.Faces[0].ID != id(205) {
		t.Fatal("dismissed filter mixed with active review", e)
	}
	empty, e := s.OrganizationFilteredPage("", "", "absent", false, true)
	if e != nil || len(empty.Faces) != 0 || empty.FaceAfter != "" {
		t.Fatal("empty search continuation", e)
	}
	if e = s.Verify(); e != nil {
		t.Fatal(e)
	}
}
