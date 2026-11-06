package web

import (
	"errors"
	"immichclone/internal/library"
	"net/http"
	"os"
	"strconv"
	"strings"
)

func curationError(w http.ResponseWriter, e error) {
	switch {
	case errors.Is(e, library.ErrCurationInput):
		http.Error(w, "Invalid organization metadata", 400)
	case errors.Is(e, library.ErrCurationConflict):
		http.Error(w, "Organization changed. Reload before saving.", 409)
	case errors.Is(e, library.ErrCurationLimit):
		http.Error(w, "Organization storage limit reached", 413)
	case errors.Is(e, library.ErrCurationCapacity):
		http.Error(w, "Not enough free space to safely save organization. Free space and retry.", 507)
	case errors.Is(e, os.ErrNotExist):
		http.Error(w, "Organization target unavailable", 404)
	default:
		http.Error(w, "Organization operation failed; reload to check current state", 500)
	}
}
func (s *Server) curationHTTP(w http.ResponseWriter, r *http.Request) bool {
	if !strings.HasPrefix(r.URL.Path, "/api/metadata/") {
		return false
	}
	if r.URL.Path == "/api/metadata/edit" && r.Method == "POST" {
		var in library.CurationEdit
		if boundedJSON(w, r, 1<<20, &in) != nil {
			curationError(w, library.ErrCurationInput)
			return true
		}
		out, e := s.Store.EditCuration(in)
		if e != nil {
			curationError(w, e)
		} else {
			jsonResponse(w, out)
		}
		return true
	}
	if r.Method != "GET" {
		w.WriteHeader(405)
		return true
	}
	if strings.HasPrefix(r.URL.Path, "/api/metadata/asset/") {
		out, e := s.Store.AssetOrganization(strings.TrimPrefix(r.URL.Path, "/api/metadata/asset/"))
		if e != nil {
			curationError(w, e)
		} else {
			jsonResponse(w, out)
		}
		return true
	}
	limit := 60
	var e error
	if v := r.URL.Query().Get("limit"); v != "" {
		limit, e = strconv.Atoi(v)
		if e != nil {
			curationError(w, library.ErrCurationInput)
			return true
		}
	}
	if r.URL.Path == "/api/metadata/album-summaries" {out,next,e:=s.Store.AlbumSummaries(r.URL.Query().Get("after"),limit);if e!=nil{curationError(w,e)}else{jsonResponse(w,map[string]any{"albums":out,"next":next})};return true}
 if r.URL.Path == "/api/metadata/albums" {
		out, next, e := s.Store.Albums(r.URL.Query().Get("after"), limit)
		if e != nil {
			curationError(w, e)
		} else {
			jsonResponse(w, map[string]any{"albums": out, "next": next})
		}
		return true
	}
	if strings.HasPrefix(r.URL.Path, "/api/metadata/albums/") {
		id := strings.TrimPrefix(r.URL.Path, "/api/metadata/albums/")
		out, e := s.Store.AlbumPage(id, r.URL.Query().Get("after"), limit)
		if e != nil {
			curationError(w, e)
		} else {
			jsonResponse(w, out)
		}
		return true
	}
	http.NotFound(w, r)
	return true
}
