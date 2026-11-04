//go:build windows

package installerfiles

import (
	"bytes"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"unsafe"

	"golang.org/x/sys/windows"
	"immichclone/internal/winfs"
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
			if kind == "inherit-only-writer" {
				bad = filepath.Join(child, "z-unsafe-directory")
				if e := os.Mkdir(bad, 0700); e != nil {
					t.Fatal(e)
				}
			} else if e := os.WriteFile(bad, []byte("unsafe code bytes"), 0600); e != nil {
				t.Fatal(e)
			}
			switch kind {
			case "ordinary-writer":
				setPrivacyDescriptor(t, bad, "D:P(A;;FA;;;SY)(A;;FA;;;BA)(A;;GW;;;BU)")
			case "service-writer":
				setPrivacyDescriptor(t, bad, "D:P(A;;FA;;;SY)(A;;FA;;;BA)(A;;GW;;;LS)")
			case "inherit-only-writer":
				// Inherit-only mutation is meaningful on a directory. Windows
				// may discard an inapplicable IO rule on a regular file, so
				// require the actual hostile grant before invoking candidate.
				setPrivacyDescriptor(t, bad, "D:P(A;OICI;FA;;;SY)(A;OICI;FA;;;BA)(A;OICIIO;GW;;;BU)")
				acl, _, e := treeSD(t, bad).DACL()
				if e != nil || acl == nil {
					t.Fatal("inherit-only directory fixture DACL missing", e)
				}
				found := false
				for i := uint32(0); i < uint32(acl.AceCount); i++ {
					var ace *windows.ACCESS_ALLOWED_ACE
					if e = windows.GetAce(acl, i, &ace); e != nil {
						t.Fatal(e)
					}
					if ace.Header.AceType == windows.ACCESS_ALLOWED_ACE_TYPE && ace.Header.AceFlags&8 != 0 && ace.Header.AceFlags&3 == 3 && (*windows.SID)(unsafe.Pointer(&ace.SidStart)).String() == "S-1-5-32-545" && uint32(ace.Mask)&0x500d0156 != 0 {
						found = true
					}
				}
				if !found {
					t.Fatal("required actual inherit-only untrusted mutation grant not established")
				}
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

// The real Program Files code root inherits its parent rather than starting with
// a protected fixture DACL. Log numeric ACL shape only if that distinct case fails.
func codeACLShape(t *testing.T, label string, sd *windows.SECURITY_DESCRIPTOR) {
	t.Helper()
	control, _, e := sd.Control()
	if e != nil {
		t.Log(label, "descriptor-control-unavailable")
		return
	}
	acl, _, e := sd.DACL()
	if e != nil || acl == nil {
		t.Log(label, "dacl-unavailable")
		return
	}
	t.Logf("%s control=%04x ace_count=%d", label, uint32(control), acl.AceCount)
	for i := uint32(0); i < uint32(acl.AceCount); i++ {
		var ace *windows.ACCESS_ALLOWED_ACE
		if e = windows.GetAce(acl, i, &ace); e != nil {
			t.Log(label, "ace-unavailable")
			return
		}
		t.Logf("%s ace=%d type=%d flags=%02x rights=%08x", label, i, ace.Header.AceType, ace.Header.AceFlags, uint32(ace.Mask))
	}
}

func TestCodeReadGrantUnprotectedInheritedRoot(t *testing.T) {
	parent := t.TempDir()
	setPrivacyDescriptor(t, parent, "D:P(A;OICI;FA;;;SY)(A;OICI;FA;;;BA)(A;OICI;FA;;;S-1-5-80-956008885-3418522649-1831038044-1853292631-2271478464)(A;OICI;0x1200a9;;;BU)(A;OICIIO;FA;;;CO)")
	root := filepath.Join(parent, "unprotected code root é")
	child := filepath.Join(root, "provider")
	for _, p := range []string{root, child} {
		if e := os.Mkdir(p, 0700); e != nil {
			t.Fatal(e)
		}
	}
	file := filepath.Join(child, "worker.bin")
	if e := os.WriteFile(file, []byte("inherited code bytes"), 0600); e != nil {
		t.Fatal(e)
	}
	beforeSD := treeSD(t, root)
	control, _, e := beforeSD.Control()
	if e != nil || control&windows.SE_DACL_PROTECTED != 0 {
		t.Fatal("unprotected-root fixture was not established", e)
	}
	beforeACL, _, e := beforeSD.DACL()
	if e != nil || beforeACL == nil {
		t.Fatal("unprotected fixture DACL unavailable", e)
	}
	inherited := false
	for i := uint32(0); i < uint32(beforeACL.AceCount); i++ {
		var ace *windows.ACCESS_ALLOWED_ACE
		if e = windows.GetAce(beforeACL, i, &ace); e != nil {
			t.Fatal(e)
		}
		if ace.Header.AceFlags&16 != 0 {
			inherited = true
		}
	}
	if !inherited {
		t.Fatal("unprotected-root fixture lacks actual inherited ACEs")
	}
	owners := map[string]string{}
	before := map[string][][]byte{}
	for _, p := range []string{root, child, file} {
		owners[p] = treeOwner(t, p)
		before[p] = codeACEBytes(t, p)
	}
	if e = GrantCodeReadAccess(root); e != nil {
		codeACLShape(t, "root-before", beforeSD)
		codeACLShape(t, "root-after", treeSD(t, root))
		codeACLShape(t, "directory-after", treeSD(t, child))
		codeACLShape(t, "file-after", treeSD(t, file))
		t.Fatal("unprotected inherited code root grant failed", e)
	}
	codeAssertServiceRX(t, root, true, false, false)
	codeAssertServiceRX(t, child, true, false, true)
	codeAssertServiceRX(t, file, false, false, true)
	for _, p := range []string{root, child, file} {
		codePreservedACEs(t, p, before[p])
		if treeOwner(t, p) != owners[p] {
			t.Fatal("unprotected code grant changed owner")
		}
	}
	assertBytes(t, file, "inherited code bytes")
	t.Log("PASS: unprotected code root with real inherited trusted-writer/reader/creator-owner policy preserves inherited ACE bytes, owners and unprotected flags while granting only intended LocalService RX")
}

func TestCodeACLFilePinsAndWriterRefusal(t *testing.T) {
	root, child, file := codeACLFixture(t)
	moved := file + ".moved"
	if e := os.Rename(file, moved); e != nil {
		t.Fatal("required file rename positive control failed", e)
	}
	if e := os.Rename(moved, file); e != nil {
		t.Fatal(e)
	}
	alias := filepath.Join(t.TempDir(), "new-hardlink")
	if e := os.Link(file, alias); e != nil {
		t.Fatal("required hard-link publication positive control failed", e)
	}
	originalInfo, e := os.Stat(file)
	if e != nil {
		t.Fatal(e)
	}
	aliasInfo, e := os.Stat(alias)
	if e != nil || !os.SameFile(originalInfo, aliasInfo) {
		t.Fatal("positive hard-link control did not select the same object", e)
	}
	if e = os.Remove(alias); e != nil {
		t.Fatal(e)
	}
	name, e := windows.UTF16PtrFromString(file)
	if e != nil {
		t.Fatal(e)
	}
	openWriter := func() (windows.Handle, error) {
		return windows.CreateFile(name, windows.GENERIC_WRITE, windows.FILE_SHARE_READ|windows.FILE_SHARE_WRITE|windows.FILE_SHARE_DELETE, nil, windows.OPEN_EXISTING, 0, 0)
	}
	writer, e := openWriter()
	if e != nil {
		t.Fatal("required data-write open positive control failed", e)
	}
	if e = windows.CloseHandle(writer); e != nil {
		t.Fatal(e)
	}
	pinBefore := map[string]string{}
	for _, p := range []string{root, child, file} {
		pinBefore[p] = treeSD(t, p).String()
	}
	held, e := winfs.OpenCodeACLTree(root)
	if e != nil {
		t.Fatal(e)
	}
	defer held.Close()
	if e = os.Rename(file, moved); e == nil {
		_ = held.Close()
		_ = os.Rename(moved, file)
		t.Fatal("held code file renamed")
	}
	if h, e := openWriter(); e == nil {
		windows.CloseHandle(h)
		t.Fatal("held code file accepted a data-write handle")
	} else if !errors.Is(e, windows.ERROR_SHARING_VIOLATION) {
		t.Fatal("code write refusal did not establish actual sharing conflict", e)
	}
	// Windows permits publishing another name for the SAME held object. That
	// is not replacement by a preexisting outside object, but the changed link
	// count must make validation refuse before an ACL operation is accepted.
	if e = os.Link(file, alias); e != nil {
		t.Fatal("required outward hard-link publication while held failed", e)
	}
	aliasInfo, e = os.Stat(alias)
	if e != nil || !os.SameFile(originalInfo, aliasInfo) {
		t.Fatal("outward hard link did not select held object", e)
	}
	if e = held.Validate(); !errors.Is(e, winfs.ErrUnsafeObject) {
		t.Fatal("validation did not refuse actual held-object link-count change", e)
	}
	for p, want := range pinBefore {
		if treeSD(t, p).String() != want {
			t.Fatal("outward-link validation changed a code ACL")
		}
	}
	if treeSD(t, alias).String() != pinBefore[file] {
		t.Fatal("outward alias descriptor differs from unchanged held object")
	}
	assertBytes(t, file, "retained code bytes")
	assertBytes(t, alias, "retained code bytes")
	if e = held.Close(); e != nil {
		t.Fatal(e)
	}
	// Delete sharing may prevent unlink while held. Remove after release, then
	// require a fresh acquisition and validation to recover with one link again.
	if e = os.Remove(alias); e != nil {
		t.Fatal(e)
	}
	recovered, e := winfs.OpenCodeACLTree(root)
	if e != nil {
		t.Fatal("removed outward alias should permit acquisition", e)
	}
	if e = recovered.Validate(); e != nil {
		recovered.Close()
		t.Fatal("removed outward alias should restore validation", e)
	}
	if e = recovered.Close(); e != nil {
		t.Fatal(e)
	}
	if e = os.Rename(file, moved); e != nil {
		t.Fatal("released code tree failed to release file rename pin", e)
	}
	if e = os.Rename(moved, file); e != nil {
		t.Fatal(e)
	}
	before := map[string]string{}
	for _, p := range []string{root, child, file} {
		before[p] = treeSD(t, p).String()
	}
	writer, e = openWriter()
	if e != nil {
		t.Fatal("existing writer control failed", e)
	}
	blocked, openErr := winfs.OpenCodeACLTree(root)
	if blocked != nil {
		blocked.Close()
	}
	closeErr := windows.CloseHandle(writer)
	if closeErr != nil {
		t.Fatal(closeErr)
	}
	if openErr == nil {
		t.Fatal("code tree accepted a preexisting data-write handle")
	}
	if !errors.Is(openErr, windows.STATUS_SHARING_VIOLATION) && !errors.Is(openErr, windows.ERROR_SHARING_VIOLATION) {
		t.Fatal("existing code writer refusal did not establish sharing conflict", openErr)
	}
	for p, want := range before {
		if treeSD(t, p).String() != want {
			t.Fatal("existing writer refusal changed a code descriptor")
		}
	}
	assertBytes(t, file, "retained code bytes")
	retry, e := winfs.OpenCodeACLTree(root)
	if e != nil {
		t.Fatal("released code writer should permit tree acquisition", e)
	}
	if e = retry.Close(); e != nil {
		t.Fatal(e)
	}
	t.Log("PASS: held code files refuse rename and data-write opens; actual outgoing hard-link publication changes link count and is refused by validation without ACL/byte changes; removal/reacquisition recovers; existing writer forces sharing refusal and release permits retry")
}

func TestCodeReadGrantLateOutgoingAliasRefusedBeforeWrite(t *testing.T) {
	root, child, file := codeACLFixture(t)
	outside := t.TempDir()
	alias := filepath.Join(outside, "new-outgoing-alias")
	sentinel := filepath.Join(outside, "preexisting-sentinel")
	if e := os.WriteFile(sentinel, []byte("preexisting outside bytes"), 0600); e != nil {
		t.Fatal(e)
	}
	if _, e := os.Stat(alias); !os.IsNotExist(e) {
		t.Fatal("outgoing alias fixture must start absent", e)
	}
	before := map[string]string{}
	for _, p := range []string{root, child, file, outside, sentinel} {
		before[p] = treeSD(t, p).String()
	}
	called, published := false, false
	e := grantCodeReadAccess(root, func() error {
		called = true
		if e := os.Link(file, alias); e != nil {
			return e
		}
		published = true
		return nil
	})
	if !called || !published {
		t.Fatal("required outgoing alias publication after planning was not exercised", e)
	}
	if !errors.Is(e, winfs.ErrUnsafeObject) {
		t.Fatal("late outgoing alias was not refused by actual link-count validation", e)
	}
	sourceInfo, e := os.Stat(file)
	if e != nil {
		t.Fatal(e)
	}
	aliasInfo, e := os.Stat(alias)
	if e != nil || !os.SameFile(sourceInfo, aliasInfo) {
		t.Fatal("new alias must select exactly the held managed object", e)
	}
	for p, want := range before {
		if treeSD(t, p).String() != want {
			t.Fatal("late outgoing alias refusal changed any planned or preexisting outside ACL")
		}
	}
	if treeSD(t, alias).String() != before[file] {
		t.Fatal("new outgoing alias did not retain exact original managed-object descriptor")
	}
	assertBytes(t, file, "retained code bytes")
	assertBytes(t, alias, "retained code bytes")
	assertBytes(t, sentinel, "preexisting outside bytes")
	if e = os.Remove(alias); e != nil {
		t.Fatal(e)
	}
	if e = GrantCodeReadAccess(root); e != nil {
		t.Fatal("removing late outgoing alias must permit ordinary grant retry", e)
	}
	assertBytes(t, sentinel, "preexisting outside bytes")
	if treeSD(t, sentinel).String() != before[sentinel] {
		t.Fatal("retry changed preexisting outside object descriptor")
	}
	t.Log("PASS: real outgoing alias created after code-grant planning refers to the held managed object; link-count validation refuses before every ACL write, preserves preexisting outside/managed ACLs and bytes, and removal permits retry")
}
