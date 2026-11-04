//go:build !windows

package main

import (
	"errors"
	"os"
	"path/filepath"
)

func readExistingAccessCode(root string) (code string, err error) {
	f, e := os.Open(filepath.Join(root, "access-code.txt"))
	if e != nil {
		return "", e
	}
	defer func() { err = errors.Join(err, f.Close()) }()
	return existingCode(f)
}
