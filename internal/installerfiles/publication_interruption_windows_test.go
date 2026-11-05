//go:build windows

package installerfiles

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"testing"
)

// This subprocess bypasses all Go deferred cleanup, as a terminated helper does.
// The hook is test-only; product invocations cannot select an interruption.
func TestPayloadProcessExitFixture(t *testing.T) {
	if os.Getenv("IMMICH_PAYLOAD_EXIT_FIXTURE") != "1" {
		return
	}
	index, e := strconv.Atoi(os.Getenv("IMMICH_PAYLOAD_EXIT_INDEX"))
	if e != nil {
		os.Exit(91)
	}
	e = publishPayload(os.Getenv("IMMICH_PAYLOAD_APP"), os.Getenv("IMMICH_PAYLOAD_SOURCE"), os.Getenv("IMMICH_PAYLOAD_BACKUP"), func(i int) error {
		if i == index {
			os.Exit(73)
		}
		return nil
	})
	if e != nil {
		os.Exit(92)
	}
	os.Exit(93)
}

func exitPublication(t *testing.T, app, source, backup string, index int) {
	t.Helper()
	exe, e := os.Executable()
	if e != nil {
		t.Fatal(e)
	}
	c := exec.Command(exe, "-test.run=^TestPayloadProcessExitFixture$")
	c.Env = append(os.Environ(), "IMMICH_PAYLOAD_EXIT_FIXTURE=1", "IMMICH_PAYLOAD_EXIT_INDEX="+strconv.Itoa(index), "IMMICH_PAYLOAD_APP="+app, "IMMICH_PAYLOAD_SOURCE="+source, "IMMICH_PAYLOAD_BACKUP="+backup)
	output, e := c.CombinedOutput()
	var exit *exec.ExitError
	if !errors.As(e, &exit) || exit.ExitCode() != 73 {
		t.Fatalf("real process-exit fixture failed: %v %s", e, output)
	}
}

func TestPayloadDurableIntentRecoversExitedMixedCopy(t *testing.T) {
	for _, index := range []int{0, 1, 5, 8} {
		t.Run(strconv.Itoa(index), func(t *testing.T) {
			app, source, backup, _ := publicationFixture(t)
			newName := `ai\nested é\new-runtime.dll`
			if e := os.WriteFile(filepath.Join(source, newName), []byte("new provider"), 0600); e != nil {
				t.Fatal(e)
			}
			exitPublication(t, app, source, backup, index)
			// A durable intent is not a successful-publication receipt.
			if e := RestorePayload(app, backup); e == nil {
				t.Fatal("mixed copy accepted as completed publication")
			}
			// Model truncation/partial write in the exact held object. WriteFile
			// preserves its identity; an identity replacement is tested below.
			if e := os.WriteFile(filepath.Join(app, "immichclone.exe"), []byte("partial"), 0600); e != nil {
				t.Fatal(e)
			}
			if e := RecoverInterruptedPayload(app, backup); e != nil {
				t.Fatal("durable interrupted recovery", e)
			}
			publicationAssertSet(t, app, "prior-")
			if _, e := os.Stat(filepath.Join(app, newName)); !errors.Is(e, os.ErrNotExist) {
				t.Fatal("new payload retained", e)
			}
			if e := RecoverInterruptedPayload(app, backup); e != nil {
				t.Fatal("idempotent recovery", e)
			}
			publicationAssertSet(t, app, "prior-")
		})
	}
}

func TestPayloadInterruptedRecoveryRefusesReplacedObject(t *testing.T) {
	app, source, backup, _ := publicationFixture(t)
	exitPublication(t, app, source, backup, 1)
	target := filepath.Join(app, "prototype-notes.txt")
	// Keep the prior object alive at another name so NTFS cannot reuse its ID.
	if e := os.Rename(target, target+".retained"); e != nil {
		t.Fatal(e)
	}
	if e := os.WriteFile(target, []byte("replacement must survive"), 0600); e != nil {
		t.Fatal(e)
	}
	before, e := os.ReadFile(filepath.Join(app, "Export-Diagnostics.ps1"))
	if e != nil {
		t.Fatal(e)
	}
	if e := RecoverInterruptedPayload(app, backup); e == nil {
		t.Fatal("replacement object accepted")
	}
	assertBytes(t, target, "replacement must survive")
	assertBytes(t, filepath.Join(app, "Export-Diagnostics.ps1"), string(before))
}

func TestPayloadInterruptedRecoveryRejectsCorruptBackupBeforeMutation(t *testing.T) {
	app, source, backup, _ := publicationFixture(t)
	exitPublication(t, app, source, backup, 5)
	if e := os.WriteFile(filepath.Join(backup, backupName(0)), []byte("corrupt snapshot"), 0600); e != nil {
		t.Fatal(e)
	}
	before, e := os.ReadFile(filepath.Join(app, "Export-Diagnostics.ps1"))
	if e != nil {
		t.Fatal(e)
	}
	if e := RecoverInterruptedPayload(app, backup); e == nil {
		t.Fatal("corrupt old source accepted")
	}
	assertBytes(t, filepath.Join(app, "Export-Diagnostics.ps1"), string(before))
}
