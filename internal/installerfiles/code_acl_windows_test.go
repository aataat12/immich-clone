//go:build windows

package installerfiles

import (
	"bytes"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"unsafe"

	"golang.org/x/sys/windows"
)

func codeACLFixture(t *testing.T) (root, child, file string) {
	t.Helper()
	root = filepath.Join(t.TempDir(), "trusted code é")
	if e := os.Mkdir(root, 0700); e != nil {
		t.Fatal(e)
	}
	setPrivacyDescriptor(t, root, "D:P(A;OICI;FA;;;SY)(A;OICI;FA;;;BA)(A;OICI;0x1200a9;;;BU)(A;OICIIO;FA;;;CO)(A;OICI;0x1200a9;;;"+rootTestReader+")")
	child = filepath.Join(root, "provider")
	if e := os.Mkdir(child, 0700); e != nil {
		t.Fatal(e)
	}
	file = filepath.Join(child, "worker.bin")
	if e := os.WriteFile(file, []byte("retained code bytes"), 0600); e != nil {
		t.Fatal(e)
	}
	return
}
func codeACEBytes(t *testing.T, path string) [][]byte {
	t.Helper()
	acl, _, e := treeSD(t, path).DACL()
	if e != nil || acl == nil {
		t.Fatal("code fixture DACL missing", e)
	}
	var out [][]byte
	for i := uint32(0); i < uint32(acl.AceCount); i++ {
		var ace *windows.ACCESS_ALLOWED_ACE
		if e = windows.GetAce(acl, i, &ace); e != nil {
			t.Fatal(e)
		}
		out = append(out, append([]byte(nil), unsafe.Slice((*byte)(unsafe.Pointer(ace)), int(ace.Header.AceSize))...))
	}
	return out
}
func codePreservedACEs(t *testing.T, path string, before [][]byte) {
	t.Helper()
	after := codeACEBytes(t, path)
	used := make([]bool, len(after))
	for _, want := range before {
		found := false
		for i, got := range after {
			if !used[i] && bytes.Equal(got, want) {
				used[i] = true
				found = true
				break
			}
		}
		if !found {
			t.Fatal("existing safe code ACE bytes/flags were removed or rewritten")
		}
	}
}
func codeAssertServiceRX(t *testing.T, path string, directory, protected, inherited bool) {
	t.Helper()
	sd := treeSD(t, path)
	control, _, e := sd.Control()
	if e != nil || (control&windows.SE_DACL_PROTECTED != 0) != protected {
		t.Fatal("code protection changed", e)
	}
	acl, _, e := sd.DACL()
	if e != nil || acl == nil {
		t.Fatal("code result has no DACL", e)
	}
	found := false
	for i := uint32(0); i < uint32(acl.AceCount); i++ {
		var ace *windows.ACCESS_ALLOWED_ACE
		if e = windows.GetAce(acl, i, &ace); e != nil {
			t.Fatal(e)
		}
		if ace.Header.AceType != windows.ACCESS_ALLOWED_ACE_TYPE {
			t.Fatal("unsupported resulting code ACE")
		}
		sid := (*windows.SID)(unsafe.Pointer(&ace.SidStart)).String()
		if sid != "S-1-5-19" {
			continue
		}
		if uint32(ace.Mask)&0x500d0156 != 0 {
			t.Fatal("LocalService gained code mutation rights")
		}
		if ace.Header.AceFlags&8 == 0 && uint32(ace.Mask)&0x1200a9 == 0x1200a9 && (ace.Header.AceFlags&16 != 0) == inherited && (!directory || ace.Header.AceFlags&3 == 3) {
			found = true
		}
	}
	if !found {
		t.Fatal("intended LocalService read/execute grant missing or inheritance flags changed")
	}
}

