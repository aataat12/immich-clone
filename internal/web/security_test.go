package web

import (
	"crypto/sha256"
	"encoding/base64"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestBundledScriptsOnlyPolicy(t *testing.T) {
	s, _ := setup(t)
	for _, path := range []string{"/", "/people", "/import", "/backups", "/build-notes"} {
		t.Run(path, func(t *testing.T) {
			w := httptest.NewRecorder()
			s.ServeHTTP(w, httptest.NewRequest("GET", "http://127.0.0.1:2283"+path, nil))
			policy := w.Header().Get("Content-Security-Policy")
			scriptPolicy := strings.Split(strings.Split(policy, "script-src ")[1], ";")[0]
			if strings.Contains(scriptPolicy, "unsafe-inline") || strings.Contains(scriptPolicy, "unsafe-eval") {
				t.Fatal("permissive script policy")
			}
			rest := w.Body.String()
			for {
				_, after, found := strings.Cut(rest, "<script>")
				if !found {
					break
				}
				script, tail, closed := strings.Cut(after, "</script>")
				if !closed {
					t.Fatal("unclosed script")
				}
				// A browser normalizes HTML CRLF/CR before checking CSP hashes.
				normalized := strings.ReplaceAll(strings.ReplaceAll(script, "\r\n", "\n"), "\r", "\n")
				digest := sha256.Sum256([]byte(normalized))
				if !strings.Contains(scriptPolicy, "'sha256-"+base64.StdEncoding.EncodeToString(digest[:])+"'") {
					t.Fatal("bundled script blocked")
				}
				changed := sha256.Sum256([]byte(script + ";alert('injected')"))
				if strings.Contains(scriptPolicy, base64.StdEncoding.EncodeToString(changed[:])) {
					t.Fatal("modified script authorized")
				}
				rest = tail
			}
			if !strings.Contains(policy, "object-src 'none'") || !strings.Contains(policy, "form-action 'self'") {
				t.Fatal("missing object/form restrictions")
			}
		})
	}
}

func TestScriptDigestHTMLLineEndings(t *testing.T) {
	want := scriptDigest("first\nsecond\n")
	for _, script := range []string{"first\r\nsecond\r\n", "first\rsecond\r"} {
		if scriptDigest(script) != want {
			t.Fatal("Windows/HTML line normalization mismatch")
		}
	}
}
