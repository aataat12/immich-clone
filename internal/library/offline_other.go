//go:build !windows

package library

import (
	"errors"
	"os"
	"path/filepath"
)

type offlineOtherFiles struct{ root string }

func openOfflineFiles(p string) (offlineFiles, error) {
	if !filepath.IsAbs(p) {
		return nil, errors.New("absolute state required")
	}
	for _, q := range []string{p, filepath.Join(p, "originals")} {
		i, e := os.Lstat(q)
		if e != nil {
			return nil, e
		}
		if !i.IsDir() || i.Mode()&os.ModeSymlink != 0 {
			return nil, errors.New("unsafe state")
		}
	}
	return &offlineOtherFiles{p}, nil
}
func (o *offlineOtherFiles) rootFile(n string) (*os.File, error) {
	return os.Open(filepath.Join(o.root, n))
}
func (o *offlineOtherFiles) original(n string) (*os.File, error) {
	return os.Open(filepath.Join(o.root, "originals", n))
}
func (o *offlineOtherFiles) validate(f *os.File) error {
	i, e := f.Stat()
	if e != nil {
		return e
	}
	if !i.Mode().IsRegular() {
		return errors.New("unsafe file")
	}
	return nil
}
func (o *offlineOtherFiles) physicalRoot() (string, error) { return filepath.EvalSymlinks(o.root) }
func (o *offlineOtherFiles) close() error                  { return nil }

func offlineMissing(e error) bool { return errors.Is(e, os.ErrNotExist) }
