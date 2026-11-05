package web

import (
	"encoding/json"
	"errors"
	"immichclone/internal/library"
	"io"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type deadlineReader struct {
	r          io.Reader
	controller *http.ResponseController
	idle       time.Duration
}

func (d deadlineReader) Read(p []byte) (int, error) {
	idle := d.idle
	if idle <= 0 {
		idle = 60 * time.Second
	}
	_ = d.controller.SetReadDeadline(time.Now().Add(idle))
	return d.r.Read(p)
}
func (s *Server) importHTTP(w http.ResponseWriter, r *http.Request) bool {
	if r.URL.Path == "/api/imports" {
		if r.Method == "GET" {
			items, e := s.Store.ImportHistory()
			if e != nil {
				http.Error(w, "import history unavailable", 500)
			} else {
				jsonResponse(w, items)
			}
			return true
		}
		if r.Method == "POST" {
			var input struct {
				Files []library.ImportFile `json:"files"`
			}
			if boundedJSON(w, r, 16<<20, &input) != nil {
				http.Error(w, "invalid or oversized import selection", 400)
				return true
			}
			m, e := s.Store.CreateImport(input.Files)
			if e != nil {
				http.Error(w, "selection rejected: use 1–50000 nonempty files, each at most 4 GiB, with distinct relative paths", 400)
			} else {
				jsonResponse(w, m)
			}
			return true
		}
	}
	if !strings.HasPrefix(r.URL.Path, "/api/imports/") {
		return false
	}
	parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/api/imports/"), "/")
	if len(parts) == 1 && r.Method == "GET" {
		after := 0
		var e error
		if v := r.URL.Query().Get("after"); v != "" {
			after, e = strconv.Atoi(v)
		}
		if e != nil {
			http.Error(w, "invalid import page", 400)
			return true
		}
		page, e := s.Store.ImportPage(parts[0], after)
		if e != nil {
			http.Error(w, "import unavailable", 404)
		} else {
			jsonResponse(w, page)
		}
		return true
	}
	if len(parts) == 2 && r.Method == "POST" {
		if e := s.Store.ControlImport(parts[0], parts[1]); e != nil {
			http.Error(w, "import control rejected; committed originals retained", 409)
		} else {
			jsonResponse(w, map[string]bool{"saved": true})
		}
		return true
	}
	if len(parts) == 3 && parts[1] == "files" && r.Method == "POST" {
		index, e := strconv.Atoi(parts[2])
		if e != nil {
			http.Error(w, "invalid file index", 400)
			return true
		}
		r.Body = http.MaxBytesReader(w, r.Body, library.MaxUpload)
		controller := http.NewResponseController(w)
		entry, e := s.Store.PutImport(r.Context(), parts[0], index, deadlineReader{r.Body, controller, s.copyIdle})
		if e != nil {
			status := 500
			var max *http.MaxBytesError
			if errors.Is(e, library.ErrTooLarge) || errors.As(e, &max) {
				status = http.StatusRequestEntityTooLarge
			}
			if errors.Is(e, library.ErrImportConflict) {
				status = 409
			}
			if errors.Is(e, library.ErrImportInput) {
				status = 400
			}
			var timeout net.Error
			if errors.As(e, &timeout) && timeout.Timeout() {
				status = http.StatusRequestTimeout
			}
			w.Header().Set("Connection", "close")
			jsonBody := map[string]any{"error": "Import file failed; check receipt and retry. Committed originals retained.", "entry": entry}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(status)
			json.NewEncoder(w).Encode(jsonBody)
		} else {
			_ = controller.SetReadDeadline(time.Time{})
			jsonResponse(w, entry)
		}
		return true
	}
	http.NotFound(w, r)
	return true
}
