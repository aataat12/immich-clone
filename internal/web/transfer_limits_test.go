package web

import (
	"bufio"
	"encoding/json"
	"fmt"
	"immichclone/internal/library"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

func copySocket(t *testing.T, s *Server, cookie *http.Cookie, paths ...string) (net.Conn, *bufio.Reader) {
	t.Helper()
	server := httptest.NewServer(s)
	t.Cleanup(server.Close)
	address := strings.TrimPrefix(server.URL, "http://")
	conn, e := net.DialTimeout("tcp", address, time.Second)
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() { conn.Close() })
	conn.SetDeadline(time.Now().Add(5 * time.Second))
	path := "/api/upload?name=stream.txt"
	if len(paths) > 0 {
		path = paths[0]
	}
	_, e = fmt.Fprintf(conn, "POST %s HTTP/1.1\r\nHost: %s\r\nCookie: session=%s\r\nX-Immich-Request: 1\r\nContent-Length: 4\r\n\r\n", path, address, cookie.Value)
	if e != nil {
		t.Fatal(e)
	}
	return conn, bufio.NewReader(conn)
}
func TestOriginalCopyIdleDeadlineAndRecovery(t *testing.T) {
	s, c := setup(t)
	s.copyIdle = 300 * time.Millisecond
	conn, reader := copySocket(t, s, c)
	conn.Write([]byte("a"))
	response, e := http.ReadResponse(reader, nil)
	if e != nil {
		t.Fatal("stalled copy did not terminate", e)
	}
	response.Body.Close()
	conn.Close()
	if response.StatusCode != 408 || !response.Close {
		t.Fatal("stalled copy not rejected/closed", response.StatusCode)
	}
	assets, e := s.Store.List()
	if e != nil || len(assets) != 0 {
		t.Fatal("partial copy was accepted")
	}
	if w := request(s, "POST", "/api/upload?name=retry.txt", "valid copy", c); w.Code != 200 {
		t.Fatal("timeout stranded the copy slot", w.Code)
	}
	if e = s.Store.Verify(); e != nil {
		t.Fatal(e)
	}
}
func TestProgressingCopyCanExceedIdleWindow(t *testing.T) {
	s, c := setup(t)
	s.copyIdle = time.Second
	conn, reader := copySocket(t, s, c)
	started := time.Now()
	for _, b := range []byte("data") {
		if _, e := conn.Write([]byte{b}); e != nil {
			t.Fatal(e)
		}
		time.Sleep(350 * time.Millisecond)
	}
	response, e := http.ReadResponse(reader, nil)
	if e != nil {
		t.Fatal(e)
	}
	response.Body.Close()
	if response.StatusCode != 200 || time.Since(started) <= s.copyIdle {
		t.Fatal("progressing copy incorrectly timed out", response.StatusCode)
	}
	if e = s.Store.Verify(); e != nil {
		t.Fatal(e)
	}
}

type heldExportWriter struct {
	*httptest.ResponseRecorder
	entered, release chan struct{}
	once             sync.Once
}

func (w *heldExportWriter) Write(p []byte) (int, error) {
	w.once.Do(func() { close(w.entered) })
	<-w.release
	return w.ResponseRecorder.Write(p)
}
func TestSlowExportBoundsOverlapWithoutBlockingGallery(t *testing.T) {
	s, c := setup(t)
	w := &heldExportWriter{ResponseRecorder: httptest.NewRecorder(), entered: make(chan struct{}), release: make(chan struct{})}
	r := httptest.NewRequest("GET", "http://127.0.0.1:2283/api/export", nil)
	r.AddCookie(c)
	done := make(chan struct{})
	go func() { defer close(done); s.ServeHTTP(w, r) }()
	select {
	case <-w.entered:
	case <-time.After(time.Second):
		t.Fatal("export never started")
	}
	defer func() { close(w.release); <-done }()
	for _, path := range []string{"/api/assets", "/api/export", "/api/backup"} {
		if result := request(s, "GET", path, "", c); result.Code != 429 || result.Header().Get("Retry-After") != "1" {
			t.Fatal("overlapping heavy operation not rejected", result.Code)
		}
	}
	for _, path := range []string{"/api/gallery?sort=capture&visibility=library", "/api/backup-job", "/api/analysis-batch"} {
		if result := request(s, "GET", path, "", c); result.Code != 200 {
			t.Fatal("independent browsing/status blocked", result.Code)
		}
	}
}

func TestBulkCopyIdleTimeoutPreservesReceipts(t *testing.T) {
	s, c := setup(t)
	s.copyIdle = 300 * time.Millisecond
	created := request(s, "POST", "/api/imports", `{"files":[{"name":"kept.txt","relative":"Synthetic/kept.txt","size":4,"modified":1},{"name":"retry.txt","relative":"Synthetic/retry.txt","size":4,"modified":1}]}`, c)
	var session library.ImportSession
	if created.Code != 200 || json.Unmarshal(created.Body.Bytes(), &session) != nil {
		t.Fatal("manifest failed")
	}
	prefix := "/api/imports/" + session.ID
	if request(s, "POST", prefix+"/start", "", c).Code != 200 || request(s, "POST", prefix+"/files/0", "kept", c).Code != 200 {
		t.Fatal("initial receipt failed")
	}
	conn, reader := copySocket(t, s, c, prefix+"/files/1")
	conn.Write([]byte("x"))
	response, e := http.ReadResponse(reader, nil)
	if e != nil {
		t.Fatal("stalled bulk copy did not terminate", e)
	}
	response.Body.Close()
	conn.Close()
	if response.StatusCode != 408 || !response.Close {
		t.Fatal("bulk timeout not rejected/closed", response.StatusCode)
	}
	page := request(s, "GET", prefix, "", c)
	var before library.ImportPage
	if page.Code != 200 || json.Unmarshal(page.Body.Bytes(), &before) != nil || before.Session.AcceptedFiles != 1 || before.Session.AcceptedBytes != 4 || before.Entries[1].AssetID != "" {
		t.Fatal("timeout falsely accepted a receipt")
	}
	for i := 0; i < 2; i++ {
		if request(s, "POST", prefix+"/files/1", "data", c).Code != 200 {
			t.Fatal("receipt retry/replay failed")
		}
	}
	page = request(s, "GET", prefix, "", c)
	var after library.ImportPage
	if json.Unmarshal(page.Body.Bytes(), &after) != nil || after.Session.AcceptedFiles != 2 || after.Session.AcceptedBytes != 8 || after.Entries[0].AssetID != before.Entries[0].AssetID {
		t.Fatal("retry/replay changed receipt accounting")
	}
	if e = s.Store.Verify(); e != nil {
		t.Fatal(e)
	}
}
