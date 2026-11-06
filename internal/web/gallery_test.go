package web

import (
	"bytes"
	"context"
	"encoding/json"
	"image"
	"image/color"
	"image/png"
	"immichclone/internal/library"
	"immichclone/internal/recognition"
	"net/url"
	"strings"
	"testing"
)

func TestGalleryProtectedBoundedAndStaleCursor(t *testing.T) {
	s, c := setup(t)
	path := "/api/gallery?sort=name&visibility=library&limit=1"
	if w := request(s, "GET", path, "", nil); w.Code != 401 {
		t.Fatal("unprotected gallery")
	}
	s.Store.Put("a.txt", strings.NewReader("first synthetic original"))
	s.Store.Put("b.txt", strings.NewReader("second synthetic original"))
	w := request(s, "GET", path, "", c)
	var page library.GalleryPage
	json.Unmarshal(w.Body.Bytes(), &page)
	if w.Code != 200 || len(page.Assets) != 1 || page.Next == "" || page.Total != 2 {
		t.Fatal(w.Body.String())
	}
	s.Store.SetHidden(page.Assets[0].ID, true)
	if w = request(s, "GET", path+"&after="+page.Next, "", c); w.Code != 409 {
		t.Fatal("stale cursor not surfaced")
	}
	if w = request(s, "GET", "/api/gallery?sort=name&visibility=library&limit=101", "", c); w.Code != 400 {
		t.Fatal("invalid size")
	}
}

func TestGallerySearchAPINamesTagsVisibilityAndCursorBinding(t *testing.T) {
	s, c := setup(t)
	put := func(name string, seed uint8) library.Asset {
		t.Helper()
		img := image.NewRGBA(image.Rect(0, 0, 100, 100))
		img.SetRGBA(0, 0, color.RGBA{seed, 0, 0, 255})
		var b bytes.Buffer
		if e := png.Encode(&b, img); e != nil {
			t.Fatal(e)
		}
		a, _, e := s.Store.Put(name, bytes.NewReader(b.Bytes()))
		if e != nil {
			t.Fatal(e)
		}
		return a
	}
	family := put("family.png", 1)
	filename := put("ALICE-filename.png", 2)
	hidden := put("private.png", 3)
	for _, face := range []struct {
		asset library.Asset
		name  string
	}{{family, "Alice"}, {family, "Bob"}, {hidden, "Alice"}} {
		if e := s.Store.ManualFace(context.Background(), face.asset.ID, []float64{.1, .1, .2, .2}, face.name); e != nil {
			t.Fatal(e)
		}
	}
	if e := s.Store.SetTags(family.ID, []string{"Reunion trip"}); e != nil {
		t.Fatal(e)
	}
	if e := s.Store.SetHidden(hidden.ID, true); e != nil {
		t.Fatal(e)
	}
	get := func(query, visibility, after string, limit int) library.GalleryPage {
		t.Helper()
		path := "/api/gallery?sort=name&visibility=" + visibility + "&q=" + url.QueryEscape(query) + "&after=" + url.QueryEscape(after)
		if limit == 1 {
			path += "&limit=1"
		}
		w := request(s, "GET", path, "", c)
		var page library.GalleryPage
		if w.Code != 200 || json.Unmarshal(w.Body.Bytes(), &page) != nil {
			t.Fatal(w.Code, w.Body.String())
		}
		return page
	}
	first := get("Alice", "library", "", 1)
	if first.Count != 2 || first.Total != 3 || first.Hidden != 1 || len(first.Assets) != 1 || first.Next == "" || first.Assets[0].ID != filename.ID {
		t.Fatalf("search count, visibility or filename match: %+v", first)
	}
	second := get("  ALICE  ", "library", first.Next, 1)
	if second.Count != 2 || len(second.Assets) != 1 || second.Assets[0].ID != family.ID || second.Next != "" {
		t.Fatalf("assigned-person match or normalized cursor failed: %+v", second)
	}
	for _, query := range []string{"Bob", "REUNION"} {
		page := get(query, "library", "", 0)
		if page.Count != 1 || len(page.Assets) != 1 || page.Assets[0].ID != family.ID {
			t.Fatalf("multi-person/custom-tag search %q: %+v", query, page)
		}
	}
	page := get("Alice", "hidden", "", 0)
	if page.Count != 1 || len(page.Assets) != 1 || page.Assets[0].ID != hidden.ID {
		t.Fatalf("hidden matches leaked or disappeared: %+v", page)
	}
	for _, path := range []string{
		"/api/gallery?sort=name&visibility=library&q=Bob&after=" + url.QueryEscape(first.Next),
		"/api/gallery?sort=name&visibility=hidden&q=Alice&after=" + url.QueryEscape(first.Next),
		"/api/gallery?sort=upload&visibility=library&q=Alice&after=" + url.QueryEscape(first.Next),
	} {
		if w := request(s, "GET", path, "", c); w.Code != 400 {
			t.Fatal("cursor accepted for a different query, visibility or order", w.Code)
		}
	}
	if e := s.Store.SetTags(family.ID, []string{"Changed tag"}); e != nil {
		t.Fatal(e)
	}
	if w := request(s, "GET", "/api/gallery?sort=name&visibility=library&q=Alice&after="+url.QueryEscape(first.Next), "", c); w.Code != 409 {
		t.Fatal("organization mutation did not invalidate filtered cursor", w.Code)
	}
	if page = get("Reunion", "library", "", 0); page.Count != 0 || len(page.Assets) != 0 || page.Next != "" {
		t.Fatal("removed tag retained in search cache")
	}
	if page = get("", "library", "", 0); page.Count != 2 || len(page.Assets) != 2 {
		t.Fatal("clearing search did not restore eligible gallery")
	}

	// Object labels participate only while they remain active.
	s.Store.SetRecognizer(func(context.Context, image.Image, string) (recognition.Result, error) {
		return recognition.Result{Protocol: 1, Objects: []recognition.Object{{Class: 15, Confidence: .99, Box: []float64{.1, .1, .4, .4}}}}, nil
	}, true)
	if e := s.Store.QueueAnalysis(filename.ID); e != nil {
		t.Fatal(e)
	}
	if worked, e := s.Store.ProcessAnalysis(context.Background()); e != nil || !worked {
		t.Fatal("object fixture analysis failed", e)
	}
	object := recognition.Classes[15]
	if page = get(object, "library", "", 0); page.Count != 1 || len(page.Assets) != 1 || page.Assets[0].ID != filename.ID {
		t.Fatal("active object tag missing from gallery search")
	}
	if e := s.Store.DismissTag(filename.ID, object, false); e != nil {
		t.Fatal(e)
	}
	if page = get(object, "library", "", 0); page.Count != 0 || len(page.Assets) != 0 {
		t.Fatal("dismissed object tag remained searchable")
	}
}

