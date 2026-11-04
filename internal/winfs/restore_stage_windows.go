//go:build windows

package winfs

import (
	"errors"
	"os"
	"sync"
	"unsafe"

	"golang.org/x/sys/windows"
)

// RestoreStage owns a single final-authority directory handle. The caller keeps
// its parent pinned. Inherited handles are adopted, not reopened by pathname.
// Private stage owner/DACL verification belongs to the restore context.
type RestoreStage struct {
	mu sync.Mutex
	h  windows.Handle
}

func (d *Directory) CreateExclusiveDirectory(name string, sd *windows.SECURITY_DESCRIPTOR) (*RestoreStage, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	if len(d.holds) == 0 {
		return nil, os.ErrClosed
	}
	if !component(name) || sd == nil {
		return nil, ErrUnsafeObject
	}
	control, _, e := sd.Control()
	if e != nil {
		return nil, e
	}
	if control&windows.SE_DACL_PROTECTED == 0 {
		return nil, ErrUnsafeObject
	}
	u, e := windows.NewNTUnicodeString(name)
	if e != nil {
		return nil, e
	}
	oa := windows.OBJECT_ATTRIBUTES{RootDirectory: d.holds[len(d.holds)-1], ObjectName: u, Attributes: windows.OBJ_CASE_INSENSITIVE | windows.OBJ_DONT_REPARSE, SecurityDescriptor: sd}
	oa.Length = uint32(unsafe.Sizeof(oa))
	var status windows.IO_STATUS_BLOCK
	var h windows.Handle
	e = windows.NtCreateFile(&h, windows.FILE_LIST_DIRECTORY|windows.FILE_READ_ATTRIBUTES|windows.READ_CONTROL|windows.DELETE|windows.SYNCHRONIZE, &oa, &status, nil, 0, windows.FILE_SHARE_READ, windows.FILE_CREATE, windows.FILE_DIRECTORY_FILE|windows.FILE_OPEN_REPARSE_POINT|windows.FILE_SYNCHRONOUS_IO_NONALERT, 0, 0)
	if e != nil {
		return nil, e
	}
	if e = checked(h, true); e != nil {
		windows.CloseHandle(h)
		return nil, e
	}
	return &RestoreStage{h: h}, nil
}

// StageFromHandle transfers ownership of the caller's inherited handle on
// success. It does not duplicate or reopen it and takes no ownership on error.
func StageFromHandle(h windows.Handle) (*RestoreStage, error) {
	if h == 0 || h == windows.InvalidHandle {
		return nil, os.ErrClosed
	}
	if e := checked(h, true); e != nil {
		return nil, e
	}
	return &RestoreStage{h: h}, nil
}
func (s *RestoreStage) Handle() windows.Handle { s.mu.Lock(); defer s.mu.Unlock(); return s.h }

// Directory is a BORROWED view. The caller must never Close it, retain it beyond
// the stage lifetime, or use it concurrently with Publish/Remove/Close.
func (s *RestoreStage) Directory() *Directory {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.h == 0 {
		return &Directory{}
	}
	return &Directory{holds: []windows.Handle{s.h}}
}
func (s *RestoreStage) Identity() (Identity, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.h == 0 {
		return Identity{}, os.ErrClosed
	}
	return (&Directory{holds: []windows.Handle{s.h}}).Identity()
}

// Native layout matches x/sys's NtSetInformationFile acceptance test. Flags are
// zero: never replace an existing name and never enable POSIX replacement.
type relativeRenameInformation struct {
	ReplaceIfExists uint32
	RootDirectory   windows.Handle
	FileNameLength  uint32
	FileName        [1]uint16
}

