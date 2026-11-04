//go:build windows

package installerfiles

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
	"unsafe"

	"golang.org/x/sys/windows"
)

const rootTestReader = "S-1-5-21-314159-271828-161803-1001"

func rootTestDescriptor(t *testing.T, path string) *windows.SECURITY_DESCRIPTOR {
	t.Helper()
	sd, e := windows.GetNamedSecurityInfo(path, windows.SE_FILE_OBJECT, windows.OWNER_SECURITY_INFORMATION|windows.DACL_SECURITY_INFORMATION)
	if e != nil {
		t.Fatal(e)
	}
	return sd
}

// Parse the actual resulting descriptor independently of production policy helpers.
func assertRootTestPolicy(t *testing.T, path string, protected, directory bool, reader string, service bool) {
	t.Helper()
	sd := rootTestDescriptor(t, path)
	control, _, e := sd.Control()
	if e != nil {
		t.Fatal(e)
	}
	if (control&windows.SE_DACL_PROTECTED != 0) != protected {
		t.Fatal("created object has unexpected inheritance protection")
	}
	acl, _, e := sd.DACL()
	if e != nil || acl == nil {
		t.Fatal("created object has no usable DACL")
	}
	want := map[string]uint32{"S-1-5-18": 0x1f01ff, "S-1-5-32-544": 0x1f01ff}
	if service {
		want["S-1-5-19"] = 0x1301bf
	}
	if reader != "" {
		if _, ok := want[reader]; !ok {
			want[reader] = 0x1200a9
		}
	}
	self, children := map[string]uint32{}, map[string]uint32{}
	for i := uint32(0); i < uint32(acl.AceCount); i++ {
		var ace *windows.ACCESS_ALLOWED_ACE
		if e = windows.GetAce(acl, i, &ace); e != nil {
			t.Fatal(e)
		}
		if ace.Header.AceType != windows.ACCESS_ALLOWED_ACE_TYPE {
			t.Fatal("created object has unsupported ACE")
		}
		sid := (*windows.SID)(unsafe.Pointer(&ace.SidStart)).String()
		rights, ok := want[sid]
		if !ok || uint32(ace.Mask)&^rights != 0 {
			t.Fatal("created object retained unexpected grant")
		}
		if ace.Header.AceFlags&8 == 0 {
			self[sid] |= uint32(ace.Mask)
		}
		if ace.Header.AceFlags&3 == 3 && ace.Header.AceFlags&12 == 0 {
			children[sid] |= uint32(ace.Mask)
		}
	}
	for sid, rights := range want {
		if self[sid] != rights || (directory && children[sid] != rights) {
			t.Fatal("created object lacks intended self/child rights")
		}
	}
}

func TestPrivateRootNestedCreationAndChildInheritance(t *testing.T) {
	for _, mode := range []struct {
		name, reader string
		service      bool
	}{{"admin-only", "", false}, {"service-reader", rootTestReader, true}, {"duplicate-admin", "S-1-5-32-544", true}, {"duplicate-service", "S-1-5-19", true}} {
		t.Run(mode.name, func(t *testing.T) {
			base := t.TempDir()
			setPrivacyDescriptor(t, base, "D:P(A;OICI;FA;;;SY)(A;OICI;FA;;;BA)(A;OICI;FA;;;WD)")
			parentBefore := rootTestDescriptor(t, base).String()
			first := filepath.Join(base, "new private é")
			root := filepath.Join(first, "nested state")
			if e := EnsurePrivateRoot(root, mode.reader, mode.service); e != nil {
				t.Fatal(e)
			}
			for _, p := range []string{first, root} {
				assertRootTestPolicy(t, p, true, true, mode.reader, mode.service)
			}
			if rootTestDescriptor(t, base).String() != parentBefore {
				t.Fatal("creation modified existing permissive parent")
			}
			child := filepath.Join(root, "inherited-child")
			if e := os.Mkdir(child, 0700); e != nil {
				t.Fatal(e)
			}
			file := filepath.Join(child, "inherited.txt")
			if e := os.WriteFile(file, []byte("child bytes"), 0600); e != nil {
				t.Fatal(e)
			}
			assertRootTestPolicy(t, child, false, true, mode.reader, mode.service)
			assertRootTestPolicy(t, file, false, false, mode.reader, mode.service)
		})
	}
	t.Log("PASS: nested private roots exclude permissive parent grants at creation and retain exact admin/service/reader child inheritance")
}

