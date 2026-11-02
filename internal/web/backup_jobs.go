package web

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"immichclone/internal/diagnostics"
	"immichclone/internal/library"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type BackupJob struct {
	ID           string                 `json:"id,omitempty"`
	Status       string                 `json:"status"`
	Error        string                 `json:"error,omitempty"`
	Progress     library.BackupProgress `json:"progress"`
	SHA256       string                 `json:"sha256,omitempty"`
	ArchiveBytes int64                  `json:"archive_bytes,omitempty"`
	Created      time.Time              `json:"created"`
}
type backupJobs struct {
	mu          sync.Mutex
	store       *library.Store
	dir         string
	job         BackupJob
	cancel      context.CancelFunc
	wg          sync.WaitGroup
	closed      bool
	downloads   int
	lastSave    time.Time
	initErr     error
	progressErr error
}

func backupID(id string) bool {
	b, e := hex.DecodeString(id)
	return e == nil && len(b) == 32 && id == strings.ToLower(id)
}
func newBackupJobs(root string, s *library.Store) *backupJobs {
	m := &backupJobs{store: s, dir: filepath.Join(root, "backups"), job: BackupJob{Status: "idle"}}
	m.initErr = m.open()
	if m.initErr != nil {
		diagnostics.Record("backup", "failed", m.initErr, 0, 0, 0)
		m.job = BackupJob{Status: "unavailable", Error: "Backup manager unavailable; the library remains usable."}
	}
	return m
}
func (m *backupJobs) path(id, suffix string) string { return filepath.Join(m.dir, id+suffix) }
func (m *backupJobs) open() error {
	if e := os.MkdirAll(m.dir, 0700); e != nil {
		return e
	}
	data, e := os.ReadFile(filepath.Join(m.dir, "status.json"))
	if e == nil {
		if len(data) > 64<<10 || json.Unmarshal(data, &m.job) != nil || !backupID(m.job.ID) {
			return errors.New("invalid backup job state")
		}
	} else if !os.IsNotExist(e) {
		return e
	}
	entries, e := os.ReadDir(m.dir)
	if e != nil {
		return e
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if strings.HasSuffix(name, ".part") && (backupID(strings.TrimSuffix(name, ".part")) || strings.HasPrefix(name, "status-")) {
			if e = os.Remove(filepath.Join(m.dir, name)); e != nil {
				return e
			}
		}
	}
	if m.job.Status == "ready" {
		info, e := os.Lstat(m.path(m.job.ID, ".zip"))
		if e != nil || !info.Mode().IsRegular() || info.Size() != m.job.ArchiveBytes || !backupID(m.job.SHA256) {
			m.job.Status = "failed"
			m.job.Error = "Completed backup is unavailable; create a new backup. Originals were retained."
			return m.saveLocked()
		}
		return nil
	}
	if m.job.Status == "queued" || m.job.Status == "snapshot" || m.job.Status == "copying" || m.job.Status == "verifying" || m.job.Status == "hashing" || m.job.Status == "cancelling" {
		m.job.Status = "interrupted"
		m.job.Error = "Backup interrupted by service restart. Start again; originals were retained."
		_ = os.Remove(m.path(m.job.ID, ".zip"))
		return m.saveLocked()
	}
	if m.job.ID != "" && m.job.Status != "cancelled" && m.job.Status != "failed" && m.job.Status != "discarded" && m.job.Status != "interrupted" {
		return errors.New("unknown backup state; retained files unchanged")
	}
	return nil
}
func (m *backupJobs) saveLocked() error {
	b, e := json.Marshal(m.job)
	if e != nil {
		return e
	}
	f, e := os.CreateTemp(m.dir, "status-*.part")
	if e != nil {
		return e
	}
	name := f.Name()
	defer os.Remove(name)
	_, e = f.Write(b)
	if e == nil {
		e = f.Sync()
	}
	ce := f.Close()
	if e == nil {
		e = ce
	}
	if e != nil {
		return e
	}
	return os.Rename(name, filepath.Join(m.dir, "status.json"))
}
func (m *backupJobs) snapshot() BackupJob { m.mu.Lock(); defer m.mu.Unlock(); return m.job }
func (m *backupJobs) progress(p library.BackupProgress) {
	m.mu.Lock()
	defer m.mu.Unlock()
	previous := m.job.Status
	m.job.Progress = p
	if m.job.Status != "cancelling" {
		m.job.Status = p.Phase
	}
	if previous != m.job.Status || time.Since(m.lastSave) >= time.Second {
		if e := m.saveLocked(); e != nil {
			m.progressErr = e
			m.job.Error = "Progress could not be saved; backup will stop safely."
			if m.cancel != nil {
				m.cancel()
			}
		}
		m.lastSave = time.Now()
	}
}
func (m *backupJobs) start() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.closed || m.initErr != nil || m.cancel != nil || m.job.Status == "ready" {
		return errors.New("backup already active or retained")
	}
	if backupID(m.job.ID) {
		if _, e := os.Lstat(m.path(m.job.ID, ".zip")); e == nil {
			return errors.New("retained backup must be discarded first")
		}
	}
	m.progressErr = nil
	previous := m.job
	id := random()
	m.job = BackupJob{ID: id, Status: "queued", Created: time.Now().UTC()}
	if e := m.saveLocked(); e != nil {
		m.job = previous
		return e
	}
	ctx, c := context.WithCancel(context.Background())
	m.cancel = c
	m.wg.Add(1)
	go m.run(ctx, id)
	return nil
}
func (m *backupJobs) run(ctx context.Context, id string) {
	defer m.wg.Done()
	part := m.path(id, ".part")
	final := m.path(id, ".zip")
	e := m.store.StagedBackup(ctx, part, m.progress)
	if e == nil {
		e = library.VerifyBackup(ctx, part, m.progress)
	}
	var sum string
	var size int64
	if e == nil {
		var f *os.File
		f, e = os.Open(part)
		if e == nil {
			info, err := f.Stat()
			if err != nil {
				e = err
			} else {
				size = info.Size()
				h := sha256.New()
				p := library.BackupProgress{Phase: "hashing", TotalBytes: uint64(size)}
				m.progress(p)
				buf := make([]byte, 64<<10)
				for e == nil {
					if e = ctx.Err(); e != nil {
						break
					}
					var n int
					n, e = f.Read(buf)
					if n > 0 {
						h.Write(buf[:n])
						p.Bytes += uint64(n)
						m.progress(p)
					}
					if e == io.EOF {
						e = nil
						break
					}
				}
				sum = hex.EncodeToString(h.Sum(nil))
			}
			f.Close()
		}
	}
	if e == nil {
		e = ctx.Err()
	}
	if e == nil {
		e = os.Link(part, final)
	}
	_ = os.Remove(part)
	m.mu.Lock()
	defer m.mu.Unlock()
	wasCancelled := ctx.Err() != nil || m.job.Status == "cancelling"
	if m.progressErr != nil {
		e = m.progressErr
		wasCancelled = false
	}
	if m.cancel != nil {
		m.cancel()
		m.cancel = nil
	}
	if e == nil && m.job.Status != "cancelling" {
		m.job.Status = "ready"
		m.job.Error = ""
		m.job.SHA256 = sum
		m.job.ArchiveBytes = size
	} else {
		_ = os.Remove(final)
		m.job.Status = "failed"
		m.job.Error = "Backup failed; no download was published. Originals were retained."
		if errors.Is(e, context.Canceled) || wasCancelled {
			m.job.Status = "cancelled"
			m.job.Error = "Backup cancelled. Originals and existing downloaded backups were retained."
			if m.closed {
				m.job.Status = "interrupted"
				m.job.Error = "Backup interrupted by service shutdown. Start again; originals retained."
			}
		}
		if errors.Is(e, library.ErrBackupCapacity) {
			m.job.Error = "Not enough application-storage space for this staged backup. No download was published."
		}
	}
	if err := m.saveLocked(); err != nil {
		_ = os.Remove(final)
		m.job.Status = "failed"
		m.job.Error = "Backup status could not be saved; no download was published."
	}
	diagnostics.Record("backup", "archive_write", e, 0, 0, 0)
}
func (m *backupJobs) cancelJob(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if id != m.job.ID || m.cancel == nil {
		return errors.New("backup is not active")
	}
	m.job.Status = "cancelling"
	e := m.saveLocked()
	m.cancel()
	return e
}
func (m *backupJobs) discard(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if id != m.job.ID || m.cancel != nil || m.downloads != 0 {
		return errors.New("backup cannot be discarded now")
	}
	if !backupID(id) {
		return errors.New("invalid backup")
	}
	if e := os.Remove(m.path(id, ".zip")); e != nil && !os.IsNotExist(e) {
		return e
	}
	m.job.Status = "discarded"
	m.job.SHA256 = ""
	m.job.ArchiveBytes = 0
	m.job.Error = ""
	return m.saveLocked()
}
func (m *backupJobs) download(id string) (*os.File, BackupJob, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if id != m.job.ID || m.job.Status != "ready" {
		return nil, m.job, errors.New("backup not ready")
	}
	f, e := os.Open(m.path(id, ".zip"))
	if e != nil {
		return nil, m.job, e
	}
	m.downloads++
	return f, m.job, nil
}
func (m *backupJobs) endDownload() { m.mu.Lock(); defer m.mu.Unlock(); m.downloads-- }
func (m *backupJobs) close() {
	m.mu.Lock()
	m.closed = true
	c := m.cancel
	m.mu.Unlock()
	if c != nil {
		c()
	}
	m.wg.Wait()
}
