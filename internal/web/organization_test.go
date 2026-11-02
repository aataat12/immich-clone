package web

import (
	"bytes"
	"context"
	"encoding/json"
	"image"
	"image/png"
	"immichclone/internal/recognition"
	"net/http/httptest"
	"testing"
)

func TestProtectedOrganizationAndManualCorrections(t *testing.T) {
	s, c := setup(t)
	if w := request(s, "GET", "/api/organization", "", nil); w.Code != 401 {
		t.Fatal("unprotected organization")
	}
	var b bytes.Buffer
	png.Encode(&b, image.NewRGBA(image.Rect(0, 0, 100, 100)))
	a, _, _ := s.Store.Put("synthetic.png", bytes.NewReader(b.Bytes()))
	s.Store.SetRecognizer(func(context.Context, image.Image, string) (recognition.Result, error) {
		v := make([]float64, 128)
		v[0] = 1
		return recognition.Result{Protocol: 1, Faces: []recognition.Face{{Box: []float64{.1, .1, .4, .4}, Confidence: .99, Embedding: v}}}, nil
	}, true)
	path := "/api/assets/" + a.ID
	for _, action := range []string{"analyze", "tags", "manual-face"} {
		if w := request(s, "POST", path+"/"+action, `{}`, nil); w.Code != 401 {
			t.Fatal("unprotected write")
		}
	}
	r := httptest.NewRequest("POST", "http://127.0.0.1:2283"+path+"/analyze", nil)
	r.AddCookie(c)
	r.Header.Set("Origin", "http://evil.example")
	r.Header.Set("X-Immich-Request", "1")
	w := httptest.NewRecorder()
	s.ServeHTTP(w, r)
	if w.Code != 403 {
		t.Fatal("cross origin AI")
	}
	if w := request(s, "POST", path+"/analyze", "", c); w.Code != 200 {
		t.Fatal(w.Body.String())
	}
	s.Store.ProcessAnalysis(context.Background())
	page, _ := s.Store.OrganizationPage("", "", "")
	id := page.Faces[0].ID
	body, _ := json.Marshal(map[string]any{"ids": []string{id}, "name": "Synthetic <person>"})
	if w := request(s, "POST", "/api/faces/label", string(body), c); w.Code != 200 {
		t.Fatal(w.Body.String())
	}
	if w := request(s, "POST", path+"/tags", `{"tags":["Synthetic tag"]}`, c); w.Code != 200 {
		t.Fatal(w.Body.String())
	}
	if w := request(s, "GET", "/api/faces/"+id+"/crop", "", c); w.Code != 200 || w.Header().Get("Content-Type") != "image/png" {
		t.Fatal("crop retrieval")
	}
	if w := request(s, "GET", "/api/organization", "", c); bytes.Contains(w.Body.Bytes(), []byte("embedding")) {
		t.Fatal("embeddings exposed to browser")
	}
	if w := request(s, "POST", path+"/manual-face", `{"box":[0.5,0.5,0.2,0.2],"name":"Manual"}`, c); w.Code != 200 {
		t.Fatal(w.Body.String())
	}
	if w := request(s, "POST", "/api/faces/label", `{"ids":[],"name":"bad"}`, c); w.Code != 400 {
		t.Fatal("invalid selection")
	}
	if e := s.Store.Verify(); e != nil {
		t.Fatal(e)
	}
}

func TestAnalysisBatchProtectionAndBuildNotes(t *testing.T) {
	s, c := setup(t)
	for _, method := range []string{"GET", "POST"} {
		if w := request(s, method, "/api/analysis-batch", `{"action":"start"}`, nil); w.Code != 401 {
			t.Fatal("unprotected batch", w.Code)
		}
	}
	r := httptest.NewRequest("POST", "http://127.0.0.1:2283/api/analysis-batch", bytes.NewBufferString(`{"action":"start"}`))
	r.AddCookie(c)
	r.Header.Set("X-Immich-Request", "1")
	r.Header.Set("Origin", "http://evil.example")
	w := httptest.NewRecorder()
	s.ServeHTTP(w, r)
	if w.Code != 403 {
		t.Fatal("cross-origin batch")
	}
	if w := request(s, "POST", "/api/analysis-batch", `{"action":"invalid"}`, c); w.Code != 409 {
		t.Fatal(w.Code)
	}
	if w := request(s, "POST", "/api/analysis-batch", `{"action":"start","unexpected":true}`, c); w.Code != 400 {
		t.Fatal("unknown field")
	}
	w = request(s, "GET", "/build-notes", "", c)
	if w.Code != 200 {
		t.Fatal(w.Code)
	}
	for _, term := range []string{"Build Notes", "Photos &amp; videos Home", "Status", "Released In", "0.1.25-prototype", "Bulk local AI analysis", "Final Windows 10 acceptance"} {
		if !bytes.Contains(w.Body.Bytes(), []byte(term)) {
			t.Fatal("missing notes", term)
		}
	}
	if bytes.Count(w.Body.Bytes(), []byte("<tr>")) != 9 {
		t.Fatal("milestone row count")
	}
}

