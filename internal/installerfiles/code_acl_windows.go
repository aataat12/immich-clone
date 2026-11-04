//go:build windows

package installerfiles

import (
	"errors"

	"golang.org/x/sys/windows"
	"immichclone/internal/winfs"
)

const codeReadAccess uint32 = 0x1200a9

func codeReadPresent(grants []aclGrant, directory bool) bool {
	var self, child uint32
	for _, g := range grants {
		if g.sid != "S-1-5-19" {
			continue
		}
		if g.flags&8 == 0 {
			self |= g.mask
		}
		if g.flags&3 == 3 && g.flags&12 == 0 {
			child |= g.mask
		}
	}
	return self&codeReadAccess == codeReadAccess && (!directory || child&codeReadAccess == codeReadAccess)
}
func sameCodeGrants(a, b []aclGrant) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// GrantCodeReadAccess replaces the existing code-root inheritable LS RX grant.
// It preserves prior safe ACEs and stops at protected descendants, notably the
// admin-only rollback subtree. It never converts private snapshots into code.
func GrantCodeReadAccess(app string) error { return grantCodeReadAccess(app, nil) }
func grantCodeReadAccess(app string, beforeApply func() error) (err error) {
	tree, e := winfs.OpenCodeACLTree(app)
	if e != nil {
		return e
	}
	defer func() { err = errors.Join(err, tree.Close()) }()
	type plan struct {
		before, after        []aclGrant
		protected, propagate bool
		owner                string
		sd                   *windows.SECURITY_DESCRIPTOR
		descriptor           string
	}
	plans := make([]plan, len(tree.Nodes))
	for i, n := range tree.Nodes {
		sd, e := n.SecurityDescriptor()
		if e != nil {
			return e
		}
		if e = trustedCodeDescriptor(sd); e != nil {
			return e
		}
		grants, protected, owner, e := descriptorGrants(sd)
		if e != nil {
			return e
		}
		for _, g := range grants {
			if g.flags & ^byte(31) != 0 {
				return ErrRefused
			}
		}
		p := &plans[i]
		p.before = grants
		p.after = append([]aclGrant(nil), grants...)
		p.protected = protected
		p.owner = owner
		p.descriptor = sd.String()
		p.propagate = i == 0 || (!protected && plans[n.Parent].propagate)
		if !p.propagate || codeReadPresent(grants, n.Directory) {
			continue
		}
		flags := byte(0)
		if n.Directory {
			flags = 3
		}
		if i != 0 {
			flags |= 16
		}
		grant := aclGrant{"S-1-5-19", codeReadAccess, flags}
		at := len(p.after)
		if i == 0 {
			for j, g := range p.after {
				if g.flags&16 != 0 {
					at = j
					break
				}
			}
		}
		p.after = append(p.after, aclGrant{})
		copy(p.after[at+1:], p.after[at:])
		p.after[at] = grant
		p.sd, e = grantDescriptor(p.after, protected)
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
	// A changed descriptor must not be overwritten from a stale preflight plan.
	for i, n := range tree.Nodes {
		sd, e := n.SecurityDescriptor()
		if e != nil {
			return e
		}
		if sd.String() != plans[i].descriptor {
			return ErrRefused
		}
	}
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
		sd, e := n.SecurityDescriptor()
		if e != nil {
			return e
		}
		if e = trustedCodeDescriptor(sd); e != nil {
			return e
		}
		grants, protected, owner, e := descriptorGrants(sd)
		if e != nil {
			return e
		}
		p := plans[i]
		if protected != p.protected || owner != p.owner || !sameCodeGrants(grants, p.after) {
			return ErrRefused
		}
		if !p.propagate && sd.String() != p.descriptor {
			return ErrRefused
		}
		if p.propagate && !codeReadPresent(grants, n.Directory) {
			return ErrRefused
		}
	}
	return tree.Validate()
}
