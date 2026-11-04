//go:build windows

package installerfiles

import (
	"fmt"
	"strings"

	"golang.org/x/sys/windows"
	"immichclone/internal/winfs"
)

// EnsurePrivateRoot creates missing components private at birth. It intentionally
// does not change any existing directory's ACL or descendants; normalization of
// existing trees is a separate installer obligation, not a root-only API claim.
func EnsurePrivateRoot(path, reader string, service bool) (err error) {
	policy := map[string]uint32{"S-1-5-18": fileFullAccess, "S-1-5-32-544": fileFullAccess}
	if service {
		policy["S-1-5-19"] = 0x1301bf
	}
	if reader != "" {
		sid, e := windows.StringToSid(reader)
		if e != nil {
			return ErrRefused
		}
		reader = sid.String()
		if _, ok := policy[reader]; !ok {
			policy[reader] = 0x1200a9
		}
	}
	var text strings.Builder
	text.WriteString("D:P")
	for sid, rights := range policy {
		fmt.Fprintf(&text, "(A;OICI;0x%x;;;%s)", rights, sid)
	}
	sd, e := windows.SecurityDescriptorFromString(text.String())
	if e != nil {
		return e
	}
	d, e := winfs.CreateDirectories(path, sd, func(d *winfs.Directory) error {
		actual, e := d.SecurityDescriptor()
		if e != nil {
			return e
		}
		return policyDescriptor(actual, true, policy)
	})
	if e != nil {
		return e
	}
	return d.Close()
}
