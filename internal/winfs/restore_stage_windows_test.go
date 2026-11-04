//go:build windows

package winfs

import (
	"errors"
	"golang.org/x/sys/windows"
	"os"
	"path/filepath"
	"testing"
)

func restorePrimitiveSD(t *testing.T) *windows.SECURITY_DESCRIPTOR {
	t.Helper()
	sd, e := windows.SecurityDescriptorFromString("D:P(A;OICI;FA;;;SY)(A;OICI;FA;;;BA)")
	if e != nil {
		t.Fatal(e)
	}
	return sd
}
func TestRestoreStageExclusivePublishAndRemoval(t *testing.T) {
	base := t.TempDir()
	parent, e := OpenDirectory(base)
	if e != nil {
		t.Fatal(e)
	}
	defer parent.Close()
	s, e := parent.CreateExclusiveDirectory("restore é", restorePrimitiveSD(t))
	if e != nil {
		t.Fatal(e)
	}
	defer s.Close()
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
	parent, e := OpenDirectory(t.TempDir())
	if e != nil {
		t.Fatal(e)
	}
	defer parent.Close()
	s, e := parent.CreateExclusiveDirectory("stage", restorePrimitiveSD(t))
	if e != nil {
		t.Fatal(e)
	}
	defer s.Close()
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