func TestCodeReadGrantPreservesReadersAndProtectedRollback(t *testing.T) {
	root, child, file := codeACLFixture(t)
	rollback := filepath.Join(root, "rollback")
	nested := filepath.Join(rollback, "nested")
	if e := os.Mkdir(rollback, 0700); e != nil {
		t.Fatal(e)
	}
	setPrivacyDescriptor(t, rollback, "D:P(A;OICI;FA;;;SY)(A;OICI;FA;;;BA)")
	if e := os.Mkdir(nested, 0700); e != nil {
		t.Fatal(e)
	}
	secret := filepath.Join(rollback, "access-code.txt")
	catalog := filepath.Join(nested, "catalog.db")
	for _, p := range []string{secret, catalog} {
		if e := os.WriteFile(p, []byte("private rollback bytes"), 0600); e != nil {
			t.Fatal(e)
		}
	}
	boundaryBefore := map[string]string{}
	for _, p := range []string{rollback, nested, secret, catalog} {
		boundaryBefore[p] = treeSD(t, p).String()
	}
	owners := map[string]string{}
	before := map[string][][]byte{}
	for _, p := range []string{root, child, file} {
		owners[p] = treeOwner(t, p)
		before[p] = codeACEBytes(t, p)
	}
	if e := GrantCodeReadAccess(root); e != nil {
		t.Fatal(e)
	}
	codeAssertServiceRX(t, root, true, true, false)
	codeAssertServiceRX(t, child, true, false, true)
	codeAssertServiceRX(t, file, false, false, true)
	for _, p := range []string{root, child, file} {
		codePreservedACEs(t, p, before[p])
		if treeOwner(t, p) != owners[p] {
			t.Fatal("code grant changed owner")
		}
	}
	for p, want := range boundaryBefore {
		if treeSD(t, p).String() != want {
			t.Fatal("protected rollback boundary or descendant ACL changed")
		}
	}
	assertBytes(t, file, "retained code bytes")
	assertBytes(t, secret, "private rollback bytes")
	assertBytes(t, catalog, "private rollback bytes")
	future := filepath.Join(child, "future.bin")
	if e := os.WriteFile(future, []byte("future"), 0600); e != nil {
		t.Fatal(e)
	}
	codeAssertServiceRX(t, future, false, false, true)
	t.Log("PASS: code grant preserves safe reader/creator-owner ACE bytes, owners and protection; LocalService RX follows unprotected inheritance only; protected rollback subtree stays private and byte-identical")
}

func TestCodeReadGrantUnsafeMutationRefusesBeforeWrites(t *testing.T) {
	for _, kind := range []string{"ordinary-writer", "service-writer", "inherit-only-writer", "deny"} {
		t.Run(kind, func(t *testing.T) {
			root, child, file := codeACLFixture(t)
			bad := filepath.Join(child, "z-unsafe.bin")
			if e := os.WriteFile(bad, []byte("unsafe code bytes"), 0600); e != nil {
				t.Fatal(e)
			}
			switch kind {
			case "ordinary-writer":
				setPrivacyDescriptor(t, bad, "D:P(A;;FA;;;SY)(A;;FA;;;BA)(A;;GW;;;BU)")
			case "service-writer":
				setPrivacyDescriptor(t, bad, "D:P(A;;FA;;;SY)(A;;FA;;;BA)(A;;GW;;;LS)")
			case "inherit-only-writer":
				setPrivacyDescriptor(t, bad, "D:P(A;;FA;;;SY)(A;;FA;;;BA)(A;OIIO;GW;;;BU)")
			case "deny":
				setPrivacyDescriptor(t, bad, "D:P(A;;FA;;;SY)(A;;FA;;;BA)(D;;GW;;;BU)")
			}
			before := map[string]string{}
			for _, p := range []string{root, child, file, bad} {
				before[p] = treeSD(t, p).String()
			}
			if e := GrantCodeReadAccess(root); e == nil {
				t.Fatal("unsafe code mutation/unsupported ACE accepted")
			}
			for p, want := range before {
				if treeSD(t, p).String() != want {
					t.Fatal("unsafe code refusal changed an earlier or unsafe ACL")
				}
			}
			assertBytes(t, file, "retained code bytes")
		})
	}
	t.Log("PASS: ordinary/LocalService/inherit-only code writers and unsupported deny rules refuse before any code ACL mutation")
}

