//go:build windows

package winfs

import (
	"errors"
	"golang.org/x/sys/windows"
	"os"
	"sort"
	"unsafe"
)

// Removal holds a complete selected subtree with read/delete authority, no data
// writes, no reparse following and no write/delete sharing. Parent stays held by
// the caller; missing selected children are reported, never created.
type Removal struct {
	h         windows.Handle
	Directory bool
	children  map[string]*Removal
}

func removalChild(parent windows.Handle, name string) (*Removal, error) {
	if !component(name) {
		return nil, ErrUnsafeObject
	}
	u, e := windows.NewNTUnicodeString(name)
	if e != nil {
		return nil, e
	}
	oa := windows.OBJECT_ATTRIBUTES{RootDirectory: parent, ObjectName: u, Attributes: windows.OBJ_CASE_INSENSITIVE | windows.OBJ_DONT_REPARSE}
	oa.Length = uint32(unsafe.Sizeof(oa))
	var status windows.IO_STATUS_BLOCK
	var h windows.Handle
	e = windows.NtCreateFile(&h, windows.FILE_GENERIC_READ|windows.DELETE, &oa, &status, nil, 0, windows.FILE_SHARE_READ, windows.FILE_OPEN, windows.FILE_OPEN_REPARSE_POINT|windows.FILE_SYNCHRONOUS_IO_NONALERT, 0, 0)
	if e != nil {
		return nil, e
	}
	var info windows.ByHandleFileInformation
	if e = windows.GetFileInformationByHandle(h, &info); e != nil {
		windows.CloseHandle(h)
		return nil, e
	}
	n := &Removal{h: h, Directory: info.FileAttributes&windows.FILE_ATTRIBUTE_DIRECTORY != 0}
	if e = checked(h, n.Directory); e != nil {
		n.Close()
		return nil, e
	}
	return n, nil
}
func (d *Directory) OpenRemoval(name string) (*Removal, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	if len(d.holds) == 0 {
		return nil, os.ErrClosed
	}
	return removalChild(d.holds[len(d.holds)-1], name)
}
func (n *Removal) Inventory() error {
	if !n.Directory {
		return nil
	}
	names, e := aclNames(n.h)
	if e != nil {
		return e
	}
	n.children = map[string]*Removal{}
	for _, name := range names {
		child, e := removalChild(n.h, name)
		if e != nil {
			return e
		}
		n.children[name] = child
		if e = child.Inventory(); e != nil {
			return e
		}
	}
	return n.Validate()
}
func (n *Removal) SecurityDescriptor() (*windows.SECURITY_DESCRIPTOR, error) {
	return windows.GetSecurityInfo(n.h, windows.SE_FILE_OBJECT, windows.OWNER_SECURITY_INFORMATION|windows.DACL_SECURITY_INFORMATION)
}
func (n *Removal) Walk(f func(*Removal) error) error {
	if e := f(n); e != nil {
		return e
	}
	for _, child := range n.children {
		if e := child.Walk(f); e != nil {
			return e
		}
	}
	return nil
}
func (n *Removal) Validate() error {
	if e := checked(n.h, n.Directory); e != nil {
		return e
	}
	if !n.Directory {
		return nil
	}
	names, e := aclNames(n.h)
	if e != nil {
		return e
	}
	if len(names) != len(n.children) {
		return ErrUnsafeObject
	}
	for _, name := range names {
		c := n.children[name]
		if c == nil {
			return ErrUnsafeObject
		}
		if e = c.Validate(); e != nil {
			return e
		}
	}
	return nil
}
func (n *Removal) Delete() error {
	if e := n.Validate(); e != nil {
		return e
	}
	names := make([]string, 0, len(n.children))
	for name := range n.children {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		if e := n.children[name].Delete(); e != nil {
			return e
		}
		delete(n.children, name)
	}
	if e := n.Validate(); e != nil {
		return e
	}
	v := byte(1)
	if e := windows.SetFileInformationByHandle(n.h, windows.FileDispositionInfo, &v, 1); e != nil {
		return e
	}
	e := windows.CloseHandle(n.h)
	n.h = 0
	return e
}
func (n *Removal) Close() (err error) {
	for _, c := range n.children {
		err = errors.Join(err, c.Close())
	}
	if n.h != 0 {
		err = errors.Join(err, windows.CloseHandle(n.h))
		n.h = 0
	}
	return err
}

// OpenMetadataFile audits only security/type/link metadata on an existing leaf.
// It intentionally does not request file-data sharing authority. It is reserved
// for the uninstaller's own active metadata files, never payload/removal targets.
func (d *Directory) OpenMetadataFile(name string) (*os.File, error) {
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
	e = windows.NtCreateFile(&h, windows.READ_CONTROL|windows.FILE_READ_ATTRIBUTES|windows.SYNCHRONIZE, &oa, &status, nil, 0, windows.FILE_SHARE_READ, windows.FILE_OPEN, windows.FILE_NON_DIRECTORY_FILE|windows.FILE_OPEN_REPARSE_POINT|windows.FILE_SYNCHRONOUS_IO_NONALERT, 0, 0)
	if e != nil {
		return nil, e
	}
	if e = checked(h, false); e != nil {
		windows.CloseHandle(h)
		return nil, e
	}
	return os.NewFile(uintptr(h), "uninstaller-metadata"), nil
}
