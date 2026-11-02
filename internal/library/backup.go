package library

import (
	"archive/zip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	bolt "go.etcd.io/bbolt"
	"immichclone/internal/diagnostics"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type Manifest struct {
	Schema int               `json:"schema"`
	Files  map[string]string `json:"sha256"`
}

// Backup blocks writes while copying a consistent database snapshot and immutable originals.
// Callers must discard output if an error occurs; HTTP callers stage before sending.
func (s *Store) Backup(w io.Writer) error { return s.BackupContext(context.Background(), w, nil) }
func (s *Store) BackupContext(ctx context.Context, w io.Writer, report func(BackupProgress)) (result error) {
	started := time.Now()
	defer func() { diagnostics.Record("backup", "archive_write", result, 0, 0, time.Since(started)) }()
	if e := s.backupLock(ctx); e != nil {
		return e
	}
	defer s.gate.Unlock()
	return s.backupLocked(ctx, w, report)
}
func (s *Store) backupLocked(ctx context.Context, w io.Writer, report func(BackupProgress)) (result error) {
	plan, e := s.backupPlanLocked()
	if e != nil {
		return e
	}
	progress := BackupProgress{Phase: "snapshot", TotalBytes: plan.PayloadBytes, TotalFiles: plan.Files}
	if report != nil {
		report(progress)
	}
	tmp, e := os.CreateTemp(filepath.Join(s.root, "staging"), "snapshot-*.db")
	if e != nil {
		return e
	}
	name := tmp.Name()
	defer tmp.Close()
	defer os.Remove(name)
	e = s.db.View(func(tx *bolt.Tx) error { _, e := tx.WriteTo(backupContextWriter{ctx: ctx, w: tmp}); return e })
	if e == nil {
		e = tmp.Sync()
	}
	ce := tmp.Close()
	if e == nil {
		e = ce
	}
	if e != nil {
		return e
	}
	info, e := os.Stat(name)
	if e != nil {
		return e
	}
	progress.Phase = "copying"
	progress.TotalBytes = plan.PayloadBytes - plan.CatalogBytes + uint64(info.Size())
	if report != nil {
		report(progress)
	}
	assets, e := s.List()
	if e != nil {
		return e
	}
	manifest := Manifest{Schema: 1, Files: map[string]string{}}
	zw := zip.NewWriter(w)
	add := func(entry, path, expected string) error {
		f, e := os.Open(path)
		if e != nil {
			return e
		}
		defer f.Close()
		h := sha256.New()
		out, e := zw.Create(entry)
		if e != nil {
			return e
		}
		if _, e = io.Copy(io.MultiWriter(out, h), backupReader{ctx: ctx, r: f, add: func(n int) {
			progress.Bytes += uint64(n)
			if report != nil {
				report(progress)
			}
		}}); e != nil {
			return e
		}
		hash := hex.EncodeToString(h.Sum(nil))
		if expected != "" && hash != expected {
			return errors.New("backup original integrity failure")
		}
		manifest.Files[entry] = hash
		progress.Files++
		if report != nil {
			report(progress)
		}
		return nil
	}
	if e = add("catalog.db", name, ""); e != nil {
		return e
	}
	if e = add("state-format.txt", filepath.Join(s.root, "state-format.txt"), ""); e != nil {
		return e
	}
	if _, e = os.Stat(filepath.Join(s.root, "access-code.txt")); e == nil {
		if e = add("access-code.txt", filepath.Join(s.root, "access-code.txt"), ""); e != nil {
			return e
		}
	} else if !errors.Is(e, os.ErrNotExist) {
		return e
	}
	for _, a := range assets {
		if e = add("originals/"+a.ID, s.Original(a.ID), a.ID); e != nil {
			return e
		}
	}
	out, e := zw.Create("manifest.json")
	if e != nil {
		return e
	}
	if e = json.NewEncoder(out).Encode(manifest); e != nil {
		return e
	}
	if e = ctx.Err(); e != nil {
		return e
	}
	return zw.Close()
}

// Restore never overwrites an existing directory. Publication happens only after
// all checksums and catalog/original relationships have been validated.
func Restore(archive, target string) (result error) {
	started := time.Now()
	defer func() { diagnostics.Record("restore", "archive_restore", result, 0, 0, time.Since(started)) }()
	if _, e := os.Lstat(target); !errors.Is(e, os.ErrNotExist) {
		return errors.New("restore destination must not exist")
	}
	zr, e := zip.OpenReader(archive)
	if e != nil {
		return e
	}
	defer zr.Close()
	if len(zr.File) > 100000 {
		return errors.New("backup exceeds 100000 entry limit")
	}
	entries := map[string]*zip.File{}
	for _, f := range zr.File {
		if _, ok := entries[f.Name]; ok {
			return errors.New("duplicate backup entry")
		}
		entries[f.Name] = f
	}
	mf := entries["manifest.json"]
	if mf == nil || mf.UncompressedSize64 > 16<<20 {
		return errors.New("missing or oversized manifest")
	}
	reader, e := mf.Open()
	if e != nil {
		return e
	}
	var m Manifest
	e = json.NewDecoder(io.LimitReader(reader, 16<<20)).Decode(&m)
	reader.Close()
	if e != nil || m.Schema != 1 {
		return errors.New("invalid manifest schema")
	}
	if len(m.Files) != len(entries)-1 || m.Files["catalog.db"] == "" || m.Files["state-format.txt"] == "" {
		return errors.New("manifest entry mismatch")
	}
	stage, e := os.MkdirTemp(filepath.Dir(target), "immich-restore-*")
	if e != nil {
		return e
	}
	defer func() {
		if result != nil {
			os.RemoveAll(stage)
		}
	}()
	for entry, expected := range m.Files {
		allowed := entry == "catalog.db" || entry == "access-code.txt" || entry == "state-format.txt" || (strings.HasPrefix(entry, "originals/") && validID(strings.TrimPrefix(entry, "originals/")))
		if !allowed || !validID(expected) {
			return errors.New("unsafe backup entry")
		}
		z := entries[entry]
		if z == nil || z.UncompressedSize64 > uint64(MaxUpload) {
			return errors.New("missing or oversized backup file")
		}
		path := filepath.Join(stage, filepath.FromSlash(entry))
		if e = os.MkdirAll(filepath.Dir(path), 0700); e != nil {
			return e
		}
		out, e := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
		if e != nil {
			return e
		}
		in, e := z.Open()
		if e != nil {
			out.Close()
			return e
		}
		h := sha256.New()
		n, copyErr := io.Copy(io.MultiWriter(out, h), io.LimitReader(in, int64(z.UncompressedSize64)+1))
		in.Close()
		if copyErr == nil {
			copyErr = out.Sync()
		}
		closeErr := out.Close()
		if copyErr != nil {
			return copyErr
		}
		if closeErr != nil {
			return closeErr
		}
		if n != int64(z.UncompressedSize64) || hex.EncodeToString(h.Sum(nil)) != expected {
			return errors.New("backup checksum mismatch")
		}
	}
	restored, e := Open(stage)
	if e != nil {
		return e
	}
	assets, e := restored.List()
	if e == nil {
		e = restored.Verify()
	}
	if e == nil {
		originals := 0
		for entry := range m.Files {
			if strings.HasPrefix(entry, "originals/") {
				originals++
			}
		}
		if originals != len(assets) {
			e = errors.New("backup catalog/original mismatch")
		}
	}
	if e == nil {
		for _, a := range assets {
			if e = restored.Retry(a.ID); e != nil {
				break
			}
		}
	}
	closeErr := restored.Close()
	if e != nil {
		return e
	}
	if closeErr != nil {
		return closeErr
	}
	return os.Rename(stage, target)
}
