//go:build windows

package media

// RunRestricted is an opt-in candidate launcher. The caller must prepare ACLs
// for the restricted SID (S-1-5-12); the token alone is not a filesystem jail.
// It deliberately shares the existing output/deadline contract while using a
// restricted primary token for the child.
import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

const (
	disableMaxPrivilege = 1
	tokenPrimary        = 1
	securityImpersonation = 2
)

var (
	advapi = windows.NewLazySystemDLL("advapi32.dll")
	createRestrictedToken = advapi.NewProc("CreateRestrictedToken")
)

func restrictedToken() (token windows.Token, err error) {
	defer func() { if recovered := recover(); recovered != nil { token = 0; err = fmt.Errorf("token panic: %v", recovered) } }()
	var base windows.Token
	if err := windows.OpenProcessToken(windows.CurrentProcess(), windows.TOKEN_DUPLICATE|windows.TOKEN_QUERY, &base); err != nil { return 0, fmt.Errorf("token: %w", err) }
	defer base.Close()
	// S-1-5-12 is the documented Restricted Code SID. Access checks for a
	// restricted token must succeed for both the normal token and this SID.
	sid, err := windows.StringToSid("S-1-5-12")
	if err != nil { return 0, fmt.Errorf("token: %w", err) }
	restricted := windows.SIDAndAttributes{Sid: sid}
	var out windows.Token
	r, _, callErr := createRestrictedToken.Call(
		uintptr(base), disableMaxPrivilege,
		0, 0, 0, 0, 1, uintptr(unsafe.Pointer(&restricted)), uintptr(unsafe.Pointer(&out)))
	runtime.KeepAlive(base)
	runtime.KeepAlive(sid)
	runtime.KeepAlive(restricted)
	if r == 0 {
		if callErr != syscall.Errno(0) { return 0, fmt.Errorf("token: %w", callErr) }
		return 0, fmt.Errorf("token: %w", windows.GetLastError())
	}
	return out, nil
}

func RunRestricted(ctx context.Context, exe string, args []string, onStart func(uint32)) (result Result, err error) {
	defer func() { if recovered := recover(); recovered != nil { result = Result{}; err = fmt.Errorf("restricted run panic: %v", recovered) } }()
	if !filepath.IsAbs(exe) { return result, errors.New("absolute executable required") }
	if err := ctx.Err(); err != nil { return result, err }
	token, err := restrictedToken()
	if err != nil { return result, fmt.Errorf("token: %w", err) }
	defer token.Close()
	r, w, err := os.Pipe(); if err != nil { return result, err }
	defer r.Close(); defer w.Close()
	nul, err := os.OpenFile("NUL", os.O_RDWR, 0); if err != nil { return result, err }
	defer nul.Close()
	for _, h := range []windows.Handle{windows.Handle(w.Fd()), windows.Handle(nul.Fd())} {
		if err = windows.SetHandleInformation(h, windows.HANDLE_FLAG_INHERIT, windows.HANDLE_FLAG_INHERIT); err != nil { return result, err }
	}
	line, err := windows.UTF16PtrFromString(windows.ComposeCommandLine(append([]string{exe}, args...))); if err != nil { return result, err }
	app, err := windows.UTF16PtrFromString(exe); if err != nil { return result, err }
	cwd, err := windows.UTF16PtrFromString(filepath.Dir(exe)); if err != nil { return result, err }
	env := syscall.StringToUTF16Ptr("PATH=" + filepath.Dir(exe) + "\x00SystemRoot=" + os.Getenv("SystemRoot") + "\x00\x00")
	startup := windows.StartupInfo{Cb: uint32(unsafe.Sizeof(windows.StartupInfo{})), Flags: windows.STARTF_USESTDHANDLES, StdInput: windows.Handle(nul.Fd()), StdOutput: windows.Handle(w.Fd()), StdErr: windows.Handle(nul.Fd())}
	var pi windows.ProcessInformation
	if err = windows.CreateProcessAsUser(token, app, line, nil, nil, true, windows.CREATE_NO_WINDOW|windows.CREATE_UNICODE_ENVIRONMENT, env, cwd, &startup, &pi); err != nil { return result, fmt.Errorf("process: %w", err) }
	job, jobErr := windows.CreateJobObject(nil, nil)
	if jobErr != nil { windows.TerminateProcess(pi.Process, 1); return result, fmt.Errorf("job: %w", jobErr) }
	defer windows.CloseHandle(job)
	limits := windows.JOBOBJECT_EXTENDED_LIMIT_INFORMATION{ProcessMemoryLimit: MemoryLimit, JobMemoryLimit: MemoryLimit}
	limits.BasicLimitInformation.LimitFlags = windows.JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE | windows.JOB_OBJECT_LIMIT_ACTIVE_PROCESS | windows.JOB_OBJECT_LIMIT_PROCESS_MEMORY | windows.JOB_OBJECT_LIMIT_JOB_MEMORY
	limits.BasicLimitInformation.ActiveProcessLimit = 1
	if _, jobErr = windows.SetInformationJobObject(job, windows.JobObjectExtendedLimitInformation, uintptr(unsafe.Pointer(&limits)), uint32(unsafe.Sizeof(limits))); jobErr != nil { windows.TerminateProcess(pi.Process, 1); return result, fmt.Errorf("job: %w", jobErr) }
	if jobErr = windows.AssignProcessToJobObject(job, pi.Process); jobErr != nil { windows.TerminateProcess(pi.Process, 1); return result, fmt.Errorf("job: %w", jobErr) }
	defer windows.CloseHandle(pi.Process); windows.CloseHandle(pi.Thread); w.Close()
	result.PID = pi.ProcessId; if onStart != nil { onStart(pi.ProcessId) }
	done := make(chan struct{}); var output []byte; var readErr error
	go func(){ output, readErr = io.ReadAll(io.LimitReader(r, OutputLimit+1)); close(done) }()
	for { state, waitErr := windows.WaitForSingleObject(pi.Process, 50); if waitErr != nil { return result, waitErr }; if state == windows.WAIT_OBJECT_0 { break }; if err = ctx.Err(); err != nil { _ = windows.TerminateProcess(pi.Process, 1); _, _ = windows.WaitForSingleObject(pi.Process, 5000); <-done; return result, err } }
	<-done; if readErr != nil { return result, readErr }; if int64(len(output)) > OutputLimit { return result, errOutputLimit }
	if err = windows.QueryInformationJobObject(job, windows.JobObjectExtendedLimitInformation, uintptr(unsafe.Pointer(&limits)), uint32(unsafe.Sizeof(limits)), nil); err != nil { return result, err }; result.Peak = limits.PeakJobMemoryUsed
	var code uint32; if err = windows.GetExitCodeProcess(pi.Process, &code); err != nil { return result, err }; if code != 0 { return result, &ChildExitError{Code: code} }; result.Output = output
	return result, nil
}

