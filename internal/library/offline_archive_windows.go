//go:build windows

package library

import (
	"errors"
	"golang.org/x/sys/windows"
	"immichclone/internal/winfs"
	"path/filepath"
	"strings"
)

// BackupFile is the CLI-only archive boundary; no source staging or pathname
// cleanup is used. A partial file is always private, even in a shared parent.
func (s *Store) BackupFile(path string) (err error) {
	if s.offline == nil || !filepath.IsAbs(path) {
		return errors.New("offline absolute archive required")
	}
	parent, e := winfs.OpenDirectory(filepath.Dir(path))
	if e != nil {
		return e
	}
	defer func() { err = errors.Join(err, parent.Close()) }()
	source, e := s.offline.files.physicalRoot()
	if e != nil {
		return e
	}
	p, e := parent.FinalPath()
	if e != nil {
		return e
	}
	source = strings.TrimRight(strings.ToLower(source), `\`)
	p = strings.TrimRight(strings.ToLower(p), `\`)
	if p == source || strings.HasPrefix(p, source+`\`) {
		return errors.New("archive must be outside source library")
	}
	user, e := windows.GetCurrentProcessToken().GetTokenUser()
	if e != nil {
		return e
	}
	sd, e := windows.SecurityDescriptorFromString("D:P(A;;FA;;;SY)(A;;FA;;;BA)(A;;FA;;;" + user.User.Sid.String() + ")")
	if e != nil {
		return e
	}
	f, e := parent.CreateExclusiveFile(filepath.Base(path), sd)
	if e != nil {
		return e
	}
	defer func() {
		if err != nil {
			if e := winfs.DeletePublicationFile(f); e != nil {
				err = errors.Join(err, ErrOfflineArchiveCleanup, e)
			}
		}
		if ce := f.Close(); ce != nil {
			err = errors.Join(err, ErrOfflineArchiveCleanup, ce)
		}
	}()
	if e = s.Backup(f); e != nil {
		return e
	}
	if e = winfs.ValidateFile(f); e != nil {
		return e
	}
	return f.Sync()
}
