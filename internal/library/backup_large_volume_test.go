package library

import (
	"archive/zip"
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// Public deterministic AES-CTR bytes are fixture entropy, not credentials.
// Regeneration is independent of the old filesystem; memory stays bounded.
func volumeFixtureReader(t *testing.T, marker byte, size int64) io.Reader {
	t.Helper()
	key := make([]byte, 32)
	key[0] = marker
	block, err := aes.NewCipher(key)
	if err != nil {
		t.Fatal(err)
	}
	return io.LimitReader(&cipher.StreamReader{S: cipher.NewCTR(block, make([]byte, aes.BlockSize)), R: volumeZeros{}}, size)
}

type volumeZeros struct{}

func (volumeZeros) Read(p []byte) (int, error) { clear(p); return len(p), nil }

func TestActualLargeVolumeBackupRestore(t *testing.T) {
	if os.Getenv("IMMICH_LARGE_BACKUP") != "1" {
		t.Skip("requires explicit disposable 18 GiB-free Windows acceptance opt-in")
	}
	parent := os.Getenv("IMMICH_LARGE_TEMP")
	if parent == "" {
		parent = t.TempDir()
	}
	free, err := AvailableBytes(parent)
	if err != nil || free < 18<<30 {
		t.Fatal("requires 18 GiB free", err)
	}
	root, err := os.MkdirTemp(parent, "immich-volume-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(root)
	ctx, cancel := context.WithTimeout(context.Background(), 24*time.Minute)
	defer cancel()
	started := time.Now()
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
	secret := []byte("synthetic-large-backup-secret-32bytes\n")
	if err = os.WriteFile(filepath.Join(state, "access-code.txt"), secret, 0600); err != nil {
		t.Fatal(err)
	}
	sizes := []int64{4 << 30, 1 << 20}
	assets := make([]Asset, 2)
	for i, size := range sizes {
		a, dup, e := s.PutWithPath("volume-fixture.bin", "SyntheticVolume/part-"+string(rune('A'+i))+".bin", volumeFixtureReader(t, byte(i+1), size))
		if e != nil || dup || a.Size != size {
			t.Fatal("real streamed fixture import", i, e)
		}
		assets[i] = a
	}
	t.Logf("Physically wrote %d original bytes via production PutWithPath in %s", sizes[0]+sizes[1], time.Since(started))
	dir := filepath.Join(state, "backups")
	if err = os.MkdirAll(dir, 0700); err != nil {
		t.Fatal(err)
	}
	partial := filepath.Join(dir, "cancel.part")
	stopctx, stop := context.WithCancel(ctx)
	var copied uint64
	err = s.StagedBackup(stopctx, partial, func(p BackupProgress) {
		if p.Phase == "copying" && p.Bytes >= 64<<20 {
			copied = p.Bytes
			stop()
		}
	})
	stop()
	if !errors.Is(err, context.Canceled) || copied < 64<<20 {
		t.Fatal("cancellation after real copying", copied, err)
	}
	if _, err = os.Stat(partial); !os.IsNotExist(err) {
		t.Fatal("partial archive retained", err)
	}
	staging, err := os.ReadDir(filepath.Join(state, "staging"))
	if err != nil || len(staging) != 0 {
		t.Fatal("snapshot staging retained", err)
	}
	t.Logf("Cancellation after %d copied bytes removed archive and snapshot; retry follows", copied)
	archive := filepath.Join(dir, "volume.zip")
	var progress BackupProgress
	then := time.Now()
	if err = s.StagedBackup(ctx, archive, func(p BackupProgress) { progress = p }); err != nil {
		t.Fatal("retry backup", err)
	}
	info, err := os.Stat(archive)
	if err != nil || info.Size() <= 4<<30 {
		t.Fatal("archive did not actually cross 4 GiB", err)
	}
	if progress.Bytes != progress.TotalBytes || progress.Files != progress.TotalFiles {
		t.Fatal("incomplete copy progress", progress)
	}
	t.Logf("Actual >4 GiB archive: %d bytes, completed payload %d bytes, backup %s", info.Size(), progress.Bytes, time.Since(then))
	z, err := zip.OpenReader(archive)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, entry := range z.File {
		if entry.Name == "originals/"+assets[0].ID {
			found = entry.UncompressedSize64 == 4<<30 && entry.CompressedSize64 > 0xffffffff
		}
	}
	if err = z.Close(); err != nil {
		t.Fatal(err)
	}
	if !found {
		t.Fatal("ZIP64 large original size fields not observed")
	}
	then = time.Now()
	if err = VerifyBackup(ctx, archive, nil); err != nil {
		t.Fatal("full archive readback", err)
	}
	t.Logf("Full production archive checksum readback: %s", time.Since(then))
	if err = s.Close(); err != nil {
		t.Fatal(err)
	}
	s = nil
	standalone := filepath.Join(root, "independent.zip")
	if err = os.Rename(archive, standalone); err != nil {
		t.Fatal(err)
	}
	if err = os.Rename(state, state+".offline"); err != nil {
		t.Fatal(err)
	}
	if _, err = os.Stat(state); !os.IsNotExist(err) {
		t.Fatal("old state still available", err)
	}
	target := filepath.Join(root, "restored")
	then = time.Now()
	if err = RestoreWithLimits(ctx, standalone, target, RestoreLimits{}); err != nil {
		t.Fatal("independent default-budget restore", err)
	}
	t.Logf("Independent restore with source offline: %s", time.Since(then))
	restored, err := Open(target)
	if err != nil {
		t.Fatal(err)
	}
	defer restored.Close()
	listed, err := restored.List()
	if err != nil || len(listed) != 2 {
		t.Fatal("restored catalog count", err)
	}
	for i, a := range assets {
		expected := sha256.New()
		if n, e := io.Copy(expected, volumeFixtureReader(t, byte(i+1), sizes[i])); e != nil || n != sizes[i] {
			t.Fatal("independent regeneration", e)
		}
		want := hex.EncodeToString(expected.Sum(nil))
		if want != a.ID {
			t.Fatal("original hash mismatch", i)
		}
		for _, path := range []string{filepath.Join(state+".offline", "originals", a.ID), restored.Original(a.ID)} {
			f, e := os.Open(path)
			if e != nil {
				t.Fatal(e)
			}
			h := sha256.New()
			n, e := io.Copy(h, f)
			ce := f.Close()
			if e != nil || ce != nil || n != sizes[i] || hex.EncodeToString(h.Sum(nil)) != want {
				t.Fatal("full streamed original verification", i, e, ce)
			}
		}
		got, e := restored.Get(a.ID)
		if e != nil || got.Size != a.Size || got.Name != a.Name || len(got.Sources) != 1 || got.Sources[0] != a.Sources[0] {
			t.Fatal("catalog roundtrip", i, e)
		}
	}
	b, err := os.ReadFile(filepath.Join(target, "access-code.txt"))
	if err != nil || string(b) != string(secret) {
		t.Fatal("required secret roundtrip", err)
	}
	t.Logf("PASS: >4 GiB actual archive, real copy cancellation/cleanup/retry, independent default-budget restore, full regenerated/source/restored hash and size verification; total %s", time.Since(started))
}
