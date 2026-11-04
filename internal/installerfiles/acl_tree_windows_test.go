//go:build windows

package installerfiles

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"unsafe"

	"golang.org/x/sys/windows"
)

func treeACLFixture(t *testing.T) (root, child, file string) {
	t.Helper()
	root = filepath.Join(t.TempDir(), "normalize é")
	if e := os.Mkdir(root, 0700); e != nil {
		t.Fatal(e)
	}
	setPrivacyDescriptor(t, root, "D:P(A;OICI;FA;;;SY)(A;OICI;FA;;;BA)(A;OICI;FA;;;WD)")
	child = filepath.Join(root, "nested")
	if e := os.Mkdir(child, 0700); e != nil {
		t.Fatal(e)
	}
	file = filepath.Join(child, "a-safe.txt")
	if e := os.WriteFile(file, []byte("retained original bytes"), 0600); e != nil {
		t.Fatal(e)
	}
	return
}
func treeSD(t *testing.T, path string) *windows.SECURITY_DESCRIPTOR {
	t.Helper()
	sd, e := windows.GetNamedSecurityInfo(path, windows.SE_FILE_OBJECT, windows.OWNER_SECURITY_INFORMATION|windows.DACL_SECURITY_INFORMATION)
	if e != nil {
		t.Fatal(e)
	}
	return sd
}
func treeSetUnprotected(t *testing.T, path, text string) {
	t.Helper()
	sd, e := windows.SecurityDescriptorFromString(text)
	if e != nil {
		t.Fatal(e)
	}
	acl, _, e := sd.DACL()
	if e != nil {
		t.Fatal(e)
	}
	if e = windows.SetNamedSecurityInfo(path, windows.SE_FILE_OBJECT, windows.DACL_SECURITY_INFORMATION|windows.UNPROTECTED_DACL_SECURITY_INFORMATION, nil, nil, acl, nil); e != nil {
		t.Fatal(e)
	}
}
func treeExplicit(t *testing.T, path string) [][]byte {
	t.Helper()
	acl, _, e := treeSD(t, path).DACL()
	if e != nil || acl == nil {
		t.Fatal("fixture DACL unavailable", e)
	}
	var out [][]byte
	for i := uint32(0); i < uint32(acl.AceCount); i++ {
		var ace *windows.ACCESS_ALLOWED_ACE
		if e = windows.GetAce(acl, i, &ace); e != nil {
			t.Fatal(e)
		}
		if ace.Header.AceFlags&16 == 0 {
			out = append(out, append([]byte(nil), unsafe.Slice((*byte)(unsafe.Pointer(ace)), int(ace.Header.AceSize))...))
		}
	}
	return out
}
func treeAssertExplicit(t *testing.T, path string, want [][]byte) {
	t.Helper()
	got := treeExplicit(t, path)
	if len(got) != len(want) {
		t.Fatal("explicit ACE count changed")
	}
	for i := range got {
		if !bytes.Equal(got[i], want[i]) {
			t.Fatal("explicit ACE bytes/flags changed")
		}
	}
}
func treeOwner(t *testing.T, path string) string {
	t.Helper()
	owner, _, e := treeSD(t, path).Owner()
	if e != nil || owner == nil {
		t.Fatal("fixture owner missing", e)
	}
	return owner.String()
}
func treeEnableLinks(t *testing.T) {
	t.Helper()
	var token windows.Token
	if e := windows.OpenProcessToken(windows.CurrentProcess(), windows.TOKEN_ADJUST_PRIVILEGES|windows.TOKEN_QUERY, &token); e != nil {
		t.Fatal(e)
	}
	defer token.Close()
	name, e := windows.UTF16PtrFromString("SeCreateSymbolicLinkPrivilege")
	if e != nil {
		t.Fatal(e)
	}
	var p windows.Tokenprivileges
	p.PrivilegeCount = 1
	if e = windows.LookupPrivilegeValue(nil, name, &p.Privileges[0].Luid); e != nil {
		t.Fatal(e)
	}
	p.Privileges[0].Attributes = windows.SE_PRIVILEGE_ENABLED
	if e = windows.AdjustTokenPrivileges(token, false, &p, 0, nil, nil); e != nil {
		t.Fatal(e)
	}
}
func treeAssertPolicy(t *testing.T, path string, directory, protected bool, reader string, service bool) {
	t.Helper()
	sd := treeSD(t, path)
	control, _, e := sd.Control()
	if e != nil || (control&windows.SE_DACL_PROTECTED != 0) != protected {
		t.Fatal("inheritance protection changed", e)
	}
	acl, _, e := sd.DACL()
	if e != nil || acl == nil {
		t.Fatal("actual DACL unavailable", e)
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
			t.Fatal("unsupported actual ACE")
		}
		sid := (*windows.SID)(unsafe.Pointer(&ace.SidStart)).String()
		rights, ok := want[sid]
		if !ok || uint32(ace.Mask)&^rights != 0 {
			t.Fatal("unsafe actual principal/rights remain")
		}
		if ace.Header.AceFlags&8 == 0 {
			self[sid] |= uint32(ace.Mask)
		}
		if ace.Header.AceFlags&3 == 3 && ace.Header.AceFlags&12 == 0 {
			children[sid] |= uint32(ace.Mask)
		}
	}
	for sid, rights := range want {
		if self[sid]&rights != rights || (directory && children[sid]&rights != rights) {
			t.Fatal("required effective or child inheritance rights missing")
		}
	}
}

