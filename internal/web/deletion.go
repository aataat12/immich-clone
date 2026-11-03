package web

import (
	"context"
	"errors"
	"immichclone/internal/library"
	"net/http"
	"os"
	"strings"
	"time"
)

func (s *Server) deletionHTTP(w http.ResponseWriter, r *http.Request) bool {
	if r.URL.Path == "/api/deletions" && r.Method == "GET" {
		jobs, e := s.Store.PendingDeletions()
		if e != nil {
			http.Error(w, "Deletion status unavailable", 500)
		} else {
			jsonResponse(w, map[string]any{"pending": jobs})
		}
		return true
	}
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	initial := len(parts) == 4 && parts[0] == "api" && parts[1] == "assets" && parts[3] == "delete"
	control := len(parts) == 4 && parts[0] == "api" && parts[1] == "deletions" && (parts[3] == "retry" || parts[3] == "keep")
	if !initial && !control {
		return false
	}
	if r.Method != "POST" {
		http.NotFound(w, r)
		return true
	}
	var input struct {
		Confirm string    `json:"confirm"`
		Added   time.Time `json:"added"`
	}
	if !organizationInput(w, r, &input) {
		return true
	}
	if input.Confirm != parts[2] {
		http.Error(w, "Deletion confirmation does not match the item", 400)
		return true
	}
	ctx, cancel := context.WithTimeout(r.Context(), 75*time.Second)
	defer cancel()
	var e error
	pending := false
	if initial {
		pending, e = s.Store.DeleteHidden(ctx, parts[2], input.Confirm, input.Added)
	} else if parts[3] == "retry" {
		e = s.Store.RetryDeletion(ctx, parts[2])
		pending = e != nil && !errors.Is(e, library.ErrDeletionBusy) && !errors.Is(e, os.ErrNotExist) && !errors.Is(e, library.ErrOrganizationInput)
	} else {
		e = s.Store.KeepDeletionConflict(parts[2])
	}
	if pending {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusAccepted)
		jsonResponse(w, map[string]bool{"pending": true})
		return true
	}
	if e != nil {
		status := 500
		message := "Deletion failed; refresh Hidden Media to inspect recovery status"
		switch {
		case errors.Is(e, library.ErrDeletionOriginal):
			status = 409
			message = e.Error()
		case errors.Is(e, library.ErrDeletionBusy):
			status = 409
			message = e.Error()
		case errors.Is(e, library.ErrDeletionConflict):
			status = 409
			message = e.Error()
		case errors.Is(e, os.ErrNotExist):
			status = 404
			message = "Item or deletion record not found"
		case errors.Is(e, library.ErrOrganizationInput):
			status = 400
			message = "Only a hidden item with matching confirmation can be deleted"
		}
		http.Error(w, message, status)
		return true
	}
	jsonResponse(w, map[string]bool{"pending": false})
	return true
}
