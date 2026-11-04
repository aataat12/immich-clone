//go:build windows

package main

import (
	"errors"
	"unsafe"

	"golang.org/x/sys/windows"
)

// Elevated offline init would resolve state and diagnostic paths with installer
// authority. Initialization belongs to the existing LocalService startup instead.
// Query failure also refuses before diagnostics or any application state opens.
func checkInitializationMode(mode string) error {
	if mode != "init" {
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
