//go:build windows

package library

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

func restoreBoundaryFixture(t *testing.T) (archive, source, id string, original []byte) {
	t.Helper()
	source, id, original = offlineFixture(t)
	s, e := OpenReadOnly(source)
	if e != nil {
		t.Fatal(e)
	}
	archive = filepath.Join(t.TempDir(), "valid private restore é.zip")
	e = s.BackupFile(archive)
	ce := s.Close()
	if e != nil || ce != nil {
		t.Fatal("valid restore archive fixture", errors.Join(e, ce))
	}
	return
}
func restoreBoundaryCall(archive, target string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	return RestoreWithLimits(ctx, archive, target, RestoreLimits{MaxBytes: 32 << 20, MemoryBytes: 256 << 20, Timeout: 12 * time.Second})
}
func restoreBoundaryPrivateDirectory(t *testing.T, path string, requireProtected bool) {
	t.Helper()
	sd, e := windows.GetNamedSecurityInfo(path, windows.SE_FILE_OBJECT, windows.OWNER_SECURITY_INFORMATION|windows.DACL_SECURITY_INFORMATION)
	if e != nil {
		t.Fatal(e)
	}
	ctrl, _, e := sd.Control()
	if e != nil || (requireProtected && ctrl&windows.SE_DACL_PROTECTED == 0) {
		t.Fatal("created restore directory is not protected/private", e)
	}
	owner, _, e := sd.Owner()
	if e != nil || owner == nil {
		t.Fatal("created restore directory owner missing", e)
	}
	user, e := windows.GetCurrentProcessToken().GetTokenUser()
	if e != nil {
		t.Fatal(e)
	}
	allowed := map[string]bool{"S-1-5-18": true, "S-1-5-32-544": true, user.User.Sid.String(): true}
	if !allowed[owner.String()] {
		t.Fatal("created restore directory has an unrelated owner")
	}
	acl, _, e := sd.DACL()
	if e != nil || acl == nil {
		t.Fatal("created restore directory DACL missing", e)
	}
	required := map[string]uint32{"S-1-5-18": 0, "S-1-5-32-544": 0}
	inherit := map[string]uint32{}
	for i := uint32(0); i < uint32(acl.AceCount); i++ {
		var ace *windows.ACCESS_ALLOWED_ACE
		if e = windows.GetAce(acl, i, &ace); e != nil {
			t.Fatal(e)
		}
		if ace.Header.AceType != windows.ACCESS_ALLOWED_ACE_TYPE {
			t.Fatal("restore private directory contains unsupported rule")
		}
		sid := (*windows.SID)(unsafe.Pointer(&ace.SidStart)).String()
		if !allowed[sid] {
			t.Fatal("restore directory permits unrelated account")
		}
		if ace.Header.AceFlags&8 == 0 {
			required[sid] |= uint32(ace.Mask)
		}
		if ace.Header.AceFlags&3 == 3 && ace.Header.AceFlags&12 == 0 {
			inherit[sid] |= uint32(ace.Mask)
		}
	}
	for _, sid := range []string{"S-1-5-18", "S-1-5-32-544"} {
		if required[sid]&0x1f01ff != 0x1f01ff || inherit[sid]&0x1f01ff != 0x1f01ff {
			t.Fatal("restore private policy does not protect current/future descendants")
		}
	}
}
func TestRestoreBoundaryValidPrivateAtBirthAndIndependentRecovery(t *testing.T) {
	archive, source, id, original := restoreBoundaryFixture(t)
	sourceBefore := offlineSnapshot(t, source)
	archiveBefore := offlineSnapshot(t, filepath.Dir(archive))
	parent := t.TempDir()
	target := filepath.Join(parent, "restored é")
	called := false
	old := restoreBoundaryStageCreated
	restoreBoundaryStageCreated = func(stage string) error {
		called = true
		restoreBoundaryPrivateDirectory(t, stage, true)
		if _, e := os.Stat(filepath.Join(stage, "catalog.db")); !errors.Is(e, os.ErrNotExist) {
			t.Fatal("private stage hook did not precede extracted catalog bytes", e)
		}
		for _, name := range []string{"originals", "staging", "previews", "faces"} {
			restoreBoundaryPrivateDirectory(t, filepath.Join(stage, name), false)
		}
		return nil
	}
	defer func() { restoreBoundaryStageCreated = old }()
	if e := restoreBoundaryCall(archive, target); e != nil {
		t.Fatal("valid private restore refused", e)
	}
	if !called {
		t.Fatal("private-at-birth control never observed actual stage")
	}
	restored, e := OpenReadOnly(target)
	if e != nil {
		t.Fatal(e)
	}
	e = restored.Verify()
	ce := restored.Close()
	if e != nil || ce != nil {
		t.Fatal(errors.Join(e, ce))
	}
	data, e := os.ReadFile(filepath.Join(target, "originals", id))
	if e != nil || string(data) != string(original) {
		t.Fatal("independent restored original differs", e)
	}
	offlineAssertSnapshot(t, source, sourceBefore)
	offlineAssertSnapshot(t, filepath.Dir(archive), archiveBefore)
	t.Log("PASS: actual restore stage and derived directories are private/protected before any catalog extraction; authentic supervised recovery verifies original bytes while source and archive remain unchanged")
}
func TestRestoreBoundaryStaticInputAliasesAndWriterRefuse(t *testing.T) {
	for _, kind := range []string{"archive-file-link", "archive-hardlink", "archive-parent-link", "target-parent-link", "target-parent-junction", "existing-directory", "existing-target-link", "existing-target-file", "archive-writer"} {
		t.Run(kind, func(t *testing.T) {
			offlineEnableLinks(t)
			archive, source, _, _ := restoreBoundaryFixture(t)
			sourceBefore := offlineSnapshot(t, source)
			parent := t.TempDir()
			outside := t.TempDir()
			sentinel := filepath.Join(outside, "sentinel.bin")
			if e := os.WriteFile(sentinel, []byte("outside restore sentinel"), 0600); e != nil {
				t.Fatal(e)
			}
			target := filepath.Join(parent, "new library")
			selected := archive
			var writer windows.Handle
			switch kind {
			case "archive-file-link":
				selected = filepath.Join(parent, "archive-link.zip")
				if e := os.Symlink(archive, selected); e != nil {
					t.Fatal(e)
				}
			case "archive-hardlink":
				selected = filepath.Join(parent, "archive-hardlink.zip")
				if e := os.Link(archive, selected); e != nil {
					t.Fatal(e)
				}
				a, e := os.Stat(archive)
				if e != nil {
					t.Fatal(e)
				}
				b, e := os.Stat(selected)
				if e != nil || !os.SameFile(a, b) {
					t.Fatal("hardlink positive identity missing", e)
				}
			case "archive-parent-link":
				p := filepath.Join(parent, "archive-parent")
				if e := os.Symlink(filepath.Dir(archive), p); e != nil {
					t.Fatal(e)
				}
				selected = filepath.Join(p, filepath.Base(archive))
			case "target-parent-link":
				p := filepath.Join(parent, "redirected-parent")
				if e := os.Symlink(outside, p); e != nil {
					t.Fatal(e)
				}
				target = filepath.Join(p, "new library")
			case "target-parent-junction":
				p := filepath.Join(parent, "junction-parent")
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()
				if e := exec.CommandContext(ctx, "cmd.exe", "/d", "/c", "mklink", "/J", p, outside).Run(); e != nil {
					t.Fatal("required actual parent junction could not be created", e)
				}
				info, e := os.Lstat(p)
				if e != nil || info.Mode()&os.ModeSymlink == 0 {
					t.Fatal("junction positive reparse control missing", e)
				}
				target = filepath.Join(p, "new library")
			case "existing-directory":
				if e := os.Mkdir(target, 0700); e != nil {
					t.Fatal(e)
				}
				if e := os.WriteFile(filepath.Join(target, "retained.bin"), []byte("existing target bytes"), 0600); e != nil {
					t.Fatal(e)
				}
			case "existing-target-link":
				if e := os.Symlink(outside, target); e != nil {
					t.Fatal(e)
				}
			case "existing-target-file":
				if e := os.WriteFile(target, []byte("existing target file bytes"), 0600); e != nil {
					t.Fatal(e)
				}
			case "archive-writer":
				u, e := windows.UTF16PtrFromString(archive)
				if e != nil {
					t.Fatal(e)
				}
				writer, e = windows.CreateFile(u, windows.GENERIC_WRITE, windows.FILE_SHARE_READ|windows.FILE_SHARE_WRITE|windows.FILE_SHARE_DELETE, nil, windows.OPEN_EXISTING, 0, 0)
				if e != nil {
					t.Fatal("required actual archive writer could not be acquired", e)
				}
				defer windows.CloseHandle(writer)
			}
			before := offlineSnapshot(t, parent)
			outsideBefore := offlineSnapshot(t, outside)
			archiveBefore := offlineSnapshot(t, filepath.Dir(archive))
			if e := restoreBoundaryCall(selected, target); e == nil {
				t.Fatal("unsafe restore selector or existing target was accepted")
			}
			offlineAssertSnapshot(t, parent, before)
			offlineAssertSnapshot(t, outside, outsideBefore)
			offlineAssertSnapshot(t, filepath.Dir(archive), archiveBefore)
			offlineAssertSnapshot(t, source, sourceBefore)
			t.Log("PASS: real restore input alias/writer or existing target refuses without target/stage publication or outside/source/archive byte and descriptor changes")
		})
	}
}
func TestRestoreBoundaryForgedLegacyWorkerCannotWrite(t *testing.T) {
	archive, source, _, _ := restoreBoundaryFixture(t)
	parent := t.TempDir()
	stage := filepath.Join(parent, "immich-restore-forged-valid")
	if e := os.Mkdir(stage, 0700); e != nil {
		t.Fatal(e)
	}
	nonce := strings.Repeat("c", 64)
	limits := RestoreLimits{MaxBytes: 32 << 20, MaxEntries: 100000, MaxManifestBytes: 16 << 20, MaxCatalogBytes: 4 << 20, MaxMetadataBytes: 64 << 20, MemoryBytes: 256 << 20, Timeout: 12 * time.Second}
	request := restoreRequest{Nonce: nonce, Archive: archive, Stage: stage, Limits: limits}
	body, e := json.Marshal(request)
	if e != nil {
		t.Fatal(e)
	}
	path := filepath.Join(stage, restoreRequestName)
	if e = os.WriteFile(path, body, 0600); e != nil {
		t.Fatal(e)
	}
	before := offlineSnapshot(t, parent)
	archiveBefore := offlineSnapshot(t, filepath.Dir(archive))
	sourceBefore := offlineSnapshot(t, source)
	exe, e := os.Executable()
	if e != nil {
		t.Fatal(e)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 12*time.Second)
	defer cancel()
	// Intentionally literal: this is the formerly admitted public request protocol,
	// not a test derived from a newly changed worker argument constant.
	command := exec.CommandContext(ctx, exe, "--immich-restore-worker-v1", path, nonce)
	e = command.Run()
	if ctx.Err() != nil {
		t.Fatal("orphan worker refusal exceeded bounded process time")
	}
	var exit *exec.ExitError
	if !errors.As(e, &exit) || exit.ExitCode() == 0 {
		t.Fatal("fully valid forged legacy worker invocation did not refuse before filesystem work")
	}
	offlineAssertSnapshot(t, parent, before)
	offlineAssertSnapshot(t, filepath.Dir(archive), archiveBefore)
	offlineAssertSnapshot(t, source, sourceBefore)
	t.Log("PASS: a caller-created valid archive, valid request, valid nonce and legitimate stage spelling cannot authorize pre-main restore writes without supervisor capabilities")
}
func TestRestoreBoundaryLateStageAliasCannotEscapeCleanup(t *testing.T) {
	offlineEnableLinks(t)
	archive, source, _, _ := restoreBoundaryFixture(t)
	parent := t.TempDir()
	outside := t.TempDir()
	sentinel := filepath.Join(outside, "retained.bin")
	if e := os.WriteFile(sentinel, []byte("retained outside stage bytes"), 0600); e != nil {
		t.Fatal(e)
	}
	outsideBefore := offlineSnapshot(t, outside)
	sourceBefore := offlineSnapshot(t, source)
	archiveBefore := offlineSnapshot(t, filepath.Dir(archive))
	target := filepath.Join(parent, "new library")
	var stage, link string
	old := restoreBoundaryStageCreated
	restoreBoundaryStageCreated = func(p string) error {
		stage = p
		restoreBoundaryPrivateDirectory(t, p, true)
		link = filepath.Join(p, "unplanned-outside-link.bin")
		if e := os.Symlink(sentinel, link); e != nil {
			return e
		}
		return errors.New("synthetic interruption after actual stage alias creation")
	}
	defer func() {
		restoreBoundaryStageCreated = old
		if link != "" {
			if _, e := os.Lstat(link); e == nil {
				_ = os.Remove(link)
			}
		}
		if stage != "" {
			_ = os.RemoveAll(stage)
		}
	}()
	e := restoreBoundaryCall(archive, target)
	if stage == "" || link == "" {
		t.Fatal("late-stage namespace control was never exercised")
	}
	if !errors.Is(e, ErrRestoreCleanup) {
		t.Fatal("unsafe late stage alias did not force explicit cleanup refusal", e)
	}
	if _, e = os.Lstat(target); !errors.Is(e, os.ErrNotExist) {
		t.Fatal("failed stage alias restore published target", e)
	}
	if _, e = os.Lstat(stage); e != nil {
		t.Fatal("unsafe cleanup did not retain its private stage for review", e)
	}
	offlineAssertSnapshot(t, outside, outsideBefore)
	offlineAssertSnapshot(t, source, sourceBefore)
	offlineAssertSnapshot(t, filepath.Dir(archive), archiveBefore)
	t.Log("PASS: actual post-creation stage alias makes cleanup refuse with retained private staging; no target is published and outside/source/archive bytes and descriptors remain unchanged")
}

