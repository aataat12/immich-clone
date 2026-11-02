package web

import (
	"bufio"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestConcurrentOriginalCopiesFailFastAndRecover(t *testing.T) {
	s, c := setup(t)
	body := &slowLoginBody{make(chan struct{}), make(chan struct{})}
	r := httptest.NewRequest("POST", "http://127.0.0.1:2283/api/upload?name=held.txt", body)
	r.AddCookie(c)
	r.Header.Set("X-Immich-Request", "1")
	done := make(chan struct{})
	go func() { defer close(done); s.ServeHTTP(httptest.NewRecorder(), r) }()
	<-body.entered
	defer func() { close(body.release); <-done }()
	for _, path := range []string{"/api/upload?name=second.txt", "/api/imports/missing/files/0"} {
		w := request(s, "POST", path, "rejected", c)
		if w.Code != 429 || w.Header().Get("Retry-After") != "1" {
			t.Fatalf("concurrent copy not rejected: %d", w.Code)
		}
	}

	for _, path := range []string{"/api/assets", "/api/analysis-batch"} {
		result := make(chan int, 1)
		go func() { result <- request(s, "GET", path, "", c).Code }()
		select {
		case status := <-result:
			if status != 200 {
				t.Fatalf("read/status unavailable: %d", status)
			}
		case <-time.After(time.Second):
			t.Fatal("read/status blocked by copy")
		}
	}

	// A rejected real connection must not wait to drain a body it never accepted.
	server := httptest.NewServer(s)
	defer server.Close()
	address := strings.TrimPrefix(server.URL, "http://")
	conn, e := net.DialTimeout("tcp", address, time.Second)
	if e != nil {
		t.Fatal(e)
	}
	defer conn.Close()
	conn.SetDeadline(time.Now().Add(2 * time.Second))
	_, e = fmt.Fprintf(conn, "POST /api/upload?name=second.txt HTTP/1.1\r\nHost: %s\r\nCookie: session=%s\r\nX-Immich-Request: 1\r\nContent-Length: 1000\r\n\r\npartial", address, c.Value)
	if e != nil {
		t.Fatal(e)
	}
	response, e := http.ReadResponse(bufio.NewReader(conn), nil)
	if e != nil {
		t.Fatal("busy response stalled on rejected body", e)
	}
	defer response.Body.Close()
	if response.StatusCode != 429 || !response.Close {
		t.Fatal("busy connection not rejected/closed")
	}
}

func TestOriginalCopySlotReleasedAfterFailure(t *testing.T) {
	s, c := setup(t)
	// Empty input fails, then a real original must still be accepted exactly once.
	if request(s, "POST", "/api/upload?name=empty.txt", "", c).Code == 200 {
		t.Fatal("empty copy succeeded")
	}
	if w := request(s, "POST", "/api/upload?name=accepted.txt", "preserved original", c); w.Code != 200 {
		t.Fatalf("copy slot stranded: %d", w.Code)
	}
	assets, e := s.Store.List()
	if e != nil || len(assets) != 1 {
		t.Fatal("rejected/failed copy created a catalog entry")
	}
	if e = s.Store.Verify(); e != nil {
		t.Fatal(e)
	}
}
