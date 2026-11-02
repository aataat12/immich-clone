package web

import (
	"bytes"
	"context"
	"encoding/json"
	"immichclone/internal/library"
	"io"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func waitBackup(t *testing.T, m *backupJobs, status string) BackupJob {
	t.Helper()
	until := time.Now().Add(10 * time.Second)
	for time.Now().Before(until) {
		j := m.snapshot()
		if j.Status == status {
			return j
		}
		if j.Status == "failed" {
			t.Fatal(j.Error)
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("backup state timeout", m.snapshot().Status)
	return BackupJob{}
}
func TestManagedBackupAuthenticationRetentionRestartAndRestore(t *testing.T) {
	s, c := setup(t)
	t.Cleanup(s.Close)
	source := []byte("synthetic managed backup original")
	a, _, _ := s.Store.Put("synthetic.bin", bytes.NewReader(source))
	for _, path := range []string{"/api/backup-plan", "/api/backup-job", "/api/backup-job/download"} {
		if w := request(s, "GET", path, "", nil); w.Code != 401 {
			t.Fatal("unprotected backup")
		}
	}
	r := httptest.NewRequest("POST", "http://127.0.0.1:2283/api/backup-job/start", nil)
	r.AddCookie(c)
	r.Header.Set("Origin", "http://evil.example")
	r.Header.Set("X-Immich-Request", "1")
	w := httptest.NewRecorder()
	s.ServeHTTP(w, r)
	if w.Code != 403 {
		t.Fatal("cross-origin backup")
	}
	if w = request(s, "POST", "/api/backup-job/start", "", c); w.Code != 200 {
		t.Fatal(w.Body.String())
	}
	j := waitBackup(t, s.backups, "ready")
	if !backupID(j.SHA256) || j.ArchiveBytes <= 0 {
		t.Fatal("missing archive checksum")
	}
	if w = request(s, "POST", "/api/backup-job/start", "", c); w.Code != 409 {
		t.Fatal("overwrote retained backup")
	}
	if w = request(s, "POST", "/api/backup-job/discard", `{"id":"`+j.ID+`"}`, c); w.Code != 400 {
		t.Fatal("unconfirmed discard")
	}
	f, _, e := s.backups.download(j.ID)
	if e != nil {
		t.Fatal(e)
	}
	b, _ := io.ReadAll(f)
	if e = s.backups.discard(j.ID); e == nil {
		t.Fatal("discarded active download")
	}
	f.Close()
	s.backups.endDownload()
	s.backups.close()
	again := newBackupJobs(s.Root, s.Store)
	defer again.close()
	if again.snapshot().Status != "ready" {
		t.Fatal("restart lost ready download")
	}
	archive := filepath.Join(t.TempDir(), "backup.zip")
	os.WriteFile(archive, b, 0600)
	if e = library.VerifyBackup(context.Background(), archive, nil); e != nil {
		t.Fatal(e)
	}
	target := filepath.Join(t.TempDir(), "restored")
	if e = library.Restore(archive, target); e != nil {
		t.Fatal(e)
	}
	restored, e := library.Open(target)
	if e != nil {
		t.Fatal(e)
	}
	defer restored.Close()
	got, _ := os.ReadFile(restored.Original(a.ID))
	if !bytes.Equal(got, source) {
		t.Fatal("managed backup restore")
	}
	if e = again.discard(j.ID); e != nil {
		t.Fatal(e)
	}
	if e = s.Store.Verify(); e != nil {
		t.Fatal("discard affected originals", e)
	}
}
func TestInterruptedBackupCleanupCannotDeleteOriginals(t *testing.T) {
	s, _ := setup(t)
	s.Close()
	id := random()
	dir := filepath.Join(s.Root, "backups")
	job := BackupJob{ID: id, Status: "copying"}
	b, _ := json.Marshal(job)
	os.WriteFile(filepath.Join(dir, "status.json"), b, 0600)
	os.WriteFile(filepath.Join(dir, id+".part"), []byte("partial"), 0600)
	m := newBackupJobs(s.Root, s.Store)
	defer m.close()
	if m.snapshot().Status != "interrupted" {
		t.Fatal("restart did not expose interruption")
	}
	if _, e := os.Stat(filepath.Join(dir, id+".part")); !os.IsNotExist(e) {
		t.Fatal("stale partial remained")
	}
	os.WriteFile(filepath.Join(dir, "status.json"), []byte(`{"id":"../../outside","status":"ready"}`), 0600)
	bad := newBackupJobs(s.Root, s.Store)
	defer bad.close()
	if bad.snapshot().Status != "unavailable" {
		t.Fatal("forged state accepted")
	}
}

func TestUnknownBackupStateRetainsCompletedFile(t *testing.T) {
	s, _ := setup(t)
	defer s.Close()
	id := random()
	path := s.backups.path(id, ".zip")
	os.WriteFile(path, []byte("retained candidate"), 0600)
	b, _ := json.Marshal(BackupJob{ID: id, Status: "unrecognized"})
	os.WriteFile(filepath.Join(s.backups.dir, "status.json"), b, 0600)
	m := newBackupJobs(s.Root, s.Store)
	defer m.close()
	if m.snapshot().Status != "unavailable" {
		t.Fatal("unknown state accepted")
	}
	if b, e := os.ReadFile(path); e != nil || string(b) != "retained candidate" {
		t.Fatal("unknown state deleted a retained file")
	}
}
