//go:build windows

package installerfiles

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"unsafe"

	"golang.org/x/sys/windows"
)

func setPrivacyDescriptor(t *testing.T, path, sddl string) {
	t.Helper()
	sd, err := windows.SecurityDescriptorFromString(sddl)
	if err != nil {
		t.Fatal(err)
	}
	acl, _, err := sd.DACL()
	if err != nil {
		t.Fatal(err)
	}
	if err = windows.SetNamedSecurityInfo(path, windows.SE_FILE_OBJECT,
		windows.DACL_SECURITY_INFORMATION|windows.PROTECTED_DACL_SECURITY_INFORMATION,
		nil, nil, acl, nil); err != nil {
		t.Fatal(err)
	}
}

// Establish the actual hostile child descriptor, rather than inferring it from
// the requested SDDL or its private parent's permissions. No raw ACL is printed.
func assertProtectedEveryoneRead(t *testing.T, path string) {
	t.Helper()
	sd, err := windows.GetNamedSecurityInfo(path, windows.SE_FILE_OBJECT, windows.DACL_SECURITY_INFORMATION)
	if err != nil {
		t.Fatal(err)
	}
	control, _, err := sd.Control()
	if err != nil || control&windows.SE_DACL_PROTECTED == 0 {
		t.Fatal("hostile child is not protected against parent inheritance")
	}
	acl, _, err := sd.DACL()
	if err != nil || acl == nil {
		t.Fatal("hostile child DACL unavailable")
	}
	found := false
	for i := uint32(0); i < uint32(acl.AceCount); i++ {
		var ace *windows.ACCESS_ALLOWED_ACE
		if err = windows.GetAce(acl, i, &ace); err != nil {
			t.Fatal(err)
		}
		if ace.Header.AceType == windows.ACCESS_ALLOWED_ACE_TYPE &&
			(*windows.SID)(unsafe.Pointer(&ace.SidStart)).String() == "S-1-1-0" &&
			uint32(ace.Mask)&0x80000001 != 0 {
			found = true
		}
	}
	if !found {
		t.Fatal("hostile child lacks the intended Everyone read grant")
	}
}

func assertPrivacyRefusal(t *testing.T, err error) {
	t.Helper()
	if !errors.Is(err, ErrRefused) {
		t.Fatal("actual-object privacy policy did not refuse the unsafe descriptor")
	}
}

func privacyBytes(t *testing.T, path string) []byte {
	t.Helper()
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return body
}

func TestProtectedReadableThirdSnapshotTargetRefusedBeforeTruncate(t *testing.T) {
	app, data, backup, _ := fixture(t)
	if err := Snapshot(app, data); err != nil {
		t.Fatal(err)
	}
	readyBefore := privacyBytes(t, filepath.Join(backup, readyName))
	writeLive(t, app, data, "latest-")
	third := filepath.Join(backup, names[2])
	setPrivacyDescriptor(t, third, "D:P(A;;FA;;;SY)(A;;FA;;;BA)(A;;GR;;;WD)")
	assertProtectedEveryoneRead(t, third)
	assertPrivacyRefusal(t, Snapshot(app, data))
	for _, name := range names {
		assertBytes(t, filepath.Join(backup, name), "baseline-"+name)
	}
	if !bytes.Equal(readyBefore, privacyBytes(t, filepath.Join(backup, readyName))) {
		t.Fatal("third-target privacy refusal changed prior snapshot readiness")
	}
	assertLive(t, app, data, "latest-")
	t.Log("PASS: protected Everyone-readable third snapshot target refused before earlier snapshot bytes or readiness changed")
}

func TestUnsafeSnapshotOrReceiptPrivacyRefusesRestoreBeforeLiveMutation(t *testing.T) {
	for _, name := range []string{readyName, names[2]} {
		t.Run(name, func(t *testing.T) {
			app, data, backup, _ := fixture(t)
			if err := Snapshot(app, data); err != nil {
				t.Fatal(err)
			}
			writeLive(t, app, data, "latest-")
			path := filepath.Join(backup, name)
			before := privacyBytes(t, path)
			setPrivacyDescriptor(t, path, "D:P(A;;FA;;;SY)(A;;FA;;;BA)(A;;GR;;;WD)")
			assertProtectedEveryoneRead(t, path)
			assertPrivacyRefusal(t, Restore(app, data))
			assertLive(t, app, data, "latest-")
			if !bytes.Equal(before, privacyBytes(t, path)) {
				t.Fatal("unsafe snapshot or receipt refusal changed source bytes")
			}
		})
	}
	t.Log("PASS: actual protected receipt and final snapshot-source privacy failures refuse restore before live mutation")
}

