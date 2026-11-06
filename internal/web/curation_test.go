package web

import (
	"encoding/json"
	"immichclone/internal/library"
	"strings"
	"testing"
)

func TestCurationHTTPAuthAtomicMutationExportAndConflict(t *testing.T) {
	s, c := setup(t)
	a, _, e := s.Store.Put("original.txt", strings.NewReader("synthetic original"))
	if e != nil {
		t.Fatal(e)
	}
	path := "/api/metadata/asset/" + a.ID
	if w := request(s, "GET", path, "", nil); w.Code != 401 {
		t.Fatal("unauthenticated metadata")
	}
	var m library.AssetOrganization
	w := request(s, "GET", path, "", c)
	if w.Code != 200 || json.Unmarshal(w.Body.Bytes(), &m) != nil {
		t.Fatal(w.Code, w.Body.String())
	}
	in := library.CurationEdit{Action: "set_asset", AssetID: a.ID, Added: a.Added, Expected: m.Token, Favorite: true, Caption: "Family <context>\nRésumé"}
	raw, _ := json.Marshal(in)
	w = request(s, "POST", "/api/metadata/edit", string(raw), c)
	if w.Code != 200 {
		t.Fatal(w.Code, w.Body.String())
	}
	if w = request(s, "POST", "/api/metadata/edit", string(raw), c); w.Code != 409 {
		t.Fatal("stale client accepted", w.Code)
	}
	if w = request(s, "POST", "/api/metadata/edit", "{\"action\":\"set_asset\",\"unknown\":true}", c); w.Code != 400 {
		t.Fatal("unknown field", w.Code)
	}
	var exported map[string]any
	w = request(s, "GET", "/api/export", "", c)
	if w.Code != 200 || json.Unmarshal(w.Body.Bytes(), &exported) != nil || exported["catalog_schema"] != "3" || exported["curation"] == nil {
		t.Fatal("incomplete export", w.Code)
	}
}