func TestNormalizeACLInheritedRightsPreservesExplicitProtectedAndBytes(t *testing.T) {
	for _, mode := range []struct {
		name, reader string
		service      bool
	}{{"admin", "", false}, {"reader-service", rootTestReader, true}} {
		t.Run(mode.name, func(t *testing.T) {
			root, child, file := treeACLFixture(t)
			treeSetUnprotected(t, child, "D:(A;OICINP;0x1;;;SY)")
			childExplicit := treeExplicit(t, child)
			if len(childExplicit) != 1 {
				t.Fatal("safe explicit inheritance-flag control not established")
			}
			treeSetUnprotected(t, file, "D:(A;;0x1;;;SY)")
			explicit := treeExplicit(t, file)
			if len(explicit) != 1 {
				t.Fatal("safe explicit positive control not established")
			}
			protected := filepath.Join(root, "protected")
			if e := os.Mkdir(protected, 0700); e != nil {
				t.Fatal(e)
			}
			policy := "D:P(A;OICI;FA;;;SY)(A;OICI;FA;;;BA)"
			if mode.service {
				policy += "(A;OICI;0x1301bf;;;LS)"
			}
			if mode.reader != "" {
				policy += fmt.Sprintf("(A;OICI;0x1200a9;;;%s)", mode.reader)
			}
			setPrivacyDescriptor(t, protected, policy)
			protectedBefore := treeSD(t, protected).String()
			owners := map[string]string{}
			for _, p := range []string{root, child, file, protected} {
				owners[p] = treeOwner(t, p)
			}
			if e := NormalizePrivateTree(root, mode.reader, mode.service); e != nil {
				t.Fatal(e)
			}
			treeAssertPolicy(t, root, true, true, mode.reader, mode.service)
			treeAssertPolicy(t, child, true, false, mode.reader, mode.service)
			treeAssertPolicy(t, file, false, false, mode.reader, mode.service)
			if treeSD(t, protected).String() != protectedBefore {
				t.Fatal("protected complete child descriptor rewritten")
			}
			treeAssertExplicit(t, file, explicit)
			treeAssertExplicit(t, child, childExplicit)
			for p, owner := range owners {
				if treeOwner(t, p) != owner {
					t.Fatal("normalization changed owner")
				}
			}
			assertBytes(t, file, "retained original bytes")
			future := filepath.Join(child, "future")
			if e := os.Mkdir(future, 0700); e != nil {
				t.Fatal(e)
			}
			treeAssertPolicy(t, future, true, false, mode.reader, mode.service)
		})
	}
	t.Log("PASS: inherited unsafe grants normalize to required admin/service/reader policy without changing explicit ACEs, protected child descriptors, owners or original bytes; future descendants inherit")
}

func TestNormalizeACLUnsafeDescendantRefusesBeforeAnyMutation(t *testing.T) {
	for _, kind := range []string{"explicit-read", "explicit-deny", "protected-read", "protected-incomplete"} {
		t.Run(kind, func(t *testing.T) {
			root, child, file := treeACLFixture(t)
			bad := filepath.Join(child, "z-unsafe.txt")
			if e := os.WriteFile(bad, []byte("unsafe fixture bytes"), 0600); e != nil {
				t.Fatal(e)
			}
			switch kind {
			case "explicit-read":
				treeSetUnprotected(t, bad, "D:(A;;GR;;;WD)")
			case "explicit-deny":
				treeSetUnprotected(t, bad, "D:(D;;GW;;;WD)")
			case "protected-read":
				setPrivacyDescriptor(t, bad, "D:P(A;;FA;;;SY)(A;;FA;;;BA)(A;;GR;;;WD)")
			case "protected-incomplete":
				setPrivacyDescriptor(t, bad, "D:P(A;;FA;;;BA)")
			}
			before := map[string]string{}
			for _, p := range []string{root, child, file, bad} {
				before[p] = treeSD(t, p).String()
			}
			if e := NormalizePrivateTree(root, "", false); e == nil {
				t.Fatal("unsafe actual descendant accepted")
			}
			for p, want := range before {
				if treeSD(t, p).String() != want {
					t.Fatal("unsafe descendant refusal changed an earlier or unsafe ACL")
				}
			}
			assertBytes(t, file, "retained original bytes")
			assertBytes(t, bad, "unsafe fixture bytes")
		})
	}
	t.Log("PASS: unsafe explicit/protected descendant policies refuse before root or earlier-child ACL mutation and preserve all bytes")
}

