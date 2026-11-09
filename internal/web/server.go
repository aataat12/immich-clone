package web

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"embed"
	"encoding/base64"
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

//go:embed index.html people.html import.html backups.html build-notes.html
var content embed.FS

// Authorize only the exact bundled inline scripts, never arbitrary injected code.
// HTML is immutable embedded content; user-controlled values are rendered as text.
var browserPolicy = bundledBrowserPolicy()

func bundledBrowserPolicy() string {
	policy := "default-src 'self'; script-src"
	for _, name := range []string{"index.html", "people.html", "import.html", "backups.html", "build-notes.html"} {
		body, err := content.ReadFile(name)
		if err != nil {
			panic(err)
		}
		rest := string(body)
		for {
			_, after, found := strings.Cut(rest, "<script>")
			if !found {
				break
			}
			script, tail, closed := strings.Cut(after, "</script>")
			if !closed {
				panic("unclosed bundled script")
			}
			digest := scriptDigest(script)
			policy += " 'sha256-" + base64.StdEncoding.EncodeToString(digest[:]) + "'"
			rest = tail
		}
	}
	return policy + "; object-src 'none'; style-src 'self' 'unsafe-inline'; img-src 'self'; connect-src 'self'; frame-ancestors 'none'; base-uri 'none'; form-action 'self'"
}

type Server struct {
	backups      *backupJobs
	Store        *library.Store
	Version      string
	Code         string
	Root         string
	mu           sync.Mutex
	copyIdle     time.Duration
	writeIdle    time.Duration
	uploadSlot   chan struct{}
	snapshotSlot chan struct{}
	sessions     map[string]time.Time
	attempts     int
	window       time.Time
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
func New(store *library.Store, root, code string, builds ...string) *Server {
	build := "prototype"
	if len(builds) > 0 {
		build = builds[0]
	}
	return &Server{Store: store, Root: root, Code: code, Version: build, sessions: map[string]time.Time{}, uploadSlot: make(chan struct{}, 1), snapshotSlot: make(chan struct{}, 1), copyIdle: 60 * time.Second, writeIdle: 60 * time.Second, backups: newBackupJobs(root, store)}
}
func jsonResponse(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v)
}

type observedResponse struct {
	http.ResponseWriter
	status   int
	bytes    int64
	idle     time.Duration
	writeErr error
}

