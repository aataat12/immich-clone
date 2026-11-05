//go:build windows

package installerfiles

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"immichclone/internal/winfs"
)

// Invoked only by the disposable Windows controller with a test executable.
// No production command/environment option selects the deletion barrier.
func TestMetadataRetirementActorBoundary(t *testing.T) {
	base := os.Getenv("IMMICH_METADATA_FIXTURE")
	if base == "" {
		t.Skip("external separate-account controller only")
	}
	app, source, data := filepath.Join(base, "app"), filepath.Join(base, "source"), filepath.Join(base, "data")
	for _, root := range []string{app, source, data} {
		if e := EnsurePrivateRoot(root, "", false); e != nil {
			t.Fatal("private fixture root refused")
		}
	}
	for _, name := range publicationTestFiles {
		for _, root := range []string{app, source} {
			path := filepath.Join(root, name)
			if e := EnsurePrivateRoot(filepath.Dir(path), "", false); e != nil {
				t.Fatal("private payload parent refused")
			}
			prefix := "prior-"
			if root == source {
				prefix = "candidate-"
			}
			if e := os.WriteFile(path, []byte(prefix+name), 0600); e != nil {
				t.Fatal("payload fixture write failed")
			}
		}
	}
	txn := app + "-upgrade-recovery"
	if e := BeginUpgrade(app, data, txn, false); e != nil {
		t.Fatal("begin fixture refused")
	}
	if e := PublishPayload(app, source, filepath.Join(txn, "payload")); e != nil {
		t.Fatal("real publication refused")
	}
	d, e := winfs.OpenDirectory(txn)
	if e != nil {
		t.Fatal("transaction open refused")
	}
	defer d.Close()
	record, e := readUpgradeRecord(d, upgradeRecordName)
	if e != nil {
		t.Fatal("pending record refused")
	}
	record.Phase = "committed"
	if e = writeUpgradeRecord(d, upgradeTerminalName, record); e != nil {
		t.Fatal("terminal write refused")
	}
	stop := errors.New("test interruption after first held deletion")
	observed := false
	e = retireUpgradeWithHook(d, func(i int) error {
		if i == 0 {
			body, _ := json.Marshal(map[string]string{"transaction": txn, "app": app})
			if e := os.WriteFile(filepath.Join(base, "held.json"), body, 0600); e != nil {
				return e
			}
			deadline := time.Now().Add(15 * time.Second)
			for time.Now().Before(deadline) {
				if _, e := os.Stat(filepath.Join(base, "continue")); e == nil {
					observed = true
					return nil
				}
				time.Sleep(10 * time.Millisecond)
			}
			return errors.New("separate-account controller did not release held barrier")
		}
		if i == 1 {
			return stop
		}
		return nil
	})
	if !observed || !errors.Is(e, stop) {
		t.Fatal("deterministic held retirement interruption was not exercised")
	}
	terminal, e := readUpgradeRecord(d, upgradeTerminalName)
	if e != nil || terminal.Phase != "committed" {
		t.Fatal("terminal was not retained after partial deletion")
	}
	files, e := os.ReadDir(filepath.Join(txn, "payload"))
	if e != nil || len(files) == 0 {
		t.Fatal("partial retirement did not retain remaining recovery files")
	}
	if e = d.Close(); e != nil {
		t.Fatal("held transaction close failed")
	}
	if kind, e := RecoverUpgrade(app, data, txn); e != nil || kind != "none" {
		t.Fatal("committed retirement retry refused")
	}
	publicationAssertSet(t, app, "candidate-")
	if _, e = os.Stat(filepath.Join(txn, upgradeTerminalName)); !errors.Is(e, os.ErrNotExist) {
		t.Fatal("terminal survived complete retirement")
	}
	if names, e := os.ReadDir(filepath.Join(txn, "payload")); e != nil || len(names) != 0 {
		t.Fatal("retirement did not leave only the accepted empty directories")
	}
	t.Log("PASS: separate-account actor reached held retirement barrier; first deletion interrupted; terminal retained; retry retired without code rollback")
}
