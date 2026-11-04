//go:build windows

package winfs

import (
	"golang.org/x/sys/windows"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
)

func fixture(t *testing.T) (string, string) {
	t.Helper()
	var token windows.Token
	if e := windows.OpenProcessToken(windows.CurrentProcess(), windows.TOKEN_ADJUST_PRIVILEGES|windows.TOKEN_QUERY, &token); e != nil {
		t.Fatal(e)
	}
	defer token.Close()
	name, e := windows.UTF16PtrFromString("SeCreateSymbolicLinkPrivilege")
	if e != nil {
		t.Fatal(e)
	}
	var privileges windows.Tokenprivileges
	privileges.PrivilegeCount = 1
	if e = windows.LookupPrivilegeValue(nil, name, &privileges.Privileges[0].Luid); e != nil {
		t.Fatal(e)
	}
	privileges.Privileges[0].Attributes = windows.SE_PRIVILEGE_ENABLED
	if e = windows.AdjustTokenPrivileges(token, false, &privileges, 0, nil, nil); e != nil {
		t.Fatal(e)
	}
	base := t.TempDir()
	inside, outside := filepath.Join(base, "managed é"), filepath.Join(base, "outside")
	for _, p := range []string{inside, outside} {
		if err := os.Mkdir(p, 0700); err != nil {
			t.Fatal(err)
		}
	}
	return inside, outside
}
func contents(t *testing.T, p, value string) {
	t.Helper()
	b, e := os.ReadFile(p)
	if e != nil || string(b) != value {
		t.Fatal("fixture bytes changed or unreadable")
	}
}

func TestActualCopyAndUnsafeAliases(t *testing.T) {
	inside, outside := fixture(t)
	for _, p := range []string{filepath.Join(inside, "source"), filepath.Join(outside, "sentinel")} {
		if e := os.WriteFile(p, []byte("unchanged"), 0600); e != nil {
			t.Fatal(e)
		}
	}
	// Demonstrate that the same fixture really permits outside writes through a
	// hard link; then restore bytes before invoking the candidate boundary.
	alias := filepath.Join(inside, "hardlink")
	if e := os.Link(filepath.Join(outside, "sentinel"), alias); e != nil {
		t.Fatal(e)
	}
	if e := os.WriteFile(alias, []byte("control"), 0600); e != nil {
		t.Fatal(e)
	}
	contents(t, filepath.Join(outside, "sentinel"), "control")
	if e := os.WriteFile(filepath.Join(outside, "sentinel"), []byte("unchanged"), 0600); e != nil {
		t.Fatal(e)
	}
	d, e := OpenDirectory(inside)
	if e != nil {
		t.Fatal(e)
	}
	defer d.Close()
	if e = Copy(d, "destination", d, "source"); e != nil {
		t.Fatal(e)
	}
	contents(t, filepath.Join(inside, "destination"), "unchanged")
	if e = Copy(d, "hardlink", d, "source"); e == nil {
		t.Fatal("hard-linked target accepted")
	}
	if e = Copy(d, "destination", d, "hardlink"); e == nil {
		t.Fatal("hard-linked source accepted")
	}
	for _, n := range []string{"..", `..\escape`, `C:\escape`, "CON", "file:stream", "destination.", "destination "} {
		if f, e := d.OpenFile(n, true); e == nil {
			f.Close()
			t.Fatal("unsafe name accepted")
		}
	}
	if e = os.Symlink(filepath.Join(outside, "sentinel"), filepath.Join(inside, "symlink")); e != nil {
		t.Fatal(e)
	}
	if e = Copy(d, "symlink", d, "source"); e == nil {
		t.Fatal("symbolic target accepted")
	}
	contents(t, filepath.Join(outside, "sentinel"), "unchanged")
	contents(t, filepath.Join(inside, "destination"), "unchanged")
	t.Log("PASS: actual Unicode/space-path copy; hard-link positive control and source/target refusal; symlink/name/device/ADS refusal; outside bytes unchanged")
}

func TestPinnedDirectoryAndAncestorReplacement(t *testing.T) {
	inside, outside := fixture(t)
	slot := filepath.Join(inside, "slot")
	moved := slot + "-moved"
	if e := os.Mkdir(slot, 0700); e != nil {
		t.Fatal(e)
	}
	if e := os.Rename(slot, moved); e != nil {
		t.Fatal(e)
	}
	if e := os.Rename(moved, slot); e != nil {
		t.Fatal(e)
	}
	d, e := OpenDirectory(slot)
	if e != nil {
		t.Fatal(e)
	}
	for i := 0; i < 100; i++ {
		if e = os.Rename(slot, moved); e == nil {
			t.Fatal("pinned leaf renamed")
		}
		if e = os.Rename(inside, inside+"-moved"); e == nil {
			t.Fatal("pinned ancestor renamed")
		}
	}
	if e = d.Close(); e != nil {
		t.Fatal(e)
	}
	if e = os.Rename(slot, moved); e != nil {
		t.Fatal("guard did not release namespace pin")
	}
	if e = os.Symlink(outside, slot); e != nil {
		t.Fatal(e)
	}
	if redirected, e := OpenDirectory(slot); e == nil {
		redirected.Close()
		t.Fatal("redirected directory accepted")
	}
	t.Log("PASS: real rename positive controls; 200 leaf/ancestor replacement refusals while pinned; release and linked-directory refusal")
}

func TestConcurrentLeafSwapCannotRedirectWrites(t *testing.T) {
	inside, outside := fixture(t)
	path := filepath.Join(outside, "sentinel")
	if e := os.WriteFile(path, []byte("outside"), 0600); e != nil {
		t.Fatal(e)
	}
	leaf := filepath.Join(inside, "slot")
	// Preflight the real symbolic-link operation before the concurrent actor.
	if e := os.Symlink(path, leaf); e != nil {
		t.Fatal(e)
	}
	if e := os.Remove(leaf); e != nil {
		t.Fatal(e)
	}
	d, e := OpenDirectory(inside)
	if e != nil {
		t.Fatal(e)
	}
	defer d.Close()
	var stop atomic.Bool
	var swaps atomic.Uint64
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		for !stop.Load() {
			_ = os.Remove(leaf)
			if os.Symlink(path, leaf) == nil {
				swaps.Add(1)
			}
			_ = os.Remove(leaf)
			if f, e := os.OpenFile(leaf, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600); e == nil {
				_, _ = f.WriteString("inside")
				f.Close()
			}
		}
	}()
	accepted, refused := 0, 0
	for i := 0; i < 2000; i++ {
		f, e := d.OpenFile("slot", true)
		if e != nil {
			refused++
			continue
		}
		if e = f.Truncate(0); e == nil {
			_, e = f.WriteString("bounded")
		}
		f.Close()
		if e != nil {
			stop.Store(true)
			wg.Wait()
			t.Fatal("validated file write failed")
		}
		accepted++
	}
	stop.Store(true)
	wg.Wait()
	if swaps.Load() == 0 {
		t.Fatal("concurrent link actor never replaced a leaf")
	}
	if accepted == 0 || refused == 0 {
		t.Fatal("race fixture did not exercise both outcomes")
	}
	contents(t, path, "outside")
	t.Logf("PASS: 2000 actual concurrent write attempts, accepted=%d refused=%d completed_link_swaps=%d; outside bytes unchanged", accepted, refused, swaps.Load())
}
