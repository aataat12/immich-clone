package web

import (
	"bytes"
	"encoding/json"
	"testing"
)

func TestDeletionHTTPConfirmationAndPendingRoutes(t *testing.T) {
	s, c := setup(t)
	a, _, e := s.Store.Put("delete-me.txt", bytes.NewBufferString("synthetic disposable original"))
	if e != nil {
		t.Fatal(e)
	}
	path := "/api/assets/" + a.ID + "/delete"
	encoded, _ := json.Marshal(map[string]any{"confirm": a.ID, "added": a.Added})
	body := string(encoded)
	if w := request(s, "POST", path, body, nil); w.Code != 401 {
		t.Fatal("unprotected deletion")
	}
	if w := request(s, "POST", path, body, c); w.Code != 400 {
		t.Fatal("visible original deleted", w.Code)
	}
	s.Store.SetHidden(a.ID, true)
	for _, input := range []string{`{}`, `{"confirm":"wrong"}`, body + `{}`, `{"confirm":"` + a.ID + `","extra":1}`} {
		if w := request(s, "POST", path, input, c); w.Code != 400 {
			t.Fatal("bad confirmation accepted", w.Code)
		}
	}
	if w := request(s, "POST", path, body, c); w.Code != 200 {
		t.Fatal(w.Code, w.Body.String())
	}
	if w := request(s, "GET", "/api/assets/"+a.ID+"/original", "", c); w.Code != 404 {
		t.Fatal("deleted original still served")
	}
	w := request(s, "GET", "/api/deletions", "", c)
	var jobs struct{ Pending []any }
	if w.Code != 200 || json.Unmarshal(w.Body.Bytes(), &jobs) != nil || len(jobs.Pending) != 0 {
		t.Fatal("bad cleanup status")
	}
	if w = request(s, "POST", "/api/deletions/"+a.ID+"/retry", body, c); w.Code != 404 {
		t.Fatal("missing recovery job accepted", w.Code)
	}
}
