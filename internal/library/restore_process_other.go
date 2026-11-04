//go:build !windows

package library

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"os/exec"
	"sync"
	"syscall"
)

type restoreCapture struct {
	mu       sync.Mutex
	b        bytes.Buffer
	limit    int64
	exceeded bool
	kill     func()
}

func (w *restoreCapture) Write(p []byte) (int, error) {
	w.mu.Lock()
	remaining := w.limit - int64(w.b.Len())
	if int64(len(p)) > remaining {
		if remaining > 0 {
			_, _ = w.b.Write(p[:int(remaining)])
		}
		w.exceeded = true
		w.mu.Unlock()
		if w.kill != nil {
			w.kill()
		}
		return 0, ErrRestoreOutputLimit
	}
	n, e := w.b.Write(p)
	w.mu.Unlock()
	return n, e
}

// Linux is a development/test host. Production Windows uses a Job Object for a
// hard memory cap. Both hosts kill and reap the separate worker on cancellation.
func runRestoreProcess(ctx context.Context, exe string, args []string, memoryBytes uint64, outputBytes int64, onStart func(uint32)) (result restoreProcessResult, err error) {
	if e := ctx.Err(); e != nil {
		return result, e
	}
	if memoryBytes < 32<<20 || outputBytes < 1 {
		return result, ErrRestoreBudget
	}
	cmd := exec.Command(exe, args...)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Env = []string{"PATH=" + os.Getenv("PATH"), "SystemRoot=" + os.Getenv("SystemRoot")}
	capture := &restoreCapture{limit: outputBytes}
	capture.kill = func() {
		if cmd.Process != nil {
			_ = syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
		}
	}
	cmd.Stdout = capture
	cmd.Stderr = io.Discard
	if e := cmd.Start(); e != nil {
		return result, e
	}
	result.PID = uint32(cmd.Process.Pid)
	if onStart != nil {
		onStart(result.PID)
	}
	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()
	select {
	case err = <-done:
	case <-ctx.Done():
		capture.kill()
		err = <-done
		if err == nil {
			err = ctx.Err()
		}
		if ctx.Err() != nil {
			err = ctx.Err()
		}
	}
	result.Reaped = true
	capture.mu.Lock()
	exceeded := capture.exceeded
	result.Output = append([]byte(nil), capture.b.Bytes()...)
	capture.mu.Unlock()
	if exceeded {
		return result, ErrRestoreOutputLimit
	}
	if err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return result, err
		}
		return result, errRestoreWorker
	}
	return result, nil
}
