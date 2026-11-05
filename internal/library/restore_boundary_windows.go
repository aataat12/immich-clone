//go:build windows

package library

import (
	"archive/zip"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"golang.org/x/sys/windows"
	"immichclone/internal/winfs"
	"io"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
	"unsafe"
)

const restoreHandleWorkerArg = "--immich-restore-handles-v2"

var restoreBoundaryStageCreated func(string) error

type restoreHandleRequest struct {
	Nonce, Stage               string
	Limits                     RestoreLimits
	StageHandle, ArchiveHandle uint64
	StageID, ArchiveID         winfs.Identity
}
type restoreBoundary struct {
	parent, archiveParent *winfs.Directory
	stage                 *winfs.RestoreStage
	archive               *os.File
	path                  string
	sd                    *windows.SECURITY_DESCRIPTOR
}

func restorePrivateDescriptor() (*windows.SECURITY_DESCRIPTOR, string, error) {
	return restoreDescriptor(true)
}

// Descendants inherit only from the protected private restore root. Keeping
// their inheritance enabled lets the installer later grant LocalService access
// through its audited held-tree normalizer, without rewriting protected children.
func restoreChildDescriptor() (*windows.SECURITY_DESCRIPTOR, string, error) {
	return restoreDescriptor(false)
}

func restoreDescriptor(protected bool) (*windows.SECURITY_DESCRIPTOR, string, error) {
	var elevated, size uint32
	if e := windows.GetTokenInformation(windows.GetCurrentProcessToken(), windows.TokenElevation, (*byte)(unsafe.Pointer(&elevated)), 4, &size); e != nil {
		return nil, "", e
	}
	if size != 4 {
		return nil, "", errors.New("unknown elevation")
	}
	owner := "S-1-5-32-544"
	text := "O:BAG:BAD:(A;OICI;FA;;;SY)(A;OICI;FA;;;BA)"
	if elevated == 0 {
		u, e := windows.GetCurrentProcessToken().GetTokenUser()
		if e != nil {
			return nil, "", e
		}
		owner = u.User.Sid.String()
		text = "O:" + owner + "D:(A;OICI;FA;;;SY)(A;OICI;FA;;;BA)(A;OICI;FA;;;" + owner + ")"
	}
	if protected {
		text = strings.Replace(text, "D:", "D:P", 1)
	} else {
		// Native file creation treats a supplied legacy DACL as protected.
		// Mark this private descendant policy for automatic inheritance too.
		text = strings.Replace(text, "D:", "D:AI", 1)
	}
	sd, e := windows.SecurityDescriptorFromString(text)
	return sd, owner, e
}
func restorePrivateDirectory(d *winfs.Directory) error {
	return restoreDirectoryPolicy(d, true)
}
func restorePrivateChildDirectory(d *winfs.Directory) error {
	return restoreDirectoryPolicy(d, false)
}
func restoreDirectoryPolicy(d *winfs.Directory, protected bool) error {
	_, owner, e := restorePrivateDescriptor()
	if e != nil {
		return e
	}
	sd, e := d.SecurityDescriptor()
	if e != nil {
		return e
	}
	actual, _, e := sd.Owner()
	if e != nil || actual == nil || actual.String() != owner {
		return errors.New("untrusted restore owner")
	}
	ctl, _, e := sd.Control()
	if e != nil || (ctl&windows.SE_DACL_PROTECTED != 0) != protected {
		return errors.New("unexpected restore inheritance")
	}
	acl, _, e := sd.DACL()
	if e != nil || acl == nil {
		return errors.New("missing private policy")
	}
	allowed := map[string]bool{"S-1-5-18": true, "S-1-5-32-544": true, owner: true}
	self := map[string]uint32{}
	children := map[string]uint32{}
	for i := uint32(0); i < uint32(acl.AceCount); i++ {
		var a *windows.ACCESS_ALLOWED_ACE
		if e = windows.GetAce(acl, i, &a); e != nil {
			return e
		}
		sid := (*windows.SID)(unsafe.Pointer(&a.SidStart)).String()
		if a.Header.AceType != windows.ACCESS_ALLOWED_ACE_TYPE || !allowed[sid] || uint32(a.Mask)&^uint32(0x1f01ff) != 0 {
			return errors.New("unexpected restore permission")
		}
		if a.Header.AceFlags&8 == 0 {
			self[sid] |= uint32(a.Mask)
		}
		if a.Header.AceFlags&3 == 3 && a.Header.AceFlags&12 == 0 {
			children[sid] |= uint32(a.Mask)
		}
	}
	for sid := range allowed {
		if self[sid]&0x1f01ff != 0x1f01ff || children[sid]&0x1f01ff != 0x1f01ff {
			return errors.New("incomplete private restore permission")
		}
	}
	return nil
}
func restoreFileIdentity(f *os.File) (winfs.Identity, error) {
	var i windows.ByHandleFileInformation
	if e := windows.GetFileInformationByHandle(windows.Handle(f.Fd()), &i); e != nil {
		return winfs.Identity{}, e
	}
	return winfs.Identity{Volume: i.VolumeSerialNumber, High: i.FileIndexHigh, Low: i.FileIndexLow}, nil
}
func (b *restoreBoundary) close() (err error) {
	if b.archive != nil {
		err = errors.Join(err, b.archive.Close())
	}
	if b.stage != nil {
		err = errors.Join(err, b.stage.Close())
	}
	if b.archiveParent != nil {
		err = errors.Join(err, b.archiveParent.Close())
	}
	if b.parent != nil {
		err = errors.Join(err, b.parent.Close())
	}
	return err
}
func restorePlatform(ctx context.Context, archive, target string, l RestoreLimits) (err error) {
	if e := ctx.Err(); e != nil {
		return e
	}
	target, e := filepath.Abs(target)
	if e != nil {
		return e
	}
	archive, e = filepath.Abs(archive)
	if e != nil {
		return e
	}
	b := &restoreBoundary{}
	defer func() { err = errors.Join(err, b.close()) }()
	b.parent, e = winfs.OpenDirectory(filepath.Dir(target))
	if e != nil {
		return e
	}
	if f, e := b.parent.OpenMetadataFile(filepath.Base(target)); e == nil {
		f.Close()
		return errors.New("restore destination exists")
	} else if !offlineMissing(e) {
		return errors.New("restore destination is not absent")
	}
	// A directory existing at the target is also refused by the exclusive rename;
	// check it here without creating or repairing any selected object.
	if d, e := b.parent.Child(filepath.Base(target), nil); e == nil {
		d.Close()
		return errors.New("restore destination exists")
	} else if !offlineMissing(e) {
		return e
	}
	b.archiveParent, e = winfs.OpenDirectory(filepath.Dir(archive))
	if e != nil {
		return e
	}
	b.archive, e = b.archiveParent.OpenFile(filepath.Base(archive), false)
	if e != nil {
		return e
	}
	// The selected nofollow archive file is now held read-only and cannot be
	// replaced or written. Release redundant directory pins: an archive inside
	// the destination parent must not retain a strict pin that blocks rename.
	if e = b.archiveParent.Close(); e != nil {
		return e
	}
	b.archiveParent = nil
	l, e = l.defaults(filepath.Dir(target))
	if e != nil {
		return e
	}
	b.sd, _, e = restorePrivateDescriptor()
	if e != nil {
		return e
	}
	random := make([]byte, 32)
	if _, e = rand.Read(random); e != nil {
		return e
	}
	nonce := hex.EncodeToString(random)
	name := "immich-restore-" + nonce
	b.path = filepath.Join(filepath.Dir(target), name)
	b.stage, e = b.parent.CreateExclusiveDirectory(name, b.sd)
	if e != nil {
		return e
	}
	cleanup := true
	defer func() {
		if cleanup {
			if e := b.stage.Remove(); e != nil {
				err = errors.Join(err, ErrRestoreCleanup)
			}
		}
	}()
	if e = restorePrivateDirectory(b.stage.Directory()); e != nil {
		return e
	}
	// Keep the initial strict parent pin until this protected, DELETE-pinned
	// child makes the parent persistently nonempty. Native rename needs WRITE
	// sharing on its target directory; nonempty directories cannot acquire a
	// reparse point. Check identity while both versions of the pin are held.
	parentID, e := b.parent.Identity()
	if e != nil {
		return e
	}
	publicationParent, e := winfs.OpenRestoreParent(filepath.Dir(target))
	if e != nil {
		return e
	}
	publicationID, e := publicationParent.Identity()
	if e != nil || publicationID != parentID {
		publicationParent.Close()
		return errors.New("restore parent identity changed")
	}
	oldParent := b.parent
	b.parent = publicationParent
	if e = oldParent.Close(); e != nil {
		return e
	}
	childSD, _, e := restoreChildDescriptor()
	if e != nil {
		return e
	}
	for _, n := range []string{"originals", "staging", "previews", "faces"} {
		d, e := b.stage.Directory().Child(n, childSD)
		if e != nil {
			return e
		}
		e = restorePrivateChildDirectory(d)
		ce := d.Close()
		if e != nil || ce != nil {
			return errors.Join(e, ce)
		}
	}
	if restoreBoundaryStageCreated != nil {
		if e = restoreBoundaryStageCreated(b.path); e != nil {
			return e
		}
	}
	stageID, e := b.stage.Identity()
	if e != nil {
		return e
	}
	archiveID, e := restoreFileIdentity(b.archive)
	if e != nil {
		return e
	}
	request := restoreHandleRequest{Nonce: nonce, Stage: b.path, Limits: l, StageHandle: uint64(b.stage.Handle()), ArchiveHandle: uint64(b.archive.Fd()), StageID: stageID, ArchiveID: archiveID}
	data, e := json.Marshal(request)
	if e != nil || len(data) > 1<<20 {
		return ErrRestoreBudget
	}
	r, w, e := os.Pipe()
	if e != nil {
		return e
	}
	defer r.Close()
	defer w.Close()
	exe, e := os.Executable()
	if e != nil {
		return e
	}
	bounded, cancel := context.WithTimeout(ctx, l.Timeout)
	defer cancel()
	mon, stop := context.WithCancel(bounded)
	done := make(chan error, 1)
	go func() { done <- b.monitor(mon, nonce, l, cancel) }()
	wrote := make(chan error, 1)
	go func() { _, e := w.Write(data); ce := w.Close(); wrote <- errors.Join(e, ce) }()
	run, e := runRestoreProcessInherited(bounded, exe, []string{restoreHandleWorkerArg, strconv.FormatUint(uint64(r.Fd()), 10)}, l.MemoryBytes, restoreOutputBytes, nil, []windows.Handle{windows.Handle(r.Fd()), windows.Handle(b.archive.Fd()), b.stage.Handle()})
	r.Close()
	w.Close()
	stop()
	monitorErr := <-done
	if !run.Reaped && run.PID != 0 {
		cleanup = false
		return errors.New("restore child not reaped; private staging retained")
	}
	var writeErr error
	select {
	case writeErr = <-wrote:
	case <-time.After(time.Second):
		return errors.New("request handoff did not complete")
	}
	if monitorErr != nil {
		return monitorErr
	}
	if ce := ctx.Err(); ce != nil {
		return ce
	}
	if e != nil {
		return e
	}
	if writeErr != nil {
		return writeErr
	}
	if e = bounded.Err(); e != nil {
		return e
	}
	var status struct{ Code string }
	if json.Unmarshal(run.Output, &status) != nil {
		return restoreStageError("worker-status-json", nil)
	}
	if status.Code == "budget" {
		return ErrRestoreBudget
	}
	if status.Code == "invalid" {
		return ErrRestoreInvalidArchive
	}
	if status.Code != "ok" {
		return restoreStageError("worker-status-code", nil)
	}
	if e = b.progress(nonce, l); e != nil {
		return e
	}
	p, e := b.stage.Directory().OpenRemoval(restoreProgressName)
	if e != nil {
		return e
	}
	if e = p.Delete(); e != nil {
		p.Close()
		return e
	}
	if e = bounded.Err(); e != nil {
		return e
	}
	if e = b.stage.Publish(b.parent, filepath.Base(target)); e != nil {
		return e
	}
	cleanup = false
	return nil
}
func (b *restoreBoundary) progress(nonce string, l RestoreLimits) error {
	f, e := b.stage.Directory().OpenFile(restoreProgressName, false)
	if offlineMissing(e) || errors.Is(e, windows.STATUS_SHARING_VIOLATION) || errors.Is(e, windows.ERROR_SHARING_VIOLATION) {
		return errRestoreProgressNotReady
	}
	if e != nil {
		return restoreStageError("progress-read", e)
	}
	p, e := io.ReadAll(io.LimitReader(f, 4097))
	ce := f.Close()
	if e != nil || ce != nil {
		return errors.Join(e, ce)
	}
	if len(p) > 4096 {
		return ErrRestoreBudget
	}
	var v restoreProgress
	if json.Unmarshal(p, &v) != nil || v.Nonce != nonce || v.BaseBytes > l.MaxBytes {
		return restoreStageError("progress-validation", nil)
	}
	cat, e := b.stage.Directory().OpenMetadataFile("catalog.db")
	if e != nil {
		return restoreStageError("catalog-stat", e)
	}
	info, e := cat.Stat()
	ce = cat.Close()
	if e != nil || ce != nil {
		return errors.Join(e, ce)
	}
	if info.Size() < 0 || uint64(info.Size()) > l.MaxCatalogBytes || uint64(info.Size()) > l.MaxBytes-v.BaseBytes {
		return ErrRestoreBudget
	}
	return nil
}
func (b *restoreBoundary) monitor(ctx context.Context, nonce string, l RestoreLimits, cancel context.CancelFunc) error {
	t := time.NewTicker(restorePollInterval)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-t.C:
			if e := b.progress(nonce, l); e != nil && !errors.Is(e, errRestoreProgressNotReady) {
				cancel()
				return e
			}
		}
	}
}

