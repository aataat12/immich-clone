package main

import (
	"errors"
	"immichclone/internal/library"
)

func runReadOnlyCommand(mode, root, archive string) (err error) {
	s, e := library.OpenReadOnly(root)
	if e != nil {
		return e
	}
	defer func() { err = errors.Join(err, s.Close()) }()
	if mode == "verify" {
		return s.Verify()
	}
	return s.BackupFile(archive)
}