func TestFaceDismissalAuthorizationAndRestore(t *testing.T) {
	s, c := setup(t)
	a, _, e := s.Store.Put("face.png", bytes.NewReader(func() []byte {
		var b bytes.Buffer
		png.Encode(&b, image.NewRGBA(image.Rect(0, 0, 100, 100)))
		return b.Bytes()
	}()))
	if e != nil {
		t.Fatal(e)
	}
	if e = s.Store.ManualFace(context.Background(), a.ID, []float64{.1, .1, .4, .4}, "Kept name"); e != nil {
		t.Fatal(e)
	}
	page, _ := s.Store.OrganizationPage("", "", "")
	id := page.Faces[0].ID
	path := "/api/faces/" + id + "/dismiss"
	if w := request(s, "POST", path, `{}`, nil); w.Code != 401 {
		t.Fatal("unprotected dismissal")
	}
	r := httptest.NewRequest("POST", "http://127.0.0.1:2283"+path, bytes.NewBufferString(`{}`))
	r.AddCookie(c)
	r.Header.Set("X-Immich-Request", "1")
	r.Header.Set("Origin", "http://evil.example")
	w := httptest.NewRecorder()
	s.ServeHTTP(w, r)
	if w.Code != 403 {
		t.Fatal("foreign origin dismissal")
	}
	if w := request(s, "POST", path, `{"unexpected":true}`, c); w.Code != 400 {
		t.Fatal("unknown dismissal input")
	}
	if w := request(s, "POST", path, `{}`, c); w.Code != 200 {
		t.Fatal(w.Body.String())
	}
	page, _ = s.Store.OrganizationPage("", "", "")
	if len(page.Faces) != 0 {
		t.Fatal("active dismissed face")
	}
	if w := request(s, "GET", "/api/organization?dismissed=1", "", c); w.Code != 200 || !bytes.Contains(w.Body.Bytes(), []byte(`"dismissed":true`)) {
		t.Fatal("dismissed review unavailable")
	}
	if w := request(s, "POST", path, `{"restore":true}`, c); w.Code != 200 {
		t.Fatal(w.Body.String())
	}
	page, _ = s.Store.OrganizationPage("", "", "")
	if len(page.Faces) != 1 || page.Faces[0].Name != "Kept name" {
		t.Fatal("restore lost name")
	}
}

func TestUnnamedPeopleFilterIsServerSide(t *testing.T) {
	s, c := setup(t)
	var b bytes.Buffer
	if e := png.Encode(&b, image.NewRGBA(image.Rect(0, 0, 100, 100))); e != nil {
		t.Fatal(e)
	}
	a, _, e := s.Store.Put("review.png", bytes.NewReader(b.Bytes()))
	if e != nil {
		t.Fatal(e)
	}
	for _, name := range []string{"Named", ""} {
		if e = s.Store.ManualFace(context.Background(), a.ID, []float64{.1, .1, .2, .2}, name); e != nil {
			t.Fatal(e)
		}
	}
	w := request(s, "GET", "/api/organization?unnamed=1", "", c)
	var page struct {
		Faces []struct {
			Name string `json:"name"`
		}
	}
	if w.Code != 200 || json.Unmarshal(w.Body.Bytes(), &page) != nil || len(page.Faces) != 1 || page.Faces[0].Name != "" {
		t.Fatal("server did not filter unnamed faces")
	}
}
