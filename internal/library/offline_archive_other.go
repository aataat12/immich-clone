//go:build !windows

package library

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
)

func (s *Store) BackupFile(path string) (err error) {
	if s.offline == nil || !filepath.IsAbs(path) {
		return errors.New("offline absolute archive required")
	}
	rel, e := filepath.Rel(s.root, path)
	if e != nil {
		return e
	}
	if rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return errors.New("archive inside source")
	}
	f, e := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if e != nil {
		return e
	}
	defer func() {
		if err != nil {
			if e := os.Remove(path); e != nil {
				err = errors.Join(err, ErrOfflineArchiveCleanup, e)
			}
		}
		err = errors.Join(err, f.Close())
	}()
	if e = s.Backup(f); e != nil {
		return e
	}
	return f.Sync()
}
