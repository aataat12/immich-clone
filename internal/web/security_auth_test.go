package web

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// Exercise the common guard for every current sensitive route family, including
// nonexistent IDs: resource existence must not bypass or precede authentication.
func TestSensitiveRouteGuardMatrix(t *testing.T) {
	s, cookie := setup(t)
	routes := []struct{ method, path string }{
		{"GET", "/api/gallery"}, {"GET", "/api/assets"}, {"GET", "/api/organization"},
		{"GET", "/api/export"}, {"GET", "/api/diagnostics"}, {"GET", "/api/backup"},
		{"GET", "/api/backup-plan"}, {"GET", "/api/backup-job"}, {"GET", "/api/backup-job/download?id=missing"},
		{"POST", "/api/backup-job/start"}, {"POST", "/api/backup-job/cancel"}, {"POST", "/api/backup-job/discard"},
		{"GET", "/api/analysis-batch"}, {"POST", "/api/analysis-batch"},
		{"GET", "/api/imports"}, {"POST", "/api/imports"}, {"GET", "/api/imports/missing"},
		{"POST", "/api/imports/missing/resume"}, {"POST", "/api/imports/missing/pause"},
		{"POST", "/api/imports/missing/cancel"}, {"POST", "/api/imports/missing/files/0"},
		{"POST", "/api/upload?name=guard.txt"}, {"POST", "/api/faces/label"},
		{"POST", "/api/faces/missing/dismiss"}, {"GET", "/api/faces/missing/context"}, {"GET", "/api/faces/missing/crop"}, {"POST", "/api/faces/missing/undo"},
	}
	for _, action := range []string{"original", "preview", "view", "retry", "hide", "unhide", "analyze", "tags", "dismiss-tag", "manual-face", "video-analysis"} {
		method := "POST"
		if action == "original" || action == "preview" || action == "view" {
			method = "GET"
		}
		routes = append(routes, struct{ method, path string }{method, "/api/assets/missing/" + action})
	}
	expired := &http.Cookie{Name: "session", Value: "expired-canary"}
	s.sessions[expired.Value] = time.Now().Add(-time.Second)
	for _, route := range routes {
		t.Run(route.method+route.path, func(t *testing.T) {
			for _, c := range []*http.Cookie{nil, expired, {Name: "session", Value: "forged-canary"}} {
				if w := request(s, route.method, route.path, `{}`, c); w.Code != 401 {
					t.Fatalf("unauthorized route returned %d", w.Code)
				}
			}
			r := httptest.NewRequest(route.method, "http://attacker.invalid"+route.path, strings.NewReader(`{}`))
			r.AddCookie(cookie)
			r.Header.Set("X-Immich-Request", "1")
			w := httptest.NewRecorder()
			s.ServeHTTP(w, r)
			if w.Code != 403 {
				t.Fatalf("foreign Host returned %d", w.Code)
			}
			if route.method == "POST" {
				for _, origin := range []string{"https://attacker.invalid", "null", "http://localhost:9999"} {
					r = httptest.NewRequest(route.method, "http://127.0.0.1:2283"+route.path, strings.NewReader(`{}`))
					r.AddCookie(cookie)
					r.Header.Set("X-Immich-Request", "1")
					r.Header.Set("Origin", origin)
					w = httptest.NewRecorder()
					s.ServeHTTP(w, r)
					if w.Code != 403 {
						t.Fatalf("foreign Origin returned %d", w.Code)
					}
				}
				r = httptest.NewRequest(route.method, "http://127.0.0.1:2283"+route.path, strings.NewReader(`{}`))
				r.AddCookie(cookie)
				w = httptest.NewRecorder()
				s.ServeHTTP(w, r)
				if w.Code != 403 {
					t.Fatalf("missing CSRF header returned %d", w.Code)
				}
			}
		})
	}
	if w := request(s, "GET", "/api/gallery?sort=capture&visibility=library", "", cookie); w.Code != 200 {
		t.Fatal("legitimate session blocked")
	}
}