// The pre-main Windows worker accepts no path-request protocol. Only an actual
// bounded child of this same image may consume the inherited request pipe.
func restoreWorkerPlatform() (bool, int) {
	if len(os.Args) < 2 {
		return false, 0
	}
	if os.Args[1] == restoreWorkerArg {
		return true, 2
	}
	if os.Args[1] != restoreHandleWorkerArg {
		return false, 0
	}
	code := func() (code int) {
		defer func() {
			if recover() != nil {
				code = 2
			}
		}()
		if len(os.Args) != 3 || !restoreWorkerContext() {
			return 2
		}
		value, e := strconv.ParseUint(os.Args[2], 10, 64)
		if e != nil {
			return 2
		}
		pipe := windows.Handle(value)
		kind, e := windows.GetFileType(pipe)
		if e != nil || kind != windows.FILE_TYPE_PIPE {
			return 2
		}
		rf := os.NewFile(uintptr(pipe), "inherited-request")
		defer rf.Close()
		data, e := io.ReadAll(io.LimitReader(rf, (1<<20)+1))
		if e != nil || len(data) > 1<<20 {
			return 2
		}
		var req restoreHandleRequest
		dec := json.NewDecoder(strings.NewReader(string(data)))
		dec.DisallowUnknownFields()
		if dec.Decode(&req) != nil || dec.Decode(new(any)) != io.EOF || !validID(req.Nonce) || !filepath.IsAbs(req.Stage) || !strings.HasPrefix(filepath.Base(req.Stage), "immich-restore-") {
			return 2
		}
		if req.Limits.MemoryBytes < 32<<20 || req.Limits.MemoryBytes >= math.MaxInt64 {
			return 2
		}
		if !restoreWorkerLimits(req.Limits.MemoryBytes) {
			return 2
		}
		stage, e := winfs.StageFromHandle(windows.Handle(req.StageHandle))
		if e != nil {
			return 2
		}
		defer stage.Close()
		id, e := stage.Identity()
		if e != nil || id != req.StageID || restorePrivateDirectory(stage.Directory()) != nil {
			return 2
		}
		parent, e := winfs.OpenDirectory(filepath.Dir(req.Stage))
		if e != nil {
			return 2
		}
		pp, e := parent.FinalPath()
		parent.Close()
		if e != nil {
			return 2
		}
		sp, e := stage.Directory().FinalPath()
		if e != nil || !strings.EqualFold(strings.TrimRight(pp, `\`)+`\`+filepath.Base(req.Stage), sp) {
			return 2
		}
		af := os.NewFile(uintptr(req.ArchiveHandle), "inherited-archive")
		defer af.Close()
		if winfs.ValidateFile(af) != nil {
			return 2
		}
		aid, e := restoreFileIdentity(af)
		if e != nil || aid != req.ArchiveID {
			return 2
		}
		info, e := af.Stat()
		if e != nil {
			return 2
		}
		fs := &restoreHandleFS{stage: stage, sd: nil}
		fs.sd, _, e = restoreChildDescriptor()
		if e != nil || fs.initial() != nil {
			return 2
		}
		setRestoreWorkerMemory(req.Limits.MemoryBytes)
		zr, e := zip.NewReader(af, info.Size())
		if e == nil {
			e = restoreArchiveReaderToStage(zr, req.Stage, req.Limits, req.Nonce, fs)
		}
		status := "ok"
		if e != nil {
			status = "invalid"
			if errors.Is(e, ErrRestoreBudget) {
				status = "budget"
			}
		}
		json.NewEncoder(os.Stdout).Encode(struct{ Code string }{status})
		return 0
	}()
	return true, code
}
func restoreWorkerContext() bool {
	var cur windows.PROCESS_BASIC_INFORMATION
	size := uint32(unsafe.Sizeof(cur))
	if windows.NtQueryInformationProcess(windows.CurrentProcess(), windows.ProcessBasicInformation, unsafe.Pointer(&cur), size, &size) != nil {
		return false
	}
	h, e := windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION, false, uint32(cur.InheritedFromUniqueProcessId))
	if e != nil {
		return false
	}
	defer windows.CloseHandle(h)
	b := make([]uint16, 32768)
	n := uint32(len(b))
	if windows.QueryFullProcessImageName(h, 0, &b[0], &n) != nil {
		return false
	}
	exe, e := os.Executable()
	if e != nil || !strings.EqualFold(windows.UTF16ToString(b[:n]), exe) {
		return false
	}
	var lim windows.JOBOBJECT_EXTENDED_LIMIT_INFORMATION
	if windows.QueryInformationJobObject(0, windows.JobObjectExtendedLimitInformation, uintptr(unsafe.Pointer(&lim)), uint32(unsafe.Sizeof(lim)), nil) != nil {
		return false
	}
	return lim.BasicLimitInformation.ActiveProcessLimit == 1 && lim.BasicLimitInformation.LimitFlags&windows.JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE != 0
}
func restoreWorkerLimits(memory uint64) bool {
	var l windows.JOBOBJECT_EXTENDED_LIMIT_INFORMATION
	if windows.QueryInformationJobObject(0, windows.JobObjectExtendedLimitInformation, uintptr(unsafe.Pointer(&l)), uint32(unsafe.Sizeof(l)), nil) != nil {
		return false
	}
	flags := uint32(windows.JOB_OBJECT_LIMIT_PROCESS_MEMORY | windows.JOB_OBJECT_LIMIT_JOB_MEMORY | windows.JOB_OBJECT_LIMIT_ACTIVE_PROCESS | windows.JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE)
	return l.BasicLimitInformation.LimitFlags&flags == flags && uint64(l.ProcessMemoryLimit) == memory && uint64(l.JobMemoryLimit) == memory
}
