package web

import (
	"encoding/json"
	"errors"
	"immichclone/internal/library"
	"io"
	"net/http"
	"strings"
)

func organizationInput(w http.ResponseWriter, r *http.Request, out any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, 64<<10)
	d := json.NewDecoder(r.Body)
	d.DisallowUnknownFields()
	if d.Decode(out) != nil {
		http.Error(w, "invalid review request", 400)
		return false
	}
	var extra any
	if !errors.Is(d.Decode(&extra), io.EOF) {
		http.Error(w, "invalid review request", 400)
		return false
	}
	return true
}
func (s *Server) organizationHTTP(w http.ResponseWriter, r *http.Request) bool {
	if r.URL.Path == "/api/analysis-batch" {
		if r.Method == "POST" {
			var input struct {
				Action string `json:"action"`
			}
			if !organizationInput(w, r, &input) {
				return true
			}
			if e := s.Store.ControlAnalysisBatch(r.Context(), input.Action); e != nil {
				http.Error(w, "batch action unavailable; refresh status and check local AI availability", 409)
				return true
			}
		} else if r.Method != "GET" {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return true
		}
		b, e := s.Store.AnalysisBatch()
		if e != nil {
			http.Error(w, "batch status unavailable", 500)
		} else {
			jsonResponse(w, b)
		}
		return true
	}

	if r.URL.Path == "/api/organization" && r.Method == "GET" {
		result, e := s.Store.OrganizationPage(r.URL.Query().Get("after"), r.URL.Query().Get("face_after"), r.URL.Query().Get("q"))
		if e != nil {
			http.Error(w, "review catalog unavailable", 400)
		} else {
			jsonResponse(w, result)
		}
		return true
	}
	if r.URL.Path == "/api/faces/label" && r.Method == "POST" {
		var input struct {
			IDs  []string `json:"ids"`
			Name string   `json:"name"`
		}
		if !organizationInput(w, r, &input) {
			return true
		}
		if e := s.Store.LabelFaces(input.IDs, input.Name); e != nil {
			http.Error(w, "face assignment failed", 400)
		} else {
			jsonResponse(w, map[string]bool{"saved": true})
		}
		return true
	}
	if strings.HasPrefix(r.URL.Path, "/api/faces/") {
		parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/api/faces/"), "/")
		if len(parts) != 2 {
			http.NotFound(w, r)
			return true
		}
		if parts[1] == "crop" && r.Method == "GET" {
			path, e := s.Store.FaceCrop(parts[0])
			if e != nil {
				http.NotFound(w, r)
			} else {
				w.Header().Set("Content-Type", "image/png")
				http.ServeFile(w, r, path)
			}
			return true
		}
		if parts[1] == "undo" && r.Method == "POST" {
			if e := s.Store.UndoFace(parts[0]); e != nil {
				http.Error(w, "undo unavailable", 400)
			} else {
				jsonResponse(w, map[string]bool{"saved": true})
			}
			return true
		}
		http.NotFound(w, r)
		return true
	}
	return false
}
func (s *Server) assetOrganization(w http.ResponseWriter, r *http.Request, id, action string) bool {
	if r.Method != "POST" {
		return false
	}
	var e error
	switch action {
	case "analyze":
		e = s.Store.QueueAnalysis(id)
	case "tags":
		var input struct {
			Tags []string `json:"tags"`
		}
		if !organizationInput(w, r, &input) {
			return true
		}
		e = s.Store.SetTags(id, input.Tags)
	case "dismiss-tag":
		var input struct {
			Tag     string `json:"tag"`
			Restore bool   `json:"restore"`
		}
		if !organizationInput(w, r, &input) {
			return true
		}
		e = s.Store.DismissTag(id, input.Tag, input.Restore)
	case "manual-face":
		var input struct {
			Box  []float64 `json:"box"`
			Name string    `json:"name"`
		}
		if !organizationInput(w, r, &input) {
			return true
		}
		e = s.Store.ManualFace(r.Context(), id, input.Box, input.Name)
	default:
		return false
	}
	if e != nil {
		status := 400
		if errors.Is(e, library.ErrOrganizationInput) {
			status = 400
		}
		http.Error(w, "review action failed: "+safeOrganizationError(e), status)
	} else {
		jsonResponse(w, map[string]bool{"saved": true})
	}
	return true
}
func safeOrganizationError(e error) string {
	if errors.Is(e, library.ErrOrganizationInput) {
		return "invalid input"
	}
	return "check photo support or try again"
}
