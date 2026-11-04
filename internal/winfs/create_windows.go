//go:build windows

package winfs

import (
	"errors"
	"path/filepath"
	"strings"

	"golang.org/x/sys/windows"
)

// CreateDirectories pins every component, creating only missing directories with
// the supplied protected descriptor in NtCreateFile itself. Existing objects are
// never repaired. Verification runs while each new object and its ancestors are
// held. On failure safely created components remain; no pathname cleanup occurs.
func CreateDirectories(path string, descriptor *windows.SECURITY_DESCRIPTOR, verifyCreated func(*Directory) error) (_ *Directory, err error) {
	if descriptor == nil || verifyCreated == nil || !filepath.IsAbs(path) {
		return nil, ErrUnsafeObject
	}
	control, _, e := descriptor.Control()
	if e != nil {
		return nil, e
	}
	if control&windows.SE_DACL_PROTECTED == 0 {
		return nil, ErrUnsafeObject
	}
	// Reject raw traversal before cleaning; convert normal slash spelling only.
	for _, part := range strings.Split(strings.ReplaceAll(path, "/", `\`), `\`) {
		if part == "." || part == ".." {
			return nil, ErrUnsafeObject
		}
	}
	path = filepath.Clean(path)
	volume := filepath.VolumeName(path)
	if len(volume) != 2 || volume[1] != ':' {
		return nil, ErrUnsafeObject
	}
	d, e := OpenDirectory(volume + `\`)
	if e != nil {
		return nil, e
	}
	defer func() {
		if err != nil {
			err = errors.Join(err, d.Close())
		}
	}()
	rest := strings.TrimPrefix(path[len(volume):], `\`)
	if rest != "" {
		for _, name := range strings.Split(rest, `\`) {
			h, created, e := relativeCreate(d.holds[len(d.holds)-1], name, true, false, descriptor)
			if e != nil {
				return nil, e
			}
			d.holds = append(d.holds, h)
			if created {
				if e = verifyCreated(d); e != nil {
					return nil, e
				}
			}
		}
	}
	return d, nil
}
