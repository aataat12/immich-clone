//go:build windows

package winfs

import (
	"errors"
	"io"
	"os"
	"strings"
	"unsafe"

	"golang.org/x/sys/windows"
)

// ACLTree retains all opened objects. Namespace validation is repeated after ACL
// changes because directory sharing pins do not prevent creation of new children.
type ACLTree struct {
	root  *ACLDirectory
	Nodes []*ACLNode
}
type ACLNode struct {
	handle    windows.Handle
	Directory bool
	Parent    int
	children  map[string]bool
}

func aclChild(parent windows.Handle, name string) (h windows.Handle, directory bool, err error) {
	if !component(name) {
		return 0, false, ErrUnsafeObject
	}
	u, e := windows.NewNTUnicodeString(name)
	if e != nil {
		return 0, false, e
	}
	oa := windows.OBJECT_ATTRIBUTES{RootDirectory: parent, ObjectName: u, Attributes: windows.OBJ_CASE_INSENSITIVE | windows.OBJ_DONT_REPARSE}
	oa.Length = uint32(unsafe.Sizeof(oa))
	var status windows.IO_STATUS_BLOCK
	e = windows.NtCreateFile(&h, windows.MAXIMUM_ALLOWED|windows.SYNCHRONIZE, &oa, &status, nil, 0, windows.FILE_SHARE_READ, windows.FILE_OPEN, windows.FILE_OPEN_REPARSE_POINT|windows.FILE_SYNCHRONOUS_IO_NONALERT, 0, 0)
	if e != nil {
		return 0, false, e
	}
	var info windows.ByHandleFileInformation
	if e = windows.GetFileInformationByHandle(h, &info); e == nil {
		directory = info.FileAttributes&windows.FILE_ATTRIBUTE_DIRECTORY != 0
		e = checked(h, directory)
	}
	if e != nil {
		windows.CloseHandle(h)
		return 0, false, e
	}
	return h, directory, nil
}

func aclNames(h windows.Handle) (_ []string, err error) {
	var duplicate windows.Handle
	if err = windows.DuplicateHandle(windows.CurrentProcess(), h, windows.CurrentProcess(), &duplicate, 0, false, windows.DUPLICATE_SAME_ACCESS); err != nil {
		return nil, err
	}
	f := os.NewFile(uintptr(duplicate), "held-directory")
	defer func() { err = errors.Join(err, f.Close()) }()
	// Go's Windows Readdirnames uses GetFileInformationByHandleEx, not a pathname
	// enumeration. The duplicate owns its lifetime; the validated original stays held.
	var names []string
	for {
		part, e := f.Readdirnames(256)
		names = append(names, part...)
		if e == io.EOF {
			return names, nil
		}
		if e != nil {
			return nil, e
		}
	}
}

func OpenACLTree(path string) (_ *ACLTree, err error) {
	root, e := OpenACLDirectory(path)
	if e != nil {
		return nil, e
	}
	t := &ACLTree{root: root, Nodes: []*ACLNode{{handle: root.handle, Directory: true, Parent: -1}}}
	defer func() {
		if err != nil {
			err = errors.Join(err, t.Close())
		}
	}()
	for i := 0; i < len(t.Nodes); i++ {
		n := t.Nodes[i]
		if !n.Directory {
			continue
		}
		names, e := aclNames(n.handle)
		if e != nil {
			return nil, e
		}
		n.children = map[string]bool{}
		for _, name := range names {
			key := strings.ToLower(name)
			if n.children[key] {
				return nil, ErrUnsafeObject
			}
			h, dir, e := aclChild(n.handle, name)
			if e != nil {
				return nil, e
			}
			n.children[key] = true
			t.Nodes = append(t.Nodes, &ACLNode{handle: h, Directory: dir, Parent: i})
		}
	}
	if e = t.Validate(); e != nil {
		return nil, e
	}
	return t, nil
}
func (n *ACLNode) SecurityDescriptor() (*windows.SECURITY_DESCRIPTOR, error) {
	if n.handle == 0 {
		return nil, os.ErrClosed
	}
	return windows.GetSecurityInfo(n.handle, windows.SE_FILE_OBJECT, windows.OWNER_SECURITY_INFORMATION|windows.DACL_SECURITY_INFORMATION)
}
func (n *ACLNode) SetDACL(sd *windows.SECURITY_DESCRIPTOR, protected bool) error {
	if n.handle == 0 {
		return os.ErrClosed
	}
	if sd == nil {
		return ErrUnsafeObject
	}
	acl, _, e := sd.DACL()
	if e != nil {
		return e
	}
	if acl == nil {
		return ErrUnsafeObject
	}
	if e = checked(n.handle, n.Directory); e != nil {
		return e
	}
	flags := windows.SECURITY_INFORMATION(windows.DACL_SECURITY_INFORMATION)
	if protected {
		flags |= windows.PROTECTED_DACL_SECURITY_INFORMATION
	}
	return windows.SetSecurityInfo(n.handle, windows.SE_FILE_OBJECT, flags, nil, nil, acl, nil)
}
func (t *ACLTree) Validate() error {
	if len(t.Nodes) == 0 {
		return os.ErrClosed
	}
	for _, n := range t.Nodes {
		if e := checked(n.handle, n.Directory); e != nil {
			return e
		}
		if !n.Directory {
			continue
		}
		names, e := aclNames(n.handle)
		if e != nil {
			return e
		}
		if len(names) != len(n.children) {
			return ErrUnsafeObject
		}
		seen := map[string]bool{}
		for _, name := range names {
			key := strings.ToLower(name)
			if !n.children[key] || seen[key] {
				return ErrUnsafeObject
			}
			seen[key] = true
		}
	}
	return nil
}
func (t *ACLTree) Close() (err error) {
	for i := len(t.Nodes) - 1; i > 0; i-- {
		if t.Nodes[i].handle != 0 {
			err = errors.Join(err, windows.CloseHandle(t.Nodes[i].handle))
			t.Nodes[i].handle = 0
		}
	}
	if t.root != nil {
		err = errors.Join(err, t.root.Close())
		t.root = nil
	}
	if len(t.Nodes) > 0 {
		t.Nodes[0].handle = 0
	}
	t.Nodes = nil
	return err
}
