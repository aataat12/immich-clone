package web

import (
	"encoding/json"
	"immichclone/internal/library"
	"net/url"
	"strings"
	"testing"
)

func TestGalleryFoundationHTTPContract(t *testing.T) {
	s, c := setup(t)
	for _, path := range []string{"/api/gallery", "/api/gallery/neighbors"} {
		if w := request(s, "GET", path, "", nil); w.Code != 401 {
			t.Fatal("unprotected", path, w.Code)
		}
	}
	for _, name := range []string{"first.jpg", "second.txt", "third.jpg"} {
		if _, _, e := s.Store.Put(name, strings.NewReader("synthetic "+name)); e != nil {
			t.Fatal(e)
		}
	}
	base := "/api/gallery?sort=name&visibility=library"
	w := request(s, "GET", base+"&filename=jpg&limit=1", "", c)
	var p library.GalleryPage
	if w.Code != 200 || json.Unmarshal(w.Body.Bytes(), &p) != nil || p.Count != 2 || len(p.Assets) != 1 || p.Snapshot == "" {
		t.Fatal(w.Code, w.Body.String())
	}
	path := "/api/gallery/neighbors?sort=name&visibility=library&filename=JPG&asset=" + p.Assets[0].ID + "&snapshot=" + url.QueryEscape(p.Snapshot)
	w = request(s, "GET", path, "", c)
	var n library.GalleryNeighbors
	if w.Code != 200 || json.Unmarshal(w.Body.Bytes(), &n) != nil || n.Previous != nil || n.Next == nil || n.Count != 2 {
		t.Fatal(w.Code, w.Body.String())
	}
	for _, suffix := range []string{"&filename=%ZZ", "&tag_any=%", "&favorite=true", "&limit=0", "&media=unknown", "&q=a&q=b", "&no_capture=maybe", "&date_basis=upload&from=2024-01-01T00:00:00", "&asset=" + p.Assets[0].ID} {
		if w := request(s, "GET", base+suffix, "", c); w.Code != 400 {
			t.Fatal(suffix, w.Code)
		}
	}
	if w := request(s, "GET", strings.Replace(path, "filename=JPG", "filename=txt", 1), "", c); w.Code != 400 {
		t.Fatal("viewer query reuse", w.Code)
	}
	if e := s.Store.SetTags(p.Assets[0].ID, []string{"edit"}); e != nil {
		t.Fatal(e)
	}
	if w := request(s, "GET", path, "", c); w.Code != 409 {
		t.Fatal("stale viewer", w.Code, w.Body.String())
	}
	if w := request(s, "GET", base+"&filename=jpg&after="+url.QueryEscape(p.Next), "", c); w.Code != 409 {
		t.Fatal("stale page", w.Code)
	}
}
