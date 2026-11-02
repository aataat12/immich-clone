package web

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"immichclone/internal/library"
	"net/http/httptest"
	"testing"
)

func TestImportAuthenticationManifestAndReceipts(t *testing.T) {
	s, c := setup(t)
	if w := request(s, "GET", "/api/imports", "", nil); w.Code != 401 {
		t.Fatal("unprotected history")
	}
	if w := request(s, "POST", "/api/imports", `{"files":[]}`, nil); w.Code != 401 {
		t.Fatal("unprotected manifest")
	}
	r := httptest.NewRequest("POST", "http://127.0.0.1:2283/api/imports", bytes.NewBufferString(`{"files":[]}`))
	r.AddCookie(c)
	r.Header.Set("Origin", "http://evil.example")
	r.Header.Set("X-Immich-Request", "1")
	w := httptest.NewRecorder()
	s.ServeHTTP(w, r)
	if w.Code != 403 {
		t.Fatal("cross-origin import")
	}
	body := `{"files":[{"name":"x.txt","relative":"Synthetic/x.txt","size":4,"modified":1}]}`
	w = request(s, "POST", "/api/imports", body, c)
	if w.Code != 200 {
		t.Fatal(w.Body.String())
	}
	var m library.ImportSession
	json.Unmarshal(w.Body.Bytes(), &m)
	prefix := "/api/imports/" + m.ID
	if w = request(s, "POST", prefix+"/files/0", "test", c); w.Code != 409 {
		t.Fatal("copy without lease")
	}
	if w = request(s, "POST", prefix+"/start", "", c); w.Code != 200 {
		t.Fatal(w.Body.String())
	}
	if w = request(s, "POST", prefix+"/files/0", "test", c); w.Code != 200 {
		t.Fatal(w.Body.String())
	}
	if w = request(s, "POST", prefix+"/finish", "", c); w.Code != 200 {
		t.Fatal(w.Body.String())
	}
	w = request(s, "GET", prefix, "", c)
	var p library.ImportPage
	json.Unmarshal(w.Body.Bytes(), &p)
	if p.Session.AcceptedFiles != 1 || p.Session.AcceptedBytes != 4 || p.BackgroundPaused {
		t.Fatal(w.Body.String())
	}
	if w = request(s, "GET", prefix+"?after=-1", "", c); w.Code != 404 {
		t.Fatal("negative page")
	}
	if e := s.Store.Verify(); e != nil {
		t.Fatal(e)
	}
	if ok, e := s.Store.ProcessOneContext(context.Background()); !ok || e != nil {
		t.Fatal("queue stalled", fmt.Sprint(e))
	}
}
