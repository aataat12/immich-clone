package web

import (
	"crypto/rand"
	"crypto/subtle"
	"embed"
	"encoding/hex"
	"encoding/json"
	"errors"
	"immichclone/internal/diagnostics"
	"immichclone/internal/library"
	"io"
	"mime"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

//go:embed index.html
var content embed.FS

type Server struct {
	Store    *library.Store
	Code     string
	Root     string
	mu       sync.Mutex
	sessions map[string]time.Time
	attempts int
	window   time.Time
}

func AccessCode(root string) (string, error) {
	path := filepath.Join(root, "access-code.txt")
	data, e := os.ReadFile(path)
	if e == nil {
		code := strings.TrimSpace(string(data))
		if len(code) != 64 {
			return "", errors.New("invalid access code file")
		}
		if _, e = hex.DecodeString(code); e != nil {
			return "", e
		}
		return code, nil
	}
	if !errors.Is(e, os.ErrNotExist) {
		return "", e
	}
	code := random()
	f, e := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if e != nil {
		return "", e
	}
	_, e = f.WriteString(code + "\n")
	if e == nil {
		e = f.Sync()
	}
	ce := f.Close()
	if e == nil {
		e = ce
	}
	return code, e
}
func random() string {
	var b [32]byte
	if _, e := rand.Read(b[:]); e != nil {
		panic(e)
	}
	return hex.EncodeToString(b[:])
}
func New(store *library.Store, root, code string) *Server {
	return &Server{Store: store, Root: root, Code: code, sessions: map[string]time.Time{}}
}
func jsonResponse(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v)
}

type observedResponse struct {
	http.ResponseWriter
	status int
	bytes  int64
}

