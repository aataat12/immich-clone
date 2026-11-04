//go:build windows

package winfs

import (
	"os"
	"path/filepath"
	"testing"
	"unsafe"

	"golang.org/x/sys/windows"
)

const aclInitial = "D:P(A;OICI;FA;;;SY)(A;OICI;FA;;;BA)(A;OICI;GR;;;WD)"
const aclFinal = "D:P(A;OICI;FA;;;SY)(A;OICI;FA;;;BA)"

func aclDescriptor(t *testing.T, text string) *windows.SECURITY_DESCRIPTOR {
	t.Helper()
	sd, e := windows.SecurityDescriptorFromString(text)
	if e != nil {
		t.Fatal(e)
	}
	return sd
}
func aclNamed(t *testing.T, path string) string {
	t.Helper()
	sd, e := windows.GetNamedSecurityInfo(path, windows.SE_FILE_OBJECT, windows.OWNER_SECURITY_INFORMATION|windows.DACL_SECURITY_INFORMATION)
	if e != nil {
		t.Fatal(e)
	}
	return sd.String()
}
func aclNamedSet(t *testing.T, path, text string) {
	t.Helper()
	sd := aclDescriptor(t, text)
	dacl, _, e := sd.DACL()
	if e != nil {
		t.Fatal(e)
	}
	if e = windows.SetNamedSecurityInfo(path, windows.SE_FILE_OBJECT, windows.DACL_SECURITY_INFORMATION|windows.PROTECTED_DACL_SECURITY_INFORMATION, nil, nil, dacl, nil); e != nil {
		t.Fatal(e)
	}
}
func aclFixture(t *testing.T) (root, child, file string) {
	t.Helper()
	root, _ = fixture(t)
	aclNamedSet(t, root, aclInitial)
	child = filepath.Join(root, "inherited child")
	if e := os.Mkdir(child, 0700); e != nil {
		t.Fatal(e)
	}
	file = filepath.Join(child, "retained.txt")
	if e := os.WriteFile(file, []byte("retained bytes"), 0600); e != nil {
		t.Fatal(e)
	}
	return
}
func aclAssertPolicy(t *testing.T, sd *windows.SECURITY_DESCRIPTOR, protected bool) {
	t.Helper()
	control, _, e := sd.Control()
	if e != nil || (control&windows.SE_DACL_PROTECTED != 0) != protected {
		t.Fatal("unexpected inheritance protection", e)
	}
	dacl, _, e := sd.DACL()
	if e != nil || dacl == nil {
		t.Fatal("object has no DACL", e)
	}
	if dacl.AceCount != 2 {
		t.Fatal("object has unexpected ACE count")
	}
	seen := map[string]bool{}
	for i := uint32(0); i < uint32(dacl.AceCount); i++ {
		var ace *windows.ACCESS_ALLOWED_ACE
		if e = windows.GetAce(dacl, i, &ace); e != nil {
			t.Fatal(e)
		}
		if ace.Header.AceType != windows.ACCESS_ALLOWED_ACE_TYPE || ace.Header.AceFlags&3 != 3 || ace.Header.AceFlags&12 != 0 || uint32(ace.Mask) != 0x1f01ff {
			t.Fatal("object ACE not fixed inheritable full access")
		}
		sid := (*windows.SID)(unsafe.Pointer(&ace.SidStart)).String()
		if sid != "S-1-5-18" && sid != "S-1-5-32-544" {
			t.Fatal("unexpected policy principal")
		}
		if seen[sid] {
			t.Fatal("duplicate principal")
		}
		seen[sid] = true
	}
	if !seen["S-1-5-18"] || !seen["S-1-5-32-544"] {
		t.Fatal("missing policy principal")
	}
}

