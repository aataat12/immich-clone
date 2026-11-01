package main

import (
	"context"
	"errors"
	"fmt"
	"golang.org/x/sys/windows"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

func helperMode(mode, root string) {
	self, _ := os.Executable()
	switch mode {
	case "memory":
		baseline, e := windows.VirtualAlloc(0, 64<<20, windows.MEM_COMMIT|windows.MEM_RESERVE, windows.PAGE_READWRITE)
		if e != nil {
			os.Exit(31)
		}
		defer windows.VirtualFree(baseline, 0, windows.MEM_RELEASE)
		allocation, e := windows.VirtualAlloc(0, 600<<20, windows.MEM_COMMIT|windows.MEM_RESERVE, windows.PAGE_READWRITE)
		if e != nil {
			fmt.Print("large_allocation_refused")
			return
		}
		windows.VirtualFree(allocation, 0, windows.MEM_RELEASE)
		fmt.Print("large_allocation_allowed")
	case "spawn":
		cmd := exec.Command(self, "-probe-stall")
		if e := cmd.Start(); e != nil {
			fmt.Print("child_refused")
			return
		}
		cmd.Process.Kill()
		cmd.Wait()
		fmt.Print("child_allowed")
	case "crash":
		os.Exit(42)
	case "parent-death":
		// os.Exit bypasses Go defers: Windows closing the parent's job handle must kill the child.
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_, e := jobCommand(ctx, self, []string{"-probe-stall"}, func(pid uint32) {
			if os.WriteFile(filepath.Join(root, "orphan-pid.txt"), []byte(strconv.FormatUint(uint64(pid), 10)), 0600) != nil {
				os.Exit(32)
			}
			os.Exit(0)
		})
		if e != nil {
			os.Exit(33)
		}
	default:
		os.Exit(34)
	}
}
func containmentChecks(root string) error {
	self, e := os.Executable()
	if e != nil {
		return e
	}
	// Controls establish that the disposable runner normally permits both operations.
	for _, c := range []struct{ mode, control, bounded string }{{"memory", "large_allocation_allowed", "large_allocation_refused"}, {"spawn", "child_allowed", "child_refused"}} {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		cmd := exec.CommandContext(ctx, self, "-probe-helper", c.mode)
		plain, e := cmd.Output()
		cancel()
		if e != nil || string(plain) != c.control {
			return errors.New("unrestricted containment control failed")
		}
		ctx, cancel = context.WithTimeout(context.Background(), 10*time.Second)
		result, e := jobCommand(ctx, self, []string{"-probe-helper", c.mode}, nil)
		cancel()
		if e != nil || string(result.output) != c.bounded || result.peak == 0 || result.peak > jobMemoryLimit {
			return errors.New("job memory/process enforcement failed")
		}
	}
	if _, e = commandBounded(5*time.Second, self, "-probe-helper", "crash"); e == nil {
		return errors.New("child crash reported success")
	}
	// Existing media commands immediately after these tests prove the service remains usable.
	parent := exec.Command(self, "-probe-helper", "parent-death", "-root", root)
	if e = parent.Run(); e != nil {
		return errors.New("parent death helper failed")
	}
	pidBytes, e := os.ReadFile(filepath.Join(root, "orphan-pid.txt"))
	if e != nil {
		return e
	}
	pid, e := strconv.ParseUint(strings.TrimSpace(string(pidBytes)), 10, 32)
	if e != nil {
		return e
	}
	h, e := windows.OpenProcess(windows.SYNCHRONIZE, false, uint32(pid))
	if errors.Is(e, windows.ERROR_INVALID_PARAMETER) {
		return nil
	}
	if e != nil {
		return e
	}
	defer windows.CloseHandle(h)
	state, e := windows.WaitForSingleObject(h, 5000)
	if e != nil || state != windows.WAIT_OBJECT_0 {
		return errors.New("child survived parent termination")
	}
	return nil
}
