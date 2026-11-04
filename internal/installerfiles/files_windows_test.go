//go:build windows

package installerfiles

import (
	"errors"
	"golang.org/x/sys/windows"
	"os"
	"path/filepath"
	"testing"
)

func fixture(t *testing.T) (app, data, backup, outside string) {
	t.Helper()
	base := t.TempDir()
	app = filepath.Join(base, "application é")
	data = filepath.Join(base, "state")
	backup = filepath.Join(app, "rollback")
	outside = filepath.Join(base, "outside")
	for _, p := range []string{app, data, backup, outside} {
		if e := os.Mkdir(p, 0700); e != nil {
			t.Fatal(e)
		}
	}
	sd, e := windows.SecurityDescriptorFromString("D:P(A;OICI;FA;;;SY)(A;OICI;FA;;;BA)")
	if e != nil {
		t.Fatal(e)
	}
	acl, _, e := sd.DACL()
	if e != nil {
		t.Fatal(e)
	}
	if e = windows.SetNamedSecurityInfo(backup, windows.SE_FILE_OBJECT, windows.DACL_SECURITY_INFORMATION|windows.PROTECTED_DACL_SECURITY_INFORMATION, nil, nil, acl, nil); e != nil {
		t.Fatal(e)
	}
	writeLive(t, app, data, "baseline-")
	return
}
func writeLive(t *testing.T, app, data, prefix string) {
	t.Helper()
	for i, n := range names {
		dir := data
		if i == 0 {
			dir = app
		}
		if e := os.WriteFile(filepath.Join(dir, n), []byte(prefix+n), 0600); e != nil {
			t.Fatal(e)
		}
	}
}
func assertLive(t *testing.T, app, data, prefix string) {
	t.Helper()
	for i, n := range names {
		dir := data
		if i == 0 {
			dir = app
		}
		assertBytes(t, filepath.Join(dir, n), prefix+n)
	}
}
func assertBytes(t *testing.T, p, want string) {
	t.Helper()
	b, e := os.ReadFile(p)
	if e != nil || string(b) != want {
		t.Fatal("fixture bytes did not match expected preservation")
	}
}

func TestActualSnapshotRestoreSet(t *testing.T) {
	app, data, backup, _ := fixture(t)
	if e := Snapshot(app, data); e != nil {
		t.Fatal(e)
	}
	for _, n := range names {
		assertBytes(t, filepath.Join(backup, n), "baseline-"+n)
	}
	writeLive(t, app, data, "changed-")
	if e := Restore(app, data); e != nil {
		t.Fatal(e)
	}
	assertLive(t, app, data, "baseline-")
	t.Log("PASS: complete native three-file snapshot/readiness/restore preserves actual binary/catalog/code bytes")
}

func TestIncompleteSnapshotRefusesRestore(t *testing.T) {
	app, data, _, _ := fixture(t)
	if e := Snapshot(app, data); e != nil {
		t.Fatal(e)
	}
	writeLive(t, app, data, "latest-")
	e := snapshot(app, data, func(i int) error {
		if i == 1 {
			return errors.New("synthetic interruption after first copy")
		}
		return nil
	})
	if e == nil {
		t.Fatal("incomplete snapshot accepted")
	}
	if e = Restore(app, data); e == nil {
		t.Fatal("incomplete snapshot allowed rollback")
	}
	assertLive(t, app, data, "latest-")
	t.Log("PASS: injected interruption after first copy invalidates readiness; rollback refused without changing live state")
}

