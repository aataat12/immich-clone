//go:build windows

// Package installerfiles coordinates the fixed installer snapshot/restore set.
// Root creation, ACL policy application and service orchestration stay with Setup.
package installerfiles

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"unsafe"

	"golang.org/x/sys/windows"
	"immichclone/internal/winfs"
)

var ErrRefused = errors.New("installer file set refused")
var names = [3]string{"immichclone.exe", "catalog.db", "access-code.txt"}

const readyName = "snapshot-ready.json"
const fileFullAccess uint32 = 0x1f01ff

type fileReceipt struct {
	SHA256 string
	Size   int64
}
type receipt struct {
	Version   int
	App, Data winfs.Identity
	Files     []fileReceipt
}
type roots struct {
	app, data, rollback *winfs.Directory
	appID, dataID       winfs.Identity
}

func within(a, b string) bool {
	r, e := filepath.Rel(a, b)
	return e == nil && r != ".." && !strings.HasPrefix(r, `..\`)
}

func openRoots(app, data string) (_ *roots, err error) {
	if !filepath.IsAbs(app) || !filepath.IsAbs(data) {
		return nil, ErrRefused
	}
	app = filepath.Clean(app)
	data = filepath.Clean(data)
	if within(app, data) || within(data, app) {
		return nil, ErrRefused
	}
	r := &roots{}
	defer func() {
		if err != nil {
			err = errors.Join(err, r.close())
		}
	}()
	if r.app, err = winfs.OpenDirectory(app); err != nil {
		return nil, err
	}
	if r.data, err = winfs.OpenDirectory(data); err != nil {
		return nil, err
	}
	appPhysical, e := r.app.FinalPath()
	if e != nil {
		return nil, e
	}
	dataPhysical, e := r.data.FinalPath()
	if e != nil {
		return nil, e
	}
	a := strings.TrimRight(strings.ToLower(appPhysical), `\`)
	b := strings.TrimRight(strings.ToLower(dataPhysical), `\`)
	if a == b || strings.HasPrefix(a, b+`\`) || strings.HasPrefix(b, a+`\`) {
		return nil, ErrRefused
	}
	if r.rollback, err = winfs.OpenDirectory(filepath.Join(app, "rollback")); err != nil {
		return nil, err
	}
	if r.appID, err = r.app.Identity(); err != nil {
		return nil, err
	}
	if r.dataID, err = r.data.Identity(); err != nil {
		return nil, err
	}
	rollbackID, e := r.rollback.Identity()
	if e != nil {
		return nil, e
	}
	if r.appID == r.dataID || r.appID == rollbackID || r.dataID == rollbackID {
		return nil, ErrRefused
	}
	if err = privateRollback(r.rollback); err != nil {
		return nil, err
	}
	return r, nil
}

// Snapshot contains the catalog and access code. Never write it into a role root
// that a different user or LocalService can read or mutate, even after a path swap.
func privateRollback(d *winfs.Directory) error {
	sd, e := d.SecurityDescriptor()
	if e != nil {
		return e
	}
	return privateDescriptor(sd, true)
}
func privateFile(f *os.File) error {
	sd, e := windows.GetSecurityInfo(windows.Handle(f.Fd()), windows.SE_FILE_OBJECT, windows.OWNER_SECURITY_INFORMATION|windows.DACL_SECURITY_INFORMATION)
	if e != nil {
		return e
	}
	return privateDescriptor(sd, false)
}
func privateDescriptor(sd *windows.SECURITY_DESCRIPTOR, directory bool) error {
	return policyDescriptor(sd, directory, map[string]uint32{"S-1-5-18": fileFullAccess, "S-1-5-32-544": fileFullAccess})
}
func policyDescriptor(sd *windows.SECURITY_DESCRIPTOR, directory bool, policy map[string]uint32) error {
	if sd == nil {
		return ErrRefused
	}
	control, _, e := sd.Control()
	if e != nil {
		return e
	}
	if directory && control&windows.SE_DACL_PROTECTED == 0 {
		return ErrRefused
	}
	owner, _, e := sd.Owner()
	if e != nil || owner == nil {
		return ErrRefused
	}
	u, e := windows.GetCurrentProcessToken().GetTokenUser()
	if e != nil {
		return e
	}
	ownerSID := owner.String()
	if ownerSID != "S-1-5-18" && ownerSID != "S-1-5-32-544" && ownerSID != u.User.Sid.String() && policy[ownerSID] == 0 {
		return ErrRefused
	}
	acl, _, e := sd.DACL()
	if e != nil || acl == nil {
		return ErrRefused
	}
	self := map[string]uint32{}
	inherited := map[string]uint32{}
	for i := uint32(0); i < uint32(acl.AceCount); i++ {
		var ace *windows.ACCESS_ALLOWED_ACE
		if e = windows.GetAce(acl, i, &ace); e != nil {
			return e
		}
		if ace.Header.AceType != windows.ACCESS_ALLOWED_ACE_TYPE {
			return ErrRefused
		}
		sid := (*windows.SID)(unsafe.Pointer(&ace.SidStart)).String()
		rights, allowed := policy[sid]
		if !allowed || uint32(ace.Mask)&^rights != 0 {
			return ErrRefused
		}
		if ace.Header.AceFlags&3 == 3 && ace.Header.AceFlags&12 == 0 {
			inherited[sid] |= uint32(ace.Mask)
		}
		if ace.Header.AceFlags&8 == 0 {
			self[sid] |= uint32(ace.Mask)
		}
	}
	for sid, rights := range policy {
		if self[sid]&rights != rights || (directory && inherited[sid]&rights != rights) {
			return ErrRefused
		}
	}
	return nil
}

func (r *roots) close() (err error) {
	for _, d := range []*winfs.Directory{r.rollback, r.data, r.app} {
		if d != nil {
			err = errors.Join(err, d.Close())
		}
	}
	return err
}

type opened struct {
	source, target [3]*os.File
	ready          *os.File
}

func (o *opened) closeSet() (err error) {
	for i := range names {
		if o.target[i] != nil {
			err = errors.Join(err, o.target[i].Close())
			o.target[i] = nil
		}
		if o.source[i] != nil {
			err = errors.Join(err, o.source[i].Close())
			o.source[i] = nil
		}
	}
	return err
}
func (o *opened) close() error {
	err := o.closeSet()
	if o.ready != nil {
		err = errors.Join(err, o.ready.Close())
		o.ready = nil
	}
	return err
}
func digest(f *os.File) (fileReceipt, error) {
	if _, e := f.Seek(0, io.SeekStart); e != nil {
		return fileReceipt{}, e
	}
	h := sha256.New()
	n, e := io.Copy(h, f)
	return fileReceipt{hex.EncodeToString(h.Sum(nil)), n}, e
}
func copyChecked(dst, src *os.File, expected fileReceipt) error {
	if _, e := src.Seek(0, io.SeekStart); e != nil {
		return e
	}
	if e := dst.Truncate(0); e != nil {
		return e
	}
	if _, e := dst.Seek(0, io.SeekStart); e != nil {
		return e
	}
	if _, e := io.Copy(dst, src); e != nil {
		return e
	}
	if e := dst.Sync(); e != nil {
		return e
	}
	got, e := digest(dst)
	if e != nil {
		return e
	}
	if got != expected {
		return ErrRefused
	}
	return nil
}

func Snapshot(app, data string) error { return snapshot(app, data, nil) }
func snapshot(app, data string, beforeCopy func(int) error) (err error) {
	r, err := openRoots(app, data)
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, r.close()) }()
	o := &opened{}
	defer func() { err = errors.Join(err, o.close()) }()
	record := receipt{Version: 1, App: r.appID, Data: r.dataID, Files: make([]fileReceipt, 3)}
	// Open every source, then every destination, before any existing byte is
	// truncated. All handles stay live until the entire copy set is verified.
	for i, name := range names {
		src := r.data
		if i == 0 {
			src = r.app
		}
		if o.source[i], err = src.OpenFile(name, false); err != nil {
			return err
		}
		if record.Files[i], err = digest(o.source[i]); err != nil {
			return err
		}
	}
	for i, name := range names {
		if o.target[i], err = r.rollback.OpenFile(name, true); err != nil {
			return err
		}
		if err = privateFile(o.target[i]); err != nil {
			return err
		}
	}
	if o.ready, err = r.rollback.OpenFile(readyName, true); err != nil {
		return err
	}
	if err = privateFile(o.ready); err != nil {
		return err
	}
	// Old/stale readiness must not survive a partially overwritten snapshot.
	if err = o.ready.Truncate(0); err != nil {
		return err
	}
	if err = o.ready.Sync(); err != nil {
		return err
	}
	for i := range names {
		if beforeCopy != nil {
			if err = beforeCopy(i); err != nil {
				return err
			}
		}
		if err = copyChecked(o.target[i], o.source[i], record.Files[i]); err != nil {
			return err
		}
	}
	if err = o.closeSet(); err != nil {
		return err
	}
	body, err := json.Marshal(record)
	if err != nil {
		return err
	}
	if _, err = o.ready.Seek(0, io.SeekStart); err != nil {
		return err
	}
	if _, err = o.ready.Write(body); err != nil {
		return err
	}
	if err = o.ready.Sync(); err != nil {
		return err
	}
	err = o.ready.Close()
	o.ready = nil
	return err
}

func Restore(app, data string) (err error) {
	r, err := openRoots(app, data)
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, r.close()) }()
	o := &opened{}
	defer func() { err = errors.Join(err, o.close()) }()
	if o.ready, err = r.rollback.OpenFile(readyName, false); err != nil {
		return err
	}
	if err = privateFile(o.ready); err != nil {
		return err
	}
	body, err := io.ReadAll(io.LimitReader(o.ready, 4097))
	if err != nil {
		return err
	}
	if len(body) > 4096 {
		return ErrRefused
	}
	var record receipt
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.DisallowUnknownFields()
	if err = decoder.Decode(&record); err != nil {
		return ErrRefused
	}
	if decoder.Decode(new(any)) != io.EOF || record.Version != 1 || len(record.Files) != 3 || record.App != r.appID || record.Data != r.dataID {
		return ErrRefused
	}
	// Verify all snapshot files before opening any live destination for writing.
	for i, name := range names {
		if o.source[i], err = r.rollback.OpenFile(name, false); err != nil {
			return err
		}
		if err = privateFile(o.source[i]); err != nil {
			return err
		}
		got, e := digest(o.source[i])
		if e != nil {
			return e
		}
		if got != record.Files[i] {
			return ErrRefused
		}
	}
	for i, name := range names {
		dst := r.data
		if i == 0 {
			dst = r.app
		}
		if o.target[i], err = dst.OpenFile(name, true); err != nil {
			return err
		}
	}
	for i := range names {
		if err = copyChecked(o.target[i], o.source[i], record.Files[i]); err != nil {
			return err
		}
	}
	return o.closeSet()
}