func TestACLNormalPropagationPositiveControl(t *testing.T) {
	root, child, file := aclFixture(t)
	beforeChild, beforeFile := aclNamed(t, child), aclNamed(t, file)
	aclNamedSet(t, root, aclFinal)
	if aclNamed(t, child) == beforeChild || aclNamed(t, file) == beforeFile {
		t.Fatal("required ordinary root ACL propagation positive control did not change unprotected descendants")
	}
	body, e := os.ReadFile(file)
	if e != nil || string(body) != "retained bytes" {
		t.Fatal("normal propagation changed bytes", e)
	}
	t.Log("PASS: ordinary named root ACL update really propagates to existing unprotected directory/file controls")
}

func TestACLHeldRootUpdateDoesNotPropagate(t *testing.T) {
	for _, locked := range []bool{false, true} {
		name := "unlocked-children"
		if locked {
			name = "locked-child"
		}
		t.Run(name, func(t *testing.T) {
			root, child, file := aclFixture(t)
			beforeChild, beforeFile := aclNamed(t, child), aclNamed(t, file)
			protected := filepath.Join(root, "protected child")
			if e := os.Mkdir(protected, 0700); e != nil {
				t.Fatal(e)
			}
			aclNamedSet(t, protected, aclInitial)
			beforeProtected := aclNamed(t, protected)
			_, outside := fixture(t)
			sentinel := filepath.Join(outside, "outside-sentinel")
			if e := os.WriteFile(sentinel, []byte("outside unchanged"), 0600); e != nil {
				t.Fatal(e)
			}
			outsideBefore, sentinelBefore := aclNamed(t, outside), aclNamed(t, sentinel)
			if e := os.Symlink(outside, filepath.Join(root, "outside alias")); e != nil {
				t.Fatal("required real child symlink unavailable", e)
			}
			writer := windows.InvalidHandle
			defer func() {
				if writer != windows.InvalidHandle {
					windows.CloseHandle(writer)
				}
			}()
			if locked {
				u, e := windows.UTF16PtrFromString(file)
				if e != nil {
					t.Fatal(e)
				}
				writer, e = windows.CreateFile(u, windows.GENERIC_READ|windows.GENERIC_WRITE, 0, nil, windows.OPEN_EXISTING, 0, 0)
				if e != nil {
					t.Fatal(e)
				}
			}
			held, e := OpenACLDirectory(root)
			if e != nil {
				t.Fatal(e)
			}
			defer held.Close()
			if e = held.SetProtectedDACL(aclDescriptor(t, aclFinal)); e != nil {
				t.Fatal(e)
			}
			sd, e := held.SecurityDescriptor()
			if e != nil {
				t.Fatal(e)
			}
			aclAssertPolicy(t, sd, true)
			if aclNamed(t, child) != beforeChild || aclNamed(t, file) != beforeFile || aclNamed(t, protected) != beforeProtected {
				t.Fatal("candidate propagated to existing child or rewrote protected child")
			}
			if aclNamed(t, outside) != outsideBefore || aclNamed(t, sentinel) != sentinelBefore {
				t.Fatal("candidate propagated through child symlink to outside target")
			}
			if locked {
				if e = windows.CloseHandle(writer); e != nil {
					t.Fatal(e)
				}
				writer = windows.InvalidHandle
			}
			contents(t, file, "retained bytes")
			contents(t, sentinel, "outside unchanged")
			future := filepath.Join(root, "future child")
			if e = os.Mkdir(future, 0700); e != nil {
				t.Fatal(e)
			}
			futureSD, e := windows.GetNamedSecurityInfo(future, windows.SE_FILE_OBJECT, windows.DACL_SECURITY_INFORMATION)
			if e != nil {
				t.Fatal(e)
			}
			aclAssertPolicy(t, futureSD, false)
		})
	}
	t.Log("PASS: held root update changes only actual root with unlocked or locked children; protected/unprotected and symlink-outside ACLs/bytes stay unchanged; future children inherit")
}

