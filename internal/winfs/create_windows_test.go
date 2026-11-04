//go:build windows

package winfs

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"golang.org/x/sys/windows"
)

func TestCreateDirectoriesPinsAndRetainsSafePartialProgress(t *testing.T) {
	base := t.TempDir()
	first := filepath.Join(base, "created é")
	target := filepath.Join(first, "second")
	sd, e := windows.SecurityDescriptorFromString("D:P(A;OICI;FA;;;SY)(A;OICI;FA;;;BA)")
	if e != nil {
		t.Fatal(e)
	}
	injected := errors.New("synthetic verification interruption")
	count := 0
	_, e = CreateDirectories(target, sd, func(d *Directory) error {
		count++
		if e := os.Rename(first, first+"-moved"); e == nil {
			t.Fatal("created ancestor renamed while held")
		}
		if e := os.Rename(base, base+"-moved"); e == nil {
			t.Fatal("existing ancestor renamed while held")
		}
		actual, e := d.SecurityDescriptor()
		if e != nil {
			return e
		}
		control, _, e := actual.Control()
		if e != nil {
			return e
		}
		if control&windows.SE_DACL_PROTECTED == 0 {
			t.Fatal("new object not protected at verification")
		}
		if count == 2 {
			return injected
		}
		return nil
	})
	if !errors.Is(e, injected) || count != 2 {
		t.Fatal("required partial progress/interruption not exercised", e)
	}
	for _, p := range []string{first, target} {
		info, e := os.Stat(p)
		if e != nil || !info.IsDir() {
			t.Fatal("safe created directory removed after failure", e)
		}
	}
	if e := os.Rename(first, first+"-moved"); e != nil {
		t.Fatal("failure leaked pinned handles", e)
	}
	t.Log("PASS: creation holds new/existing ancestors during verification; interrupted verification retains protected partial progress and releases handles")
}