func TestRestoreBoundaryForgedHandleWorkerCannotWrite(t *testing.T) {
	archive, source, _, _ := restoreBoundaryFixture(t)
	parent := t.TempDir()
	if e := os.WriteFile(filepath.Join(parent, "retained.bin"), []byte("forged worker sentinel"), 0600); e != nil {
		t.Fatal(e)
	}
	before := offlineSnapshot(t, parent)
	archiveBefore := offlineSnapshot(t, filepath.Dir(archive))
	sourceBefore := offlineSnapshot(t, source)
	exe, e := os.Executable()
	if e != nil {
		t.Fatal(e)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 12*time.Second)
	defer cancel()
	// Numeric zero supplies no inherited request-pipe authority. Run the real
	// pre-main entrypoint directly, without the supervisor's job or capabilities.
	command := exec.CommandContext(ctx, exe, "--immich-restore-handles-v2", "0")
	command.Dir = parent
	e = command.Run()
	if ctx.Err() != nil {
		t.Fatal("forged handle worker did not refuse promptly")
	}
	var exit *exec.ExitError
	if !errors.As(e, &exit) || exit.ExitCode() == 0 {
		t.Fatal("forged handle worker accepted absent supervisor authority", e)
	}
	offlineAssertSnapshot(t, parent, before)
	offlineAssertSnapshot(t, filepath.Dir(archive), archiveBefore)
	offlineAssertSnapshot(t, source, sourceBefore)
	t.Log("PASS: actual v2 pre-main invocation without a supervisor job or inherited request pipe refuses before any writes")
}

