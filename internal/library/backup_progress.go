package library

import (
	"archive/zip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"immichclone/internal/diagnostics"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

var ErrBackupCapacity = errors.New("insufficient space for a staged backup")

type BackupPlan struct {
	Originals         int    `json:"originals"`
	Files             int    `json:"files"`
	CatalogBytes      uint64 `json:"catalog_bytes"`
	OriginalBytes     uint64 `json:"original_bytes"`
	PayloadBytes      uint64 `json:"payload_bytes"`
	ArchiveUpperBound uint64 `json:"archive_upper_bound"`
	RequiredFree      uint64 `json:"required_free"`
	Available         uint64 `json:"available"`
	Fits              bool   `json:"fits"`
}
type BackupProgress struct {
	Phase      string `json:"phase"`
	Bytes      uint64 `json:"bytes"`
	TotalBytes uint64 `json:"total_bytes"`
	Files      int    `json:"files"`
	TotalFiles int    `json:"total_files"`
}
type backupReader struct {
	ctx context.Context
	r   io.Reader
	add func(int)
}

func (r backupReader) Read(p []byte) (int, error) {
	if e := r.ctx.Err(); e != nil {
		return 0, e
	}
	n, e := r.r.Read(p)
	if n > 0 && r.add != nil {
		r.add(n)
	}
	return n, e
}
func (s *Store) backupLock(ctx context.Context) error {
	for {
		if e := ctx.Err(); e != nil {
			return e
		}
		if s.gate.TryLock() {
			return nil
		}
		timer := time.NewTimer(50 * time.Millisecond)
		select {
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		case <-timer.C:
		}
	}
}
func (s *Store) backupPlanLocked() (BackupPlan, error) {
	var p BackupPlan
	if s.deletionPending("") {
		return p, ErrDeletionPending
	}
	assets, e := s.List()
	if e != nil {
		return p, e
	}
	if len(assets) > 99996 {
		return p, errors.New("backup entry count exceeds restore limit")
	}
	p.Originals = len(assets)
	p.Files = len(assets) + 2
	for _, a := range assets {
		if a.Size <= 0 || a.Size > MaxUpload {
			return p, errors.New("original size outside backup limits")
		}
		p.OriginalBytes += uint64(a.Size)
	}
	info, e := os.Stat(filepath.Join(s.root, "catalog.db"))
	if e != nil {
		return p, e
	}
	if info.Size() > MaxUpload {
		return p, errors.New("catalog exceeds restore entry limit")
	}
	dbSize := uint64(info.Size())
	p.CatalogBytes = dbSize
	p.PayloadBytes = p.OriginalBytes + dbSize
	for _, name := range []string{"state-format.txt", "access-code.txt"} {
		info, e := os.Stat(filepath.Join(s.root, name))
		if os.IsNotExist(e) && name == "access-code.txt" {
			continue
		}
		if e != nil {
			return p, e
		}
		if info.Size() > 4096 {
			return p, errors.New("backup configuration exceeds bound")
		}
		p.PayloadBytes += uint64(info.Size())
		if name == "access-code.txt" {
			p.Files++
		}
	}
	// Deflate expansion, per-entry/ZIP64 headers, maximum manifest and a free-space reserve.
	p.ArchiveUpperBound = p.PayloadBytes + p.PayloadBytes/50 + uint64(p.Files)*4096 + (16 << 20)
	p.RequiredFree = p.ArchiveUpperBound + dbSize + (64 << 20)
	return p, nil
}

// BackupPlan is advisory until checked again under the backup write lock.
func (s *Store) BackupPlan(ctx context.Context) (BackupPlan, error) {
	if e := s.backupLock(ctx); e != nil {
		return BackupPlan{}, e
	}
	defer s.gate.Unlock()
	p, e := s.backupPlanLocked()
	if e != nil {
		return p, e
	}
	p.Available, e = AvailableBytes(s.root)
	p.Fits = e == nil && p.Available >= p.RequiredFree
	return p, e
}

// StagedBackup is the web-managed same-volume path. Other Backup writers retain their existing contract.
func (s *Store) StagedBackup(ctx context.Context, path string, report func(BackupProgress)) (retErr error) {
	started := time.Now()
	defer func() { diagnostics.Record("backup", "archive_write", retErr, 0, 0, time.Since(started)) }()
	expected, e := filepath.Abs(filepath.Join(s.root, "backups"))
	if e != nil {
		return e
	}
	actual, e := filepath.Abs(filepath.Dir(path))
	if e != nil || actual != expected {
		return errors.New("backup destination outside managed staging")
	}
	if e := s.backupLock(ctx); e != nil {
		return e
	}
	defer s.gate.Unlock()
	p, e := s.backupPlanLocked()
	if e == nil {
		p.Available, e = AvailableBytes(s.root)
	}
	if e != nil {
		return e
	}
	if p.Available < p.RequiredFree {
		return ErrBackupCapacity
	}
	f, e := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if e != nil {
		return e
	}
	ok := false
	defer func() {
		f.Close()
		if !ok {
			os.Remove(path)
		}
	}()
	e = s.backupLocked(ctx, &backupBudget{w: f, remaining: p.ArchiveUpperBound}, report)
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
	ok = true
	return nil
}

type backupBudget struct {
	w         io.Writer
	remaining uint64
}

func (b *backupBudget) Write(p []byte) (int, error) {
	if uint64(len(p)) > b.remaining {
		return 0, ErrBackupCapacity
	}
	n, e := b.w.Write(p)
	b.remaining -= uint64(n)
	return n, e
}

// VerifyBackup reads every archived byte back, without creating a second restored copy.
// This verifies the archive's manifest/checksums; independent restore is tested separately.
func VerifyBackup(ctx context.Context, path string, report func(BackupProgress)) error {
	z, e := zip.OpenReader(path)
	if e != nil {
		return e
	}
	defer z.Close()
	if len(z.File) > 100000 {
		return errors.New("backup entry limit")
	}
	entries := map[string]*zip.File{}
	for _, f := range z.File {
		if entries[f.Name] != nil {
			return errors.New("duplicate backup entry")
		}
		entries[f.Name] = f
	}
	mf := entries["manifest.json"]
	if mf == nil || mf.UncompressedSize64 > 16<<20 {
		return errors.New("invalid backup manifest")
	}
	in, e := mf.Open()
	if e != nil {
		return e
	}
	var m Manifest
	e = json.NewDecoder(io.LimitReader(in, 16<<20)).Decode(&m)
	in.Close()
	if e != nil || m.Schema != 1 || len(m.Files) != len(entries)-1 || m.Files["catalog.db"] == "" || m.Files["state-format.txt"] == "" {
		return errors.New("backup manifest mismatch")
	}
	progress := BackupProgress{Phase: "verifying", TotalFiles: len(m.Files)}
	for name, hash := range m.Files {
		allowed := name == "catalog.db" || name == "state-format.txt" || name == "access-code.txt" || (strings.HasPrefix(name, "originals/") && validID(strings.TrimPrefix(name, "originals/")))
		f := entries[name]
		if !allowed || !validID(hash) || f == nil || f.UncompressedSize64 > uint64(MaxUpload) {
			return errors.New("invalid backup entry")
		}
		progress.TotalBytes += f.UncompressedSize64
	}
	if report != nil {
		report(progress)
	}
	for name, want := range m.Files {
		if e = ctx.Err(); e != nil {
			return e
		}
		in, e := entries[name].Open()
		if e != nil {
			return e
		}
		h := sha256.New()
		n, e := io.Copy(h, backupReader{ctx, in, func(n int) {
			progress.Bytes += uint64(n)
			if report != nil {
				report(progress)
			}
		}})
		in.Close()
		if e != nil {
			return e
		}
		if uint64(n) != entries[name].UncompressedSize64 || hex.EncodeToString(h.Sum(nil)) != want {
			return errors.New("backup read-back checksum mismatch")
		}
		progress.Files++
		if report != nil {
			report(progress)
		}
	}
	return ctx.Err()
}

type backupContextWriter struct {
	ctx context.Context
	w   io.Writer
}

func (w backupContextWriter) Write(p []byte) (int, error) {
	if e := w.ctx.Err(); e != nil {
		return 0, e
	}
	return w.w.Write(p)
}
