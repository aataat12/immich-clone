//go:build windows

package installerfiles

import (
	"bytes"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"golang.org/x/sys/windows"
)

// Keep this independent from the publisher's admitted-name table: a change to
// the product allowlist must not silently change the acceptance fixture too.
var publicationTestFiles = []string{
	"Manage-Service.ps1", "Storage-Security.ps1", "installer-files.exe",
	"immichclone.exe", "Export-Diagnostics.ps1", "THIRD-PARTY-LICENSES.txt",
	"prototype-notes.txt", `media\nested é\decoder.bin`, `ai\nested é\model.onnx`,
}

func publicationFixture(t *testing.T) (app, source, backup, outside string) {
	t.Helper()
	base := t.TempDir()
	app, source, backup, outside = filepath.Join(base, "application é"), filepath.Join(base, "staged payload é"), filepath.Join(base, "private code backup"), filepath.Join(base, "outside")
	for _, p := range []string{app, source, backup, outside} {
		if e := os.Mkdir(p, 0700); e != nil {
			t.Fatal(e)
		}
		setPrivacyDescriptor(t, p, "D:P(A;OICI;FA;;;SY)(A;OICI;FA;;;BA)")
	}
	for _, name := range publicationTestFiles {
		for _, root := range []string{app, source} {
			file := filepath.Join(root, name)
			if e := os.MkdirAll(filepath.Dir(file), 0700); e != nil {
				t.Fatal(e)
			}
			prefix := "prior-"
			if root == source {
				prefix = "candidate-"
			}
			if e := os.WriteFile(file, []byte(prefix+name), 0600); e != nil {
				t.Fatal(e)
			}
		}
	}
	return
}

func publicationAssertSet(t *testing.T, app, prefix string) {
	t.Helper()
	for _, name := range publicationTestFiles {
		assertBytes(t, filepath.Join(app, name), prefix+name)
	}
}

func TestPayloadPublicationAndFullCodeRestore(t *testing.T) {
	app, source, backup, _ := publicationFixture(t)
	unknown := filepath.Join(app, "unrelated-user-file.txt")
	if e := os.WriteFile(unknown, []byte("retained unrelated bytes"), 0600); e != nil {
		t.Fatal(e)
	}
	newName := `ai\nested é\new-runtime.dll`
	if e := os.WriteFile(filepath.Join(source, newName), []byte("new provider bytes"), 0600); e != nil {
		t.Fatal(e)
	}
	if e := PublishPayload(app, source, backup); e != nil {
		t.Fatal("normal complete nested payload publication refused", e)
	}
	publicationAssertSet(t, app, "candidate-")
	assertBytes(t, filepath.Join(app, newName), "new provider bytes")
	assertBytes(t, unknown, "retained unrelated bytes")
	if e := RestorePayload(app, backup); e != nil {
		t.Fatal("verified full code restoration refused", e)
	}
	publicationAssertSet(t, app, "prior-")
	if _, e := os.Stat(filepath.Join(app, newName)); !errors.Is(e, os.ErrNotExist) {
		t.Fatal("restoration did not remove a newly published provider file", e)
	}
	assertBytes(t, unknown, "retained unrelated bytes")
	t.Log("PASS: complete native nested Unicode payload verified; restoration recovers every prior script/runtime/provider byte, removes newly published files, and preserves unrelated installed content")
}

func TestPayloadCopyInterruptionRestoresHeldCodeSet(t *testing.T) {
	app, source, backup, _ := publicationFixture(t)
	calls := 0
	e := publishPayload(app, source, backup, func(i int) error {
		calls++
		if i == 1 {
			return errors.New("synthetic publication interruption after first copy")
		}
		return nil
	})
	if e == nil || calls < 2 {
		t.Fatal("partial copy interruption was not exercised and refused")
	}
	publicationAssertSet(t, app, "prior-")
	t.Log("PASS: injected partial native payload publication refuses completion and automatically restores all held prior code bytes")
}

