package diagnostics

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
)

func TestPrivacyErrorCodesAndRotation(t *testing.T) {
	root := t.TempDir()
	if e := Init(root, "0.1.3-prototype", "application"); e != nil {
		t.Fatal(e)
	}
	private := "PRIVATE_MACHINE_SENTINEL PRIVATE_PATH_SENTINEL PRIVATE_TOKEN_SENTINEL"
	Record(private, private, &os.PathError{Op: "open", Path: private, Err: syscall.EACCES}, 403, 123, 0)
	Record("upload", "original_write", errors.New(private), 500, 0, 0)
	var b bytes.Buffer
	if e := Export(&b); e != nil {
		t.Fatal(e)
	}
	if strings.Contains(b.String(), "PRIVATE_") {
		t.Fatal("sensitive input reached diagnostic log")
	}
	if !strings.Contains(b.String(), "permission_denied") || !strings.Contains(b.String(), "native_code") || !strings.Contains(b.String(), "http_status") {
		t.Fatal("missing useful error detail")
	}
	path := filepath.Join(root, "application.jsonl")
	if e := os.WriteFile(path, bytes.Repeat([]byte("x"), int(Limit)), 0600); e != nil {
		t.Fatal(e)
	}
	Record("startup", "ready", nil, 0, 0, 0)
	if info, e := os.Stat(path + ".previous"); e != nil || info.Size() != Limit {
		t.Fatal("rotation did not retain previous file")
	}
	if info, e := os.Stat(path); e != nil || info.Size() > Limit {
		t.Fatal("active file exceeded cap")
	}
	global.Lock()
	global.root = ""
	global.Unlock()
}
func TestLoggingFailureDoesNotPanic(t *testing.T) {
	root := t.TempDir()
	if e := Init(root, "0.1.3-prototype", "application"); e != nil {
		t.Fatal(e)
	}
	os.RemoveAll(root)
	Record("upload", "failed", syscall.ENOSPC, 500, 0, 0)
	global.Lock()
	failed := global.failed
	global.root = ""
	global.Unlock()
	if !failed {
		t.Fatal("log write failure not surfaced")
	}
}

type testChildExit struct{}

func (testChildExit) Error() string          { return "private path and stderr must not be logged" }
func (testChildExit) NativeExitCode() uint32 { return 0xc000001d }
func TestNativeChildExitClassification(t *testing.T) {
	category, code := classify(testChildExit{})
	if category != "native_child_exit" || code != 0xc000001d {
		t.Fatal("lost numeric native failure", category, code)
	}
}