func TestUnsafeThirdSnapshotFileBeforeTruncate(t *testing.T) {
	app, data, backup, outside := fixture(t)
	if e := Snapshot(app, data); e != nil {
		t.Fatal(e)
	}
	writeLive(t, app, data, "latest-")
	sentinel := filepath.Join(outside, "sentinel")
	if e := os.WriteFile(sentinel, []byte("outside"), 0600); e != nil {
		t.Fatal(e)
	}
	third := filepath.Join(backup, names[2])
	if e := os.Remove(third); e != nil {
		t.Fatal(e)
	}
	if e := os.Link(sentinel, third); e != nil {
		t.Fatal(e)
	}
	if e := Snapshot(app, data); e == nil {
		t.Fatal("unsafe third snapshot destination accepted")
	}
	for _, n := range names[:2] {
		assertBytes(t, filepath.Join(backup, n), "baseline-"+n)
	}
	assertBytes(t, sentinel, "outside")
	assertLive(t, app, data, "latest-")
	t.Log("PASS: unsafe third snapshot target refused before truncating first two; outside bytes preserved")
}

func TestUnsafeThirdRestoreTargetBeforeTruncate(t *testing.T) {
	app, data, _, outside := fixture(t)
	if e := Snapshot(app, data); e != nil {
		t.Fatal(e)
	}
	writeLive(t, app, data, "latest-")
	sentinel := filepath.Join(outside, "sentinel")
	if e := os.WriteFile(sentinel, []byte("outside"), 0600); e != nil {
		t.Fatal(e)
	}
	third := filepath.Join(data, names[2])
	if e := os.Remove(third); e != nil {
		t.Fatal(e)
	}
	if e := os.Link(sentinel, third); e != nil {
		t.Fatal(e)
	}
	if e := Restore(app, data); e == nil {
		t.Fatal("unsafe third live target accepted")
	}
	assertBytes(t, filepath.Join(app, names[0]), "latest-"+names[0])
	assertBytes(t, filepath.Join(data, names[1]), "latest-"+names[1])
	assertBytes(t, sentinel, "outside")
	t.Log("PASS: unsafe third live destination refused before first restoration write; outside bytes preserved")
}

func TestCorruptSnapshotAndChangedIdentityRefused(t *testing.T) {
	app, data, backup, _ := fixture(t)
	if e := Snapshot(app, data); e != nil {
		t.Fatal(e)
	}
	writeLive(t, app, data, "latest-")
	if e := os.WriteFile(filepath.Join(backup, names[2]), []byte("corrupt"), 0600); e != nil {
		t.Fatal(e)
	}
	if e := Restore(app, data); e == nil {
		t.Fatal("corrupt snapshot restored")
	}
	assertLive(t, app, data, "latest-")
	if e := Snapshot(app, data); e != nil {
		t.Fatal(e)
	}
	if e := os.Rename(data, data+"-old"); e != nil {
		t.Fatal(e)
	}
	if e := os.Mkdir(data, 0700); e != nil {
		t.Fatal(e)
	}
	writeLive(t, app, data, "replacement-")
	if e := Restore(app, data); e == nil {
		t.Fatal("changed data-root identity accepted")
	}
	assertLive(t, app, data, "replacement-")
	t.Log("PASS: all snapshot hashes checked before live mutation; replacement data-root identity refused")
}

func TestUnsafeRoleRootsRefused(t *testing.T) {
	app, data, backup, _ := fixture(t)
	if e := Snapshot(app, app); e == nil {
		t.Fatal("equal roles accepted")
	}
	if e := Snapshot(app, backup); e == nil {
		t.Fatal("overlapping roles accepted")
	}
	sd, e := windows.SecurityDescriptorFromString("D:P(A;OICI;FA;;;SY)(A;OICI;FA;;;BA)(A;OICI;GR;;;WD)")
	if e != nil {
		t.Fatal(e)
	}
	acl, _, e := sd.DACL()
	if e != nil {
		t.Fatal(e)
	}
	if e = windows.SetNamedSecurityInfo(backup, windows.SE_FILE_OBJECT, windows.DACL_SECURITY_INFORMATION|windows.PROTECTED_DACL_SECURITY_INFORMATION, nil, nil, acl, nil); e != nil {
		t.Fatal(e)
	}
	if e = Snapshot(app, data); e == nil {
		t.Fatal("publicly readable snapshot root accepted")
	}
	assertLive(t, app, data, "baseline-")
	t.Log("PASS: equal/overlapping roles and nonprivate actual rollback-root DACL refused")
}
