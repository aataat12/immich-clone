//go:build windows

package installerfiles

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"golang.org/x/sys/windows"
	"immichclone/internal/winfs"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

var payloadRootNames = []string{"Manage-Service.ps1", "Storage-Security.ps1", "installer-files.exe", "immichclone.exe", "Export-Diagnostics.ps1", "THIRD-PARTY-LICENSES.txt", "prototype-notes.txt"}

const publicationReceipt = "publication-ready.json"

type payloadEntry struct {
	Path     string
	HadOld   bool
	Old, New fileReceipt
}
type payloadReceipt struct {
	Version int
	App     winfs.Identity
	Files   []payloadEntry
}
type payloadFile struct {
	entry         payloadEntry
	src, dst, old *os.File
	created       bool
}
type payloadScan struct {
	d     *winfs.Directory
	names []string
}
type payloadSet struct {
	scans  []payloadScan
	dirs   []*winfs.Directory
	files  []*payloadFile
	ready  *os.File
	backup *winfs.Directory
	app    *winfs.Directory
}

func (p *payloadSet) close() (err error) {
	if p.ready != nil {
		err = errors.Join(err, p.ready.Close())
		p.ready = nil
	}
	for _, f := range p.files {
		for _, v := range []*os.File{f.old, f.dst, f.src} {
			if v != nil {
				err = errors.Join(err, v.Close())
			}
		}
	}
	for i := len(p.dirs) - 1; i >= 0; i-- {
		err = errors.Join(err, p.dirs[i].Close())
	}
	return err
}
func trustedDirectory(d *winfs.Directory) error {
	sd, e := d.SecurityDescriptor()
	if e != nil {
		return e
	}
	return trustedCodeDescriptor(sd)
}
func trustedFile(f *os.File) error {
	sd, e := fileDescriptor(f)
	if e != nil {
		return e
	}
	return trustedCodeDescriptor(sd)
}
func codeCreationDescriptor() (*windows.SECURITY_DESCRIPTOR, error) {
	return windows.SecurityDescriptorFromString("D:P(A;OICI;FA;;;SY)(A;OICI;FA;;;BA)(A;OICI;0x1200a9;;;LS)")
}
func (p *payloadSet) child(parent *winfs.Directory, name string, create bool) (*winfs.Directory, error) {
	var sd *windows.SECURITY_DESCRIPTOR
	var e error
	if create {
		sd, e = codeCreationDescriptor()
		if e != nil {
			return nil, e
		}
	}
	d, e := parent.Child(name, sd)
	if e != nil {
		return nil, e
	}
	p.dirs = append(p.dirs, d)
	if e = trustedDirectory(d); e != nil {
		return nil, e
	}
	return d, nil
}
func disjointPayload(a, b *winfs.Directory) error {
	x, e := a.FinalPath()
	if e != nil {
		return e
	}
	y, e := b.FinalPath()
	if e != nil {
		return e
	}
	x = strings.TrimRight(strings.ToLower(x), `\`)
	y = strings.TrimRight(strings.ToLower(y), `\`)
	if x == y || strings.HasPrefix(x, y+`\`) || strings.HasPrefix(y, x+`\`) {
		return ErrRefused
	}
	return nil
}
func openPayloadRoots(app, backup string, create bool) (p *payloadSet, err error) {
	p = &payloadSet{}
	held := p
	defer func() {
		if err != nil {
			err = errors.Join(err, held.close())
		}
	}()
	sd, e := codeCreationDescriptor()
	if e != nil {
		return nil, e
	}
	if create {
		p.app, e = winfs.CreateDirectories(app, sd, trustedDirectory)
	} else {
		p.app, e = winfs.OpenDirectory(app)
	}
	if e != nil {
		return nil, e
	}
	p.dirs = append(p.dirs, p.app)
	if e = trustedDirectory(p.app); e != nil {
		return nil, e
	}
	p.backup, e = winfs.OpenDirectory(backup)
	if e != nil {
		return nil, e
	}
	p.dirs = append(p.dirs, p.backup)
	if e = privateRollback(p.backup); e != nil {
		return nil, e
	}
	if e = disjointPayload(p.app, p.backup); e != nil {
		return nil, e
	}
	return p, nil
}
func payloadPath(path string) bool {
	if path == "" || strings.ContainsAny(path, ":/\x00") || strings.Contains(path, `\\`) {
		return false
	}
	parts := strings.Split(path, `\`)
	for _, part := range parts {
		if part == "" || part == "." || part == ".." || strings.HasSuffix(part, ".") || strings.HasSuffix(part, " ") {
			return false
		}
	}
	if len(parts) == 1 {
		for _, name := range payloadRootNames {
			if path == name {
				return true
			}
		}
		return false
	}
	return parts[0] == "media" || parts[0] == "ai"
}
func (p *payloadSet) targetDirectory(path string, cache map[string]*winfs.Directory, create bool) (*winfs.Directory, error) {
	parent := p.app
	prefix := ""
	parts := strings.Split(path, `\`)
	for _, part := range parts[:len(parts)-1] {
		prefix += part + `\`
		if d := cache[prefix]; d != nil {
			parent = d
			continue
		}
		d, e := p.child(parent, part, create)
		if e != nil {
			return nil, e
		}
		cache[prefix] = d
		parent = d
	}
	return parent, nil
}

// inventory collects a closed fixed-root payload, recursively enumerating held
// provider directories. No arbitrary root files, rollback data or uninstaller.
func (p *payloadSet) inventory(src *winfs.Directory, prefix string) error {
	names, e := src.Names()
	if e != nil {
		return e
	}
	sort.Strings(names)
	p.scans = append(p.scans, payloadScan{src, names})
	expected := map[string]bool{}
	if prefix == "" {
		for _, n := range payloadRootNames {
			expected[n] = true
		}
		expected["media"] = true
		expected["ai"] = true
		if len(names) != len(expected) {
			return ErrRefused
		}
	}
	seen := map[string]bool{}
	for _, name := range names {
		key := strings.ToLower(name)
		if seen[key] {
			return ErrRefused
		}
		seen[key] = true
		if prefix == "" && !expected[name] {
			return ErrRefused
		}
		path := prefix + name
		if prefix == "" && (name == "media" || name == "ai") {
			d, e := p.child(src, name, false)
			if e != nil {
				return e
			}
			if e = p.inventory(d, path+`\`); e != nil {
				return e
			}
			continue
		}
		if !payloadPath(path) {
			return ErrRefused
		}
		f, e := src.OpenFile(name, false)
		if e != nil {
			if prefix == "" || !errors.Is(e, windows.STATUS_FILE_IS_A_DIRECTORY) {
				return e
			}
			// Only a real directory can take this branch. Links and failed file opens do
			// not acquire a permissive path fallback; Child independently opens nofollow.
			d, de := p.child(src, name, false)
			if de != nil {
				return e
			}
			if e = p.inventory(d, path+`\`); e != nil {
				return e
			}
			continue
		}
		row := &payloadFile{entry: payloadEntry{Path: path}, src: f}
		p.files = append(p.files, row)
		if e = trustedFile(f); e != nil {
			return e
		}
		if row.entry.New, e = digest(f); e != nil {
			return e
		}
	}
	again, e := src.Names()
	if e != nil {
		return e
	}
	sort.Strings(again)
	if !equalPayloadNames(names, again) {
		return ErrRefused
	}
	return nil
}
func (p *payloadSet) validate() error {
	for _, scan := range p.scans {
		names, e := scan.d.Names()
		if e != nil {
			return e
		}
		sort.Strings(names)
		if !equalPayloadNames(names, scan.names) {
			return ErrRefused
		}
		if e = trustedDirectory(scan.d); e != nil {
			return e
		}
	}
	for _, f := range p.files {
		for _, v := range []*os.File{f.src, f.dst} {
			if v != nil {
				if e := winfs.ValidateFile(v); e != nil {
					return e
				}
				if e := trustedFile(v); e != nil {
					return e
				}
			}
		}
	}
	return nil
}
func equalPayloadNames(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
func mapDirectories(m map[string]*winfs.Directory) (out []*winfs.Directory) {
	for _, d := range m {
		out = append(out, d)
	}
	return
}
func backupName(i int) string { return fmt.Sprintf("file-%06d.bin", i) }
func (p *payloadSet) restoreHeld() error {
	// All originals were saved and verified before mutation. Never fall back to
	// resolving destination names on an error; rollback uses the held objects.
	var err error
	for _, f := range p.files {
		if f.dst == nil {
			continue
		}
		if f.created {
			err = errors.Join(err, winfs.DeletePublicationFile(f.dst))
			f.created = false
			continue
		}
		if f.old != nil {
			if e := winfs.ValidateFile(f.dst); e != nil {
				err = errors.Join(err, e)
				continue
			}
			err = errors.Join(err, copyChecked(f.dst, f.old, f.entry.Old))
		}
	}
	return err
}
func PublishPayload(app, source, backup string) error {
	return publishPayload(app, source, backup, nil)
}
func publishPayload(app, source, backup string, beforeCopy func(int) error) (err error) {
	if within(app, source) || within(source, app) || within(backup, source) || within(source, backup) || within(app, backup) || within(backup, app) {
		return ErrRefused
	}
	p, e := openPayloadRoots(app, backup, true)
	if e != nil {
		return e
	}
	defer func() { err = errors.Join(err, p.close()) }()
	defer func() {
		if err != nil {
			for _, f := range p.files {
				if f.created && f.dst != nil {
					err = errors.Join(err, winfs.DeletePublicationFile(f.dst))
				}
			}
		}
	}()
	contents, e := p.backup.Names()
	if e != nil {
		return e
	}
	if len(contents) != 0 {
		return ErrRefused
	}
	src, e := winfs.OpenDirectory(source)
	if e != nil {
		return e
	}
	p.dirs = append(p.dirs, src)
	if e = trustedDirectory(src); e != nil {
		return e
	}
	if e = disjointPayload(src, p.app); e != nil {
		return e
	}
	if e = disjointPayload(src, p.backup); e != nil {
		return e
	}
	if e = p.inventory(src, ""); e != nil {
		return e
	}
	sort.Slice(p.files, func(i, j int) bool { return p.files[i].entry.Path < p.files[j].entry.Path })
	cache := map[string]*winfs.Directory{}
	// Open and audit the ENTIRE destination set before truncating an old byte.
	for _, f := range p.files {
		d, e := p.targetDirectory(f.entry.Path, cache, true)
		if e != nil {
			return e
		}
		name := filepath.Base(f.entry.Path)
		f.dst, f.created, e = d.PublicationFile(name)
		if e != nil {
			return e
		}
		if e = trustedFile(f.dst); e != nil {
			return e
		}
		f.entry.HadOld = !f.created
	}
	for _, d := range append([]*winfs.Directory{p.app}, mapDirectories(cache)...) {
		names, e := d.Names()
		if e != nil {
			return e
		}
		sort.Strings(names)
		p.scans = append(p.scans, payloadScan{d, names})
	}
	// If preflight refuses, only safely-created empty leaves/directories may remain.
	// They are never executed and no existing contents have been changed.
	for i, f := range p.files {
		if !f.entry.HadOld {
			continue
		}
		f.entry.Old, e = digest(f.dst)
		if e != nil {
			return e
		}
		f.old, e = p.backup.OpenFile(backupName(i), true)
		if e != nil {
			return e
		}
		if e = privateFile(f.old); e != nil {
			return e
		}
		if e = copyChecked(f.old, f.dst, f.entry.Old); e != nil {
			return e
		}
	}
	p.ready, e = p.backup.OpenFile(publicationReceipt, true)
	if e != nil {
		return e
	}
	if e = privateFile(p.ready); e != nil {
		return e
	}
	changed := false
	defer func() {
		if err != nil && changed {
			err = errors.Join(err, p.restoreHeld())
			if p.ready != nil {
				err = errors.Join(err, p.ready.Truncate(0), p.ready.Sync())
			}
		}
	}()
	for i, f := range p.files {
		if beforeCopy != nil {
			if e = beforeCopy(i); e != nil {
				return e
			}
		}
		if e = p.validate(); e != nil {
			return e
		}
		changed = true
		if e = copyChecked(f.dst, f.src, f.entry.New); e != nil {
			return e
		}
	}
	if e = p.validate(); e != nil {
		return e
	}
	id, e := p.app.Identity()
	if e != nil {
		return e
	}
	receipt := payloadReceipt{Version: 1, App: id}
	for _, f := range p.files {
		receipt.Files = append(receipt.Files, f.entry)
	}
	body, e := json.Marshal(receipt)
	if e != nil {
		return e
	}
	// A close failure is an incomplete operation; no completed receipt and no
	// service restart. Checked copy failures above still restore through held files.
	changed = false
	var closeErr error
	for _, f := range p.files {
		for _, slot := range []**os.File{&f.src, &f.dst, &f.old} {
			if *slot != nil {
				closeErr = errors.Join(closeErr, (*slot).Close())
				*slot = nil
			}
		}
	}
	if closeErr != nil {
		return closeErr
	}
	if _, e = p.ready.Write(body); e != nil {
		return e
	}
	if e = p.ready.Sync(); e != nil {
		return e
	}
	err = p.ready.Close()
	p.ready = nil
	return err
}
func RestorePayload(app, backup string) (err error) {
	p, e := openPayloadRoots(app, backup, false)
	if e != nil {
		return e
	}
	defer func() { err = errors.Join(err, p.close()) }()
	p.ready, e = p.backup.OpenFile(publicationReceipt, false)
	if e != nil {
		return e
	}
	if e = privateFile(p.ready); e != nil {
		return e
	}
	body, e := io.ReadAll(io.LimitReader(p.ready, 4*1024*1024+1))
	if e != nil {
		return e
	}
	if len(body) > 4*1024*1024 {
		return ErrRefused
	}
	var record payloadReceipt
	dec := json.NewDecoder(bytes.NewReader(body))
	dec.DisallowUnknownFields()
	if dec.Decode(&record) != nil || dec.Decode(new(any)) != io.EOF || record.Version != 1 || len(record.Files) < len(payloadRootNames) {
		return ErrRefused
	}
	id, e := p.app.Identity()
	if e != nil {
		return e
	}
	if id != record.App {
		return ErrRefused
	}
	cache := map[string]*winfs.Directory{}
	seen := map[string]bool{}
	// Verify all recovery sources, all selected objects and new payload digests
	// before restoring any byte or deleting a newly-created payload object.
	for i, row := range record.Files {
		if !payloadPath(row.Path) || seen[strings.ToLower(row.Path)] {
			return ErrRefused
		}
		seen[strings.ToLower(row.Path)] = true
		f := &payloadFile{entry: row, created: !row.HadOld}
		p.files = append(p.files, f)
		if row.HadOld {
			f.old, e = p.backup.OpenFile(backupName(i), false)
			if e != nil {
				return e
			}
			if e = privateFile(f.old); e != nil {
				return e
			}
			got, e := digest(f.old)
			if e != nil {
				return e
			}
			if got != row.Old {
				return ErrRefused
			}
		}
		d, e := p.targetDirectory(row.Path, cache, false)
		if e != nil {
			return e
		}
		var created bool
		f.dst, created, e = d.ExistingPublicationFile(filepath.Base(row.Path))
		if e != nil {
			return e
		}
		if created {
			return ErrRefused
		}
		if e = trustedFile(f.dst); e != nil {
			return e
		}
		got, e := digest(f.dst)
		if e != nil {
			return e
		}
		if got != row.New {
			return ErrRefused
		}
	}
	return p.restoreHeld()
}
