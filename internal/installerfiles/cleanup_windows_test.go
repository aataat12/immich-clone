//go:build windows

package installerfiles

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"golang.org/x/sys/windows"
	"immichclone/internal/winfs"
)

// Independent removal contract: changing the product allowlist cannot silently
// redefine the expected removal set or authorize unrelated user-file deletion.
var cleanupTestFiles = []string{
	"Manage-Service.ps1", "Storage-Security.ps1", "installer-files.exe", "immichclone.exe",
	"Export-Diagnostics.ps1", "THIRD-PARTY-LICENSES.txt", "prototype-notes.txt",
	`media\nested é\decoder.bin`, `ai\nested é\model.onnx`,
	`rollback\immichclone.exe`, `rollback\catalog.db`, `rollback\access-code.txt`, `rollback\snapshot-ready.json`,
}

func cleanupFixture(t *testing.T) (app, data, outside string) {
	t.Helper()
	base := t.TempDir()
	app = filepath.Join(base, "application é")
	data = filepath.Join(base, "retained library é")
	outside = filepath.Join(base, "outside")
	for _, root := range []string{app, data, outside} {
		if e := os.Mkdir(root, 0700); e != nil {
			t.Fatal(e)
		}
		setPrivacyDescriptor(t, root, "D:P(A;OICI;FA;;;SY)(A;OICI;FA;;;BA)")
	}
	for _, name := range cleanupTestFiles {
		p := filepath.Join(app, name)
		if e := os.MkdirAll(filepath.Dir(p), 0700); e != nil {
			t.Fatal(e)
		}
		if e := os.WriteFile(p, []byte("managed-"+name), 0600); e != nil {
			t.Fatal(e)
		}
	}
	setPrivacyDescriptor(t, filepath.Join(app, "rollback"), "D:P(A;OICI;FA;;;SY)(A;OICI;FA;;;BA)")
	for _, name := range []string{`originals\original.bin`, "catalog.db", "access-code.txt"} {
		p := filepath.Join(data, name)
		if e := os.MkdirAll(filepath.Dir(p), 0700); e != nil {
			t.Fatal(e)
		}
		if e := os.WriteFile(p, []byte("retained-data-"+name), 0600); e != nil {
			t.Fatal(e)
		}
	}
	return
}
func cleanupAssertData(t *testing.T, data string) {
	t.Helper()
	for _, name := range []string{`originals\original.bin`, "catalog.db", "access-code.txt"} {
		assertBytes(t, filepath.Join(data, name), "retained-data-"+name)
	}
}
func cleanupAssertOld(t *testing.T, app string) {
	t.Helper()
	for _, name := range cleanupTestFiles {
		assertBytes(t, filepath.Join(app, name), "managed-"+name)
	}
}
func TestCleanupRemovesOnlyManagedPayloadAndKnownRollback(t *testing.T) {
	app, data, _ := cleanupFixture(t)
	retained := []string{"unrelated user file.txt", `rollback\unrelated recovery note.txt`}
	before := map[string]string{}
	for _, name := range retained {
		p := filepath.Join(app, name)
		if e := os.WriteFile(p, []byte("retained-unrelated-"+name), 0600); e != nil {
			t.Fatal(e)
		}
		before[name] = treeSD(t, p).String()
	}
	if e := AuditCleanup(app, data); e != nil {
		t.Fatal("safe stationary cleanup audit refused", e)
	}
	cleanupAssertOld(t, app)
	cleanupAssertData(t, data)
	if e := Cleanup(app, data); e != nil {
		t.Fatal("native stopped-service cleanup refused", e)
	}
	for _, name := range cleanupTestFiles {
		if _, e := os.Stat(filepath.Join(app, name)); !errors.Is(e, os.ErrNotExist) {
			t.Fatal("managed payload or known rollback file remains", e)
		}
	}
	for _, dir := range []string{"media", "ai"} {
		if _, e := os.Stat(filepath.Join(app, dir)); !errors.Is(e, os.ErrNotExist) {
			t.Fatal("empty managed provider directory remains", e)
		}
	}
	for _, name := range retained {
		p := filepath.Join(app, name)
		assertBytes(t, p, "retained-unrelated-"+name)
		if treeSD(t, p).String() != before[name] {
			t.Fatal("cleanup changed unrelated descriptor")
		}
	}
	cleanupAssertData(t, data)
	t.Log("PASS: native nested Unicode cleanup removes fixed payload/provider trees/known rollback files, preserves unrelated root and rollback bytes/descriptors, and leaves library originals/catalog/access code unchanged")
}
func TestCleanupUnsafeActualAliasesRefuseBeforeEarlierDeletion(t *testing.T) {
	for _, kind := range []string{"file-link", "hardlink", "provider-directory-link"} {
		t.Run(kind, func(t *testing.T) {
			treeEnableLinks(t)
			app, data, outside := cleanupFixture(t)
			sentinel := filepath.Join(outside, "outside.bin")
			if e := os.WriteFile(sentinel, []byte("outside sentinel"), 0600); e != nil {
				t.Fatal(e)
			}
			victim := filepath.Join(app, "prototype-notes.txt")
			if kind == "provider-directory-link" {
				victim = filepath.Join(app, "ai")
				if e := os.RemoveAll(victim); e != nil {
					t.Fatal(e)
				}
				if e := os.Symlink(outside, victim); e != nil {
					t.Fatal(e)
				}
			} else {
				if e := os.Remove(victim); e != nil {
					t.Fatal(e)
				}
				var e error
				if kind == "hardlink" {
					e = os.Link(sentinel, victim)
				} else {
					e = os.Symlink(sentinel, victim)
				}
				if e != nil {
					t.Fatal(e)
				}
			}
			sd := treeSD(t, sentinel).String()
			if e := AuditCleanup(app, data); e == nil {
				t.Fatal("unsafe actual cleanup alias passed stationary audit")
			}
			if e := Cleanup(app, data); e == nil {
				t.Fatal("unsafe actual cleanup alias was deleted")
			}
			assertBytes(t, filepath.Join(app, "Manage-Service.ps1"), "managed-Manage-Service.ps1")
			assertBytes(t, filepath.Join(app, "immichclone.exe"), "managed-immichclone.exe")
			assertBytes(t, sentinel, "outside sentinel")
			if treeSD(t, sentinel).String() != sd {
				t.Fatal("refused cleanup changed outside descriptor")
			}
			cleanupAssertData(t, data)
			t.Log("PASS: actual cleanup alias refuses stationary audit and native removal before earlier payload deletion with outside bytes/descriptors and library preserved")
		})
	}
}
func TestCleanupUnsafeMutationGrantRefusesBeforeEarlierDeletion(t *testing.T) {
	app, data, _ := cleanupFixture(t)
	victim := filepath.Join(app, `ai\nested é\model.onnx`)
	setPrivacyDescriptor(t, victim, "D:P(A;;FA;;;SY)(A;;FA;;;BA)(A;;GW;;;BU)")
	before := treeSD(t, victim).String()
	if e := AuditCleanup(app, data); e == nil {
		t.Fatal("actual untrusted mutation grant passed stationary audit")
	}
	if e := Cleanup(app, data); e == nil {
		t.Fatal("actual untrusted mutation grant accepted for removal")
	}
	cleanupAssertOld(t, app)
	cleanupAssertData(t, data)
	if treeSD(t, victim).String() != before {
		t.Fatal("refused cleanup repaired unsafe descriptor")
	}
	t.Log("PASS: actual ordinary-user code mutation permission refuses all cleanup before prior payload bytes or descriptors change")
}
func TestCleanupLateChildNameRefusesBeforeFirstDeletion(t *testing.T) {
	app, data, _ := cleanupFixture(t)
	late := filepath.Join(app, `media\nested é\late-name.bin`)
	called := false
	e := cleanup(app, data, func() error { called = true; return os.WriteFile(late, []byte("late child bytes"), 0600) })
	if !called {
		t.Fatal("late-child hook was not exercised")
	}
	assertBytes(t, late, "late child bytes")
	if e == nil {
		t.Fatal("late actual child addition accepted for cleanup")
	}
	cleanupAssertOld(t, app)
	cleanupAssertData(t, data)
	t.Log("PASS: actual late child publication after complete cleanup planning refuses before first selected-object deletion and preserves every old payload and library byte")
}
func TestCleanupExistingWriterRefusesBeforeEarlierDeletion(t *testing.T) {
	app, data, _ := cleanupFixture(t)
	victim := filepath.Join(app, `ai\nested é\model.onnx`)
	u, e := windows.UTF16PtrFromString(victim)
	if e != nil {
		t.Fatal(e)
	}
	h, e := windows.CreateFile(u, windows.GENERIC_WRITE, windows.FILE_SHARE_READ|windows.FILE_SHARE_WRITE|windows.FILE_SHARE_DELETE, nil, windows.OPEN_EXISTING, 0, 0)
	if e != nil {
		t.Fatal("real unrelated writer positive control failed", e)
	}
	audit := AuditCleanup(app, data)
	e = Cleanup(app, data)
	if ce := windows.CloseHandle(h); ce != nil {
		t.Fatal(ce)
	}
	if audit == nil || e == nil {
		t.Fatal("existing writer failed to force stationary/native sharing refusal", audit, e)
	}
	cleanupAssertOld(t, app)
	cleanupAssertData(t, data)
	if e := Cleanup(app, data); e != nil {
		t.Fatal("released unrelated writer should permit normal cleanup", e)
	}
	cleanupAssertData(t, data)
	t.Log("PASS: actual preexisting code data writer forces sharing refusal before earlier deletion; release permits normal cleanup without touching library data")
}
func TestCleanupDataRoleOverlapRefuses(t *testing.T) {
	for _, kind := range []string{"same-root", "data-inside-provider", "data-is-app-parent"} {
		t.Run(kind, func(t *testing.T) {
			app, data, _ := cleanupFixture(t)
			selectedData := data
			switch kind {
			case "same-root":
				selectedData = app
			case "data-inside-provider":
				selectedData = filepath.Join(app, `media\nested é`)
			case "data-is-app-parent":
				selectedData = filepath.Dir(app)
			}
			if e := AuditCleanup(app, selectedData); e == nil {
				t.Fatal("overlapping cleanup/data role passed stationary audit")
			}
			if e := Cleanup(app, selectedData); e == nil {
				t.Fatal("overlapping cleanup/data role accepted for deletion")
			}
			cleanupAssertOld(t, app)
			cleanupAssertData(t, data)
		})
	}
	t.Log("PASS: same and nested cleanup/library roles refuse before any selected payload or retained data deletion")
}

func TestCleanupAuditWithPersistentUninstallerDirectoryPin(t *testing.T) {
	app, data, _ := cleanupFixture(t)
	// The production uninstaller keeps its selected app-directory pin live until
	// builtin metadata removal finishes. A native audit must coexist with it.
	pin, e := winfs.OpenDirectory(app)
	if e != nil {
		t.Fatal(e)
	}
	defer pin.Close()
	if e = AuditCleanup(app, data); e != nil {
		t.Fatal("read-only stationary audit conflicts with persistent uninstaller directory pin", e)
	}
	cleanupAssertOld(t, app)
	cleanupAssertData(t, data)
	t.Log("PASS: read-only cleanup audit coexists with the uninstaller's persistent no-write/delete-sharing app-directory pin without requesting conflicting mutation authority")
}
