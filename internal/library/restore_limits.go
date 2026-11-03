package library

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"runtime/debug"
	"strings"
	"syscall"
	"time"
)

var ErrRestoreBudget = errors.New("restore budget exceeded; increase the explicit recovery budget or provide more destination space")
var ErrRestoreOutputLimit = errors.New("restore worker output exceeds limit")
var ErrRestoreInvalidArchive = errors.New("restore archive failed validation")
var ErrRestoreWorkerFailed = errors.New("restore worker failed or exceeded memory/process limits")
var ErrRestoreCleanup = errors.New("restore staging cleanup requires attention")
var errRestoreWorker = ErrRestoreWorkerFailed
var errRestoreProgressNotReady = errors.New("restore progress not ready")

// Zero fields select finite defaults. Original blobs retain the existing 4GiB
// format limit; MaxCatalogBytes is independently configurable for recovery.
type RestoreLimits struct {
	MaxBytes         uint64
	MaxEntries       uint64
	MaxManifestBytes uint64
	MaxCatalogBytes  uint64
	MaxMetadataBytes uint64
	MemoryBytes      uint64
	Timeout          time.Duration
}

const restoreReserve = uint64(64 << 20)
const restoreOutputBytes = int64(4096)
const restoreWorkerArg = "--immich-restore-worker-v1"
const restoreRequestName = ".restore-request-v1.json"
const restoreProgressName = ".restore-progress-v1.json"
const restorePollInterval = 20 * time.Millisecond

type restoreRequest struct {
	Nonce, Archive, Stage string
	Limits                RestoreLimits
}
type restoreProgress struct {
	Nonce     string
	BaseBytes uint64
}
type restoreProcessResult struct {
	Output []byte
	Peak   uint64
	PID    uint32
	Reaped bool
}

func (l RestoreLimits) defaults(parent string) (RestoreLimits, error) {
	if l.Timeout < 0 {
		return l, errors.New("restore timeout must be positive")
	}
	if l.Timeout == 0 {
		l.Timeout = 30 * time.Minute
	}
	if l.MaxEntries == 0 {
		l.MaxEntries = 100000
	}
	if l.MaxManifestBytes == 0 {
		l.MaxManifestBytes = 16 << 20
	}
	if l.MaxCatalogBytes == 0 {
		l.MaxCatalogBytes = uint64(MaxUpload)
	}
	if l.MaxMetadataBytes == 0 {
		l.MaxMetadataBytes = 64 << 20
	}
	if l.MemoryBytes == 0 {
		l.MemoryBytes = 512 << 20
	}
	available, e := AvailableBytes(parent)
	if e != nil {
		return l, e
	}
	if available <= restoreReserve {
		return l, ErrRestoreBudget
	}
	capacity := available - restoreReserve
	if l.MaxBytes == 0 {
		l.MaxBytes = capacity
	}
	if l.MaxBytes > capacity || l.MaxEntries > uint64(math.MaxInt) || l.MaxManifestBytes >= math.MaxInt64 || l.MaxCatalogBytes >= math.MaxInt64 || l.MemoryBytes >= math.MaxInt64 || l.MemoryBytes < 32<<20 {
		return l, ErrRestoreBudget
	}
	return l, nil
}

