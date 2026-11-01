package web

import (
	"bytes"
	"encoding/json"
	"image"
	"image/png"
	"immichclone/internal/library"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestFullImageViewAndRange(t *testing.T) {
	s, c := setup(t)
	var source bytes.Buffer
	if e := png.Encode(&source, image.NewRGBA(image.Rect(0, 0, 8, 8))); e != nil {
		t.Fatal(e)
	}
	a, _, e := s.Store.Put("view.png", bytes.NewReader(source.Bytes()))
	if e != nil {
		t.Fatal(e)
	}
	r := httptest.NewRequest("GET", "http://127.0.0.1:2283/api/assets/"+a.ID+"/view", nil)
	r.AddCookie(c)
	r.Header.Set("Range", "bytes=0-7")
	w := httptest.NewRecorder()
	s.ServeHTTP(w, r)
	if w.Code != 206 || !bytes.Equal(w.Body.Bytes(), source.Bytes()[:8]) || w.Header().Get("Content-Type") != "image/png" {
		t.Fatal("full-size media range retrieval failed")
	}
	if e = s.Store.Verify(); e != nil {
		t.Fatal(e)
	}
}

func request(s *Server, method, path, body string, cookie *http.Cookie) *httptest.ResponseRecorder {
	r := httptest.NewRequest(method, "http://127.0.0.1:2283"+path, bytes.NewBufferString(body))
	r.Header.Set("X-Immich-Request", "1")
	if cookie != nil {
		r.AddCookie(cookie)
	}
	w := httptest.NewRecorder()
	s.ServeHTTP(w, r)
	return w
}
func setup(t *testing.T) (*Server, *http.Cookie) {
	t.Helper()
	root := filepath.Join(t.TempDir(), "state")
	store, e := library.Open(root)
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() { store.Close() })
	code, e := AccessCode(root)
	if e != nil {
		t.Fatal(e)
	}
	s := New(store, root, code)
	w := request(s, "POST", "/api/login", `{"code":"`+code+`"}`, nil)
	if w.Code != 200 {
		t.Fatal("login failed", w.Body.String())
	}
	cookies := w.Result().Cookies()
	if len(cookies) != 1 || !cookies[0].HttpOnly || cookies[0].SameSite != http.SameSiteStrictMode {
		t.Fatal("insecure cookie")
	}
	return s, cookies[0]
}
func TestAuthenticationHostOriginAndPrivacy(t *testing.T) {
	s, c := setup(t)
	if w := request(s, "GET", "/api/assets", "", nil); w.Code != 401 {
		t.Fatal("unauthorized catalog exposed")
	}
	r := httptest.NewRequest("POST", "http://127.0.0.1:2283/api/upload", bytes.NewBufferString("bad"))
	r.AddCookie(c)
	r.Header.Set("X-Immich-Request", "1")
	r.Header.Set("Origin", "https://attacker.example")
	w := httptest.NewRecorder()
	s.ServeHTTP(w, r)
	if w.Code != 403 {
		t.Fatal("foreign origin accepted")
	}
	r = httptest.NewRequest("GET", "http://attacker.example/api/assets", nil)
	r.AddCookie(c)
	w = httptest.NewRecorder()
	s.ServeHTTP(w, r)
	if w.Code != 403 {
		t.Fatal("DNS rebinding host accepted")
	}
	r = httptest.NewRequest("POST", "http://127.0.0.1:2283/api/upload", bytes.NewBufferString("bad"))
	r.AddCookie(c)
	w = httptest.NewRecorder()
	s.ServeHTTP(w, r)
	if w.Code != 403 {
		t.Fatal("missing CSRF header accepted")
	}
	w = request(s, "GET", "/", "", nil)
	if bytes.Contains(w.Body.Bytes(), []byte(s.Code)) {
		t.Fatal("secret exposed by UI")
	}
	for i := 0; i < 11; i++ {
		w = request(s, "POST", "/api/login", `{"code":"wrong"}`, nil)
	}
	if w.Code != 429 {
		t.Fatal("login throttle failed")
	}
}
func TestHTTPUploadDownloadExportBackupRestore(t *testing.T) {
	s, c := setup(t)
	original := "synthetic original bytes"
	w := request(s, "POST", "/api/upload?name=sample.txt", original, c)
	if w.Code != 200 {
		t.Fatal(w.Code, w.Body.String())
	}
	var result struct {
		Asset     library.Asset `json:"asset"`
		Duplicate bool          `json:"duplicate"`
	}
	if e := json.Unmarshal(w.Body.Bytes(), &result); e != nil {
		t.Fatal(e)
	}
	w = request(s, "GET", "/api/assets/"+result.Asset.ID+"/original", "", c)
	if w.Code != 200 || w.Body.String() != original || w.Header().Get("Content-Disposition") == "" {
		t.Fatal("original download mismatch")
	}
	w = request(s, "POST", "/api/upload?name=again.txt", original, c)
	json.Unmarshal(w.Body.Bytes(), &result)
	if !result.Duplicate {
		t.Fatal("HTTP duplicate not detected")
	}
	w = request(s, "GET", "/api/export", "", c)
	if w.Code != 200 || !bytes.Contains(w.Body.Bytes(), []byte(result.Asset.ID)) {
		t.Fatal("catalog export failed")
	}
	w = request(s, "GET", "/api/backup", "", c)
	if w.Code != 200 {
		t.Fatal(w.Body.String())
	}
	path := filepath.Join(t.TempDir(), "backup.zip")
	if e := os.WriteFile(path, w.Body.Bytes(), 0600); e != nil {
		t.Fatal(e)
	}
	target := filepath.Join(t.TempDir(), "restored")
	if e := library.Restore(path, target); e != nil {
		t.Fatal(e)
	}
	restored, e := library.Open(target)
	if e != nil {
		t.Fatal(e)
	}
	defer restored.Close()
	f, e := os.Open(restored.Original(result.Asset.ID))
	if e != nil {
		t.Fatal(e)
	}
	data, _ := io.ReadAll(f)
	f.Close()
	if string(data) != original {
		t.Fatal("backup original mismatch")
	}
}

func TestDiagnosticDownloadIsAuthenticated(t *testing.T) {
	s, c := setup(t)
	if w := request(s, "GET", "/api/diagnostics", "", nil); w.Code != 401 {
		t.Fatal("diagnostic log exposed without login")
	}
	w := request(s, "GET", "/api/diagnostics?private_query=DO_NOT_LOG_THIS", "", c)
	if w.Code != 200 || !bytes.Contains(w.Body.Bytes(), []byte("Immich Clone diagnostic log")) || bytes.Contains(w.Body.Bytes(), []byte("DO_NOT_LOG_THIS")) {
		t.Fatal("diagnostic export failed or leaked query")
	}
}