func renameHeldNoReplace(h windows.Handle, parent windows.Handle, name string) error {
	if !component(name) {
		return ErrUnsafeObject
	}
	u, e := windows.UTF16FromString(name)
	if e != nil {
		return e
	}
	n := len(u) - 1
	var layout relativeRenameInformation
	offset := int(unsafe.Offsetof(layout.FileName))
	b := make([]byte, offset+n*2)
	info := (*relativeRenameInformation)(unsafe.Pointer(&b[0]))
	info.RootDirectory = parent
	info.FileNameLength = uint32(n * 2)
	copy(unsafe.Slice((*uint16)(unsafe.Pointer(&b[offset])), n), u[:n])
	var status windows.IO_STATUS_BLOCK
	return windows.NtSetInformationFile(h, &status, &b[0], uint32(len(b)), windows.FileRenameInformation)
}
func (s *RestoreStage) Publish(parent *Directory, name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	parent.mu.RLock()
	defer parent.mu.RUnlock()
	if s.h == 0 || len(parent.holds) == 0 {
		return os.ErrClosed
	}
	if e := checked(s.h, true); e != nil {
		return e
	}
	d := &Directory{holds: []windows.Handle{s.h}}
	before, e := d.Identity()
	if e != nil {
		return e
	}
	if e = renameHeldNoReplace(s.h, parent.holds[len(parent.holds)-1], name); e != nil {
		return e
	}
	if e = checked(s.h, true); e != nil {
		return e
	}
	after, e := d.Identity()
	if e != nil {
		return e
	}
	if before != after {
		return ErrUnsafeObject
	}
	return nil
}
func (s *RestoreStage) Remove() (err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.h == 0 {
		return os.ErrClosed
	}
	if e := checked(s.h, true); e != nil {
		return e
	}
	names, e := aclNames(s.h)
	if e != nil {
		return e
	}
	root := &Removal{h: s.h, Directory: true, children: map[string]*Removal{}}
	// Descendants are ALL held and checked before the first deletion. The root
	// remains stage-owned; only descendants are closed by error cleanup below.
	defer func() {
		for _, child := range root.children {
			err = errors.Join(err, child.Close())
		}
	}()
	for _, name := range names {
		child, e := removalChild(s.h, name)
		if e != nil {
			return e
		}
		root.children[name] = child
		if e = child.Inventory(); e != nil {
			return e
		}
	}
	if e = root.Validate(); e != nil {
		return e
	}
	e = root.Delete()
	s.h = root.h
	return e
}
func (s *RestoreStage) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.h == 0 {
		return nil
	}
	h := s.h
	s.h = 0
	return windows.CloseHandle(h)
}

// RenameFileNoReplace publishes only this selected held single-link file under
// this directory. The caller supplies a DELETE-capable file and owns its close.
func (d *Directory) RenameFileNoReplace(f *os.File, name string) error {
	d.mu.RLock()
	defer d.mu.RUnlock()
	if len(d.holds) == 0 {
		return os.ErrClosed
	}
	if e := ValidateFile(f); e != nil {
		return e
	}
	var before windows.ByHandleFileInformation
	if e := windows.GetFileInformationByHandle(windows.Handle(f.Fd()), &before); e != nil {
		return e
	}
	if e := renameHeldNoReplace(windows.Handle(f.Fd()), d.holds[len(d.holds)-1], name); e != nil {
		return e
	}
	if e := ValidateFile(f); e != nil {
		return e
	}
	var after windows.ByHandleFileInformation
	if e := windows.GetFileInformationByHandle(windows.Handle(f.Fd()), &after); e != nil {
		return e
	}
	if before.VolumeSerialNumber != after.VolumeSerialNumber || before.FileIndexHigh != after.FileIndexHigh || before.FileIndexLow != after.FileIndexLow {
		return ErrUnsafeObject
	}
	return nil
}

// OpenSharedReadFile is ONLY for immutable fixed progress records published by
// the trusted child while its DELETE-capable creation handle may remain live.
// It grants no writes/deletes; payload/source files retain the strict read API.
func (d *Directory) OpenSharedReadFile(name string) (*os.File, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	if len(d.holds) == 0 {
		return nil, os.ErrClosed
	}
	if !component(name) {
		return nil, ErrUnsafeObject
	}
	u, e := windows.NewNTUnicodeString(name)
	if e != nil {
		return nil, e
	}
	oa := windows.OBJECT_ATTRIBUTES{RootDirectory: d.holds[len(d.holds)-1], ObjectName: u, Attributes: windows.OBJ_CASE_INSENSITIVE | windows.OBJ_DONT_REPARSE}
	oa.Length = uint32(unsafe.Sizeof(oa))
	var status windows.IO_STATUS_BLOCK
	var h windows.Handle
	e = windows.NtCreateFile(&h, windows.FILE_GENERIC_READ, &oa, &status, nil, 0, windows.FILE_SHARE_READ|windows.FILE_SHARE_WRITE|windows.FILE_SHARE_DELETE, windows.FILE_OPEN, windows.FILE_NON_DIRECTORY_FILE|windows.FILE_OPEN_REPARSE_POINT|windows.FILE_SYNCHRONOUS_IO_NONALERT, 0, 0)
	if e != nil {
		return nil, e
	}
	if e = checked(h, false); e != nil {
		windows.CloseHandle(h)
		return nil, e
	}
	return os.NewFile(uintptr(h), "restore-progress"), nil
}
