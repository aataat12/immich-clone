package web

import (
	"context"
	"net/http"
	"time"
)

func (s *Server) Close() { s.backups.close() }
func (s *Server) backupHTTP(w http.ResponseWriter, r *http.Request) bool {
	switch r.URL.Path {
	case "/api/backup-plan":
		if r.Method != "GET" {
			break
		}
		ctx, c := context.WithTimeout(r.Context(), 2*time.Second)
		defer c()
		p, e := s.Store.BackupPlan(ctx)
		if e != nil {
			http.Error(w, "Backup planning unavailable while the library is busy, or a backup limit was reached.", 409)
		} else {
			jsonResponse(w, p)
		}
		return true
	case "/api/backup-job":
		if r.Method == "GET" {
			jsonResponse(w, s.backups.snapshot())
			return true
		}
	case "/api/backup-job/start":
		if r.Method == "POST" {
			if e := s.backups.start(); e != nil {
				http.Error(w, "Finish, download and discard the retained temporary backup before starting another.", 409)
			} else {
				jsonResponse(w, s.backups.snapshot())
			}
			return true
		}
	case "/api/backup-job/cancel", "/api/backup-job/discard":
		if r.Method != "POST" {
			break
		}
		var input struct {
			ID      string `json:"id"`
			Confirm bool   `json:"confirm"`
		}
		if !organizationInput(w, r, &input) {
			return true
		}
		var e error
		if r.URL.Path == "/api/backup-job/cancel" {
			e = s.backups.cancelJob(input.ID)
		} else {
			if !input.Confirm {
				http.Error(w, "Confirm deletion of the temporary backup copy.", 400)
				return true
			}
			e = s.backups.discard(input.ID)
		}
		if e != nil {
			http.Error(w, "Backup control rejected; refresh status before retrying.", 409)
		} else {
			jsonResponse(w, s.backups.snapshot())
		}
		return true
	case "/api/backup-job/download":
		if r.Method != "GET" {
			break
		}
		f, job, e := s.backups.download(r.URL.Query().Get("id"))
		if e != nil {
			http.Error(w, "Backup download is not ready.", 409)
			return true
		}
		defer s.backups.endDownload()
		defer f.Close()
		w.Header().Set("Content-Type", "application/zip")
		w.Header().Set("Content-Disposition", `attachment; filename="immich-backup.zip"`)
		w.Header().Set("X-Backup-SHA256", job.SHA256)
		http.ServeContent(w, r, "immich-backup.zip", job.Created, f)
		return true
	}
	return false
}
