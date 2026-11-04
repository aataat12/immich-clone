//go:build windows

package library

import (
	"errors"
	"golang.org/x/sys/windows"
	"immichclone/internal/winfs"
	"os"
)

type offlineWindowsFiles struct{ root, originals *winfs.Directory }

func openOfflineFiles(path string) (_ offlineFiles, err error) {
	r, e := winfs.OpenDirectory(path)
	if e != nil {
		return nil, e
	}
	o := &offlineWindowsFiles{root: r}
	defer func() {
		if err != nil {
			err = errors.Join(err, o.close())
		}
	}()
	o.originals, e = r.Child("originals", nil)
	if e != nil {
		return nil, e
	}
	return o, nil
}
func (o *offlineWindowsFiles) rootFile(n string) (*os.File, error) { return o.root.OpenFile(n, false) }
func (o *offlineWindowsFiles) original(n string) (*os.File, error) {
	return o.originals.OpenFile(n, false)
}
func (o *offlineWindowsFiles) validate(f *os.File) error     { return winfs.ValidateFile(f) }
func (o *offlineWindowsFiles) physicalRoot() (string, error) { return o.root.FinalPath() }
func (o *offlineWindowsFiles) close() error {
	var e error
	if o.originals != nil {
		e = o.originals.Close()
	}
	return errors.Join(e, o.root.Close())
}

func offlineMissing(e error) bool {
	return errors.Is(e, os.ErrNotExist) || errors.Is(e, windows.STATUS_OBJECT_NAME_NOT_FOUND) || errors.Is(e, windows.STATUS_OBJECT_PATH_NOT_FOUND)
}