func TestCodeReadGrantAliasesAndLateNamesPreserveOutside(t *testing.T) {
	treeEnableLinks(t)
	for _, kind := range []string{"directory-link", "file-link", "hardlink", "late-file", "late-link"} {
		t.Run(kind, func(t *testing.T) {
			root, child, file := codeACLFixture(t)
			outside := t.TempDir()
			sentinel := filepath.Join(outside, "sentinel")
			if e := os.WriteFile(sentinel, []byte("outside unchanged"), 0600); e != nil {
				t.Fatal(e)
			}
			alias := filepath.Join(child, "z-alias")
			publish := func() error {
				switch kind {
				case "directory-link", "late-link":
					return os.Symlink(outside, alias)
				case "file-link":
					return os.Symlink(sentinel, alias)
				case "hardlink":
					return os.Link(sentinel, alias)
				default:
					return os.WriteFile(alias, []byte("late"), 0600)
				}
			}
			if kind != "late-file" && kind != "late-link" {
				if e := publish(); e != nil {
					t.Fatal("required actual alias fixture unavailable", e)
				}
			}
			before := map[string]string{}
			for _, p := range []string{root, child, file, outside, sentinel} {
				before[p] = treeSD(t, p).String()
			}
			var e error
			if kind == "late-file" || kind == "late-link" {
				called := false
				e = grantCodeReadAccess(root, func() error { called = true; return publish() })
				if !called {
					t.Fatal("late-publication hook not reached", e)
				}
			} else {
				e = GrantCodeReadAccess(root)
			}
			if e == nil {
				t.Fatal("code grant accepted alias or late namespace")
			}
			for p, want := range before {
				if treeSD(t, p).String() != want {
					t.Fatal("code alias/late-name refusal changed earlier or outside ACL")
				}
			}
			assertBytes(t, file, "retained code bytes")
			assertBytes(t, sentinel, "outside unchanged")
		})
	}
	t.Log("PASS: actual directory/file links, hard links and deterministic late child publication refuse before code ACL writes; outside and earlier descriptors/bytes preserved")
}

func TestCodeReadGrantRunningOwnImage(t *testing.T) {
	if os.Getenv("IMMICH_CODE_ACL_SELF_IMAGE") == "1" {
		if e := GrantCodeReadAccess(os.Getenv("IMMICH_CODE_ACL_SELF_ROOT")); e != nil {
			t.Fatal("running image code ACL operation failed", e)
		}
		return
	}
	root, _, _ := codeACLFixture(t)
	self, e := os.Executable()
	if e != nil {
		t.Fatal(e)
	}
	src, e := os.Open(self)
	if e != nil {
		t.Fatal(e)
	}
	defer src.Close()
	target := filepath.Join(root, "running-code-test.exe")
	dst, e := os.Create(target)
	if e != nil {
		t.Fatal(e)
	}
	if _, e = io.Copy(dst, src); e != nil {
		dst.Close()
		t.Fatal(e)
	}
	if e = dst.Close(); e != nil {
		t.Fatal(e)
	}
	before, e := os.ReadFile(target)
	if e != nil {
		t.Fatal(e)
	}
	process := exec.Command(target, "-test.run=^TestCodeReadGrantRunningOwnImage$", "-test.v")
	process.Env = append(os.Environ(), "IMMICH_CODE_ACL_SELF_IMAGE=1", "IMMICH_CODE_ACL_SELF_ROOT="+root)
	output, e := process.CombinedOutput()
	if e != nil {
		t.Fatal("actual running image control failed", e, string(output))
	}
	after, e := os.ReadFile(target)
	if e != nil || !bytes.Equal(before, after) {
		t.Fatal("running image ACL grant changed executable bytes", e)
	}
	codeAssertServiceRX(t, target, false, false, true)
	t.Log("PASS: actual running copy of current Windows test executable successfully grants code read access including its own image, preserving all executable bytes")
}
