//go:build windows

package winfs

import (
	"golang.org/x/sys/windows"
	"os"
	"unsafe"
)

// Names enumerates the actual held directory, never a re-resolved path.
func (d *Directory) Names() ([]string, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	if len(d.holds) == 0 {
		return nil, os.ErrClosed
	}
	return aclNames(d.holds[len(d.holds)-1])
}

// Child opens one component. A descriptor allows creation with final permissions
// at birth; callers must retain the parent and verify the actual child policy.
func (d *Directory) Child(name string, descriptor *windows.SECURITY_DESCRIPTOR) (*Directory, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	if len(d.holds) == 0 {
		return nil, os.ErrClosed
	}
	h, _, e := relativeCreate(d.holds[len(d.holds)-1], name, true, false, descriptor)
	if e != nil {
		return nil, e
	}
	return &Directory{holds: []windows.Handle{h}}, nil
}

// PublicationFile opens without truncation, with DELETE authority for exact
// newly-created-file rollback. Sharing excludes other data writers and renames.
func (d *Directory) PublicationFile(name string) (*os.File, bool, error) {
	return d.publicationFile(name, true)
}
func (d *Directory) ExistingPublicationFile(name string) (*os.File, bool, error) {
	return d.publicationFile(name, false)
}
func (d *Directory) publicationFile(name string, create bool) (*os.File, bool, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	if len(d.holds) == 0 {
		return nil, false, os.ErrClosed
	}
	if !component(name) {
		return nil, false, ErrUnsafeObject
	}
	u, e := windows.NewNTUnicodeString(name)
	if e != nil {
		return nil, false, e
	}
	oa := windows.OBJECT_ATTRIBUTES{RootDirectory: d.holds[len(d.holds)-1], ObjectName: u, Attributes: windows.OBJ_CASE_INSENSITIVE | windows.OBJ_DONT_REPARSE}
	oa.Length = uint32(unsafe.Sizeof(oa))
	var status windows.IO_STATUS_BLOCK
	var h windows.Handle
	disposition := uint32(windows.FILE_OPEN)
	if create {
		disposition = windows.FILE_OPEN_IF
	}
	e = windows.NtCreateFile(&h, windows.FILE_GENERIC_READ|windows.FILE_GENERIC_WRITE|windows.DELETE, &oa, &status, nil, 0, windows.FILE_SHARE_READ, disposition, windows.FILE_NON_DIRECTORY_FILE|windows.FILE_OPEN_REPARSE_POINT|windows.FILE_SYNCHRONOUS_IO_NONALERT, 0, 0)
	if e != nil {
		return nil, false, e
	}
	if e = checked(h, false); e != nil {
		windows.CloseHandle(h)
		return nil, false, e
	}
	return os.NewFile(uintptr(h), "publication-file"), status.Information == 2, nil
}
func ValidateFile(f *os.File) error { return checked(windows.Handle(f.Fd()), false) }

// DeletePublicationFile marks only the already held single-link file for deletion.
func DeletePublicationFile(f *os.File) error {
	if e := ValidateFile(f); e != nil {
		return e
	}
	v := byte(1)
	return windows.SetFileInformationByHandle(windows.Handle(f.Fd()), windows.FileDispositionInfo, &v, 1)
}
