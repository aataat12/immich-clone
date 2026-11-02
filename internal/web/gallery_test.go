package web

import (
	"encoding/json"
	"immichclone/internal/library"
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
