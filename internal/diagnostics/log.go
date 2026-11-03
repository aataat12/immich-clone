// Package diagnostics records a fixed-schema, allowlisted support log. It never
// serializes an error's message, request URL, headers, body, path or asset ID.
package diagnostics

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"sync"
	"sync/atomic"
	"syscall"
	"time"
)

const Limit int64 = 4 << 20

var global struct {
	sync.Mutex
	root, version, file string
	failed              bool
}
var sequence atomic.Uint64
var events = map[string]bool{"recognition": true, "startup": true, "shutdown": true, "service": true, "http": true, "upload": true, "preview": true, "backup": true, "restore": true, "verify": true, "logging": true}
var stages = map[string]bool{"ai_provider_verify": true, "ai_process": true, "ai_faces": true, "ai_objects": true, "begin": true, "complete": true, "catalog_open": true, "access_code": true, "listen": true, "ready": true, "stop": true, "failed": true, "original_write": true, "preview_generate": true, "media_provider_verify": true, "media_probe": true, "media_decode": true, "archive_write": true, "archive_restore": true, "integrity": true, "running": true, "requested": true, "assets": true, "login": true, "export": true, "original": true, "view": true, "retry": true, "hide": true, "unhide": true, "diagnostics": true, "health": true, "ui": true, "unknown": true}
var safeVersion = regexp.MustCompile(`^\d+\.\d+\.\d+(?:-[a-z]+)?$`)

func DefaultDir() string {
	if runtime.GOOS == "windows" {
		return filepath.Join(os.Getenv("ProgramData"), "ImmichCloneDiagnostics")
	}
	return filepath.Join(os.TempDir(), "immichclone-diagnostics")
}
func Init(root, version, channel string) error {
	global.Lock()
	defer global.Unlock()
	if channel != "commands" {
		channel = "application"
	}
	if !safeVersion.MatchString(version) {
		version = "prototype"
	}
	if e := os.MkdirAll(root, 0700); e != nil {
		return e
	}
	global.root, global.version, global.file, global.failed = root, version, channel+".jsonl", false
	f, e := os.OpenFile(filepath.Join(root, global.file), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
	if e != nil {
		global.failed = true
		return e
	}
	return f.Close()
}
func Root() string { global.Lock(); defer global.Unlock(); return global.root }
func classify(e error) (string, int64) {
	if e == nil {
		return "none", 0
	}
	var child interface{ NativeExitCode() uint32 }
	if errors.As(e, &child) {
		return "native_child_exit", int64(child.NativeExitCode())
	}
	var native syscall.Errno
	errors.As(e, &native)
	switch {
	case errors.Is(e, context.Canceled):
		return "cancelled", int64(native)
	case errors.Is(e, context.DeadlineExceeded):
		return "media_timeout", int64(native)
	case errors.Is(e, os.ErrPermission):
		return "permission_denied", int64(native)
	case errors.Is(e, os.ErrNotExist):
		return "not_found", int64(native)
	case errors.Is(e, os.ErrExist):
		return "already_exists", int64(native)
	case errors.Is(e, syscall.ENOSPC) || (runtime.GOOS == "windows" && native == 112):
		return "disk_full", int64(native)
	case errors.Is(e, syscall.EADDRINUSE) || (runtime.GOOS == "windows" && native == 10048):
		return "address_in_use", int64(native)
	}
	// Match only exact, fixed messages owned by this program. Never emit raw text.
	switch e.Error() {
	case "local AI provider unavailable; originals remain available":
		return "ai_provider_unavailable", int64(native)
	case "optional media provider unavailable; original remains available":
		return "media_provider_unavailable", int64(native)
	case "child output exceeds limit":
		return "media_output_limit", int64(native)
	case "media preview time limit exceeded; original remains available":
		return "media_timeout", int64(native)
	case "HDR/high-bit-depth preview unsupported; original remains available", "video/image codec unsupported; original remains available":
		return "unsupported_preview", int64(native)
	case "media preview failed; original remains available":
		return "media_failed", int64(native)
	case "timeout":
		return "database_locked", int64(native)
	case "unknown state format", "unsupported schema; refusing migration":
		return "unsupported_schema", int64(native)
	case "refusing an unrelated nonempty data directory":
		return "unrelated_data_directory", int64(native)
	case "preview unsupported in this build; original remains available":
		return "unsupported_preview", int64(native)
	case "original checksum mismatch", "backup checksum mismatch", "backup original integrity failure":
		return "checksum_mismatch", int64(native)
	case "malformed image", "image decode failed":
		return "malformed_media", int64(native)
	}
	return "operation_failed", int64(native)
}
func Record(event, stage string, e error, status int, n int64, duration time.Duration) {
	global.Lock()
	defer global.Unlock()
	if global.root == "" {
		return
	}
	if !events[event] {
		event = "logging"
	}
	if !stages[stage] {
		stage = "unknown"
	}
	category, native := classify(e)
	typ := ""
	if e != nil {
		typ = fmt.Sprintf("%T", e)
	}
	record := struct {
		Time     string `json:"time_utc"`
		Version  string `json:"version"`
		Sequence uint64 `json:"sequence"`
		Event    string `json:"event"`
		Stage    string `json:"stage"`
		Error    string `json:"error_category"`
		Type     string `json:"error_type,omitempty"`
		Native   int64  `json:"native_code,omitempty"`
		Status   int    `json:"http_status,omitempty"`
		Bytes    int64  `json:"bytes,omitempty"`
		Duration int64  `json:"duration_ms,omitempty"`
	}{time.Now().UTC().Format(time.RFC3339Nano), global.version, sequence.Add(1), event, stage, category, typ, native, status, n, duration.Milliseconds()}
	b, err := json.Marshal(record)
	if err != nil {
		global.failed = true
		return
	}
	b = append(b, '\n')
	path := filepath.Join(global.root, global.file)
	if info, err := os.Stat(path); err == nil && info.Size()+int64(len(b)) > Limit {
		if err = os.Remove(path + ".previous"); err != nil && !errors.Is(err, os.ErrNotExist) {
			global.failed = true
			return
		}
		if err = os.Rename(path, path+".previous"); err != nil {
			global.failed = true
			return
		}
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		global.failed = true
		return
	}
	_, err = f.Write(b)
	if err == nil {
		err = f.Sync()
	}
	ce := f.Close()
	if err == nil {
		err = ce
	}
	if err != nil {
		global.failed = true
	}
}

// Export holds the writer lock so each JSON line remains complete during export.
func Export(w io.Writer) error {
	global.Lock()
	defer global.Unlock()
	if _, e := io.WriteString(w, "Immich Clone diagnostic log; no hardware inventory, paths, media names, hashes, credentials or request contents.\n"); e != nil {
		return e
	}
	if global.root == "" {
		_, e := io.WriteString(w, "Logging was not initialized.\n")
		return e
	}
	if global.failed {
		if _, e := io.WriteString(w, "WARNING: a diagnostic write failed; some events may be missing.\n"); e != nil {
			return e
		}
	}
	for _, name := range []string{"installer.jsonl.previous", "installer.jsonl", "application.jsonl.previous", "application.jsonl", "commands.jsonl.previous", "commands.jsonl"} {
		f, e := os.Open(filepath.Join(global.root, name))
		if errors.Is(e, os.ErrNotExist) {
			continue
		}
		if e != nil {
			return e
		}
		if _, e = io.WriteString(w, "\n--- "+name+" ---\n"); e == nil {
			_, e = io.Copy(w, io.LimitReader(f, Limit))
		}
		f.Close()
		if e != nil {
			return e
		}
	}
	return nil
}
