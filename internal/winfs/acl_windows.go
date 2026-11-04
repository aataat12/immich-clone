//go:build windows

package winfs

import (
	"errors"
	"os"
	"path/filepath"
	"sync"
	"unsafe"

	"golang.org/x/sys/windows"
)

// ACLDirectory is an isolated candidate for single-object ACL updates. Installer
// integration must separately audit/pin descendants and preserve inherited ACEs.
// MAXIMUM_ALLOWED is intentional: SetSecurityInfo documents that this suppresses
// implicit propagation to children. Actual Windows acceptance is required.
type ACLDirectory struct {
	mu        sync.Mutex
	directory *Directory
	handle    windows.Handle
}

func OpenACLDirectory(path string) (_ *ACLDirectory, err error) {
	// Pin the parent chain, then open the leaf once with its final access mask.
	// Keeping a read-only leaf open while requesting MAXIMUM_ALLOWED would
	// conflict with our own no-write/delete-sharing pin.
	path = filepath.Clean(path)
	if !filepath.IsAbs(path) || filepath.Dir(path) == path {
		return nil, ErrUnsafeObject
	}
	name := filepath.Base(path)
	if !component(name) {
		return nil, ErrUnsafeObject
	}
	d, err := OpenDirectory(filepath.Dir(path))
	if err != nil {
		return nil, err
	}
	a := &ACLDirectory{directory: d}
	defer func() {
		if err != nil {
			err = errors.Join(err, a.Close())
		}
	}()
	u, err := windows.NewNTUnicodeString(name)
	if err != nil {
		return nil, err
	}
	oa := windows.OBJECT_ATTRIBUTES{RootDirectory: d.holds[len(d.holds)-1], ObjectName: u, Attributes: windows.OBJ_CASE_INSENSITIVE | windows.OBJ_DONT_REPARSE}
	oa.Length = uint32(unsafe.Sizeof(oa))
	var status windows.IO_STATUS_BLOCK
	err = windows.NtCreateFile(&a.handle, windows.MAXIMUM_ALLOWED|windows.SYNCHRONIZE, &oa, &status, nil, 0, windows.FILE_SHARE_READ, windows.FILE_OPEN, windows.FILE_DIRECTORY_FILE|windows.FILE_OPEN_REPARSE_POINT|windows.FILE_SYNCHRONOUS_IO_NONALERT, 0, 0)
	if err != nil {
		return nil, err
	}
	if err = checked(a.handle, true); err != nil {
		return nil, err
	}
	return a, nil
}

func (a *ACLDirectory) SecurityDescriptor() (*windows.SECURITY_DESCRIPTOR, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.handle == 0 {
		return nil, os.ErrClosed
	}
	return windows.GetSecurityInfo(a.handle, windows.SE_FILE_OBJECT, windows.OWNER_SECURITY_INFORMATION|windows.DACL_SECURITY_INFORMATION)
}

// SetProtectedDACL changes only the held directory's protected DACL. It never
// requests an owner/SACL change, accepts no null DACL, and has no pathname fallback.
// It does not normalize existing descendants or certify an installer operation.
func (a *ACLDirectory) SetProtectedDACL(sd *windows.SECURITY_DESCRIPTOR) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.handle == 0 {
		return os.ErrClosed
	}
	if sd == nil {
		return ErrUnsafeObject
	}
	control, _, e := sd.Control()
	if e != nil {
		return e
	}
	if control&windows.SE_DACL_PROTECTED == 0 {
		return ErrUnsafeObject
	}
	acl, _, e := sd.DACL()
	if e != nil {
		return e
	}
	if acl == nil {
		return ErrUnsafeObject
	}
	return windows.SetSecurityInfo(a.handle, windows.SE_FILE_OBJECT, windows.DACL_SECURITY_INFORMATION|windows.PROTECTED_DACL_SECURITY_INFORMATION, nil, nil, acl, nil)
}

func (a *ACLDirectory) Close() error {
	a.mu.Lock()
	defer a.mu.Unlock()
	var err error
	if a.handle != 0 {
		err = windows.CloseHandle(a.handle)
		a.handle = 0
	}
	if a.directory != nil {
		err = errors.Join(err, a.directory.Close())
		a.directory = nil
	}
	return err
}
