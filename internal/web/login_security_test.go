package web

import (
	"io"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

type slowLoginBody struct{ entered, release chan struct{} }

func (b *slowLoginBody) Read(p []byte) (int, error) { close(b.entered); <-b.release; return 0, io.EOF }
func (b *slowLoginBody) Close() error               { return nil }

func TestSlowLoginDoesNotBlockExistingSession(t *testing.T) {
	s, c := setup(t)
	body := &slowLoginBody{make(chan struct{}), make(chan struct{})}
	r := httptest.NewRequest("POST", "http://127.0.0.1:2283/api/login", body)
	r.Header.Set("X-Immich-Request", "1")
	done := make(chan struct{})
	go func() { defer close(done); s.ServeHTTP(httptest.NewRecorder(), r) }()
	<-body.entered
	result := make(chan int, 1)
	go func() { result <- request(s, "GET", "/api/assets", "", c).Code }()
	select {
	case code := <-result:
		if code != 200 {
			t.Errorf("existing session returned %d", code)
		}
	case <-time.After(time.Second):
		t.Error("slow login blocked existing-session authentication")
	}
	close(body.release)
	<-done
}

func TestLoginRejectsExtraAndOversizedJSON(t *testing.T) {
	s, _ := setup(t)
	valid := `{"code":"` + s.Code + `"}`
	for _, body := range []string{valid + `{}`, valid + strings.Repeat(" ", 1025), `{"code":"` + s.Code + `","unexpected":true}`} {
		w := request(s, "POST", "/api/login", body, nil)
		if w.Code != 401 || len(w.Result().Cookies()) != 0 {
			t.Fatal("malformed login created a session")
		}
	}
	if w := request(s, "POST", "/api/login", valid, nil); w.Code != 200 {
		t.Fatal("valid login rejected")
	}
}
