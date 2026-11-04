//go:build windows

package installerfiles

import (
	"errors"
	"golang.org/x/sys/windows"
	"immichclone/internal/winfs"
	"os"
	"sort"
)

// AuditCleanup is read-only, including application state. It precedes service
// stop and native removal acquisition, which independently repeats actual audits.
func AuditCleanup(app, data string) (err error) {
	root, e := winfs.OpenDirectory(app)
	if e != nil {
		return e
	}
	defer func() { err = errors.Join(err, root.Close()) }()
	state, e := winfs.OpenDirectory(data)
	if e != nil {
		return e
	}
	defer func() { err = errors.Join(err, state.Close()) }()
	if e = disjointPayload(root, state); e != nil {
		return e
	}
	return auditCleanupDirectory(root)

}
func auditCleanupDirectory(d *winfs.Directory) (err error) {
	if e := trustedDirectory(d); e != nil {
		return e
	}
	names, e := d.Names()
	if e != nil {
		return e
	}
	sort.Strings(names)
	for _, name := range names {
		f, e := d.OpenFile(name, false)
		if e == nil {
			e = trustedFile(f)
			ce := f.Close()
			if e != nil || ce != nil {
				return errors.Join(e, ce)
			}
			continue
		}
		if !errors.Is(e, windows.STATUS_FILE_IS_A_DIRECTORY) {
			return e
		}
		child, e := d.Child(name, nil)
		if e != nil {
			return e
		}
		e = auditCleanupDirectory(child)
		ce := child.Close()
		if e != nil || ce != nil {
			return errors.Join(e, ce)
		}
	}
	again, e := d.Names()
	if e != nil {
		return e
	}
	sort.Strings(again)
	if !equalPayloadNames(names, again) {
		return ErrRefused
	}
	return nil
}
func missingCleanup(e error) bool {
	return errors.Is(e, os.ErrNotExist) || errors.Is(e, windows.STATUS_OBJECT_NAME_NOT_FOUND) || errors.Is(e, windows.STATUS_OBJECT_PATH_NOT_FOUND)
}
func Cleanup(app, data string) error { return cleanup(app, data, nil) }
func cleanup(app, data string, beforeDelete func() error) (err error) {
	d, e := winfs.OpenDirectory(app)
	if e != nil {
		return e
	}
	defer func() { err = errors.Join(err, d.Close()) }()
	state, e := winfs.OpenDirectory(data)
	if e != nil {
		return e
	}
	defer func() { err = errors.Join(err, state.Close()) }()
	if e = disjointPayload(d, state); e != nil {
		return e
	}
	if e = trustedDirectory(d); e != nil {
		return e
	}
	type selected struct {
		name string
		n    *winfs.Removal
	}
	var targets []selected
	defer func() {
		for _, t := range targets {
			err = errors.Join(err, t.n.Close())
		}
	}()
	add := func(parent *winfs.Directory, name string, dir, recursive bool) error {
		n, e := parent.OpenRemoval(name)
		if missingCleanup(e) {
			return nil
		}
		if e != nil {
			return e
		}
		targets = append(targets, selected{name, n})
		if n.Directory != dir {
			return ErrRefused
		}
		if recursive {
			if e = n.Inventory(); e != nil {
				return e
			}
		}
		return n.Walk(func(n *winfs.Removal) error {
			sd, e := n.SecurityDescriptor()
			if e != nil {
				return e
			}
			return trustedCodeDescriptor(sd)
		})
	}
	for _, name := range payloadRootNames {
		if e = add(d, name, false, false); e != nil {
			return e
		}
	}
	for _, name := range []string{"media", "ai"} {
		if e = add(d, name, true, true); e != nil {
			return e
		}
	}
	rollback, e := d.Child("rollback", nil)
	if e != nil && !missingCleanup(e) {
		return e
	}
	if e == nil {
		defer func() { err = errors.Join(err, rollback.Close()) }()
		if e = privateRollback(rollback); e != nil {
			return e
		}
		for _, name := range []string{"immichclone.exe", "catalog.db", "access-code.txt", readyName} {
			if e = add(rollback, name, false, false); e != nil {
				return e
			}
		}
	}
	// All stationary targets, outside roles and actual descriptors are verified
	// before the first deletion. Late-name checks have no recursive path fallback.
	if beforeDelete != nil {
		if e = beforeDelete(); e != nil {
			return e
		}
	}
	for _, t := range targets {
		if e = t.n.Validate(); e != nil {
			return e
		}
		if e = t.n.Walk(func(n *winfs.Removal) error {
			sd, e := n.SecurityDescriptor()
			if e != nil {
				return e
			}
			return trustedCodeDescriptor(sd)
		}); e != nil {
			return e
		}
	}
	sort.SliceStable(targets, func(i, j int) bool { return targets[i].name < targets[j].name })
	for _, t := range targets {
		if e = t.n.Delete(); e != nil {
			return e
		}
	}
	// Keep app/rollback directories, unknown files, state and diagnostics. Inno
	// owns only its own metadata and registration; native partial errors propagate.
	return nil
}
