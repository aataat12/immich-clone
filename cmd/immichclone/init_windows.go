//go:build windows

package main

import (
	"errors"
	"unsafe"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/svc"
)

// Elevated offline init would resolve state and diagnostic paths with installer
// authority. Initialization belongs to the existing LocalService startup instead.
// Query failure also refuses before diagnostics or any application state opens.
func checkInitializationMode(mode string) error {
	if mode == "service" {
		user, e := windows.GetCurrentProcessToken().GetTokenUser()
		if e != nil {
			return e
		}
		if user.User.Sid.String() != "S-1-5-19" {
			return errors.New("service account refused")
		}
		managed, e := svc.IsWindowsService()
		if e != nil {
			return e
		}
		if !managed {
			return errors.New("direct service mode refused")
		}
		return nil
	}
	if mode != "init" && mode != "serve" {
		return nil
	}
	var elevated uint32
	var size uint32
	if e := windows.GetTokenInformation(windows.GetCurrentProcessToken(), windows.TokenElevation, (*byte)(unsafe.Pointer(&elevated)), uint32(unsafe.Sizeof(elevated)), &size); e != nil {
		return e
	}
	if size != uint32(unsafe.Sizeof(elevated)) || elevated != 0 {
		return errors.New("elevated initialization refused")
	}
	return nil
}
