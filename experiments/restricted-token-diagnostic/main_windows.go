//go:build windows

package main

import (
	"encoding/json"
	"fmt"
	"os"
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
	PanicRecovered bool `json:"panic_recovered"`
	Error string `json:"error,omitempty"`
}

func writeReport(r report) {
	b, _ := json.MarshalIndent(r, "", "  ")
	_ = os.WriteFile("restricted-token-diagnostic.json", b, 0600)
	fmt.Println(string(b))
}

func main() {
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
	if err := windows.OpenProcessToken(windows.CurrentProcess(), windows.TOKEN_DUPLICATE|windows.TOKEN_QUERY, &base); err != nil {
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
	_ = out.Close()
	r.Stage = "complete"
	writeReport(r)
}

