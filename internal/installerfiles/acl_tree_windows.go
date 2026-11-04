//go:build windows

package installerfiles

import (
	"errors"
	"fmt"
	"strings"
	"unsafe"

	"golang.org/x/sys/windows"
	"immichclone/internal/winfs"
)

type aclGrant struct {
	sid   string
	mask  uint32
	flags byte
}

func storagePolicy(reader string, service bool) (map[string]uint32, error) {
	p := map[string]uint32{"S-1-5-18": fileFullAccess, "S-1-5-32-544": fileFullAccess}
	if service {
		p["S-1-5-19"] = 0x1301bf
	}
	if reader != "" {
		sid, e := windows.StringToSid(reader)
		if e != nil {
			return nil, ErrRefused
		}
		if _, ok := p[sid.String()]; !ok {
			p[sid.String()] = 0x1200a9
		}
	}
	return p, nil
}
func descriptorGrants(sd *windows.SECURITY_DESCRIPTOR) ([]aclGrant, bool, string, error) {
	if sd == nil {
		return nil, false, "", ErrRefused
	}
	control, _, e := sd.Control()
	if e != nil {
		return nil, false, "", e
	}
	owner, _, e := sd.Owner()
	if e != nil || owner == nil {
		return nil, false, "", ErrRefused
	}
	acl, _, e := sd.DACL()
	if e != nil || acl == nil {
		return nil, false, "", ErrRefused
	}
	var grants []aclGrant
	for i := uint32(0); i < uint32(acl.AceCount); i++ {
		var ace *windows.ACCESS_ALLOWED_ACE
		if e = windows.GetAce(acl, i, &ace); e != nil {
			return nil, false, "", e
		}
		// Unsupported inherited ACEs may be discarded only on unprotected children,
		// matching preflight policy. Represent them without dereferencing SID layout.
		if ace.Header.AceType != windows.ACCESS_ALLOWED_ACE_TYPE {
			grants = append(grants, aclGrant{"", ^uint32(0), ace.Header.AceFlags})
			continue
		}
		sid := (*windows.SID)(unsafe.Pointer(&ace.SidStart)).String()
		grants = append(grants, aclGrant{sid, uint32(ace.Mask), ace.Header.AceFlags})
	}
	return grants, control&windows.SE_DACL_PROTECTED != 0, owner.String(), nil
}
func auditGrants(grants []aclGrant, protected, dir, root, final bool, owner, current string, policy map[string]uint32) error {
	if owner != current && policy[owner] == 0 {
		return ErrRefused
	}
	if root && !final {
		return nil
	}
	if root && !protected {
		return ErrRefused
	}
	self, child := map[string]uint32{}, map[string]uint32{}
	for _, g := range grants {
		if !protected && !final && g.flags&16 != 0 {
			continue
		}
		rights, ok := policy[g.sid]
		if g.flags & ^byte(31) != 0 {
			return ErrRefused
		}
		if !ok || g.mask&^rights != 0 {
			return ErrRefused
		}
		if g.flags&8 == 0 {
			self[g.sid] |= g.mask
		}
		if g.flags&3 == 3 && g.flags&12 == 0 {
			child[g.sid] |= g.mask
		}
	}
	if protected || final {
		for sid, rights := range policy {
			if self[sid]&rights != rights || (dir && child[sid]&rights != rights) {
				return ErrRefused
			}
		}
	}
	return nil
}
func grantDescriptor(grants []aclGrant, protected bool) (*windows.SECURITY_DESCRIPTOR, error) {
	var b strings.Builder
	b.WriteString("D:")
	if protected {
		b.WriteString("P")
	}
	for _, g := range grants {
		var flags strings.Builder
		for _, f := range []struct {
			mask byte
			name string
		}{{1, "OI"}, {2, "CI"}, {4, "NP"}, {8, "IO"}, {16, "ID"}} {
			if g.flags&f.mask != 0 {
				flags.WriteString(f.name)
			}
		}
		fmt.Fprintf(&b, "(A;%s;0x%x;;;%s)", flags.String(), g.mask, g.sid)
	}
	return windows.SecurityDescriptorFromString(b.String())
}

func NormalizePrivateTree(root, reader string, service bool) error {
	return normalizePrivateTree(root, reader, service, nil)
}
func normalizePrivateTree(root, reader string, service bool, beforeApply func() error) (err error) {
	policy, e := storagePolicy(reader, service)
	if e != nil {
		return e
	}
	user, e := windows.GetCurrentProcessToken().GetTokenUser()
	if e != nil {
		return e
	}
	tree, e := winfs.OpenACLTree(root)
	if e != nil {
		return e
	}
	defer func() { err = errors.Join(err, tree.Close()) }()
	type plan struct {
		sd        *windows.SECURITY_DESCRIPTOR
		protected bool
		owner     string
	}
	plans := make([]plan, len(tree.Nodes))
	for i, n := range tree.Nodes {
		actual, e := n.SecurityDescriptor()
		if e != nil {
			return e
		}
		grants, protected, owner, e := descriptorGrants(actual)
		if e != nil {
			return e
		}
		if e = auditGrants(grants, protected, n.Directory, i == 0, false, owner, user.User.Sid.String(), policy); e != nil {
			return e
		}
		plans[i] = plan{protected: protected, owner: owner}
		if i != 0 && protected {
			continue
		}
		desired := []aclGrant{}
		if i != 0 {
			for _, g := range grants {
				if g.flags&16 == 0 {
					desired = append(desired, g)
				}
			}
		}
		flags := byte(0)
		if n.Directory {
			flags = 3
		}
		if i != 0 {
			flags |= 16
		}
		for sid, rights := range policy {
			desired = append(desired, aclGrant{sid, rights, flags})
		}
		plans[i].protected = i == 0
		plans[i].sd, e = grantDescriptor(desired, i == 0)
		if e != nil {
			return e
		}
	}
	if beforeApply != nil {
		if e = beforeApply(); e != nil {
			return e
		}
	}
	if e = tree.Validate(); e != nil {
		return e
	}
	// Parent-first: newly created descendants inherit final policy. Previously
	// protected, valid descendants remain byte-for-byte untouched.
	for i, n := range tree.Nodes {
		if plans[i].sd != nil {
			if e = n.SetDACL(plans[i].sd, plans[i].protected); e != nil {
				return e
			}
		}
	}
	if e = tree.Validate(); e != nil {
		return e
	}
	for i, n := range tree.Nodes {
		actual, e := n.SecurityDescriptor()
		if e != nil {
			return e
		}
		grants, protected, owner, e := descriptorGrants(actual)
		if e != nil {
			return e
		}
		if owner != plans[i].owner || protected != plans[i].protected {
			return ErrRefused
		}
		if e = auditGrants(grants, protected, n.Directory, i == 0, true, owner, user.User.Sid.String(), policy); e != nil {
			return e
		}
	}
	return tree.Validate()
}
