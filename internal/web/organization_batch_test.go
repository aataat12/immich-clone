package web

import (
	"encoding/json"
	"immichclone/internal/library"
	"strings"
	"testing"
)

func TestOrganizationBatchHTTPAuthPreviewAndReplay(t *testing.T) {
	s, c := setup(t)
	a, _, e := s.Store.Put("batch.txt", strings.NewReader("synthetic batch source"))
	if e != nil {
		t.Fatal(e)
	}
	q := library.GalleryQuery{Sort: "name", Visibility: "library"}
	p, _ := s.Store.QueryGalleryPage(q, "", 60)
	in := library.OrganizationBatchRequest{Action: "favorite_on", Query: q, Snapshot: p.Snapshot, Targets: []library.OrganizationTarget{{ID: a.ID, Added: a.Added}}}
	raw, _ := json.Marshal(in)
	if w := request(s, "POST", "/api/organization/batches", string(raw), nil); w.Code != 401 {
		t.Fatal("auth", w.Code)
	}
	w := request(s, "POST", "/api/organization/batches", string(raw), c)
	var b library.OrganizationBatch
	if w.Code != 200 || json.Unmarshal(w.Body.Bytes(), &b) != nil {
		t.Fatal(w.Code, w.Body.String())
	}
	path := "/api/organization/batches/" + b.ID
	w = request(s, "POST", path, `{"action":"continue","step":0}`, c)
	if w.Code != 200 {
		t.Fatal(w.Code, w.Body.String())
	}
	first := w.Body.String()
	w = request(s, "POST", path, `{"action":"continue","step":0}`, c)
	if w.Code != 200 || w.Body.String() != first {
		t.Fatal("duplicate continuation")
	}
	w = request(s, "GET", "/api/gallery?sort=name&visibility=library&favorite=true", "", c)
	if w.Code != 200 {
		t.Fatal("favorite query", w.Code)
	}
	w = request(s, "GET", "/api/organization/batches/"+strings.Repeat("f", 64), "", c)
	if w.Code != 410 {
		t.Fatal("expired", w.Code)
	}
}
