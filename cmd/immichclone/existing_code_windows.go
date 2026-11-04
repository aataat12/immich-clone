//go:build windows

package main

import (
	"errors"
	"immichclone/internal/winfs"
)

// Reading an existing code never initializes state, opens the catalog or logs.
// Both directory chain and single-link leaf remain held without following links.
func readExistingAccessCode(root string) (code string, err error) {
	d, e := winfs.OpenDirectory(root)
	if e != nil {
		return "", e
	}
	defer func() { err = errors.Join(err, d.Close()) }()
	f, e := d.OpenFile("access-code.txt", false)
	if e != nil {
		return "", e
	}
	defer func() { err = errors.Join(err, f.Close()) }()
	return existingCode(f)
}