func TestPayloadUnsafeIncomingAliasesPreserveAllOldAndOutsideBytes(t *testing.T) {
	for _, where := range []string{"source-file-link", "target-file-link", "source-hardlink", "target-hardlink", "source-provider-link", "target-provider-link"} {
		t.Run(where, func(t *testing.T) {
			treeEnableLinks(t)
			app, source, backup, outside := publicationFixture(t)
			sentinel := filepath.Join(outside, "outside.bin")
			if e := os.WriteFile(sentinel, []byte("outside must stay unchanged"), 0600); e != nil {
				t.Fatal(e)
			}
			root := app
			if where == "source-file-link" || where == "source-hardlink" || where == "source-provider-link" {
				root = source
			}
			victim := filepath.Join(root, "prototype-notes.txt")
			if where == "source-provider-link" || where == "target-provider-link" {
				victim = filepath.Join(root, "ai")
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
				if where == "source-hardlink" || where == "target-hardlink" {
					e = os.Link(sentinel, victim)
				} else {
					e = os.Symlink(sentinel, victim)
				}
				if e != nil {
					t.Fatal(e)
				}
			}
			outsideSD := treeSD(t, sentinel).String()
			if e := PublishPayload(app, source, backup); e == nil {
				t.Fatal("unsafe payload source/destination alias was accepted")
			}
			// An earlier root target must remain untouched regardless of which
			// later leaf or provider directory was aliased.
			assertBytes(t, filepath.Join(app, "Manage-Service.ps1"), "prior-Manage-Service.ps1")
			assertBytes(t, filepath.Join(app, "immichclone.exe"), "prior-immichclone.exe")
			assertBytes(t, sentinel, "outside must stay unchanged")
			if treeSD(t, sentinel).String() != outsideSD {
				t.Fatal("refused publication changed outside security descriptor")
			}
			t.Log("PASS: actual incoming payload alias refused before earlier code truncation with outside bytes and descriptor unchanged")
		})
	}
}

func TestPayloadExistingWriterRefusesBeforeEarlierCodeTruncation(t *testing.T) {
	app, source, backup, _ := publicationFixture(t)
	victim := filepath.Join(app, `ai\nested é\model.onnx`)
	u, e := windows.UTF16PtrFromString(victim)
	if e != nil {
		t.Fatal(e)
	}
	h, e := windows.CreateFile(u, windows.GENERIC_WRITE, windows.FILE_SHARE_READ|windows.FILE_SHARE_WRITE|windows.FILE_SHARE_DELETE, nil, windows.OPEN_EXISTING, 0, 0)
	if e != nil {
		t.Fatal("real unrelated writer positive control failed", e)
	}
	e = PublishPayload(app, source, backup)
	if closeErr := windows.CloseHandle(h); closeErr != nil {
		t.Fatal(closeErr)
	}
	if e == nil {
		t.Fatal("existing writable target handle did not force publication refusal")
	}
	publicationAssertSet(t, app, "prior-")
	t.Log("PASS: actual preexisting provider data writer forces payload preflight refusal before any earlier code byte is changed")
}

func TestPayloadSourceAllowlistAndRoleOverlapRefused(t *testing.T) {
	for _, kind := range []string{"unknown-root-file", "missing-required-file", "required-file-is-empty-directory", "source-is-target", "backup-inside-source", "source-inside-target"} {
		t.Run(kind, func(t *testing.T) {
			app, source, backup, _ := publicationFixture(t)
			switch kind {
			case "unknown-root-file":
				if e := os.WriteFile(filepath.Join(source, "unexpected.exe"), []byte("must not publish"), 0600); e != nil {
					t.Fatal(e)
				}
			case "missing-required-file":
				if e := os.Remove(filepath.Join(source, "Storage-Security.ps1")); e != nil {
					t.Fatal(e)
				}
			case "required-file-is-empty-directory":
				file := filepath.Join(source, "Storage-Security.ps1")
				if e := os.Remove(file); e != nil {
					t.Fatal(e)
				}
				if e := os.Mkdir(file, 0700); e != nil {
					t.Fatal(e)
				}
			case "source-is-target":
				source = app
			case "backup-inside-source":
				backup = filepath.Join(source, "private backup")
				if e := os.Mkdir(backup, 0700); e != nil {
					t.Fatal(e)
				}
			case "source-inside-target":
				newSource := filepath.Join(app, "nested payload")
				if e := os.Rename(source, newSource); e != nil {
					t.Fatal(e)
				}
				source = newSource
			}
			if e := PublishPayload(app, source, backup); e == nil {
				t.Fatal("invalid payload shape or overlapping role accepted")
			}
			publicationAssertSet(t, app, "prior-")
		})
	}
	t.Log("PASS: payload fixed-root shape and disjoint physical roles enforced before old code bytes change")
}