// RestoreWithLimits supervises the whole untrusted parse/check/reindex operation
// out of process. The parent never opens the restored bbolt catalog.
func RestoreWithLimits(ctx context.Context, archive, target string, l RestoreLimits) (result error) {
	started := time.Now()
	defer func() { recordRestore(result, time.Since(started)) }()
	if e := ctx.Err(); e != nil {
		return e
	}
	if _, e := os.Lstat(target); !errors.Is(e, os.ErrNotExist) {
		return errors.New("restore destination must not exist")
	}
	var e error
	target, e = filepath.Abs(target)
	if e != nil {
		return e
	}
	archive, e = filepath.Abs(archive)
	if e != nil {
		return e
	}
	l, e = l.defaults(filepath.Dir(target))
	if e != nil {
		return e
	}
	stage, e := os.MkdirTemp(filepath.Dir(target), "immich-restore-*")
	if e != nil {
		return e
	}
	cleanup := true
	defer func() {
		if cleanup {
			if ce := os.RemoveAll(stage); ce != nil {
				result = errors.Join(result, ErrRestoreCleanup)
			}
		}
	}()
	nonceBytes := make([]byte, 32)
	if _, e = rand.Read(nonceBytes); e != nil {
		return e
	}
	req := restoreRequest{Nonce: hex.EncodeToString(nonceBytes), Archive: archive, Stage: stage, Limits: l}
	data, e := json.Marshal(req)
	if e != nil {
		return e
	}
	if len(data) > 1<<20 {
		return ErrRestoreBudget
	}
	requestPath := filepath.Join(stage, restoreRequestName)
	if e = os.WriteFile(requestPath, data, 0600); e != nil {
		return e
	}
	exe, e := os.Executable()
	if e != nil {
		return e
	}
	bounded, cancel := context.WithTimeout(ctx, l.Timeout)
	defer cancel()
	monitorCtx, stopMonitor := context.WithCancel(bounded)
	var monitorErr error
	monitorDone := make(chan struct{})
	go func() {
		defer close(monitorDone)
		monitorErr = monitorRestoreGrowth(monitorCtx, stage, req.Nonce, l, cancel)
	}()
	run, e := runRestoreProcess(bounded, exe, []string{restoreWorkerArg, requestPath, req.Nonce}, l.MemoryBytes, restoreOutputBytes, nil)
	stopMonitor()
	<-monitorDone
	if !run.Reaped && run.PID != 0 {
		cleanup = false
		return errors.New("restore worker termination was not confirmed; private staging retained")
	}
	if monitorErr != nil {
		return monitorErr
	}
	if ce := ctx.Err(); ce != nil {
		return ce
	}
	if e != nil {
		return e
	}
	if e = bounded.Err(); e != nil {
		return e
	}
	var status struct{ Code string }
	if json.Unmarshal(run.Output, &status) != nil {
		return restoreStageError("worker-status-json", nil)
	}
	if status.Code == "budget" {
		return ErrRestoreBudget
	}
	if status.Code == "invalid" {
		return ErrRestoreInvalidArchive
	}
	if status.Code != "ok" {
		return restoreStageError("worker-status-code", nil)
	}
	// Full stage traversal/checking already completed inside the bounded worker.
	// Parent reads only this fixed-size status and one catalog stat.
	if e = checkRestoreCatalogProgress(stage, req.Nonce, l); e != nil {
		return e
	}
	if e = os.Remove(requestPath); e != nil {
		return e
	}
	if e = os.Remove(filepath.Join(stage, restoreProgressName)); e != nil {
		return e
	}
	if e = bounded.Err(); e != nil {
		return e
	}
	if _, e = os.Lstat(target); !errors.Is(e, os.ErrNotExist) {
		return errors.New("restore destination must not exist")
	}
	if e = os.Rename(stage, target); e != nil {
		return e
	}
	cleanup = false
	return nil
}

func init() {
	if len(os.Args) < 2 || os.Args[1] != restoreWorkerArg {
		return
	}
	code := func() (code int) {
		defer func() {
			if recover() != nil {
				code = 2
			}
		}()
		if len(os.Args) != 4 || !validID(os.Args[3]) {
			return 2
		}
		path := os.Args[2]
		if !filepath.IsAbs(path) || filepath.Base(path) != restoreRequestName {
			return 2
		}
		f, e := os.Open(path)
		if e != nil {
			return 2
		}
		b, e := io.ReadAll(io.LimitReader(f, (1<<20)+1))
		f.Close()
		if e != nil || len(b) > 1<<20 {
			return 2
		}
		var r restoreRequest
		if json.Unmarshal(b, &r) != nil || subtle.ConstantTimeCompare([]byte(r.Nonce), []byte(os.Args[3])) != 1 || r.Stage != filepath.Dir(path) || !strings.HasPrefix(filepath.Base(r.Stage), "immich-restore-") {
			return 2
		}
		if r.Limits.MemoryBytes < 32<<20 || r.Limits.MemoryBytes >= math.MaxInt64 {
			return 2
		}
		debug.SetMemoryLimit(int64(r.Limits.MemoryBytes - r.Limits.MemoryBytes/4))
		e = restoreArchiveToStage(r.Archive, r.Stage, r.Limits, r.Nonce)
		status := "ok"
		if e != nil {
			status = "invalid"
			if errors.Is(e, ErrRestoreBudget) {
				status = "budget"
			}
		}
		_ = json.NewEncoder(os.Stdout).Encode(struct{ Code string }{status})
		return 0
	}()
	os.Exit(code)
}

