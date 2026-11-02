package library

// This bounds resources; it is not a filesystem or security-token sandbox.
// JOB_LIST assigns the child at creation, before it can execute any input.
import (
	"context"
	"errors"
	"golang.org/x/sys/windows"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"unicode/utf16"
	"unsafe"
)

const restoreJobListAttribute = 0x0002000d // PROC_THREAD_ATTRIBUTE_JOB_LIST, Windows 10+

func runRestoreProcess(ctx context.Context, exe string, args []string, memoryBytes uint64, outputBytes int64, onStart func(uint32)) (result restoreProcessResult, err error) {
	if memoryBytes < 32<<20 || outputBytes < 1 {
		return result, ErrRestoreBudget
	}
	if !filepath.IsAbs(exe) {
		return result, errors.New("absolute executable required")
	}
	if e := ctx.Err(); e != nil {
		return result, e
	}
	job, e := windows.CreateJobObject(nil, nil)
	if e != nil {
		return result, e
	}
	defer windows.CloseHandle(job)
	limits := windows.JOBOBJECT_EXTENDED_LIMIT_INFORMATION{ProcessMemoryLimit: uintptr(memoryBytes), JobMemoryLimit: uintptr(memoryBytes)}
	limits.BasicLimitInformation.LimitFlags = windows.JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE | windows.JOB_OBJECT_LIMIT_ACTIVE_PROCESS | windows.JOB_OBJECT_LIMIT_PROCESS_MEMORY | windows.JOB_OBJECT_LIMIT_JOB_MEMORY
	limits.BasicLimitInformation.ActiveProcessLimit = 1
	if _, e = windows.SetInformationJobObject(job, windows.JobObjectExtendedLimitInformation, uintptr(unsafe.Pointer(&limits)), uint32(unsafe.Sizeof(limits))); e != nil {
		return result, e
	}
	r, w, e := os.Pipe()
	if e != nil {
		return result, e
	}
	defer r.Close()
	defer w.Close()
	nul, e := os.OpenFile("NUL", os.O_RDWR, 0)
	if e != nil {
		return result, e
	}
	defer nul.Close()
	inherited := []windows.Handle{windows.Handle(w.Fd()), windows.Handle(nul.Fd())}
	for _, h := range inherited {
		if e = windows.SetHandleInformation(h, windows.HANDLE_FLAG_INHERIT, windows.HANDLE_FLAG_INHERIT); e != nil {
			return result, e
		}
	}
	attributes, e := windows.NewProcThreadAttributeList(2)
	if e != nil {
		return result, e
	}
	defer attributes.Delete()
	jobs := []windows.Handle{job}
	if e = attributes.Update(restoreJobListAttribute, unsafe.Pointer(&jobs[0]), unsafe.Sizeof(jobs[0])); e != nil {
		return result, e
	}
	if e = attributes.Update(windows.PROC_THREAD_ATTRIBUTE_HANDLE_LIST, unsafe.Pointer(&inherited[0]), uintptr(len(inherited))*unsafe.Sizeof(inherited[0])); e != nil {
		return result, e
	}
	startup := windows.StartupInfoEx{ProcThreadAttributeList: attributes.List()}
	startup.Cb = uint32(unsafe.Sizeof(startup))
	startup.Flags = windows.STARTF_USESTDHANDLES
	startup.StdInput = inherited[1]
	startup.StdOutput = inherited[0]
	startup.StdErr = inherited[1]
	application, e := windows.UTF16PtrFromString(exe)
	if e != nil {
		return result, e
	}
	line, e := windows.UTF16PtrFromString(windows.ComposeCommandLine(append([]string{exe}, args...)))
	if e != nil {
		return result, e
	}
	cwd, e := windows.UTF16PtrFromString(filepath.Dir(exe))
	if e != nil {
		return result, e
	}
	environment := utf16.Encode([]rune("PATH=" + filepath.Dir(exe) + "\x00SystemRoot=" + os.Getenv("SystemRoot") + "\x00\x00"))
	var process windows.ProcessInformation
	e = windows.CreateProcess(application, line, nil, nil, true, windows.EXTENDED_STARTUPINFO_PRESENT|windows.CREATE_NO_WINDOW|windows.CREATE_UNICODE_ENVIRONMENT, &environment[0], cwd, &startup.StartupInfo, &process)
	runtime.KeepAlive(environment)
	runtime.KeepAlive(jobs)
	runtime.KeepAlive(inherited)
	if e != nil {
		return result, e
	}
	defer windows.CloseHandle(process.Process)
	windows.CloseHandle(process.Thread)
	result.PID = process.ProcessId
	w.Close()
	if onStart != nil {
		onStart(process.ProcessId)
	}
	type readResult struct {
		b []byte
		e error
	}
	readDone := make(chan readResult, 1)
	go func() {
		b, e := io.ReadAll(io.LimitReader(r, outputBytes+1))
		if int64(len(b)) > outputBytes {
			e = ErrRestoreOutputLimit
			_ = windows.TerminateJobObject(job, 1)
		}
		readDone <- readResult{b, e}
	}()
	for {
		state, e := windows.WaitForSingleObject(process.Process, 50)
		if e != nil {
			_ = windows.TerminateJobObject(job, 1)
			r.Close()
			<-readDone
			return result, e
		}
		if state == windows.WAIT_OBJECT_0 {
			break
		}
		if e = ctx.Err(); e != nil {
			_ = windows.TerminateJobObject(job, 1)
			state, waitErr := windows.WaitForSingleObject(process.Process, 5000)
			if waitErr != nil || state != windows.WAIT_OBJECT_0 {
				r.Close()
				<-readDone
				return result, errors.New("child termination not confirmed")
			}
			err = e
			break
		}
	}
	result.Reaped = true
	read := <-readDone
	if e = windows.QueryInformationJobObject(job, windows.JobObjectExtendedLimitInformation, uintptr(unsafe.Pointer(&limits)), uint32(unsafe.Sizeof(limits)), nil); e != nil {
		return result, e
	}
	result.Peak = uint64(limits.PeakJobMemoryUsed)
	if err != nil {
		return result, err
	}
	if read.e != nil {
		return result, read.e
	}
	var code uint32
	if e = windows.GetExitCodeProcess(process.Process, &code); e != nil {
		return result, e
	}
	if code != 0 {
		return result, restoreWorkerFailure(code)
	}
	result.Output = read.b
	return result, nil
}
