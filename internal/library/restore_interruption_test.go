package library

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// Unlike the earlier stage-appearance test, cancellation requires an actual
// partly extracted original. Only disposable fixture directories are watched.
func TestActualRestoreCancellationDuringExtractionAndRetry(t *testing.T) {
	if os.Getenv("IMMICH_RESTORE_INTERRUPTION") != "1" {
		t.Skip("explicit disposable Windows restore interruption acceptance only")
	}
	parent := os.Getenv("IMMICH_RESTORE_TEMP")
	if parent == "" {
		parent = t.TempDir()
	}
	free, err := AvailableBytes(parent)
	if err != nil || free < 3<<30 {
		t.Fatal("requires 3 GiB free", err)
	}
	root, err := os.MkdirTemp(parent, "immich-restore-interruption-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(root)
	ctx, finish := context.WithTimeout(context.Background(), 3*time.Minute)
	defer finish()
	state := filepath.Join(root, "state")
	s, err := Open(state)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if s != nil {
			s.Close()
		}
	}()
	const size = int64(512 << 20)
	a, dup, err := s.PutWithPath("interruption.bin", "SyntheticRestore/interruption.bin", volumeFixtureReader(t, 9, size))
	if err != nil || dup || a.Size != size {
		t.Fatal("real fixture import", err)
	}
	secret := []byte("synthetic-interrupted-restore-secret\n")
	if err = os.WriteFile(filepath.Join(state, "access-code.txt"), secret, 0600); err != nil {
		t.Fatal(err)
	}
	if err = os.MkdirAll(filepath.Join(state, "backups"), 0700); err != nil {
		t.Fatal(err)
	}
	staged := filepath.Join(state, "backups", "fixture.zip")
	if err = s.StagedBackup(ctx, staged, nil); err != nil {
		t.Fatal(err)
	}
	if err = s.Close(); err != nil {
		t.Fatal(err)
	}
	s = nil
	archive := filepath.Join(root, "standalone.zip")
	if err = os.Rename(staged, archive); err != nil {
		t.Fatal(err)
	}
	if err = os.Rename(state, state+".offline"); err != nil {
		t.Fatal(err)
	}
	if _, err = os.Stat(state); !os.IsNotExist(err) {
		t.Fatal("old state remains", err)
	}
	hashFile := func(path string) string {
		t.Helper()
		f, e := os.Open(path)
		if e != nil {
			t.Fatal(e)
		}
		h := sha256.New()
		_, e = io.Copy(h, f)
		ce := f.Close()
		if e != nil || ce != nil {
			t.Fatal(e, ce)
		}
		return hex.EncodeToString(h.Sum(nil))
	}
	before := hashFile(archive)
	recovery := filepath.Join(root, "recovery")
	if err = os.Mkdir(recovery, 0700); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(recovery, "restored")
	interrupted, cancel := context.WithCancel(ctx)
	defer cancel()
	observed := make(chan int64, 1)
	watcherDone := make(chan struct{})
	go func() {
		defer close(watcherDone)
		ticker := time.NewTicker(time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-interrupted.Done():
				observed <- 0
				return
			case <-ticker.C:
				stages, e := os.ReadDir(recovery)
				if e != nil {
					continue
				}
				for _, stage := range stages {
					if !stage.IsDir() || !strings.HasPrefix(stage.Name(), "immich-restore-") {
						continue
					}
					info, e := os.Stat(filepath.Join(recovery, stage.Name(), "originals", a.ID))
					if e == nil && info.Size() >= 16<<20 && info.Size() < size {
						observed <- info.Size()
						cancel()
						return
					}
				}
			}
		}
	}()
	started := time.Now()
	err = RestoreWithLimits(interrupted, archive, target, RestoreLimits{})
	cancel()
	<-watcherDone
	extracted := <-observed
	if extracted < 16<<20 || extracted >= size || !errors.Is(err, context.Canceled) {
		t.Fatal("mid-extraction cancellation not proved", extracted, err)
	}
	entries, e := os.ReadDir(recovery)
	if e != nil || len(entries) != 0 {
		t.Fatal("cancelled restore retained stage or published target", e)
	}
	if hashFile(archive) != before {
		t.Fatal("cancelled restore changed archive")
	}
	t.Logf("Actual mid-extraction cancellation at %d/%d original bytes; no target or staging; archive unchanged; elapsed %s", extracted, size, time.Since(started))
	started = time.Now()
	if err = RestoreWithLimits(ctx, archive, target, RestoreLimits{}); err != nil {
		t.Fatal("default-budget independent retry", err)
	}
	restored, err := Open(target)
	if err != nil {
		t.Fatal(err)
	}
	defer restored.Close()
	all, err := restored.List()
	if err != nil || len(all) != 1 {
		t.Fatal("restored catalog count", err)
	}
	got, err := restored.Get(a.ID)
	if err != nil || got.Name != a.Name || got.Size != size || len(got.Sources) != 1 || got.Sources[0] != a.Sources[0] {
		t.Fatal("restored catalog metadata", err)
	}
	expected := sha256.New()
	if n, e := io.Copy(expected, volumeFixtureReader(t, 9, size)); e != nil || n != size {
		t.Fatal("independent fixture regeneration", e)
	}
	want := hex.EncodeToString(expected.Sum(nil))
	if want != a.ID || hashFile(restored.Original(a.ID)) != want || hashFile(filepath.Join(state+".offline", "originals", a.ID)) != want {
		t.Fatal("full original hash mismatch")
	}
	info, err := os.Stat(restored.Original(a.ID))
	if err != nil || info.Size() != size {
		t.Fatal("restored size", err)
	}
	b, err := os.ReadFile(filepath.Join(target, "access-code.txt"))
	if err != nil || string(b) != string(secret) {
		t.Fatal("required secret", err)
	}
	if hashFile(archive) != before {
		t.Fatal("retry changed archive")
	}
	entries, err = os.ReadDir(recovery)
	if err != nil || len(entries) != 1 || entries[0].Name() != "restored" {
		t.Fatal("retry retained extra staging", err)
	}
	t.Logf("PASS: independent default-budget retry with source offline; full regenerated/source/restored hashes, exact size, catalog alias and required secret; no extra staging; retry %s", time.Since(started))
}
