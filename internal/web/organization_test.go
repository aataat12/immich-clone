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
