//go:build windows

package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	advapi = windows.NewLazySystemDLL("advapi32.dll")
	createRestrictedToken = advapi.NewProc("CreateRestrictedToken")
)

type report struct {
	Stage string `json:"stage"`
	OpenProcessToken bool `json:"open_process_token"`
	SidCreated bool `json:"sid_created"`
	NativeReturned bool `json:"native_returned"`
	TokenCreated bool `json:"token_created"`
	ProcessCreated bool `json:"process_created"`
	ProcessExitedZero bool `json:"process_exited_zero"`
	ProcessExitCode uint32 `json:"process_exit_code"`
	PanicRecovered bool `json:"panic_recovered"`
	Error string `json:"error,omitempty"`
}

func writeReport(r report) {
	b, _ := json.MarshalIndent(r, "", "  ")
	_ = os.WriteFile("restricted-token-diagnostic.json", b, 0600)
	fmt.Println(string(b))
}

func main() {
	if len(os.Args) > 1 && os.Args[1] == "child" { return }
	r := report{Stage: "start"}
	defer func() {
		if p := recover(); p != nil {
			r.PanicRecovered = true
			r.Error = fmt.Sprint(p)
			writeReport(r)
		}
	}()

	var base windows.Token
	r.Stage = "open_process_token"
	if err := windows.OpenProcessToken(windows.CurrentProcess(), windows.TOKEN_ASSIGN_PRIMARY|windows.TOKEN_DUPLICATE|windows.TOKEN_QUERY, &base); err != nil {
		r.Error = err.Error(); writeReport(r); return
	}
	defer base.Close()
	r.OpenProcessToken = true

	r.Stage = "create_restricted_sid"
	sid, err := windows.StringToSid("S-1-5-12")
	if err != nil { r.Error = err.Error(); writeReport(r); return }
	r.SidCreated = true
	restricted := windows.SIDAndAttributes{Sid: sid}
	var out windows.Token

	r.Stage = "create_restricted_token_call"
	result, _, callErr := createRestrictedToken.Call(uintptr(base), 1, 0, 0, 0, 0, 1, uintptr(unsafe.Pointer(&restricted)), uintptr(unsafe.Pointer(&out)))
	r.NativeReturned = true
	runtime.KeepAlive(base)
	runtime.KeepAlive(sid)
	runtime.KeepAlive(restricted)
	if result == 0 {
		if callErr == syscall.Errno(0) { callErr = windows.GetLastError() }
		r.Error = callErr.Error(); writeReport(r); return
	}
	r.TokenCreated = true
	r.Stage = "create_process_as_user"
	exe, err := os.Executable()
	if err != nil { r.Error = err.Error(); _ = out.Close(); writeReport(r); return }
	app, err := windows.UTF16PtrFromString(exe)
	if err != nil { r.Error = err.Error(); _ = out.Close(); writeReport(r); return }
	line, err := windows.UTF16PtrFromString(windows.ComposeCommandLine([]string{exe, "child"}))
	if err != nil { r.Error = err.Error(); _ = out.Close(); writeReport(r); return }
	cwd, err := windows.UTF16PtrFromString(filepath.Dir(exe))
	if err != nil { r.Error = err.Error(); _ = out.Close(); writeReport(r); return }
	env := syscall.StringToUTF16Ptr("PATH=" + os.Getenv("PATH") + "\x00SystemRoot=" + os.Getenv("SystemRoot") + "\x00\x00")
	startup := windows.StartupInfo{Cb: uint32(unsafe.Sizeof(windows.StartupInfo{}))}
	var pi windows.ProcessInformation
	if err = windows.CreateProcessAsUser(out, app, line, nil, nil, false, windows.CREATE_NO_WINDOW|windows.CREATE_UNICODE_ENVIRONMENT, env, cwd, &startup, &pi); err != nil { r.Error = err.Error(); _ = out.Close(); writeReport(r); return }
	r.ProcessCreated = true
	_, _ = windows.WaitForSingleObject(pi.Process, 5000)
	var code uint32
	if windows.GetExitCodeProcess(pi.Process, &code) == nil { r.ProcessExitCode = code; r.ProcessExitedZero = code == 0 }
	_ = windows.CloseHandle(pi.Process)
	_ = windows.CloseHandle(pi.Thread)
	_ = out.Close()
	r.Stage = "complete"
	writeReport(r)
}