func TestNormalizeACLReparseAndHardlinkOutsidePreserved(t *testing.T) {
	treeEnableLinks(t)
	for _, kind := range []string{"directory-link", "file-link", "hardlink"} {
		t.Run(kind, func(t *testing.T) {
			root, _, file := treeACLFixture(t)
			outside := t.TempDir()
			sentinel := filepath.Join(outside, "sentinel")
			if e := os.WriteFile(sentinel, []byte("outside unchanged"), 0600); e != nil {
				t.Fatal(e)
			}
			alias := filepath.Join(root, "z-outside")
			var e error
			switch kind {
			case "directory-link":
				e = os.Symlink(outside, alias)
			case "file-link":
				e = os.Symlink(sentinel, alias)
			case "hardlink":
				e = os.Link(sentinel, alias)
			}
			if e != nil {
				t.Fatal("required actual alias fixture unavailable", e)
			}
			rootBefore, outsideBefore, fileBefore := treeSD(t, root).String(), treeSD(t, outside).String(), treeSD(t, sentinel).String()
			if e = NormalizePrivateTree(root, "", false); e == nil {
				t.Fatal("unsafe outside alias accepted")
			}
			if treeSD(t, root).String() != rootBefore || treeSD(t, outside).String() != outsideBefore || treeSD(t, sentinel).String() != fileBefore {
				t.Fatal("alias refusal changed managed/outside descriptors")
			}
			assertBytes(t, sentinel, "outside unchanged")
			assertBytes(t, file, "retained original bytes")
		})
	}
	t.Log("PASS: real directory/file reparse and hard-link descendants refuse before ACL writes; outside and managed descriptors/bytes remain unchanged")
}

func TestNormalizeACLLateNamespaceChangeRefusedBeforeWrite(t *testing.T) {
	treeEnableLinks(t)
	for _, kind := range []string{"file", "directory-link"} {
		t.Run(kind, func(t *testing.T) {
			root, child, file := treeACLFixture(t)
			outside := t.TempDir()
			sentinel := filepath.Join(outside, "sentinel")
			if e := os.WriteFile(sentinel, []byte("outside unchanged"), 0600); e != nil {
				t.Fatal(e)
			}
			before := map[string]string{}
			for _, p := range []string{root, child, file, outside, sentinel} {
				before[p] = treeSD(t, p).String()
			}
			called := false
			e := normalizePrivateTree(root, "", false, func() error {
				called = true
				late := filepath.Join(child, "late")
				if kind == "file" {
					return os.WriteFile(late, []byte("late unsafe owner fixture"), 0600)
				}
				return os.Symlink(outside, late)
			})
			if !called || e == nil {
				t.Fatal("required late namespace change/refusal not exercised", e)
			}
			for p, want := range before {
				if treeSD(t, p).String() != want {
					t.Fatal("late namespace refusal changed an earlier or outside ACL")
				}
			}
			assertBytes(t, file, "retained original bytes")
			assertBytes(t, sentinel, "outside unchanged")
		})
	}
	t.Log("PASS: deterministic late child/file-link publication between planning and apply refuses before root/earlier ACL writes and preserves outside targets")
}

func TestNormalizeACLExistingExclusiveWriterRefusedBeforeMutation(t *testing.T) {
	root, child, file := treeACLFixture(t)
	beforeRoot, beforeChild, beforeFile := treeSD(t, root).String(), treeSD(t, child).String(), treeSD(t, file).String()
	u, e := windows.UTF16PtrFromString(file)
	if e != nil {
		t.Fatal(e)
	}
	h, e := windows.CreateFile(u, windows.GENERIC_READ|windows.GENERIC_WRITE, 0, nil, windows.OPEN_EXISTING, 0, 0)
	if e != nil {
		t.Fatal(e)
	}
	e = NormalizePrivateTree(root, "", false)
	if closeErr := windows.CloseHandle(h); closeErr != nil {
		t.Fatal(closeErr)
	}
	if e == nil {
		t.Fatal("active exclusive writer accepted")
	}
	if treeSD(t, root).String() != beforeRoot || treeSD(t, child).String() != beforeChild || treeSD(t, file).String() != beforeFile {
		t.Fatal("sharing refusal changed ACLs")
	}
	assertBytes(t, file, "retained original bytes")
	if e = NormalizePrivateTree(root, "", false); e != nil {
		t.Fatal("released writer should permit normal normalization", e)
	}
	t.Log("PASS: actual exclusive data writer blocks normalization without descriptor/byte mutation; released writer allows normalization")
}