func TestACLDirectoryPinsAndReparseRefusal(t *testing.T) {
	root, _, _ := aclFixture(t)
	renamed := root + " moved"
	if e := os.Rename(root, renamed); e != nil {
		t.Fatal("required rename positive control unavailable", e)
	}
	if e := os.Rename(renamed, root); e != nil {
		t.Fatal(e)
	}
	held, e := OpenACLDirectory(root)
	if e != nil {
		t.Fatal(e)
	}
	if e = os.Rename(root, renamed); e == nil {
		t.Fatal("held ACL root renamed")
	}
	parent := filepath.Dir(root)
	if e = os.Rename(parent, parent+" moved"); e == nil {
		t.Fatal("held ACL ancestor renamed")
	}
	if e = held.Close(); e != nil {
		t.Fatal(e)
	}
	if e = os.Rename(root, renamed); e != nil {
		t.Fatal("ACL close did not release pins", e)
	}
	if e = os.Rename(renamed, root); e != nil {
		t.Fatal(e)
	}
	outside := filepath.Join(t.TempDir(), "outside")
	if e = os.Mkdir(outside, 0700); e != nil {
		t.Fatal(e)
	}
	sentinel := filepath.Join(outside, "sentinel")
	if e = os.WriteFile(sentinel, []byte("outside unchanged"), 0600); e != nil {
		t.Fatal(e)
	}
	outsideACL, fileACL := aclNamed(t, outside), aclNamed(t, sentinel)
	alias := filepath.Join(filepath.Dir(root), "alias")
	if e = os.Symlink(outside, alias); e != nil {
		t.Fatal("required real symlink unavailable", e)
	}
	for _, path := range []string{alias, filepath.Join(alias, "child")} {
		if h, e := OpenACLDirectory(path); e == nil {
			h.Close()
			t.Fatal("ACL opener followed reparse component")
		}
	}
	if aclNamed(t, outside) != outsideACL || aclNamed(t, sentinel) != fileACL {
		t.Fatal("reparse refusal changed outside ACLs")
	}
	body, e := os.ReadFile(sentinel)
	if e != nil || string(body) != "outside unchanged" {
		t.Fatal("reparse refusal changed outside bytes", e)
	}
	t.Log("PASS: real root/ancestor rename refusal while held; actual symlink root/ancestor refusal preserves outside descriptors and bytes")
}

func TestACLDirectoryExistingWriterRefusedWithoutMutation(t *testing.T) {
	root, child, file := aclFixture(t)
	rootBefore, childBefore, fileBefore := aclNamed(t, root), aclNamed(t, child), aclNamed(t, file)
	u, e := windows.UTF16PtrFromString(root)
	if e != nil {
		t.Fatal(e)
	}
	writer, e := windows.CreateFile(u, windows.GENERIC_WRITE, windows.FILE_SHARE_READ|windows.FILE_SHARE_WRITE|windows.FILE_SHARE_DELETE, nil, windows.OPEN_EXISTING, windows.FILE_FLAG_BACKUP_SEMANTICS|windows.FILE_FLAG_OPEN_REPARSE_POINT, 0)
	if e != nil {
		t.Fatal("required directory writer positive control unavailable", e)
	}
	defer func() {
		if writer != windows.InvalidHandle {
			windows.CloseHandle(writer)
		}
	}()
	if held, e := OpenACLDirectory(root); e == nil {
		held.Close()
		t.Fatal("ACL pin admitted already-held directory writer")
	}
	if aclNamed(t, root) != rootBefore || aclNamed(t, child) != childBefore || aclNamed(t, file) != fileBefore {
		t.Fatal("sharing refusal changed descriptors")
	}
	contents(t, file, "retained bytes")
	if e = windows.CloseHandle(writer); e != nil {
		t.Fatal(e)
	}
	writer = windows.InvalidHandle
	held, e := OpenACLDirectory(root)
	if e != nil {
		t.Fatal("writer release did not restore ACL access", e)
	}
	if e = held.Close(); e != nil {
		t.Fatal(e)
	}
	t.Log("PASS: real existing directory writer forces fail-closed ACL sharing refusal with unchanged root/child descriptors and bytes; released writer permits open")
}
