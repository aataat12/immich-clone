//go:build windows

// Package winfs provides candidate handle-bound filesystem operations. It is not
// used by the application or installer until their integration gates pass.
package winfs

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"unsafe"

	"golang.org/x/sys/windows"
)

var ErrUnsafeObject = errors.New("filesystem boundary refused unsafe object")

// Directory pins each ancestor and the selected directory against deletion,
// replacement and reparse conversion. Names are opened one component at a time
// relative to the preceding handle, never by re-resolving a checked string path.
type Directory struct {
	mu    sync.RWMutex
	holds []windows.Handle
}

func component(name string) bool {
	return name != "." && name != ".." && name != "" &&
		!strings.ContainsAny(name, `\/:`+"\x00") &&
		!strings.HasSuffix(name, ".") && !strings.HasSuffix(name, " ") && filepath.IsLocal(name)
}

func checked(h windows.Handle, directory bool) error {
	var info windows.ByHandleFileInformation
	if err := windows.GetFileInformationByHandle(h, &info); err != nil {
		return err
	}
	if info.FileAttributes&windows.FILE_ATTRIBUTE_REPARSE_POINT != 0 ||
		(info.FileAttributes&windows.FILE_ATTRIBUTE_DIRECTORY != 0) != directory ||
		(!directory && info.NumberOfLinks != 1) {
		return ErrUnsafeObject
	}
	return nil
}

func relative(parent windows.Handle, name string, directory, write bool) (windows.Handle, error) {
	if !component(name) {
		return 0, ErrUnsafeObject
	}
	u, err := windows.NewNTUnicodeString(name)
	if err != nil {
		return 0, err
	}
	oa := windows.OBJECT_ATTRIBUTES{RootDirectory: parent, ObjectName: u, Attributes: windows.OBJ_CASE_INSENSITIVE | windows.OBJ_DONT_REPARSE}
	oa.Length = uint32(unsafe.Sizeof(oa))
	var status windows.IO_STATUS_BLOCK
	var h windows.Handle
	access := uint32(windows.FILE_LIST_DIRECTORY | windows.FILE_READ_ATTRIBUTES | windows.READ_CONTROL | windows.SYNCHRONIZE)
	options := uint32(windows.FILE_OPEN_REPARSE_POINT | windows.FILE_SYNCHRONOUS_IO_NONALERT)
	disposition := uint32(windows.FILE_OPEN)
	if directory {
		options |= windows.FILE_DIRECTORY_FILE
	} else {
		options |= windows.FILE_NON_DIRECTORY_FILE
		access = windows.FILE_GENERIC_READ
		if write {
			access |= windows.FILE_GENERIC_WRITE
			disposition = windows.FILE_OPEN_IF
		}
	}
	// Do not truncate on open. Inspect the actual opened object first. No WRITE or
	// DELETE sharing means a validated file cannot be rewritten or redirected.
	err = windows.NtCreateFile(&h, access, &oa, &status, nil, 0, windows.FILE_SHARE_READ, disposition, options, 0, 0)
	if err != nil {
		return 0, err
	}
	if err = checked(h, directory); err != nil {
		windows.CloseHandle(h)
		return 0, err
	}
	return h, nil
}

// OpenDirectory accepts local absolute drive paths only. It refuses links in
// every component, including the selected leaf, and releases all handles on error.
func OpenDirectory(path string) (*Directory, error) {
	if !filepath.IsAbs(path) {
		return nil, ErrUnsafeObject
	}
	path = filepath.Clean(path)
	volume := filepath.VolumeName(path)
	if len(volume) != 2 || volume[1] != ':' {
		return nil, ErrUnsafeObject
	}
	u, err := windows.UTF16PtrFromString(volume + `\`)
	if err != nil {
		return nil, err
	}
	h, err := windows.CreateFile(u, windows.FILE_LIST_DIRECTORY|windows.FILE_READ_ATTRIBUTES|windows.READ_CONTROL|windows.SYNCHRONIZE,
		windows.FILE_SHARE_READ, nil, windows.OPEN_EXISTING,
		windows.FILE_FLAG_BACKUP_SEMANTICS|windows.FILE_FLAG_OPEN_REPARSE_POINT, 0)
	if err != nil {
		return nil, err
	}
	d := &Directory{holds: []windows.Handle{h}}
	if err = checked(h, true); err != nil {
		d.Close()
		return nil, err
	}
	rest := strings.TrimPrefix(path[len(volume):], `\`)
	if rest != "" {
		for _, name := range strings.Split(rest, `\`) {
			h, err = relative(h, name, true, false)
			if err != nil {
				d.Close()
				return nil, err
			}
			d.holds = append(d.holds, h)
		}
	}
	return d, nil
}

func (d *Directory) Close() error {
	d.mu.Lock()
	defer d.mu.Unlock()
	var err error
	for i := len(d.holds) - 1; i >= 0; i-- {
		err = errors.Join(err, windows.CloseHandle(d.holds[i]))
	}
	d.holds = nil
	return err
}

// OpenFile opens a single leaf relative to the pinned directory. A writable open
// can create a new file, but never truncates an existing object before validation.
// The returned file remains bound to the selected object even after Directory.Close.
func (d *Directory) OpenFile(name string, write bool) (*os.File, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	if len(d.holds) == 0 {
		return nil, os.ErrClosed
	}
	h, err := relative(d.holds[len(d.holds)-1], name, false, write)
	if err != nil {
		return nil, err
	}
	return os.NewFile(uintptr(h), "managed-file"), nil
}

// Copy copies through validated handles. It does not provide atomic publication
// or installer rollback orchestration; those are separate integration gates.
func Copy(destination *Directory, destinationName string, source *Directory, sourceName string) error {
	r, err := source.OpenFile(sourceName, false)
	if err != nil {
		return err
	}
	defer r.Close()
	w, err := destination.OpenFile(destinationName, true)
	if err != nil {
		return err
	}
	defer w.Close()
	if err = w.Truncate(0); err != nil {
		return err
	}
	if _, err = io.Copy(w, r); err != nil {
		return err
	}
	return w.Sync()
}