func TestGallerySearchAPIRejectsInvalidQueriesAndPages(t *testing.T) {
	s, c := setup(t)
	base := "/api/gallery?sort=name&visibility=library"
	if w := request(s, "GET", base+"&q=Alice", "", nil); w.Code != 401 {
		t.Fatal("search bypassed authentication")
	}
	for _, query := range []string{strings.Repeat("x", 129), strings.Repeat("é", 65), "bad\nquery", "bad\x00query", string([]byte{255})} {
		if w := request(s, "GET", base+"&q="+url.QueryEscape(query), "", c); w.Code != 400 {
			t.Fatal("invalid query accepted", w.Code)
		}
	}
	for _, query := range []string{strings.Repeat("x", 128), strings.Repeat("é", 64), "   "} {
		if w := request(s, "GET", base+"&q="+url.QueryEscape(query), "", c); w.Code != 200 {
			t.Fatal("valid bounded query rejected", w.Code)
		}
	}
	for _, suffix := range []string{"&limit=0", "&limit=101", "&limit=bad", "&after=malformed", "&after=" + strings.Repeat("x", 4097)} {
		if w := request(s, "GET", base+"&q=Alice"+suffix, "", c); w.Code != 400 {
			t.Fatal("invalid filtered page accepted", w.Code)
		}
	}
}

func TestGallerySearchAPIExcludesSuggestionsAndDismissedPeople(t *testing.T) {
	s, c := setup(t)
	s.Store.SetRecognizer(func(context.Context, image.Image, string) (recognition.Result, error) {
		v := make([]float64, 128)
		v[0] = 1
		return recognition.Result{Protocol: 1, Faces: []recognition.Face{{Box: []float64{.1, .1, .4, .4}, Confidence: .99, Embedding: v}}}, nil
	}, true)
	var referenceID, candidateID, referenceFaceID string
	for i, name := range []string{"reference.png", "candidate.png"} {
		img := image.NewRGBA(image.Rect(0, 0, 100, 100))
		img.SetRGBA(0, 0, color.RGBA{uint8(i + 1), 0, 0, 255})
		var data bytes.Buffer
		if e := png.Encode(&data, img); e != nil {
			t.Fatal(e)
		}
		a, _, e := s.Store.Put(name, bytes.NewReader(data.Bytes()))
		if e != nil {
			t.Fatal(e)
		}
		if e = s.Store.QueueAnalysis(a.ID); e != nil {
			t.Fatal(e)
		}
		if worked, e := s.Store.ProcessAnalysis(context.Background()); e != nil || !worked {
			t.Fatal("face fixture analysis failed", e)
		}
		if i == 0 {
			referenceID = a.ID
			page, e := s.Store.OrganizationPage("", "", "")
			if e != nil || len(page.Faces) != 1 {
				t.Fatal("reference face missing", e)
			}
			referenceFaceID = page.Faces[0].ID
			if e = s.Store.LabelFaces([]string{referenceFaceID}, "KnownReviewPerson"); e != nil {
				t.Fatal(e)
			}
		} else {
			candidateID = a.ID
		}
	}
	review, e := s.Store.OrganizationPage("", "", "")
	if e != nil {
		t.Fatal(e)
	}
	foundSuggestion := false
	for _, face := range review.Faces {
		if face.AssetID == candidateID && face.Name == "" && face.Suggestion == "KnownReviewPerson" {
			foundSuggestion = true
		}
	}
	if !foundSuggestion {
		t.Fatal("fixture did not produce an unconfirmed suggestion")
	}
	path := "/api/gallery?sort=name&visibility=library&q=KnownReviewPerson"
	w := request(s, "GET", path, "", c)
	var page library.GalleryPage
	if w.Code != 200 || json.Unmarshal(w.Body.Bytes(), &page) != nil || page.Count != 1 || len(page.Assets) != 1 || page.Assets[0].ID != referenceID {
		t.Fatal("unconfirmed suggestion counted as assigned person", w.Body.String())
	}
	if e = s.Store.DismissFace(referenceFaceID, false); e != nil {
		t.Fatal(e)
	}
	w = request(s, "GET", path, "", c)
	if w.Code != 200 || json.Unmarshal(w.Body.Bytes(), &page) != nil || page.Count != 0 || len(page.Assets) != 0 {
		t.Fatal("dismissed person remained searchable", w.Body.String())
	}
}