func TestRollbackRootWithoutRequiredInheritanceRefused(t *testing.T) {
	app, data, backup, _ := fixture(t)
	// Both privileged principals retain full self access, but only one principal
	// (or neither) inherits to new children. Root-self checks alone accept these.
	for _, sddl := range []string{
		"D:P(A;;FA;;;SY)(A;;FA;;;BA)",
		"D:P(A;OICI;FA;;;SY)(A;;FA;;;BA)",
		"D:P(A;;FA;;;SY)(A;OICI;FA;;;BA)",
	} {
		setPrivacyDescriptor(t, backup, sddl)
		assertPrivacyRefusal(t, Snapshot(app, data))
		assertLive(t, app, data, "baseline-")
		for _, name := range append(names[:], readyName) {
			if _, err := os.Stat(filepath.Join(backup, name)); !errors.Is(err, os.ErrNotExist) {
				t.Fatal("root inheritance refusal created a snapshot file or receipt")
			}
		}
	}
	t.Log("PASS: protected rollback roots missing either privileged principal's child inheritance refused before snapshot publication")
}

// Create one exact, process-visible DOS drive alias and remove only that same
// definition during cleanup. No existing mapping is overwritten or removed.
func privacyDriveAlias(t *testing.T, target string) string {
	t.Helper()
	mask, err := windows.GetLogicalDrives()
	if err != nil {
		t.Fatal(err)
	}
	var device *uint16
	var letter string
	for _, candidate := range "ZYXWVUTSRQ" {
		if mask&(uint32(1)<<uint(candidate-'A')) != 0 {
			continue
		}
		name, e := windows.UTF16PtrFromString(string(candidate) + ":")
		if e != nil {
			t.Fatal(e)
		}
		var current [512]uint16
		if _, e = windows.QueryDosDevice(name, &current[0], uint32(len(current))); !errors.Is(e, windows.ERROR_FILE_NOT_FOUND) {
			continue
		}
		device, letter = name, string(candidate)+":"
		break
	}
	if device == nil {
		t.Fatal("no unused drive alias available for required overlap control")
	}
	rawTarget, err := windows.UTF16PtrFromString(`\??\` + target)
	if err != nil {
		t.Fatal(err)
	}
	flags := uint32(windows.DDD_RAW_TARGET_PATH | windows.DDD_NO_BROADCAST_SYSTEM)
	if err = windows.DefineDosDevice(flags, device, rawTarget); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if e := windows.DefineDosDevice(flags|windows.DDD_REMOVE_DEFINITION|windows.DDD_EXACT_MATCH_ON_REMOVE, device, rawTarget); e != nil {
			t.Error("owned drive alias could not be removed exactly")
		}
	})
	return letter + `\`
}

func TestDriveAliasPhysicalRoleOverlapRefusedBeforeMutation(t *testing.T) {
	for _, appThroughAlias := range []bool{true, false} {
		label := "data-through-alias"
		if appThroughAlias {
			label = "application-through-alias"
		}
		t.Run(label, func(t *testing.T) {
			app, data, backup, _ := fixture(t)
			realApp, realData := app, data
			aliasParent := filepath.Join(app, "alias-parent")
			leaf := "state"
			if appThroughAlias {
				aliasParent = filepath.Join(data, "alias-parent")
				leaf = "application"
			}
			physicalRole := filepath.Join(aliasParent, leaf)
			if err := os.MkdirAll(physicalRole, 0700); err != nil {
				t.Fatal(err)
			}
			if appThroughAlias {
				realApp = physicalRole
				backup = filepath.Join(realApp, "rollback")
				if err := os.Mkdir(backup, 0700); err != nil {
					t.Fatal(err)
				}
				setPrivacyDescriptor(t, backup, "D:P(A;OICI;FA;;;SY)(A;OICI;FA;;;BA)")
			} else {
				realData = physicalRole
			}
			writeLive(t, realApp, realData, "aliased-")
			alias := filepath.Join(privacyDriveAlias(t, aliasParent), leaf)
			if appThroughAlias {
				app = alias
			} else {
				data = alias
			}
			// This must be a real bypass of lexical checks, not an ordinary nested
			// path refused before testing the actual held-object final paths.
			if within(app, data) || within(data, app) {
				t.Fatal("alias fixture did not bypass lexical overlap checks")
			}
			assertLive(t, realApp, realData, "aliased-")
			assertPrivacyRefusal(t, Snapshot(app, data))
			assertLive(t, realApp, realData, "aliased-")
			for _, name := range append(names[:], readyName) {
				if _, err := os.Stat(filepath.Join(backup, name)); !errors.Is(err, os.ErrNotExist) {
					t.Fatal("physically overlapping aliased roles created snapshot state")
				}
			}
		})
	}
	t.Log("PASS: actual DOS drive aliases bypass lexical overlap checks but physical application/data ancestry is refused before mutation")
}
