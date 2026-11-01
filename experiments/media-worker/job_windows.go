package main

// This bounds resources; it is not a filesystem or security-token sandbox.
// JOB_LIST assigns the child at creation, before it can execute any input.
import (
	"context"
	"errors"
	"fmt"
	"golang.org/x/sys/windows"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"sync/atomic"
	"time"
	"unicode/utf16"
	"unsafe"
)

var peakCommit atomic.Uint64

const jobListAttribute = 0x0002000d // PROC_THREAD_ATTRIBUTE_JOB_LIST, Windows 10+
const jobMemoryLimit = 512 << 20
const outputLimit = 1 << 20

var errOutputLimit = errors.New("child output exceeds limit")

type jobResult struct {
	output []byte
	peak   uintptr
	pid    uint32
}

func jobCommand(ctx context.Context, exe string, args []string, onStart func(uint32)) (result jobResult, err error) {
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
	limits := windows.JOBOBJECT_EXTENDED_LIMIT_INFORMATION{ProcessMemoryLimit: jobMemoryLimit, JobMemoryLimit: jobMemoryLimit}
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
	if e = attributes.Update(jobListAttribute, unsafe.Pointer(&jobs[0]), unsafe.Sizeof(jobs[0])); e != nil {
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
	result.pid = process.ProcessId
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
		b, e := io.ReadAll(io.LimitReader(r, outputLimit+1))
		if len(b) > outputLimit {
			e = errOutputLimit
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
	read := <-readDone
	if e = windows.QueryInformationJobObject(job, windows.JobObjectExtendedLimitInformation, uintptr(unsafe.Pointer(&limits)), uint32(unsafe.Sizeof(limits)), nil); e != nil {
		return result, e
	}
	result.peak = limits.PeakJobMemoryUsed
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
		return result, fmt.Errorf("child exit code %d", code)
	}
	result.output = read.b
	return result, nil
}
func command(exe string, args ...string) ([]byte, error) {
	return commandBounded(30*time.Second, exe, args...)
}
func commandBounded(budget time.Duration, exe string, args ...string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), budget)
	defer cancel()
	r, e := jobCommand(ctx, exe, args, nil)
	for previous := peakCommit.Load(); uint64(r.peak) > previous; previous = peakCommit.Load() {
		if peakCommit.CompareAndSwap(previous, uint64(r.peak)) {
			break
		}
	}
	return r.output, e
}
