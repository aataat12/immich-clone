package web

import (
	"bytes"
	"encoding/json"
	"net/http/httptest"
	"testing"
)

func TestActualBuildAndProtectedVisibility(t *testing.T) {
	base, c := setup(t)
	s := New(base.Store, base.Root, base.Code, "0.1.11-prototype")
	// Share only the authenticated test session, as a normal server constructor does.
	s.sessions = base.sessions
	health := request(s, "GET", "/health", "", nil)
	var h struct{ Build string }
	if e := json.Unmarshal(health.Body.Bytes(), &h); e != nil || h.Build != "0.1.11-prototype" {
		t.Fatal("health did not expose configured executable build")
	}
	a, _, e := s.Store.Put("synthetic.txt", bytes.NewBufferString("synthetic visibility original"))
	if e != nil {
		t.Fatal(e)
	}
	path := "/api/assets/" + a.ID
	if w := request(s, "POST", path+"/hide", "", nil); w.Code != 401 {
		t.Fatal("unauthenticated hide permitted")
	}
	if w := request(s, "POST", path+"/hide", "", c); w.Code != 200 {
		t.Fatal("hide failed")
	}
	a, _ = s.Store.Get(a.ID)
	if !a.Hidden {
		t.Fatal("hide not persisted")
	}
	r := httptest.NewRequest("POST", "http://127.0.0.1:2283"+path+"/unhide", nil)
	r.AddCookie(c)
	r.Header.Set("X-Immich-Request", "1")
	r.Header.Set("Origin", "http://invalid.example")
	w := httptest.NewRecorder()
	s.ServeHTTP(w, r)
	if w.Code != 403 {
		t.Fatal("cross-origin unhide permitted")
	}
	if w := request(s, "GET", path+"/original", "", c); w.Code != 200 || w.Body.String() != "synthetic visibility original" {
		t.Fatal("hidden original is unavailable or changed")
	}
	if w := request(s, "POST", path+"/unhide", "", c); w.Code != 200 {
		t.Fatal("unhide failed")
	}
	a, _ = s.Store.Get(a.ID)
	if a.Hidden {
		t.Fatal("unhide not persisted")
	}
}