func TestPayloadCorruptBackupRefusesBeforeLiveRestoration(t *testing.T) {
	app, source, backup, _ := publicationFixture(t)
	if e := PublishPayload(app, source, backup); e != nil {
		t.Fatal(e)
	}
	// Find the saved old provider by bytes, without coupling this independent
	// corruption control to the implementation's internal backup filenames.
	var saved string
	want := []byte("prior-" + `ai\nested é\model.onnx`)
	e := filepath.WalkDir(backup, func(path string, entry fs.DirEntry, e error) error {
		if e != nil || entry.IsDir() {
			return e
		}
		body, e := os.ReadFile(path)
		if e != nil {
			return e
		}
		if bytes.Equal(body, want) {
			saved = path
		}
		return nil
	})
	if e != nil || saved == "" {
		t.Fatal("actual old provider backup positive control missing", e)
	}
	if e = os.WriteFile(saved, []byte("corrupt retained provider snapshot"), 0600); e != nil {
		t.Fatal(e)
	}
	if e = RestorePayload(app, backup); e == nil {
		t.Fatal("corrupt complete-code backup accepted")
	}
	publicationAssertSet(t, app, "candidate-")
	t.Log("PASS: actual corrupt saved provider is refused before any candidate script/runtime/provider byte is restored")
}

func TestPayloadMissingLiveTargetRestoreDoesNotCreateOnRefusal(t *testing.T) {
	app, source, backup, _ := publicationFixture(t)
	if e := PublishPayload(app, source, backup); e != nil {
		t.Fatal(e)
	}
	missing := filepath.Join(app, "prototype-notes.txt")
	if e := os.Remove(missing); e != nil {
		t.Fatal(e)
	}
	if e := RestorePayload(app, backup); e == nil {
		t.Fatal("missing live target must refuse automatic restoration")
	}
	if _, e := os.Stat(missing); !errors.Is(e, os.ErrNotExist) {
		t.Fatal("refused restoration unexpectedly created a missing live payload file", e)
	}
	for _, name := range publicationTestFiles {
		if filepath.Join(app, name) != missing {
			assertBytes(t, filepath.Join(app, name), "candidate-"+name)
		}
	}
	t.Log("PASS: missing live payload target refuses complete-code restoration without creating it or modifying earlier candidate bytes")
}

func TestPayloadUntrustedActualMutationGrantRefusesBeforeTruncation(t *testing.T) {
	for _, sourceSide := range []bool{false, true} {
		app, source, backup, _ := publicationFixture(t)
		root := app
		if sourceSide {
			root = source
		}
		victim := filepath.Join(root, `ai\nested é\model.onnx`)
		setPrivacyDescriptor(t, victim, "D:P(A;;FA;;;SY)(A;;FA;;;BA)(A;;FA;;;BU)")
		before := treeSD(t, victim).String()
		if e := PublishPayload(app, source, backup); e == nil {
			t.Fatal("actual ordinary-user code mutation grant accepted")
		}
		publicationAssertSet(t, app, "prior-")
		if treeSD(t, victim).String() != before {
			t.Fatal("payload refusal unexpectedly repaired unsafe descriptor")
		}
	}
	t.Log("PASS: actual untrusted source/destination code mutation grants refuse before any old code truncation or descriptor repair")
}