func TestRestoreBoundaryLateExistingTargetCannotBeReplaced(t *testing.T) {
	archive, source, _, _ := restoreBoundaryFixture(t)
	parent := t.TempDir()
	target := filepath.Join(parent, "late existing library")
	sourceBefore := offlineSnapshot(t, source)
	archiveBefore := offlineSnapshot(t, filepath.Dir(archive))
	called := false
	var targetBefore map[string]offlineObserved
	old := restoreBoundaryStageCreated
	restoreBoundaryStageCreated = func(stage string) error {
		called = true
		restoreBoundaryPrivateDirectory(t, stage, true)
		if e := os.Mkdir(target, 0700); e != nil {
			return e
		}
		if e := os.WriteFile(filepath.Join(target, "retained.bin"), []byte("late target original bytes"), 0600); e != nil {
			return e
		}
		targetBefore = offlineSnapshot(t, target)
		return nil
	}
	defer func() { restoreBoundaryStageCreated = old }()
	// The hook creates the real target after initial absence checks. Allow the
	// authentic child to extract; publication must still refuse replacement.
	e := restoreBoundaryCall(archive, target)
	if !called || e == nil {
		t.Fatal("late existing target did not refuse publication", e)
	}
	if !errors.Is(e, windows.STATUS_OBJECT_NAME_COLLISION) {
		t.Fatal("late target test did not reach native no-replace publication", e)
	}
	offlineAssertSnapshot(t, target, targetBefore)
	data, readErr := os.ReadFile(filepath.Join(target, "retained.bin"))
	if readErr != nil || string(data) != "late target original bytes" {
		t.Fatal("late target was changed", readErr)
	}
	entries, readErr := os.ReadDir(target)
	if readErr != nil || len(entries) != 1 || entries[0].Name() != "retained.bin" {
		t.Fatal("restored payload entered late target", readErr)
	}
	offlineAssertSnapshot(t, source, sourceBefore)
	offlineAssertSnapshot(t, filepath.Dir(archive), archiveBefore)
	t.Log("PASS: a real target created after initial checks survives authenticated worker completion and no-replace publication refusal")
}
