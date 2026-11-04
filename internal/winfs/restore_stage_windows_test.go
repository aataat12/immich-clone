//go:build windows

package winfs

import (
	"encoding/binary"
	"errors"
	"golang.org/x/sys/windows"
	"os"
	"path/filepath"
	"testing"
	"unicode/utf16"
)

func restorePrimitiveSD(t *testing.T) *windows.SECURITY_DESCRIPTOR {
	t.Helper()
	sd, e := windows.SecurityDescriptorFromString("D:P(A;OICI;FA;;;SY)(A;OICI;FA;;;BA)")
	if e != nil {
		t.Fatal(e)
	}
	return sd
}
func TestRestoreParentDriveRoot(t *testing.T) {
	root := filepath.VolumeName(t.TempDir()) + `\`
	strict, e := OpenDirectory(root)
	if e != nil {
		t.Fatal(e)
	}
	defer strict.Close()
	expected, e := strict.Identity()
	if e != nil {
		t.Fatal(e)
	}
	parent, e := OpenRestoreParent(root)
	if e != nil {
		t.Fatal(e)
	}
	defer parent.Close()
	actual, e := parent.Identity()
	if e != nil || actual != expected {
		t.Fatal("drive-root identity changed", e)
	}
}
func TestRestoreStageExclusivePublishAndRemoval(t *testing.T) {
	base := t.TempDir()
	parent, e := OpenDirectory(base)
	if e != nil {
		t.Fatal(e)
	}
	defer parent.Close()
	strict, e := OpenDirectory(base)
	if e != nil {
		t.Fatal(e)
	}
	expected, e := strict.Identity()
	strict.Close()
	actual, ae := parent.Identity()
	if e != nil || ae != nil || actual != expected {
		t.Fatal("restore parent identity changed", e, ae)
	}
	if e = os.Rename(base, base+"-redirected"); e == nil {
		t.Fatal("restore parent renamed while pinned")
	}
	s, e := parent.CreateExclusiveDirectory("restore é", restorePrimitiveSD(t))
	if e != nil {
		t.Fatal(e)
	}
	defer s.Close()
	shared, e := OpenRestoreParent(base)
	if e != nil {
		t.Fatal(e)
	}
	if e = parent.Close(); e != nil {
		shared.Close()
		t.Fatal(e)
	}
	parent = shared
	defer parent.Close()
	before, e := s.Identity()
	if e != nil {
		t.Fatal(e)
	}
	sd, e := s.Directory().SecurityDescriptor()
	if e != nil {
		t.Fatal(e)
	}
	control, _, e := sd.Control()
	if e != nil || control&windows.SE_DACL_PROTECTED == 0 {
		t.Fatal("stage was not private/protected at creation", e)
	}
	if other, e := parent.CreateExclusiveDirectory("restore é", restorePrimitiveSD(t)); e == nil {
		other.Close()
		t.Fatal("exclusive directory replaced existing stage")
	}
	if e = os.Rename(filepath.Join(base, "restore é"), filepath.Join(base, "redirected")); e == nil {
		t.Fatal("stage root renamed while held")
	}
	child, e := s.Directory().Child("nested", restorePrimitiveSD(t))
	if e != nil {
		t.Fatal(e)
	}
	f, e := child.CreateExclusiveFile("payload.bin", restorePrimitiveSD(t))
	if e != nil {
		t.Fatal(e)
	}
	if _, e = f.WriteString("restored bytes"); e != nil {
		t.Fatal(e)
	}
	if e = f.Close(); e != nil {
		t.Fatal(e)
	}
	if e = child.Close(); e != nil {
		t.Fatal(e)
	}
	existing := filepath.Join(base, "existing")
	if e = os.Mkdir(existing, 0700); e != nil {
		t.Fatal(e)
	}
	if e = os.WriteFile(filepath.Join(existing, "sentinel"), []byte("outside unchanged"), 0600); e != nil {
		t.Fatal(e)
	}
	if e = s.Publish(parent, "existing"); e == nil {
		t.Fatal("publish replaced an existing target")
	}
	if b, e := os.ReadFile(filepath.Join(existing, "sentinel")); e != nil || string(b) != "outside unchanged" {
		t.Fatal("collision changed target", e)
	}
	if e = s.Publish(parent, "completed é"); e != nil {
		t.Fatal("native relative publication failed", e)
	}
	after, e := s.Identity()
	if e != nil || after != before {
		t.Fatal("publication changed actual stage identity", e)
	}
	if _, e = os.Stat(filepath.Join(base, "restore é")); !errors.Is(e, os.ErrNotExist) {
		t.Fatal("old stage name still present", e)
	}
	if e = s.Remove(); e != nil {
		t.Fatal("held bottom-up stage removal failed", e)
	}
	if _, e = os.Stat(filepath.Join(base, "completed é")); !errors.Is(e, os.ErrNotExist) {
		t.Fatal("selected stage not removed", e)
	}
	t.Log("PASS: exclusive protected stage pins its identity, refuses overwrite, publishes relative Unicode name without identity change and removes selected held tree bottom-up")
}
func TestRestoreProgressRelativeRenameAndSharedRead(t *testing.T) {
	base := t.TempDir()
	parent, e := OpenDirectory(base)
	if e != nil {
		t.Fatal(e)
	}
	defer parent.Close()
	s, e := parent.CreateExclusiveDirectory("stage", restorePrimitiveSD(t))
	if e != nil {
		t.Fatal(e)
	}
	defer s.Close()
	shared, e := OpenRestoreParent(base)
	if e != nil {
		t.Fatal(e)
	}
	if e = parent.Close(); e != nil {
		shared.Close()
		t.Fatal(e)
	}
	parent = shared
	defer parent.Close()
	d := s.Directory()
	f, e := d.CreateExclusiveFile("progress.part", restorePrimitiveSD(t))
	if e != nil {
		t.Fatal(e)
	}
	defer f.Close()
	if _, e = f.WriteString("bounded progress bytes"); e != nil {
		t.Fatal(e)
	}
	if e = f.Sync(); e != nil {
		t.Fatal(e)
	}
	if e = d.RenameFileNoReplace(f, "progress.json"); e != nil {
		t.Fatal(e)
	}
	if strict, e := d.OpenFile("progress.json", false); e == nil {
		strict.Close()
		t.Fatal("strict source read unexpectedly shared live publication DELETE handle")
	}
	read, e := d.OpenSharedReadFile("progress.json")
	if e != nil {
		t.Fatal("progress read could not coexist with live DELETE publication handle", e)
	}
	b := make([]byte, 64)
	n, e := read.Read(b)
	ce := read.Close()
	if e != nil || ce != nil || string(b[:n]) != "bounded progress bytes" {
		t.Fatal("relative progress read changed bytes", e, ce)
	}
	collision, e := d.CreateExclusiveFile("existing.json", restorePrimitiveSD(t))
	if e != nil {
		t.Fatal(e)
	}
	collision.Close()
	if e = d.RenameFileNoReplace(f, "existing.json"); e == nil {
		t.Fatal("relative file publication replaced target")
	}
	t.Log("PASS: fixed progress rename is no-replace; shared read survives live DELETE publication handle without weakening strict source-reader sharing")
}

// Mount-point buffer: REPARSE_DATA_BUFFER's 8-byte header, 8-byte mount-point
// fields, then UTF-16 substitute/print names, each terminated by NUL.
func restoreMountPointBuffer(target string) []byte {
	substitute := utf16.Encode([]rune(`\??\` + target))
	printName := utf16.Encode([]rune(target))
	b := make([]byte, 16+2*(len(substitute)+1+len(printName)+1))
	binary.LittleEndian.PutUint32(b, windows.IO_REPARSE_TAG_MOUNT_POINT)
	binary.LittleEndian.PutUint16(b[4:], uint16(len(b)-8))
	binary.LittleEndian.PutUint16(b[10:], uint16(len(substitute)*2))
	binary.LittleEndian.PutUint16(b[12:], uint16((len(substitute)+1)*2))
	binary.LittleEndian.PutUint16(b[14:], uint16(len(printName)*2))
	for i, v := range substitute {
		binary.LittleEndian.PutUint16(b[16+2*i:], v)
	}
	start := 16 + 2*(len(substitute)+1)
	for i, v := range printName {
		binary.LittleEndian.PutUint16(b[start+2*i:], v)
	}
	return b
}
func TestRestoreParentHeldStageRejectsReparseConversion(t *testing.T) {
	base := t.TempDir()
	outside := filepath.Join(base, "outside")
	empty := filepath.Join(base, "empty-control")
	destination := filepath.Join(base, "destination")
	for _, p := range []string{outside, empty, destination} {
		if e := os.Mkdir(p, 0700); e != nil {
			t.Fatal(e)
		}
	}
	buffer := restoreMountPointBuffer(outside)
	openWriter := func(path string) (windows.Handle, error) {
		u, e := windows.UTF16PtrFromString(path)
		if e != nil {
			return 0, e
		}
		return windows.CreateFile(u, windows.GENERIC_WRITE, windows.FILE_SHARE_READ|windows.FILE_SHARE_WRITE|windows.FILE_SHARE_DELETE, nil, windows.OPEN_EXISTING, windows.FILE_FLAG_BACKUP_SEMANTICS|windows.FILE_FLAG_OPEN_REPARSE_POINT, 0)
	}
	set := func(h windows.Handle) error {
		var n uint32
		return windows.DeviceIoControl(h, windows.FSCTL_SET_REPARSE_POINT, &buffer[0], uint32(len(buffer)), nil, 0, &n, nil)
	}
	control, e := openWriter(empty)
	if e != nil {
		t.Fatal(e)
	}
	if e = set(control); e != nil {
		windows.CloseHandle(control)
		t.Fatal("positive control could not set mount-point reparse", e)
	}
	// Remove only the reparse record, never traverse the outside target.
	deleteBuffer := make([]byte, 8)
	binary.LittleEndian.PutUint32(deleteBuffer, windows.IO_REPARSE_TAG_MOUNT_POINT)
	var n uint32
	e = windows.DeviceIoControl(control, windows.FSCTL_DELETE_REPARSE_POINT, &deleteBuffer[0], 8, nil, 0, &n, nil)
	windows.CloseHandle(control)
	if e != nil {
		t.Fatal("positive-control cleanup", e)
	}
	strict, e := OpenDirectory(destination)
	if e != nil {
		t.Fatal(e)
	}
	defer strict.Close()
	if h, e := openWriter(destination); e == nil {
		windows.CloseHandle(h)
		t.Fatal("strict empty parent permitted write handle")
	}
	stage, e := strict.CreateExclusiveDirectory("held-stage", restorePrimitiveSD(t))
	if e != nil {
		t.Fatal(e)
	}
	defer stage.Close()
	expected, e := strict.Identity()
	if e != nil {
		t.Fatal(e)
	}
	parent, e := OpenRestoreParent(destination)
	if e != nil {
		t.Fatal(e)
	}
	defer parent.Close()
	if e = strict.Close(); e != nil {
		t.Fatal(e)
	}
	attacker, e := openWriter(destination)
	if e != nil {
		t.Fatal("write-sharing parent did not admit positive attack handle", e)
	}
	e = set(attacker)
	windows.CloseHandle(attacker)
	if !errors.Is(e, windows.ERROR_DIR_NOT_EMPTY) {
		t.Fatal("held stage did not prevent parent reparse conversion", e)
	}
	actual, e := parent.Identity()
	if e != nil || actual != expected {
		t.Fatal("parent identity changed", e)
	}
	if e = checked(parent.holds[len(parent.holds)-1], true); e != nil {
		t.Fatal("parent became reparse", e)
	}
	if e = stage.Remove(); e != nil {
		t.Fatal(e)
	}
	t.Log("PASS: real mount-point control succeeds on empty directory; strict parent rejects writer, held private stage makes write-sharing parent reject conversion without identity change")
}
