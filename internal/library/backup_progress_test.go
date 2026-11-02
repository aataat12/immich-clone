package library

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestStagedBackupCapacityProgressReadBackAndIndependentRestore(t *testing.T) {
	s, root := openTest(t)
	source := fixture(t)
	a, _, _ := s.Put("photo.png", bytes.NewReader(source))
	dir := filepath.Join(root, "backups")
	os.MkdirAll(dir, 0700)
	plan, e := s.BackupPlan(context.Background())
	if e != nil || !plan.Fits || plan.Originals != 1 || plan.RequiredFree < plan.ArchiveUpperBound || plan.OriginalBytes != uint64(len(source)) {
		t.Fatal("plan incorrect", e)
	}
	path := filepath.Join(dir, "candidate.part")
	var last BackupProgress
	seen := false
	e = s.StagedBackup(context.Background(), path, func(p BackupProgress) {
		if p.Phase == "copying" {
			seen = true
			last = p
		}
	})
	if e != nil || !seen || last.Bytes != last.TotalBytes || last.Files != last.TotalFiles {
		t.Fatal("progress not complete", last, e)
	}
	if e = VerifyBackup(context.Background(), path, func(p BackupProgress) { last = p }); e != nil || last.Phase != "verifying" || last.Bytes != last.TotalBytes {
		t.Fatal("read-back failed", last, e)
	}
	target := filepath.Join(t.TempDir(), "independent")
	if e = Restore(path, target); e != nil {
		t.Fatal(e)
	}
	s.Close()
	os.Rename(root, root+".offline")
	r, e := Open(target)
	if e != nil {
		t.Fatal(e)
	}
	defer r.Close()
	actual, e := os.ReadFile(r.Original(a.ID))
	if e != nil || !bytes.Equal(actual, source) {
		t.Fatal("restore bytes")
	}
	if e = r.Verify(); e != nil {
		t.Fatal(e)
	}
}
func TestBackupCancellationBudgetAndTruncatedReadback(t *testing.T) {
	s, root := openTest(t)
	s.Put("original.bin", bytes.NewReader(bytes.Repeat([]byte("synthetic"), 1<<15)))
	dir := filepath.Join(root, "backups")
	os.MkdirAll(dir, 0700)
	path := filepath.Join(dir, "cancel.part")
	ctx, cancel := context.WithCancel(context.Background())
	e := s.StagedBackup(ctx, path, func(p BackupProgress) {
		if p.Phase == "copying" && p.Bytes > 0 {
			cancel()
		}
	})
	if !errors.Is(e, context.Canceled) {
		t.Fatal("cancel ignored", e)
	}
	if _, e = os.Stat(path); !os.IsNotExist(e) {
		t.Fatal("cancelled staging retained")
	}
	if e = s.Verify(); e != nil {
		t.Fatal(e)
	}
	s.gate.Lock()
	ctx, cancel = context.WithTimeout(context.Background(), 20*time.Millisecond)
	e = s.BackupContext(ctx, &bytes.Buffer{}, nil)
	cancel()
	s.gate.Unlock()
	if !errors.Is(e, context.DeadlineExceeded) {
		t.Fatal("queued cancel blocked", e)
	}
	var b bytes.Buffer
	budget := backupBudget{w: &b, remaining: 3}
	if _, e = budget.Write([]byte("1234")); !errors.Is(e, ErrBackupCapacity) || b.Len() != 0 {
		t.Fatal("budget accepted excess")
	}
	if e = s.StagedBackup(context.Background(), path, nil); e != nil {
		t.Fatal(e)
	}
	info, _ := os.Stat(path)
	os.Truncate(path, info.Size()-10)
	if e = VerifyBackup(context.Background(), path, nil); e == nil {
		t.Fatal("truncated archive verified")
	}
	if e = s.Verify(); e != nil {
		t.Fatal(e)
	}
}
