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
	if s.offline != nil {
		return s.backupOffline(ctx, w)
	}
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
func Restore(archive, target string) error {
	return RestoreWithLimits(context.Background(), archive, target, RestoreLimits{})
}
func recordRestore(result error, elapsed time.Duration) {
	diagnostics.Record("restore", "archive_restore", result, 0, 0, elapsed)
}
func restoreArchiveToStage(archive, stage string, limits RestoreLimits, nonce string) (result error) {
	zr, e := zip.OpenReader(archive)
	if e != nil {
		return e
	}
	defer zr.Close()
	return restoreArchiveReaderToStage(&zr.Reader, stage, limits, nonce, nil)
}

// restoreStageIO confines payload creation/progress/final checks to held Windows
// staging authority. Non-Windows development keeps the existing implementation.
type restoreStageIO interface {
	Create(string) (*os.File, error)
	Progress(string, uint64) error
	Check(RestoreLimits) error
}

func restoreArchiveReaderToStage(zr *zip.Reader, stage string, limits RestoreLimits, nonce string, fs restoreStageIO) (result error) {
	var e error
	if uint64(len(zr.File)) > limits.MaxEntries {
		return ErrRestoreBudget
	}
	entries := map[string]*zip.File{}
	var declared uint64
	metadata := uint64(len(zr.Comment))
	if metadata > limits.MaxMetadataBytes {
		return ErrRestoreBudget
	}
	for _, f := range zr.File {
		for _, cost := range []uint64{46, uint64(len(f.Name)), uint64(len(f.Extra)), uint64(len(f.Comment))} {
			metadata, e = restoreAddWithin(metadata, cost, limits.MaxMetadataBytes)
			if e != nil {
				return e
			}
		}
		declared, e = restoreAddWithin(declared, f.UncompressedSize64, limits.MaxBytes)
		if e != nil {
			return e
		}
		if _, ok := entries[f.Name]; ok {
			return errors.New("duplicate backup entry")
		}
		if (f.Name != "manifest.json" && !backupPayloadName(f.Name)) || !f.Mode().IsRegular() {
			return errors.New("unsafe backup entry")
		}
		fileLimit := uint64(MaxUpload)
		if f.Name == "catalog.db" {
			fileLimit = limits.MaxCatalogBytes
		}
		if f.Name == "manifest.json" {
			fileLimit = limits.MaxManifestBytes
		}
		if f.UncompressedSize64 > fileLimit {
			return ErrRestoreBudget
		}
		entries[f.Name] = f
	}
	mf := entries["manifest.json"]
	if mf == nil || mf.UncompressedSize64 > limits.MaxManifestBytes {
		return errors.New("missing or oversized manifest")
	}
	reader, e := mf.Open()
	if e != nil {
		return e
	}
	// Read through EOF so ZIP CRC/size errors cannot hide behind a complete JSON
	// prefix. Unmarshal also rejects any non-whitespace trailing JSON/data.
	data, readErr := io.ReadAll(io.LimitReader(reader, int64(limits.MaxManifestBytes)+1))
	closeErr := reader.Close()
	if readErr != nil || closeErr != nil || uint64(len(data)) > limits.MaxManifestBytes || uint64(len(data)) != mf.UncompressedSize64 {
		return errors.New("invalid backup manifest")
	}
	var m Manifest
	e = json.Unmarshal(data, &m)
	if e != nil || m.Schema != 1 {
		return errors.New("invalid manifest schema")
	}
	if len(m.Files) != len(entries)-1 || m.Files["catalog.db"] == "" || m.Files["state-format.txt"] == "" {
		return errors.New("manifest entry mismatch")
	}
	// Validate the entire payload index before creating extraction state.
	for entry, expected := range m.Files {
		if !backupPayloadName(entry) || !validID(expected) || entries[entry] == nil {
			return errors.New("unsafe or missing backup entry")
		}
	}
	remaining := limits.MaxBytes - uint64(len(data))
	for entry, expected := range m.Files {
		z := entries[entry]
		if z == nil {
			return errors.New("missing backup file")
		}
		path := filepath.Join(stage, filepath.FromSlash(entry))
		var out *os.File
		if fs != nil {
			out, e = fs.Create(entry)
		} else {
			if e = os.MkdirAll(filepath.Dir(path), 0700); e != nil {
				return e
			}
			out, e = os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
		}
		if e != nil {
			return e
		}
		in, e := z.Open()
		if e != nil {
			out.Close()
			return e
		}
		h := sha256.New()
		bounded := restoreBudgetWriter{Writer: io.MultiWriter(out, h), Remaining: remaining}
		n, copyErr := io.Copy(&bounded, io.LimitReader(in, int64(z.UncompressedSize64)+1))
		remaining = bounded.Remaining
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
	base := declared - entries["catalog.db"].UncompressedSize64
	if fs != nil {
		e = fs.Progress(nonce, base)
	} else {
		e = writeRestoreProgress(stage, nonce, base)
	}
	if e != nil {
		return e
	}
	// Read-only structural checking is untrusted work too: it remains inside the
	// same hard-limited process before Open can rebuild/write derived indexes.
	catalog, e := bolt.Open(filepath.Join(stage, "catalog.db"), 0600, &bolt.Options{ReadOnly: true, Timeout: time.Second})
	if e != nil {
		return e
	}
	checkErr := catalog.View(func(tx *bolt.Tx) error {
		invalid := false
		for problem := range tx.Check() {
			if problem != nil {
				invalid = true
			}
		}
		if invalid {
			return errors.New("invalid backup catalog structure")
		}
		return nil
	})
	closeErr = catalog.Close()
	if checkErr != nil {
		return checkErr
	}
	if closeErr != nil {
		return closeErr
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
	closeErr = restored.Close()
	if e != nil {
		return e
	}
	if closeErr != nil {
		return closeErr
	}
	if fs != nil {
		return fs.Check(limits)
	}
	return checkRestoreStage(stage, limits)
}

// Backup payload names are format-defined, never arbitrary extraction paths.
func backupPayloadName(name string) bool {
	return name == "catalog.db" || name == "access-code.txt" || name == "state-format.txt" ||
		(strings.HasPrefix(name, "originals/") && validID(strings.TrimPrefix(name, "originals/")))
}
