package library

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"golang.org/x/sys/windows"
)

// NTFS publication can make a name visible while a delete-access handle is
// still open. A reader must share DELETE, even though it only requests READ.
func TestRestoreProgressReadDuringDeleteAccessHandle(t *testing.T) {
	stage := t.TempDir()
	if e := writeRestoreProgress(stage, "handoff-nonce", 1); e != nil {
		t.Fatal(e)
	}
	path := filepath.Join(stage, restoreProgressName)
	name, e := windows.UTF16PtrFromString(path)
	if e != nil {
		t.Fatal(e)
	}
	h, e := windows.CreateFile(name, windows.DELETE, windows.FILE_SHARE_READ|windows.FILE_SHARE_WRITE|windows.FILE_SHARE_DELETE, nil, windows.OPEN_EXISTING, windows.FILE_ATTRIBUTE_NORMAL, 0)
	if e != nil {
		t.Fatal(e)
	}
	defer windows.CloseHandle(h)
	f, e := os.Open(path)
	if f != nil {
		f.Close()
	}
	if !errors.Is(e, windows.ERROR_SHARING_VIOLATION) {
		t.Fatal("baseline did not reproduce sharing violation", e)
	}
	b, e := readRestoreProgress(path)
	if e != nil {
		t.Fatalf("production progress read failed with baseline sharing conflict: %s", RestoreFailureDetail(restoreStageError("progress-read", e)))
	}
	if string(b) != "{\"Nonce\":\"handoff-nonce\",\"BaseBytes\":1}" {
		t.Fatal("progress bytes changed")
	}
	t.Log("PASS: baseline os.Open gives sharing violation 32; production reader succeeds with delete-access handle live")
}
