//go:build windows

package library

import (
	"encoding/json"
	"errors"
	"golang.org/x/sys/windows"
	"immichclone/internal/winfs"
	"os"
	"runtime/debug"
	"strings"
)

type restoreHandleFS struct {
	stage *winfs.RestoreStage
	sd    *windows.SECURITY_DESCRIPTOR
}

func setRestoreWorkerMemory(n uint64) { debug.SetMemoryLimit(int64(n - n/4)) }
func (f *restoreHandleFS) initial() error {
	names, e := f.stage.Directory().Names()
	if e != nil {
		return e
	}
	if len(names) != 4 {
		return errors.New("unexpected initial stage")
	}
	allowed := map[string]bool{"originals": true, "staging": true, "previews": true, "faces": true}
	for _, name := range names {
		if !allowed[name] {
			return errors.New("unexpected initial stage")
		}
		d, e := f.stage.Directory().Child(name, nil)
		if e != nil {
			return e
		}
		e = restorePrivateDirectory(d)
		children, ne := d.Names()
		ce := d.Close()
		if e != nil || ne != nil || ce != nil {
			return errors.Join(e, ne, ce)
		}
		if len(children) != 0 {
			return errors.New("nonempty initial stage")
		}
	}
	return nil
}
func (f *restoreHandleFS) Create(entry string) (*os.File, error) {
	if !backupPayloadName(entry) {
		return nil, errors.New("unexpected payload")
	}
	d := f.stage.Directory()
	var child *winfs.Directory
	if strings.HasPrefix(entry, "originals/") {
		var e error
		child, e = d.Child("originals", nil)
		if e != nil {
			return nil, e
		}
		defer child.Close()
		d = child
		entry = strings.TrimPrefix(entry, "originals/")
	}
	return d.CreateExclusiveFile(entry, f.sd)
}
func (f *restoreHandleFS) Progress(nonce string, base uint64) (err error) {
	d := f.stage.Directory()
	file, e := d.CreateExclusiveFile(restoreProgressName+".part", f.sd)
	if e != nil {
		return e
	}
	defer func() { err = errors.Join(err, file.Close()) }()
	b, e := json.Marshal(restoreProgress{nonce, base})
	if e != nil {
		return e
	}
	if _, e = file.Write(b); e != nil {
		return e
	}
	if e = file.Sync(); e != nil {
		return e
	}
	return d.RenameFileNoReplace(file, restoreProgressName)
}
func (f *restoreHandleFS) Check(l RestoreLimits) error {
	names, e := f.stage.Directory().Names()
	if e != nil {
		return e
	}
	var total uint64
	for _, name := range names {
		if name == restoreProgressName {
			continue
		}
		if name == "originals" || name == "staging" || name == "previews" || name == "faces" {
			d, e := f.stage.Directory().Child(name, nil)
			if e != nil {
				return e
			}
			e = restorePrivateDirectory(d)
			if e != nil {
				d.Close()
				return e
			}
			children, e := d.Names()
			if e != nil {
				d.Close()
				return e
			}
			if name != "originals" && len(children) != 0 {
				d.Close()
				return errors.New("unexpected derived payload")
			}
			for _, n := range children {
				if !validID(n) {
					d.Close()
					return errors.New("unexpected original name")
				}
				v, e := d.OpenFile(n, false)
				if e != nil {
					d.Close()
					return e
				}
				info, e := v.Stat()
				ce := v.Close()
				if e != nil || ce != nil {
					d.Close()
					return errors.Join(e, ce)
				}
				if info.Size() < 0 || info.Size() > MaxUpload {
					d.Close()
					return ErrRestoreBudget
				}
				total, e = restoreAddWithin(total, uint64(info.Size()), l.MaxBytes)
				if e != nil {
					d.Close()
					return e
				}
			}
			if e = d.Close(); e != nil {
				return e
			}
			continue
		}
		if name != "catalog.db" && name != "state-format.txt" && name != "access-code.txt" {
			return errors.New("unexpected stage name")
		}
		v, e := f.stage.Directory().OpenFile(name, false)
		if e != nil {
			return e
		}
		info, e := v.Stat()
		ce := v.Close()
		if e != nil || ce != nil {
			return errors.Join(e, ce)
		}
		if info.Size() < 0 || (name == "catalog.db" && uint64(info.Size()) > l.MaxCatalogBytes) {
			return ErrRestoreBudget
		}
		total, e = restoreAddWithin(total, uint64(info.Size()), l.MaxBytes)
		if e != nil {
			return e
		}
	}
	return nil
}
