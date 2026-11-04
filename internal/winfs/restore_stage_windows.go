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

// RestoreStage owns a single final-authority directory handle. The caller keeps
// its parent pinned. Inherited handles are adopted, not reopened by pathname.
// Private stage owner/DACL verification belongs to the restore context.
type RestoreStage struct {
	mu sync.Mutex
	h  windows.Handle
}

// OpenRestoreParent pins the same nofollow chain as OpenDirectory. Only its
// final read-only handle shares WRITE so native relative rename can open the
// target directory for adding an entry. DELETE remains unshared at every level.
// Before releasing any strict pin or using this for publication, the caller must
// already hold a protected private stage child with DELETE sharing denied. That
// child keeps the parent nonempty, preventing reparse conversion while WRITE is
// shared. Opening an empty parent this way alone is not a safe restore boundary.
func OpenRestoreParent(path string) (_ *Directory, err error) {
	d, err := OpenDirectory(path)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err != nil {
			d.Close()
		}
	}()
	before, err := d.Identity()
	if err != nil {
		return nil, err
	}
	access := uint32(windows.FILE_LIST_DIRECTORY | windows.FILE_READ_ATTRIBUTES | windows.READ_CONTROL | windows.SYNCHRONIZE)
	var h windows.Handle
	if len(d.holds) == 1 {
		var u *uint16
		u, err = windows.UTF16PtrFromString(filepath.VolumeName(filepath.Clean(path)) + `\`)
		if err == nil {
			h, err = windows.CreateFile(u, access, windows.FILE_SHARE_READ|windows.FILE_SHARE_WRITE, nil, windows.OPEN_EXISTING, windows.FILE_FLAG_BACKUP_SEMANTICS|windows.FILE_FLAG_OPEN_REPARSE_POINT, 0)
		}
	} else {
		var u *windows.NTUnicodeString
		u, err = windows.NewNTUnicodeString(filepath.Base(filepath.Clean(path)))
		if err == nil {
			oa := windows.OBJECT_ATTRIBUTES{RootDirectory: d.holds[len(d.holds)-2], ObjectName: u, Attributes: windows.OBJ_CASE_INSENSITIVE | windows.OBJ_DONT_REPARSE}
			oa.Length = uint32(unsafe.Sizeof(oa))
			var status windows.IO_STATUS_BLOCK
			err = windows.NtCreateFile(&h, access, &oa, &status, nil, 0, windows.FILE_SHARE_READ|windows.FILE_SHARE_WRITE, windows.FILE_OPEN, windows.FILE_DIRECTORY_FILE|windows.FILE_OPEN_REPARSE_POINT|windows.FILE_SYNCHRONOUS_IO_NONALERT, 0, 0)
		}
	}
	if err != nil {
		return nil, err
	}
	if err = checked(h, true); err != nil {
		windows.CloseHandle(h)
		return nil, err
	}
	after, e := (&Directory{holds: []windows.Handle{h}}).Identity()
	if e != nil || before != after {
		windows.CloseHandle(h)
		if e != nil {
			return nil, e
		}
		return nil, ErrUnsafeObject
	}
	old := d.holds[len(d.holds)-1]
	d.holds[len(d.holds)-1] = h
	if err = windows.CloseHandle(old); err != nil {
		return nil, err
	}
	return d, nil
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
	e = windows.NtCreateFile(&h, windows.FILE_LIST_DIRECTORY|windows.FILE_READ_ATTRIBUTES|windows.READ_CONTROL|windows.DELETE|windows.SYNCHRONIZE, &oa, &status, nil, 0, windows.FILE_SHARE_READ|windows.FILE_SHARE_WRITE, windows.FILE_CREATE, windows.FILE_DIRECTORY_FILE|windows.FILE_OPEN_REPARSE_POINT|windows.FILE_SYNCHRONOUS_IO_NONALERT, 0, 0)
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
