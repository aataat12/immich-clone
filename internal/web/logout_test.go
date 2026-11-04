package web

import (
	"net/http"
	"testing"
)

func TestLogoutRevokesOnlyCurrentSession(t *testing.T) {
	s, first := setup(t)
	w := request(s, "POST", "/api/login", `{"code":"`+s.Code+`"}`, nil)
	second := w.Result().Cookies()[0]
	for _, path := range []string{"/login", "/logout"} {
		if request(s, "GET", path, "", first).Code != 200 {
			t.Fatal("missing authentication page")
		}
	}
	if request(s, "GET", "/api/assets", "", first).Code != 200 {
		t.Fatal("GET page signed out session")
	}
	w = request(s, "POST", "/api/logout", "", first)
	if w.Code != 200 {
		t.Fatal("logout failed")
	}
	cookies := w.Result().Cookies()
	if len(cookies) != 1 || cookies[0].Name != "session" || cookies[0].MaxAge >= 0 || cookies[0].Path != "/" || !cookies[0].HttpOnly || cookies[0].SameSite != http.SameSiteStrictMode {
		t.Fatal("logout cookie not securely cleared")
	}
	if request(s, "GET", "/api/assets", "", first).Code != 401 {
		t.Fatal("replayed revoked cookie accepted")
	}
	if request(s, "GET", "/api/assets", "", second).Code != 200 {
		t.Fatal("unrelated session revoked")
	}
	if request(s, "POST", "/api/logout", "", nil).Code != 200 {
		t.Fatal("logout not idempotent")
	}
}