func (w *observedResponse) WriteHeader(status int) {
	if w.status == 0 {
		w.status = status
		w.renewWriteDeadline()
		w.ResponseWriter.WriteHeader(status)
	}
}
func (w *observedResponse) Write(b []byte) (int, error) {
	if w.status == 0 {
		w.WriteHeader(200)
	}
	w.renewWriteDeadline()
	n, e := w.ResponseWriter.Write(b)
	if e != nil && w.writeErr == nil {
		w.writeErr = e
	}
	w.bytes += int64(n)
	return n, e
}
func (w *observedResponse) renewWriteDeadline() {
	if w.idle > 0 {
		_ = http.NewResponseController(w.ResponseWriter).SetWriteDeadline(time.Now().Add(w.idle))
	}
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
		case "original", "preview", "view", "retry", "hide", "unhide":
			if last == "preview" {
				return "preview_generate"
			}
			return last
		}
	}
	return "unknown"
}
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	observed := &observedResponse{ResponseWriter: w, idle: s.writeIdle}
	w = observed
	started := time.Now()
	defer func() {
		status := observed.status
		if status == 0 {
			status = 200
		}
		diagnostics.Record("http", requestStage(r.URL.Path), observed.writeErr, status, observed.bytes, time.Since(started))
	}()

	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("X-Frame-Options", "DENY")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Content-Security-Policy", browserPolicy)
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
	if (r.URL.Path == "/" || r.URL.Path == "/login" || r.URL.Path == "/logout" || r.URL.Path == "/people" || r.URL.Path == "/import" || r.URL.Path == "/backups" || r.URL.Path == "/build-notes") && r.Method == "GET" {
		name := "index.html"
		if r.URL.Path == "/people" {
			name = "people.html"
		}
		if r.URL.Path == "/import" {
			name = "import.html"
		}
		if r.URL.Path == "/backups" {
			name = "backups.html"
		}
		if r.URL.Path == "/build-notes" {
			name = "build-notes.html"
		}
		b, _ := content.ReadFile(name)
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write(b)
		return
	}
	if r.URL.Path == "/health" && r.Method == "GET" {
		jsonResponse(w, map[string]string{"status": "ok", "build": s.Version})
		return
	}
	if r.URL.Path == "/api/login" && r.Method == "POST" {
		s.login(w, r)
		return
	}
	if r.URL.Path == "/api/logout" && r.Method == "POST" {
		if cookie, err := r.Cookie("session"); err == nil {
			s.mu.Lock()
			delete(s.sessions, cookie.Value)
			s.mu.Unlock()
		}
		http.SetCookie(w, &http.Cookie{Name: "session", Value: "", Path: "/", HttpOnly: true, SameSite: http.SameSiteStrictMode, MaxAge: -1, Expires: time.Unix(1, 0)})
		jsonResponse(w, map[string]string{"status": "signed out"})
		return
	}
	if !s.authenticated(r) {
		http.Error(w, "sign in required", 401)
		return
	}
	// Copying is already serialized by the store. Do not accumulate waiting
	// requests (and their bodies) behind that writer, including bulk receipts.
	if originalCopyRequest(r) {
		select {
		case s.uploadSlot <- struct{}{}:
			defer func() { <-s.uploadSlot }()
		default:
			w.Header().Set("Retry-After", "1")
			w.Header().Set("Connection", "close")
			http.Error(w, "An original copy is already active. Retry when it finishes; accepted files are retained.", http.StatusTooManyRequests)
			return
		}
	}
	if r.Method == http.MethodGet && (r.URL.Path == "/api/assets" || r.URL.Path == "/api/export" || r.URL.Path == "/api/backup" || r.URL.Path == "/api/album-suggestions") {
		select {
		case s.snapshotSlot <- struct{}{}:
			defer func() { <-s.snapshotSlot }()
		default:
			w.Header().Set("Retry-After", "1")
			http.Error(w, "Another large catalog request or legacy backup is already active. Retry when it finishes.", http.StatusTooManyRequests)
			return
		}
	}
	if s.deletionHTTP(w, r) {
		return
	}
	if s.backupHTTP(w, r) {
		return
	}
	if s.importHTTP(w, r) {
		return
	}
	if s.organizationHTTP(w, r) {
		return
	}
	if s.organizationBatchHTTP(w, r) {
		return
	}
	if s.curationHTTP(w, r) {
		return
	}
	if s.galleryHTTP(w, r) {
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
		exported, e := s.Store.OrganizationExport()
		if e != nil {
			catalogExportError(w, e)
			return
		}
		w.Header().Set("Content-Disposition", `attachment; filename="catalog.json"`)
		jsonResponse(w, exported)
	case r.URL.Path == "/api/upload" && r.Method == "POST":
		r.Body = http.MaxBytesReader(w, r.Body, library.MaxUpload)
		name := r.URL.Query().Get("name")
		controller := http.NewResponseController(w)
		a, duplicate, e := s.Store.PutWithPath(name, r.URL.Query().Get("relative"), deadlineReader{r.Body, controller, s.copyIdle})
		if e != nil {
			status := 500
			var max *http.MaxBytesError
			if errors.Is(e, library.ErrTooLarge) || errors.As(e, &max) {
				status = 413
			}
			var timeout net.Error
			if errors.As(e, &timeout) && timeout.Timeout() {
				status = http.StatusRequestTimeout
			}
			w.Header().Set("Connection", "close")
			http.Error(w, "upload failed; no success recorded", status)
			return
		}
		_ = controller.SetReadDeadline(time.Time{})
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
		if s.assetOrganization(w, r, a.ID, parts[1]) {
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
		case parts[1] == "managed-location" && r.Method == "GET":
			if !a.Hidden {
				http.Error(w, "Open Hidden items to inspect managed storage.", 409)
				return
			}
			location, err := filepath.Abs(s.Store.Original(a.ID))
			if err != nil {
				http.Error(w, "Managed location unavailable", 500)
				return
			}
			jsonResponse(w, map[string]string{"path": location, "name": a.Name})
		case parts[1] == "preview" && r.Method == "GET" && a.Preview == "ready":
			w.Header().Set("Content-Type", "image/png")
			http.ServeFile(w, r, s.Store.PreviewPath(a.ID))
		case (parts[1] == "hide" || parts[1] == "unhide") && r.Method == "POST":
			hidden := parts[1] == "hide"
			if e = s.Store.SetHidden(a.ID, hidden); e != nil {
				http.Error(w, "visibility update failed", 500)
				return
			}
			jsonResponse(w, map[string]bool{"hidden": hidden})
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

func catalogExportError(w http.ResponseWriter, e error) {
	if errors.Is(e, library.ErrImportHistoryExportLimit) {
		http.Error(w, "Recorded import history is too large for JSON export. Use a full backup to preserve all records.", http.StatusRequestEntityTooLarge)
		return
	}
	http.Error(w, "catalog unavailable", http.StatusInternalServerError)
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
	// Reserve the attempt under the mutex, but never hold session access hostage
	// while reading an unauthenticated request body.
	s.mu.Lock()
	now := time.Now()
	if now.Sub(s.window) > time.Minute {
		s.attempts = 0
		s.window = now
	}
	s.attempts++
	limited := s.attempts > 10
	s.mu.Unlock()
	if limited {
		w.Header().Set("Retry-After", "60")
		w.Header().Set("Connection", "close")
		http.Error(w, "try again in one minute", 429)
		return
	}
	controller := http.NewResponseController(w)
	_ = controller.SetReadDeadline(time.Now().Add(10 * time.Second))
	r.Body = http.MaxBytesReader(w, r.Body, 1024)
	var input struct {
		Code string `json:"code"`
	}
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	var extra any
	if decoder.Decode(&input) != nil || !errors.Is(decoder.Decode(&extra), io.EOF) || subtle.ConstantTimeCompare([]byte(input.Code), []byte(s.Code)) != 1 {
		// Do not clear a failed read deadline before net/http finishes the
		// response: draining an incomplete body could otherwise stall again.
		w.Header().Set("Connection", "close")
		http.Error(w, "incorrect access code", 401)
		return
	}
	_ = controller.SetReadDeadline(time.Time{})
	s.mu.Lock()
	defer s.mu.Unlock()
	now = time.Now()
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
	name := filepath.Join(s.Root, "backups", random()+".part")
	defer os.Remove(name)
	if e := s.Store.StagedBackup(r.Context(), name, nil); e != nil {
		http.Error(w, "Backup failed or application-storage capacity is insufficient; no archive delivered.", 507)
		return
	}
	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", `attachment; filename="immich-backup.zip"`)
	http.ServeFile(w, r, name)
}

// HTML parsing normalizes CRLF and CR, including Windows checkout line endings.
func scriptDigest(script string) [32]byte {
	return sha256.Sum256([]byte(strings.ReplaceAll(strings.ReplaceAll(script, "\r\n", "\n"), "\r", "\n")))
}

func originalCopyRequest(r *http.Request) bool {
	if r.Method != http.MethodPost {
		return false
	}
	if r.URL.Path == "/api/upload" {
		return true
	}
	parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/api/imports/"), "/")
	return strings.HasPrefix(r.URL.Path, "/api/imports/") && len(parts) == 3 && parts[1] == "files"
}