func TestPrivateRootExistingDirectoryIsNotACLRepair(t *testing.T) {
	root := t.TempDir()
	setPrivacyDescriptor(t, root, "D:P(A;OICI;FA;;;SY)(A;OICI;FA;;;BA)(A;OICI;FA;;;WD)")
	file := filepath.Join(root, "existing.txt")
	if e := os.WriteFile(file, []byte("existing bytes"), 0600); e != nil {
		t.Fatal(e)
	}
	before := rootTestDescriptor(t, root).String()
	childBefore := rootTestDescriptor(t, file).String()
	if e := EnsurePrivateRoot(root, rootTestReader, true); e != nil {
		t.Fatal(e)
	}
	if before != rootTestDescriptor(t, root).String() || childBefore != rootTestDescriptor(t, file).String() {
		t.Fatal("creation action silently normalized existing tree")
	}
	assertBytes(t, file, "existing bytes")
	t.Log("PASS: existing unsafe directory and child ACLs/bytes remain unchanged; creation action does not claim ACL repair")
}

func TestPrivateRootReparseRefusesOutsideChanges(t *testing.T) {
	for _, leaf := range []bool{true, false} {
		name := "ancestor"
		if leaf {
			name = "leaf"
		}
		t.Run(name, func(t *testing.T) {
			base := t.TempDir()
			outside := filepath.Join(base, "outside")
			if e := os.Mkdir(outside, 0700); e != nil {
				t.Fatal(e)
			}
			sentinel := filepath.Join(outside, "sentinel")
			if e := os.WriteFile(sentinel, []byte("outside untouched"), 0600); e != nil {
				t.Fatal(e)
			}
			before := rootTestDescriptor(t, outside).String()
			fileBefore := rootTestDescriptor(t, sentinel).String()
			alias := filepath.Join(base, "alias")
			if e := os.Symlink(outside, alias); e != nil {
				t.Fatal("required actual directory symlink control unavailable", e)
			}
			target := alias
			if !leaf {
				target = filepath.Join(alias, "must-not-create")
			}
			if e := EnsurePrivateRoot(target, rootTestReader, true); e == nil {
				t.Fatal("reparse directory accepted")
			}
			if rootTestDescriptor(t, outside).String() != before || rootTestDescriptor(t, sentinel).String() != fileBefore {
				t.Fatal("reparse refusal changed outside ACLs")
			}
			assertBytes(t, sentinel, "outside untouched")
			if _, e := os.Stat(filepath.Join(outside, "must-not-create")); !os.IsNotExist(e) {
				t.Fatal("creation traversed outside symlink")
			}
		})
	}
	t.Log("PASS: actual directory symlink at root or ancestor refuses without outside ACL/byte mutation")
}

func TestPrivateRootInvalidReaderAndNonDirectoryRefused(t *testing.T) {
	base := t.TempDir()
	target := filepath.Join(base, "invalid reader must not create")
	for _, reader := range []string{"not-a-sid", "S-1-5-21-1);D:(A;;FA;;;WD)", "S-1-5-"} {
		if e := EnsurePrivateRoot(target, reader, true); e == nil {
			t.Fatal("invalid reader accepted")
		}
		if _, e := os.Stat(target); !os.IsNotExist(e) {
			t.Fatal("invalid reader caused root creation")
		}
	}
	file := filepath.Join(base, "ordinary-file")
	body := []byte("retain ordinary file")
	if e := os.WriteFile(file, body, 0600); e != nil {
		t.Fatal(e)
	}
	before := rootTestDescriptor(t, file).String()
	for _, target := range []string{file, filepath.Join(file, "child")} {
		if e := EnsurePrivateRoot(target, "", false); e == nil {
			t.Fatal("non-directory component accepted")
		}
	}
	if !bytes.Equal(privacyBytes(t, file), body) || rootTestDescriptor(t, file).String() != before {
		t.Fatal("non-directory refusal changed existing file")
	}
	t.Log("PASS: invalid readers refuse before creation and ordinary-file components preserve existing bytes/ACLs")
}

func TestPrivateRootLaterRefusalLeavesOnlyPrivateProgress(t *testing.T) {
	base := t.TempDir()
	setPrivacyDescriptor(t, base, "D:P(A;OICI;FA;;;SY)(A;OICI;FA;;;BA)(A;OICI;FA;;;WD)")
	before := rootTestDescriptor(t, base).String()
	first := filepath.Join(base, "private partial")
	// A trailing-dot component is rejected by the component boundary. Implementations
	// may prevalidate it or discover it after creating the earlier private directory.
	if e := EnsurePrivateRoot(filepath.Join(first, "invalid."), rootTestReader, true); e == nil {
		t.Fatal("later unsafe component accepted")
	}
	if info, e := os.Stat(first); e == nil {
		if !info.IsDir() {
			t.Fatal("partial root is not directory")
		}
		assertRootTestPolicy(t, first, true, true, rootTestReader, true)
		entries, e := os.ReadDir(first)
		if e != nil || len(entries) != 0 {
			t.Fatal("later refusal left unexpected descendants")
		}
	} else if !os.IsNotExist(e) {
		t.Fatal(e)
	}
	if rootTestDescriptor(t, base).String() != before {
		t.Fatal("partial refusal changed existing ancestor")
	}
	t.Log("PASS: later unsafe component refuses; any retained creation progress remains private and existing ancestor is unchanged")
}
