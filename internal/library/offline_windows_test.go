//go:build windows

package library

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"unsafe"

	bolt "go.etcd.io/bbolt"
	"golang.org/x/sys/windows"
)

type offlineObserved struct {
	Descriptor string
	Hash       string
	Link       string
	Directory  bool
}

func offlineSnapshot(t *testing.T, root string) map[string]offlineObserved {
	t.Helper()
	out := map[string]offlineObserved{}
	e := filepath.WalkDir(root, func(path string, d fs.DirEntry, e error) error {
		if e != nil {
			return e
		}
		rel, e := filepath.Rel(root, path)
		if e != nil {
			return e
		}
		u, e := windows.UTF16PtrFromString(path)
		if e != nil {
			return e
		}
		attributes, e := windows.GetFileAttributes(u)
		if e != nil {
			return e
		}
		if attributes&windows.FILE_ATTRIBUTE_REPARSE_POINT != 0 {
			// Go does not classify every Windows mount point as a symlink.
			// Snapshot the held reparse object itself, never its target bytes.
			h, e := windows.CreateFile(u, windows.READ_CONTROL|windows.FILE_READ_ATTRIBUTES, windows.FILE_SHARE_READ|windows.FILE_SHARE_WRITE|windows.FILE_SHARE_DELETE, nil, windows.OPEN_EXISTING, windows.FILE_FLAG_BACKUP_SEMANTICS|windows.FILE_FLAG_OPEN_REPARSE_POINT, 0)
			if e != nil {
				return e
			}
			defer windows.CloseHandle(h)
			var info windows.ByHandleFileInformation
			if e = windows.GetFileInformationByHandle(h, &info); e != nil {
				return e
			}
			if info.FileAttributes&windows.FILE_ATTRIBUTE_REPARSE_POINT == 0 {
				return errors.New("snapshot reparse object changed")
			}
			sd, e := windows.GetSecurityInfo(h, windows.SE_FILE_OBJECT, windows.OWNER_SECURITY_INFORMATION|windows.DACL_SECURITY_INFORMATION)
			if e != nil {
				return e
			}
			buffer := make([]byte, 16<<10)
			var returned uint32
			if e = windows.DeviceIoControl(h, windows.FSCTL_GET_REPARSE_POINT, nil, 0, &buffer[0], uint32(len(buffer)), &returned, nil); e != nil {
				return e
			}
			if returned < 8 || returned > uint32(len(buffer)) {
				return errors.New("snapshot reparse data incomplete")
			}
			out[rel] = offlineObserved{Descriptor: sd.String(), Link: hex.EncodeToString(buffer[:returned]), Directory: info.FileAttributes&windows.FILE_ATTRIBUTE_DIRECTORY != 0}
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		sd, e := windows.GetNamedSecurityInfo(path, windows.SE_FILE_OBJECT, windows.OWNER_SECURITY_INFORMATION|windows.DACL_SECURITY_INFORMATION)
		if e != nil {
			return e
		}
		v := offlineObserved{Descriptor: sd.String(), Directory: d.IsDir()}
		if d.Type()&os.ModeSymlink != 0 {
			v.Link, e = os.Readlink(path)
			if e != nil {
				return e
			}
		} else if !d.IsDir() {
			b, e := os.ReadFile(path)
			if e != nil {
				return e
			}
			sum := sha256.Sum256(b)
			v.Hash = hex.EncodeToString(sum[:])
		}
		out[rel] = v
		return nil
	})
	if e != nil {
		t.Fatal("synthetic source snapshot failed", e)
	}
	return out
}
func offlineAssertSnapshot(t *testing.T, root string, before map[string]offlineObserved) {
	t.Helper()
	if !reflect.DeepEqual(before, offlineSnapshot(t, root)) {
		t.Fatal("read-only operation changed source namespace, security descriptors or file bytes")
	}
}
func offlineFixture(t *testing.T) (root, id string, original []byte) {
	t.Helper()
	root = filepath.Join(t.TempDir(), "existing library é")
	s, e := Open(root)
	if e != nil {
		t.Fatal(e)
	}
	original = fixture(t)
	a, _, e := s.Put("private synthetic photo.png", bytes.NewReader(original))
	if e != nil {
		s.Close()
		t.Fatal(e)
	}
	id = a.ID
	// Leave the preview queue pending. A normal read/write Open rebuilds derived
	// catalog data; the byte-for-byte snapshot below detects that side effect.
	if e = s.Close(); e != nil {
		t.Fatal(e)
	}
	if e = os.WriteFile(filepath.Join(root, "access-code.txt"), []byte(strings.Repeat("a", 64)+"\n"), 0600); e != nil {
		t.Fatal(e)
	}
	if e = os.WriteFile(filepath.Join(root, "staging", "retained-unrelated.bin"), []byte("existing staging sentinel"), 0600); e != nil {
		t.Fatal(e)
	}
	return
}
func offlineCheckArchive(t *testing.T, b []byte, id string, original []byte) {
	t.Helper()
	zr, e := zip.NewReader(bytes.NewReader(b), int64(len(b)))
	if e != nil {
		t.Fatal("read-only backup is not a ZIP", e)
	}
	entries := map[string][]byte{}
	for _, f := range zr.File {
		if _, ok := entries[f.Name]; ok {
			t.Fatal("duplicate archive entry")
		}
		r, e := f.Open()
		if e != nil {
			t.Fatal(e)
		}
		body, re := io.ReadAll(r)
		ce := r.Close()
		if re != nil || ce != nil {
			t.Fatal("backup entry readback failed", errors.Join(re, ce))
		}
		entries[f.Name] = body
	}
	for _, name := range []string{"catalog.db", "state-format.txt", "access-code.txt", "originals/" + id, "manifest.json"} {
		if _, ok := entries[name]; !ok {
			t.Fatal("backup omits a required independent recovery payload")
		}
	}
	if !bytes.Equal(entries["originals/"+id], original) {
		t.Fatal("backup original differs")
	}
	if string(entries["state-format.txt"]) != "immichclone-schema-1\n" || string(entries["access-code.txt"]) != strings.Repeat("a", 64)+"\n" {
		t.Fatal("backup did not preserve marker/code")
	}
	var manifest Manifest
	if e = json.Unmarshal(entries["manifest.json"], &manifest); e != nil || manifest.Schema != 1 {
		t.Fatal("invalid backup manifest", e)
	}
	if len(manifest.Files) != len(entries)-1 {
		t.Fatal("manifest coverage mismatch")
	}
	for name, want := range manifest.Files {
		sum := sha256.Sum256(entries[name])
		if hex.EncodeToString(sum[:]) != want {
			t.Fatal("backup manifest checksum mismatch")
		}
	}
}
func TestOfflineReadOnlyVerifyAndBackupPreserveWholeSource(t *testing.T) {
	root, id, original := offlineFixture(t)
	before := offlineSnapshot(t, root)
	s, e := OpenReadOnly(root)
	if e != nil {
		t.Fatal("valid existing library read-only open failed", e)
	}
	if e = s.Verify(); e != nil {
		s.Close()
		t.Fatal("read-only original verification failed", e)
	}
	var archive bytes.Buffer
	if e = s.Backup(&archive); e != nil {
		s.Close()
		t.Fatal("read-only backup failed", e)
	}
	if e = s.Close(); e != nil {
		t.Fatal(e)
	}
	offlineCheckArchive(t, archive.Bytes(), id, original)
	offlineAssertSnapshot(t, root, before)
	t.Log("PASS: read-only verify and archive creation preserve every source byte, descriptor and directory entry including catalog preview queue and staging sentinel; independent ZIP/manifest/original readback passed")
}
func TestOfflineMissingOrUnsupportedStateNeverInitialized(t *testing.T) {
	for _, kind := range []string{"missing-root", "missing-marker", "unsupported-marker", "missing-catalog", "unsupported-catalog-schema", "missing-catalog-buckets"} {
		t.Run(kind, func(t *testing.T) {
			root, _, _ := offlineFixture(t)
			if kind == "missing-root" {
				root = filepath.Join(filepath.Dir(root), "must remain absent")
				if s, e := OpenReadOnly(root); e == nil {
					s.Close()
					t.Fatal("missing source initialized")
				}
				if _, e := os.Stat(root); !errors.Is(e, os.ErrNotExist) {
					t.Fatal("missing source directory was created", e)
				}
				return
			}
			if kind == "missing-marker" {
				if e := os.Remove(filepath.Join(root, "state-format.txt")); e != nil {
					t.Fatal(e)
				}
			}
			if kind == "unsupported-marker" {
				if e := os.WriteFile(filepath.Join(root, "state-format.txt"), []byte("unsupported-schema\n"), 0600); e != nil {
					t.Fatal(e)
				}
			}
			if kind == "missing-catalog" {
				if e := os.Remove(filepath.Join(root, "catalog.db")); e != nil {
					t.Fatal(e)
				}
			}
			if kind == "unsupported-catalog-schema" {
				db, e := bolt.Open(filepath.Join(root, "catalog.db"), 0600, nil)
				if e != nil {
					t.Fatal(e)
				}
				e = db.Update(func(tx *bolt.Tx) error { return tx.Bucket([]byte("meta")).Put([]byte("schema"), []byte("3")) })
				ce := db.Close()
				if e != nil || ce != nil {
					t.Fatal(errors.Join(e, ce))
				}
			}
			if kind == "missing-catalog-buckets" {
				if e := os.Remove(filepath.Join(root, "catalog.db")); e != nil {
					t.Fatal(e)
				}
				db, e := bolt.Open(filepath.Join(root, "catalog.db"), 0600, nil)
				if e != nil {
					t.Fatal(e)
				}
				if e = db.Close(); e != nil {
					t.Fatal(e)
				}
			}
			before := offlineSnapshot(t, root)
			if s, e := OpenReadOnly(root); e == nil {
				s.Close()
				t.Fatal("missing/unsupported source accepted")
			}
			offlineAssertSnapshot(t, root, before)
		})
	}
	t.Log("PASS: absent source, missing marker/catalog and unsupported state refuse without directory, marker, schema or catalog initialization")
}
func offlineEnableLinks(t *testing.T) {
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
func offlineRefused(t *testing.T, root, target string) {
	t.Helper()
	s, e := OpenReadOnly(root)
	if e != nil {
		return
	}
	defer s.Close()
	if target != "access-code.txt" {
		if e = s.Verify(); e == nil {
			t.Fatal("unsafe read-only source was accepted by verification")
		}
	}
	var archive bytes.Buffer
	if e = s.Backup(&archive); e == nil {
		t.Fatal("unsafe read-only backup source was accepted")
	}
}
func TestOfflineActualSourceAliasesRefuseWithoutOutsideChanges(t *testing.T) {
	for _, kind := range []string{"root-link", "originals-directory-link", "marker-link", "catalog-link", "original-link", "code-link", "marker-hardlink", "catalog-hardlink", "original-hardlink", "code-hardlink"} {
		t.Run(kind, func(t *testing.T) {
			offlineEnableLinks(t)
			root, id, _ := offlineFixture(t)
			outside := t.TempDir()
			if kind == "root-link" {
				before := offlineSnapshot(t, root)
				link := filepath.Join(outside, "linked library")
				if e := os.Symlink(root, link); e != nil {
					t.Fatal(e)
				}
				offlineRefused(t, link, "root")
				offlineAssertSnapshot(t, root, before)
				return
			}

			if kind == "originals-directory-link" {
				selected := filepath.Join(root, "originals")
				outsideDir := filepath.Join(outside, "actual originals")
				if e := os.Rename(selected, outsideDir); e != nil {
					t.Fatal(e)
				}
				if e := os.Symlink(outsideDir, selected); e != nil {
					t.Fatal(e)
				}
				info, e := os.Stat(selected)
				if e != nil {
					t.Fatal(e)
				}
				other, e := os.Stat(outsideDir)
				if e != nil || !os.SameFile(info, other) {
					t.Fatal("actual directory alias positive control failed", e)
				}
				before := offlineSnapshot(t, root)
				outsideBefore := offlineSnapshot(t, outside)
				offlineRefused(t, root, "originals directory")
				offlineAssertSnapshot(t, root, before)
				offlineAssertSnapshot(t, outside, outsideBefore)
				return
			}
			target := "state-format.txt"
			if strings.HasPrefix(kind, "catalog-") {
				target = "catalog.db"
			}
			if strings.HasPrefix(kind, "original-") {
				target = filepath.Join("originals", id)
			}
			if strings.HasPrefix(kind, "code-") {
				target = "access-code.txt"
			}
			selected := filepath.Join(root, target)
			body, e := os.ReadFile(selected)
			if e != nil {
				t.Fatal(e)
			}
			outsideFile := filepath.Join(outside, "retained-source.bin")
			if e = os.WriteFile(outsideFile, body, 0600); e != nil {
				t.Fatal(e)
			}
			if e = os.Remove(selected); e != nil {
				t.Fatal(e)
			}
			if strings.HasSuffix(kind, "hardlink") {
				e = os.Link(outsideFile, selected)
			} else {
				e = os.Symlink(outsideFile, selected)
			}
			if e != nil {
				t.Fatal(e)
			}
			// Link creation and a same-object file identity check are the positive controls;
			// all source bytes remain valid so refusal cannot be attributed to bad data.
			info, e := os.Stat(selected)
			if e != nil {
				t.Fatal(e)
			}
			outInfo, e := os.Stat(outsideFile)
			if e != nil || !os.SameFile(info, outInfo) {
				t.Fatal("actual alias identity positive control failed", e)
			}
			before := offlineSnapshot(t, root)
			outsideBefore := offlineSnapshot(t, outside)
			offlineRefused(t, root, target)
			offlineAssertSnapshot(t, root, before)
			offlineAssertSnapshot(t, outside, outsideBefore)
		})
	}
	t.Log("PASS: actual root/marker/catalog/original/code links and hardlinks refuse through valid-data fixtures without source or outside bytes/descriptor mutation")
}
func TestOfflineActualSourceWriterForcesRefusal(t *testing.T) {
	for _, role := range []string{"catalog.db", "state-format.txt", "access-code.txt", "original"} {
		t.Run(role, func(t *testing.T) {
			root, id, _ := offlineFixture(t)
			target := role
			if role == "original" {
				target = filepath.Join("originals", id)
			}
			name, e := windows.UTF16PtrFromString(filepath.Join(root, target))
			if e != nil {
				t.Fatal(e)
			}
			h, e := windows.CreateFile(name, windows.GENERIC_WRITE, windows.FILE_SHARE_READ|windows.FILE_SHARE_WRITE|windows.FILE_SHARE_DELETE, nil, windows.OPEN_EXISTING, 0, 0)
			if e != nil {
				t.Fatal("actual writer positive control acquisition failed", e)
			}
			defer func() {
				if h != 0 {
					windows.CloseHandle(h)
				}
			}()
			before := offlineSnapshot(t, root)
			offlineRefused(t, root, target)
			offlineAssertSnapshot(t, root, before)
			if e = windows.CloseHandle(h); e != nil {
				t.Fatal(e)
			}
			h = 0
			s, e := OpenReadOnly(root)
			if e != nil {
				t.Fatal("writer release should permit safe retry", e)
			}
			var archive bytes.Buffer
			e = s.Backup(&archive)
			ce := s.Close()
			if e != nil || ce != nil {
				t.Fatal("released-writer retry failed", errors.Join(e, ce))
			}
		})
	}
	t.Log("PASS: real preexisting source data writers force read-only refusal without mutation; release permits a complete backup retry")
}

type offlineFailWriter struct{}

func (offlineFailWriter) Write([]byte) (int, error) {
	return 0, errors.New("synthetic archive write refusal")
}
func TestOfflineBackupFailureAndCancellationPreserveSource(t *testing.T) {
	root, _, _ := offlineFixture(t)
	before := offlineSnapshot(t, root)
	s, e := OpenReadOnly(root)
	if e != nil {
		t.Fatal(e)
	}
	if e = s.Backup(offlineFailWriter{}); e == nil {
		s.Close()
		t.Fatal("archive output failure was reported successful")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	var archive bytes.Buffer
	if e = s.BackupContext(ctx, &archive, nil); !errors.Is(e, context.Canceled) {
		s.Close()
		t.Fatal("cancelled read-only backup did not return cancellation", e)
	}
	if e = s.Close(); e != nil {
		t.Fatal(e)
	}
	offlineAssertSnapshot(t, root, before)
	t.Log("PASS: failed archive output and cancelled read-only backup return failure without source/catalog/staging/descriptor mutation")
}

func offlinePrivateArchive(t *testing.T, path string) {
	t.Helper()
	sd, e := windows.GetNamedSecurityInfo(path, windows.SE_FILE_OBJECT, windows.OWNER_SECURITY_INFORMATION|windows.DACL_SECURITY_INFORMATION)
	if e != nil {
		t.Fatal(e)
	}
	control, _, e := sd.Control()
	if e != nil || control&windows.SE_DACL_PROTECTED == 0 {
		t.Fatal("archive descriptor is not protected/private", e)
	}
	acl, _, e := sd.DACL()
	if e != nil || acl == nil {
		t.Fatal("archive DACL missing", e)
	}
	caller, e := windows.GetCurrentProcessToken().GetTokenUser()
	if e != nil {
		t.Fatal(e)
	}
	admitted := map[string]bool{"S-1-5-18": true, "S-1-5-32-544": true, caller.User.Sid.String(): true}
	for i := uint32(0); i < uint32(acl.AceCount); i++ {
		var ace *windows.ACCESS_ALLOWED_ACE
		if e = windows.GetAce(acl, i, &ace); e != nil {
			t.Fatal(e)
		}
		if ace.Header.AceType != windows.ACCESS_ALLOWED_ACE_TYPE {
			t.Fatal("unexpected archive permission type")
		}
		sid := (*windows.SID)(unsafe.Pointer(&ace.SidStart)).String()
		if !admitted[sid] {
			t.Fatal("archive grants access outside caller/administrators/SYSTEM")
		}
	}
}
func TestOfflineBackupFilePrivateArchiveAndIndependentRestore(t *testing.T) {
	root, id, original := offlineFixture(t)
	before := offlineSnapshot(t, root)
	output := t.TempDir()
	archive := filepath.Join(output, "private backup é.zip")
	sentinel := filepath.Join(output, "retained-outside.bin")
	if e := os.WriteFile(sentinel, []byte("retained outside sentinel"), 0600); e != nil {
		t.Fatal(e)
	}
	s, e := OpenReadOnly(root)
	if e != nil {
		t.Fatal(e)
	}
	e = s.BackupFile(archive)
	ce := s.Close()
	if e != nil || ce != nil {
		t.Fatal("private held archive failed", errors.Join(e, ce))
	}
	offlineAssertSnapshot(t, root, before)
	offlinePrivateArchive(t, archive)
	b, e := os.ReadFile(archive)
	if e != nil {
		t.Fatal(e)
	}
	offlineCheckArchive(t, b, id, original)
	target := filepath.Join(t.TempDir(), "independently restored é")
	if e = Restore(archive, target); e != nil {
		t.Fatal("independent restoration of private offline archive failed", e)
	}
	restored, e := OpenReadOnly(target)
	if e != nil {
		t.Fatal(e)
	}
	e = restored.Verify()
	ce = restored.Close()
	if e != nil || ce != nil {
		t.Fatal("independent restored original verification failed", errors.Join(e, ce))
	}
	offlineAssertSnapshot(t, root, before)
	got, e := os.ReadFile(filepath.Join(target, "originals", id))
	if e != nil || !bytes.Equal(got, original) {
		t.Fatal("independent restored original changed", e)
	}
	got, e = os.ReadFile(sentinel)
	if e != nil || string(got) != "retained outside sentinel" {
		t.Fatal("backup changed unrelated outside sentinel", e)
	}
	entries, e := os.ReadDir(output)
	if e != nil || len(entries) != 2 {
		t.Fatal("successful backup left unexpected outside staging/files", e)
	}
	t.Log("PASS: exclusive held backup archive is private to caller/admin/SYSTEM, source stays byte/ACL/namespace-identical, independent Restore and Verify recover original, unrelated outside bytes preserved")
}
func TestOfflineBackupFileExistingAndInsideSourceRefuse(t *testing.T) {
	for _, kind := range []string{"existing-archive", "existing-leaf-link", "existing-leaf-hardlink", "source-root", "source-child"} {
		t.Run(kind, func(t *testing.T) {
			root, _, _ := offlineFixture(t)
			output := t.TempDir()
			archive := filepath.Join(output, "new.zip")
			if kind == "existing-archive" {
				if e := os.WriteFile(archive, []byte("existing archive must never be truncated"), 0600); e != nil {
					t.Fatal(e)
				}
			}
			if kind == "existing-leaf-link" || kind == "existing-leaf-hardlink" {
				offlineEnableLinks(t)
				sentinel := filepath.Join(output, "outside-existing.bin")
				if e := os.WriteFile(sentinel, []byte("outside archive leaf sentinel"), 0600); e != nil {
					t.Fatal(e)
				}
				var e error
				if kind == "existing-leaf-link" {
					e = os.Symlink(sentinel, archive)
				} else {
					e = os.Link(sentinel, archive)
				}
				if e != nil {
					t.Fatal(e)
				}
				info, e := os.Stat(archive)
				if e != nil {
					t.Fatal(e)
				}
				other, e := os.Stat(sentinel)
				if e != nil || !os.SameFile(info, other) {
					t.Fatal("actual output leaf alias positive control failed", e)
				}
			}
			if kind == "source-root" {
				archive = filepath.Join(root, "must-not-exist.zip")
			}
			if kind == "source-child" {
				archive = filepath.Join(root, "previews", "must-not-exist.zip")
			}
			before := offlineSnapshot(t, root)
			outsideBefore := offlineSnapshot(t, output)
			s, e := OpenReadOnly(root)
			if e != nil {
				t.Fatal(e)
			}
			e = s.BackupFile(archive)
			ce := s.Close()
			if e == nil {
				t.Fatal("existing or source-contained archive target accepted")
			}
			if ce != nil {
				t.Fatal(ce)
			}
			offlineAssertSnapshot(t, root, before)
			offlineAssertSnapshot(t, output, outsideBefore)
		})
	}
	t.Log("PASS: exclusive archive creation refuses existing file and source-root/nested outputs without truncation or namespace changes")
}
func TestOfflineBackupFileRedirectedParentsRefuseBeforeOutsideCreation(t *testing.T) {
	for _, kind := range []string{"parent-link", "source-link"} {
		t.Run(kind, func(t *testing.T) {
			offlineEnableLinks(t)
			root, _, _ := offlineFixture(t)
			outside := t.TempDir()
			container := t.TempDir()
			link := filepath.Join(container, "redirected parent")
			destination := outside
			if kind == "source-link" {
				destination = root
			}
			if e := os.Symlink(destination, link); e != nil {
				t.Fatal(e)
			}
			info, e := os.Stat(link)
			if e != nil {
				t.Fatal(e)
			}
			target, e := os.Stat(destination)
			if e != nil || !os.SameFile(info, target) {
				t.Fatal("actual archive-parent alias positive control missing", e)
			}
			before := offlineSnapshot(t, root)
			outsideBefore := offlineSnapshot(t, outside)
			s, e := OpenReadOnly(root)
			if e != nil {
				t.Fatal(e)
			}
			e = s.BackupFile(filepath.Join(link, "must-not-exist.zip"))
			ce := s.Close()
			if e == nil {
				t.Fatal("redirected archive parent accepted")
			}
			if ce != nil {
				t.Fatal(ce)
			}
			offlineAssertSnapshot(t, root, before)
			offlineAssertSnapshot(t, outside, outsideBefore)
		})
	}
	t.Log("PASS: actual redirected archive parents and source-root aliases refuse before selected/outside archive creation")
}
func TestOfflineBackupFileCorruptOriginalRemovesExactNewArchive(t *testing.T) {
	root, id, _ := offlineFixture(t)
	bad := filepath.Join(root, "originals", id)
	if e := os.WriteFile(bad, []byte("actual checksum corruption"), 0600); e != nil {
		t.Fatal(e)
	}
	before := offlineSnapshot(t, root)
	output := t.TempDir()
	sentinel := filepath.Join(output, "unrelated.bin")
	if e := os.WriteFile(sentinel, []byte("outside retained bytes"), 0600); e != nil {
		t.Fatal(e)
	}
	outsideBefore := offlineSnapshot(t, output)
	archive := filepath.Join(output, "failed-private.zip")
	s, e := OpenReadOnly(root)
	if e != nil {
		t.Fatal(e)
	}
	e = s.BackupFile(archive)
	ce := s.Close()
	if e == nil {
		t.Fatal("corrupted original produced successful archive")
	}
	if ce != nil {
		t.Fatal(ce)
	}
	if _, e = os.Stat(archive); !errors.Is(e, os.ErrNotExist) {
		t.Fatal("failed exact new archive was not removed", e)
	}
	offlineAssertSnapshot(t, root, before)
	offlineAssertSnapshot(t, output, outsideBefore)
	t.Log("PASS: actual original checksum failure removes only the newly held private archive and preserves all source and unrelated outside bytes/descriptors")
}

func TestOfflineOptionalMissingAccessCodePreservedWithoutGeneration(t *testing.T) {
	root, id, original := offlineFixture(t)
	if e := os.Remove(filepath.Join(root, "access-code.txt")); e != nil {
		t.Fatal(e)
	}
	before := offlineSnapshot(t, root)
	s, e := OpenReadOnly(root)
	if e != nil {
		t.Fatal("optional missing code must not prevent read-only open", e)
	}
	if e = s.Verify(); e != nil {
		s.Close()
		t.Fatal("optional missing code must not prevent original verification", e)
	}
	var archive bytes.Buffer
	e = s.Backup(&archive)
	ce := s.Close()
	if e != nil || ce != nil {
		t.Fatal("optional missing code must permit backup without generation", errors.Join(e, ce))
	}
	zr, e := zip.NewReader(bytes.NewReader(archive.Bytes()), int64(archive.Len()))
	if e != nil {
		t.Fatal(e)
	}
	entries := map[string][]byte{}
	for _, f := range zr.File {
		r, e := f.Open()
		if e != nil {
			t.Fatal(e)
		}
		b, re := io.ReadAll(r)
		ce := r.Close()
		if re != nil || ce != nil {
			t.Fatal(errors.Join(re, ce))
		}
		entries[f.Name] = b
	}
	if _, ok := entries["access-code.txt"]; ok {
		t.Fatal("backup generated an absent optional access code")
	}
	if !bytes.Equal(entries["originals/"+id], original) || string(entries["state-format.txt"]) != "immichclone-schema-1\n" || len(entries["catalog.db"]) == 0 {
		t.Fatal("optional-code backup lost required recovery content")
	}
	var m Manifest
	if e = json.Unmarshal(entries["manifest.json"], &m); e != nil {
		t.Fatal(e)
	}
	if len(m.Files) != len(entries)-1 {
		t.Fatal("optional-code manifest coverage mismatch")
	}
	for name, want := range m.Files {
		sum := sha256.Sum256(entries[name])
		if hex.EncodeToString(sum[:]) != want {
			t.Fatal("optional-code manifest checksum mismatch")
		}
	}
	offlineAssertSnapshot(t, root, before)
	if _, e = os.Stat(filepath.Join(root, "access-code.txt")); !errors.Is(e, os.ErrNotExist) {
		t.Fatal("optional code was generated", e)
	}
	t.Log("PASS: existing format with absent optional code remains verifiable/backuppable without source mutation or replacement-code generation; archive required payloads and manifest validate")
}
func TestOfflineBackupDeclaredOriginalSizeMismatchRefusesAndCleans(t *testing.T) {
	root, id, original := offlineFixture(t)
	source, e := Open(root)
	if e != nil {
		t.Fatal(e)
	}
	a, e := source.Get(id)
	if e != nil {
		source.Close()
		t.Fatal(e)
	}
	a.Size = int64(len(original)) + 17
	e = source.save(a)
	ce := source.Close()
	if e != nil || ce != nil {
		t.Fatal("size-mismatch fixture setup failed", errors.Join(e, ce))
	}
	// The original content and content-addressed ID are still valid: only the
	// catalog's declared size disagrees, isolating the archive completeness check.
	b, e := os.ReadFile(filepath.Join(root, "originals", id))
	if e != nil {
		t.Fatal(e)
	}
	sum := sha256.Sum256(b)
	if hex.EncodeToString(sum[:]) != id {
		t.Fatal("declared-size positive control lost valid original checksum")
	}
	before := offlineSnapshot(t, root)
	output := t.TempDir()
	outsideBefore := offlineSnapshot(t, output)
	archive := filepath.Join(output, "must-refuse-size-mismatch.zip")
	s, e := OpenReadOnly(root)
	if e != nil {
		t.Fatal(e)
	}
	e = s.BackupFile(archive)
	ce = s.Close()
	if e == nil {
		t.Fatal("declared original size mismatch was accepted")
	}
	if ce != nil {
		t.Fatal(ce)
	}
	if _, e = os.Stat(archive); !errors.Is(e, os.ErrNotExist) {
		t.Fatal("size-mismatch failure left a new archive", e)
	}
	offlineAssertSnapshot(t, root, before)
	offlineAssertSnapshot(t, output, outsideBefore)
	t.Log("PASS: valid content checksum with deliberately wrong declared size refuses backup and removes exact new private archive without source/outside changes")
}