func (w *observedResponse) WriteHeader(status int) {
	if w.status == 0 {
		w.status = status
		w.ResponseWriter.WriteHeader(status)
	}
}
func (w *observedResponse) Write(b []byte) (int, error) {
	if w.status == 0 {
		w.WriteHeader(200)
	}
	n, e := w.ResponseWriter.Write(b)
	w.bytes += int64(n)
	return n, e
}
func (w *observedResponse) Unwrap() http.ResponseWriter { return w.ResponseWriter }
func requestStage(path string) string {
	switch path {
	case "/":
		return "ui"
	case "/health":
		return "health"
	case "/api/assets":
		return "assets"
	case "/api/login":
		return "login"
	case "/api/upload":
		return "original_write"
	case "/api/backup":
		return "archive_write"
	case "/api/export":
		return "export"
	case "/api/diagnostics":
		return "diagnostics"
	}
	if strings.HasPrefix(path, "/api/assets/") {
		switch last := path[strings.LastIndex(path, "/")+1:]; last {
		case "original", "preview", "view", "retry":
			if last == "preview" {
				return "preview_generate"
			}
			return last
		}
	}
	return "unknown"
}
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	observed := &observedResponse{ResponseWriter: w}
	w = observed
	started := time.Now()
	defer func() {
		status := observed.status
		if status == 0 {
			status = 200
		}
		diagnostics.Record("http", requestStage(r.URL.Path), nil, status, observed.bytes, time.Since(started))
	}()

	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("X-Frame-Options", "DENY")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Content-Security-Policy", "default-src 'self'; script-src 'self' 'unsafe-inline'; style-src 'self' 'unsafe-inline'; img-src 'self'; connect-src 'self'; frame-ancestors 'none'; base-uri 'none'")
	host, _, e := net.SplitHostPort(r.Host)
	if e != nil {
		host = r.Host
	}
	if host != "127.0.0.1" && host != "localhost" && host != "::1" {
		http.Error(w, "loopback host required", 403)
		return
	}
	if r.Method != "GET" && r.Method != "HEAD" {
		origin := r.Header.Get("Origin")
		if origin != "" && origin != "http://"+r.Host {
			http.Error(w, "origin rejected", 403)
			return
		}
		if r.Header.Get("X-Immich-Request") != "1" {
			http.Error(w, "request header required", 403)
			return
		}
	}
	if r.URL.Path == "/" && r.Method == "GET" {
		b, _ := content.ReadFile("index.html")
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write(b)
		return
	}
	if r.URL.Path == "/health" && r.Method == "GET" {
		jsonResponse(w, map[string]string{"status": "ok", "build": "prototype"})
		return
	}
	if r.URL.Path == "/api/login" && r.Method == "POST" {
		s.login(w, r)
		return
	}
	if !s.authenticated(r) {
		http.Error(w, "sign in required", 401)
		return
	}
	switch {
	case r.URL.Path == "/api/assets" && r.Method == "GET":
		assets, e := s.Store.List()
		if e != nil {
			http.Error(w, "catalog unavailable", 500)
			return
		}
		jsonResponse(w, assets)
	case r.URL.Path == "/api/diagnostics" && r.Method == "GET":
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.Header().Set("Content-Disposition", `attachment; filename="ImmichClone-diagnostics.log"`)
		if e := diagnostics.Export(w); e != nil {
			diagnostics.Record("logging", "failed", e, 0, 0, 0)
		}
	case r.URL.Path == "/api/export" && r.Method == "GET":
		assets, e := s.Store.List()
		if e != nil {
			http.Error(w, "catalog unavailable", 500)
			return
		}
		w.Header().Set("Content-Disposition", `attachment; filename="catalog.json"`)
		jsonResponse(w, map[string]any{"schema": 1, "assets": assets})
	case r.URL.Path == "/api/upload" && r.Method == "POST":
		r.Body = http.MaxBytesReader(w, r.Body, library.MaxUpload)
		name := r.URL.Query().Get("name")
		a, duplicate, e := s.Store.PutWithPath(name, r.URL.Query().Get("relative"), r.Body)
		if e != nil {
			status := 500
			var max *http.MaxBytesError
			if errors.Is(e, library.ErrTooLarge) || errors.As(e, &max) {
				status = 413
			}
			http.Error(w, "upload failed; no success recorded", status)
			return
		}
		jsonResponse(w, map[string]any{"asset": a, "duplicate": duplicate})
	case r.URL.Path == "/api/backup" && r.Method == "GET":
		s.backup(w, r)
	case strings.HasPrefix(r.URL.Path, "/api/assets/"):
		parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/api/assets/"), "/")
		if len(parts) != 2 {
			http.NotFound(w, r)
			return
		}
		a, e := s.Store.Get(parts[0])
		if e != nil {
			http.NotFound(w, r)
			return
		}
		switch {
		case parts[1] == "view" && r.Method == "GET" && (a.MIME == "image/jpeg" || a.MIME == "image/png" || a.MIME == "image/gif" || a.MIME == "image/bmp" || a.MIME == "video/mp4"):
			w.Header().Set("Content-Type", a.MIME)
			w.Header().Set("Content-Disposition", mime.FormatMediaType("inline", map[string]string{"filename": a.Name}))
			http.ServeFile(w, r, s.Store.Original(a.ID))
		case parts[1] == "original" && r.Method == "GET":
			w.Header().Set("Content-Disposition", mime.FormatMediaType("attachment", map[string]string{"filename": a.Name}))
			w.Header().Set("Content-Type", "application/octet-stream")
			http.ServeFile(w, r, s.Store.Original(a.ID))
		case parts[1] == "preview" && r.Method == "GET" && a.Preview == "ready":
			w.Header().Set("Content-Type", "image/png")
			http.ServeFile(w, r, s.Store.PreviewPath(a.ID))
		case parts[1] == "retry" && r.Method == "POST":
			if e = s.Store.Retry(a.ID); e != nil {
				http.Error(w, "retry failed", 500)
				return
			}
			jsonResponse(w, map[string]string{"status": "queued"})
		default:
			http.NotFound(w, r)
		}
	default:
		http.NotFound(w, r)
	}
}
func (s *Server) authenticated(r *http.Request) bool {
	c, e := r.Cookie("session")
	if e != nil {
		return false
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	expires, ok := s.sessions[c.Value]
	return ok && time.Now().Before(expires)
}
func (s *Server) login(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now()
	if now.Sub(s.window) > time.Minute {
		s.attempts = 0
		s.window = now
	}
	s.attempts++
	if s.attempts > 10 {
		http.Error(w, "try again in one minute", 429)
		return
	}
	var input struct {
		Code string `json:"code"`
	}
	if json.NewDecoder(io.LimitReader(r.Body, 1024)).Decode(&input) != nil || subtle.ConstantTimeCompare([]byte(input.Code), []byte(s.Code)) != 1 {
		http.Error(w, "incorrect access code", 401)
		return
	}
	for k, expiry := range s.sessions {
		if now.After(expiry) {
			delete(s.sessions, k)
		}
	}
	if len(s.sessions) >= 32 {
		http.Error(w, "too many active sessions", 429)
		return
	}
	token := random()
	s.sessions[token] = now.Add(12 * time.Hour)
	http.SetCookie(w, &http.Cookie{Name: "session", Value: token, Path: "/", HttpOnly: true, SameSite: http.SameSiteStrictMode, MaxAge: 43200})
	jsonResponse(w, map[string]string{"status": "ok"})
}
func (s *Server) backup(w http.ResponseWriter, r *http.Request) {
	f, e := os.CreateTemp(s.Root, "download-backup-*.zip")
	if e != nil {
		http.Error(w, "backup staging unavailable", 500)
		return
	}
	name := f.Name()
	defer os.Remove(name)
	e = s.Store.Backup(f)
	if e == nil {
		e = f.Sync()
	}
	ce := f.Close()
	if e == nil {
		e = ce
	}
	if e != nil {
		http.Error(w, "backup failed; no archive delivered", 500)
		return
	}
	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", `attachment; filename="immich-backup.zip"`)
	http.ServeFile(w, r, name)
}
