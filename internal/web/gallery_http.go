package web

import (
	"errors"
	"immichclone/internal/library"
	"net/http"
	"net/url"
	"strconv"
)

func galleryRequest(r *http.Request) (library.GalleryQuery, error) {
	p, e := url.ParseQuery(r.URL.RawQuery)
	if e != nil {
		return library.GalleryQuery{}, library.ErrGalleryInput
	}
	singles := map[string]bool{"source_folder": true, "source_mode": true, "source_missing": true, "import_batch": true, "favorite": true, "album": true, "sort": true, "visibility": true, "q": true, "filename": true, "media": true, "date_basis": true, "from": true, "until": true, "no_capture": true, "after": true, "limit": true, "asset": true, "snapshot": true}
	if r.URL.Path == "/api/gallery/sources" {
		singles["source_after"] = true
	}
	if r.URL.Path == "/api/gallery/calendar" {
		for _, name := range []string{"calendar_basis", "calendar_zone", "year", "month"} {
			singles[name] = true
		}
	}
	lists := map[string]bool{"person_any": true, "person_all": true, "person_exclude": true, "tag_any": true, "tag_all": true, "tag_exclude": true}
	// Reject unsupported or ambiguous fields instead of silently broadening results.
	for key, values := range p {
		if (!singles[key] && !lists[key]) || (singles[key] && len(values) != 1) || len(values) > 32 {
			e = library.ErrGalleryInput
		}
	}
	noCapture := p.Get("no_capture")
	if noCapture != "" && noCapture != "true" && noCapture != "false" {
		e = library.ErrGalleryInput
	}
	sourceMissing := p.Get("source_missing")
	if sourceMissing != "" && sourceMissing != "true" && sourceMissing != "false" {
		e = library.ErrGalleryInput
	}
	q := library.GalleryQuery{SourceFolder: p.Get("source_folder"), SourceMode: p.Get("source_mode"), SourceMissing: sourceMissing == "true", ImportBatch: p.Get("import_batch"), Favorite: p.Get("favorite"), Album: p.Get("album"), Sort: p.Get("sort"), Visibility: p.Get("visibility"), Text: p.Get("q"), Filename: p.Get("filename"), Media: p.Get("media"), DateBasis: p.Get("date_basis"), From: p.Get("from"), Until: p.Get("until"), NoCapture: noCapture == "true", People: library.GalleryLabels{Any: p["person_any"], All: p["person_all"], Exclude: p["person_exclude"]}, Tags: library.GalleryLabels{Any: p["tag_any"], All: p["tag_all"], Exclude: p["tag_exclude"]}}
	if e != nil {
		return q, e
	}
	return q.Normalize()
}

func galleryError(w http.ResponseWriter, e error) {
	switch {
	case errors.Is(e, library.ErrGalleryChanged):
		http.Error(w, "Gallery results changed. Refresh from the first page.", 409)
	case errors.Is(e, library.ErrGalleryInput):
		http.Error(w, "invalid gallery search or page", 400)
	default:
		http.Error(w, "gallery search unavailable; try Refresh", 500)
	}
}

func (s *Server) galleryHTTP(w http.ResponseWriter, r *http.Request) bool {
	if r.URL.Path != "/api/gallery" && r.URL.Path != "/api/gallery/neighbors" && r.URL.Path != "/api/gallery/anchor" && r.URL.Path != "/api/gallery/sources" && r.URL.Path != "/api/gallery/calendar" {
		return false
	}
	if r.Method != "GET" {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return true
	}
	q, e := galleryRequest(r)
	if e != nil {
		galleryError(w, e)
		return true
	}
	p := r.URL.Query()
	if r.URL.Path == "/api/gallery/sources" || r.URL.Path == "/api/gallery/calendar" {
		if p.Has("after") || p.Has("asset") || p.Has("snapshot") {
			galleryError(w, library.ErrGalleryInput)
			return true
		}
		if r.URL.Path == "/api/gallery/sources" {
			limit := 100
			if p.Has("limit") {
				limit, e = strconv.Atoi(p.Get("limit"))
			}
			if e != nil {
				galleryError(w, library.ErrGalleryInput)
				return true
			}
			result, e := s.Store.GallerySources(q, p.Get("source_after"), limit)
			if e != nil {
				galleryError(w, e)
			} else {
				jsonResponse(w, result)
			}
			return true
		}
		if p.Has("limit") {
			galleryError(w, library.ErrGalleryInput)
			return true
		}
		year, month := 0, 0
		if p.Has("year") {
			year, e = strconv.Atoi(p.Get("year"))
			if e != nil || year < 1 {
				galleryError(w, library.ErrGalleryInput)
				return true
			}
		}
		if p.Has("month") {
			month, e = strconv.Atoi(p.Get("month"))
			if e != nil || month < 1 {
				galleryError(w, library.ErrGalleryInput)
				return true
			}
		}
		result, e := s.Store.GalleryCalendar(q, p.Get("calendar_basis"), p.Get("calendar_zone"), year, month)
		if e != nil {
			galleryError(w, e)
		} else {
			jsonResponse(w, result)
		}
		return true
	}
	if r.URL.Path == "/api/gallery/anchor" {
		limit := 60
		if p.Has("after") {
			galleryError(w, library.ErrGalleryInput)
			return true
		}
		if p.Has("limit") {
			limit, e = strconv.Atoi(p.Get("limit"))
			if e != nil {
				galleryError(w, library.ErrGalleryInput)
				return true
			}
		}
		anchor, e := s.Store.GalleryAnchorPage(q, p.Get("asset"), p.Get("snapshot"), limit)
		if e != nil {
			galleryError(w, e)
		} else {
			jsonResponse(w, anchor)
		}
		return true
	}
	if r.URL.Path == "/api/gallery/neighbors" {
		if p.Has("after") || p.Has("limit") {
			galleryError(w, library.ErrGalleryInput)
			return true
		}
		neighbors, e := s.Store.GalleryNeighbors(q, p.Get("asset"), p.Get("snapshot"))
		if e != nil {
			galleryError(w, e)
		} else {
			jsonResponse(w, neighbors)
		}
		return true
	}
	if p.Has("asset") || p.Has("snapshot") {
		galleryError(w, library.ErrGalleryInput)
		return true
	}
	limit := 60
	if p.Has("limit") {
		limit, e = strconv.Atoi(p.Get("limit"))
	}
	if e != nil {
		galleryError(w, library.ErrGalleryInput)
		return true
	}
	page, e := s.Store.QueryGalleryPage(q, p.Get("after"), limit)
	if e != nil {
		galleryError(w, e)
	} else {
		jsonResponse(w, page)
	}
	return true
}