func restoreAddWithin(total, n, limit uint64) (uint64, error) {
	if n > limit || total > limit-n {
		return total, ErrRestoreBudget
	}
	return total + n, nil
}

type restoreBudgetWriter struct {
	Writer    io.Writer
	Remaining uint64
}

func (w *restoreBudgetWriter) Write(p []byte) (int, error) {
	over := uint64(len(p)) > w.Remaining
	if over {
		p = p[:int(w.Remaining)]
	}
	n, e := w.Writer.Write(p)
	if n < 0 || n > len(p) {
		return 0, errors.New("invalid restore write count")
	}
	w.Remaining -= uint64(n)
	if e != nil {
		return n, e
	}
	if n != len(p) {
		return n, io.ErrShortWrite
	}
	if over {
		return n, ErrRestoreBudget
	}
	return n, nil
}

func writeRestoreProgress(stage, nonce string, base uint64) error {
	b, e := json.Marshal(restoreProgress{nonce, base})
	if e != nil {
		return e
	}
	temp := filepath.Join(stage, restoreProgressName+".part")
	if e = os.WriteFile(temp, b, 0600); e != nil {
		return e
	}
	return os.Rename(temp, filepath.Join(stage, restoreProgressName))
}
func checkRestoreStage(stage string, l RestoreLimits) error {
	var total uint64
	return filepath.WalkDir(stage, func(path string, d os.DirEntry, e error) error {
		if e != nil {
			return e
		}
		if d.IsDir() {
			return nil
		}
		if filepath.Dir(path) == stage && (d.Name() == restoreRequestName || d.Name() == restoreProgressName) {
			return nil
		}
		if d.Type()&os.ModeSymlink != 0 {
			return errRestoreWorker
		}
		info, e := d.Info()
		if e != nil {
			return e
		}
		if !info.Mode().IsRegular() || info.Size() < 0 {
			return errRestoreWorker
		}
		if filepath.Dir(path) == stage && d.Name() == "catalog.db" && uint64(info.Size()) > l.MaxCatalogBytes {
			return ErrRestoreBudget
		}
		total, e = restoreAddWithin(total, uint64(info.Size()), l.MaxBytes)
		return e
	})
}

func restoreWorkerFailure(code uint32) error {
	return fmt.Errorf("%w (exit %d)", ErrRestoreWorkerFailed, code)
}

func readRestoreProgress(path string) ([]byte, error) {
	f, e := os.Open(path)
	if e != nil {
		return nil, e
	}
	defer f.Close()
	return io.ReadAll(io.LimitReader(f, 4097))
}

func checkRestoreCatalogProgress(stage, nonce string, l RestoreLimits) error {
	p, e := readRestoreProgress(filepath.Join(stage, restoreProgressName))
	if errors.Is(e, os.ErrNotExist) {
		return errRestoreProgressNotReady
	}
	if e != nil {
		return restoreStageError("progress-read", e)
	}
	if len(p) > 4096 {
		return ErrRestoreBudget
	}
	var progress restoreProgress
	if json.Unmarshal(p, &progress) != nil || progress.Nonce != nonce || progress.BaseBytes > l.MaxBytes {
		return restoreStageError("progress-validation", nil)
	}
	info, e := os.Stat(filepath.Join(stage, "catalog.db"))
	if e != nil {
		return restoreStageError("catalog-stat", e)
	}
	if info.Size() < 0 || uint64(info.Size()) > l.MaxCatalogBytes || uint64(info.Size()) > l.MaxBytes-progress.BaseBytes {
		return ErrRestoreBudget
	}
	return nil
}
func monitorRestoreGrowth(ctx context.Context, stage, nonce string, l RestoreLimits, cancel context.CancelFunc) error {
	ticker := time.NewTicker(restorePollInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			e := checkRestoreCatalogProgress(stage, nonce, l)
			if errors.Is(e, errRestoreProgressNotReady) {
				continue
			}
			if e != nil {
				cancel()
				return e
			}
		}
	}
}

// Fixed stage and numeric OS code only; never include private paths, contents or raw errors.
func restoreStageError(stage string, cause error) error {
	var code syscall.Errno
	_ = errors.As(cause, &code)
	return fmt.Errorf("%w [%s:%d]", ErrRestoreWorkerFailed, stage, uint64(code))
}
