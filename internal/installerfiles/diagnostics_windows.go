//go:build windows

package installerfiles

import (
	"encoding/json"
	"errors"
	"io"
	"os"
	"strings"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
	"immichclone/internal/winfs"
)

const diagnosticLimit int64 = 4 * 1024 * 1024

func label(value, values string) bool {
	for _, v := range strings.Split(values, "|") {
		if value == v {
			return true
		}
	}
	return false
}
func validStage(stage string) bool {
	return label(stage, "inspect service|audit code permissions|audit storage permissions|snapshot previous installation|validate state location|configure state ACL|initialize state|register LocalService|configure automatic startup|start service|verify service health")
}
func fileDescriptor(f *os.File) (*windows.SECURITY_DESCRIPTOR, error) {
	return windows.GetSecurityInfo(windows.Handle(f.Fd()), windows.SE_FILE_OBJECT, windows.OWNER_SECURITY_INFORMATION|windows.DACL_SECURITY_INFORMATION)
}
func diagnosticPolicy() (map[string]uint32, error) {
	u, e := windows.GetCurrentProcessToken().GetTokenUser()
	if e != nil {
		return nil, e
	}
	p := map[string]uint32{"S-1-5-18": fileFullAccess, "S-1-5-32-544": fileFullAccess, "S-1-5-19": 0x1301bf}
	if _, ok := p[u.User.Sid.String()]; !ok {
		p[u.User.Sid.String()] = 0x1200a9
	}
	return p, nil
}

// Diagnostic appends only fixed installer labels/numbers. Both rotation leaves
// are pinned and audited before bytes change; no pathname rename or fallback.
func Diagnostic(root, event, action, stage string, result int32) (err error) {
	if !label(event, "installer_begin|upgrade_prepare|installer_complete|native_operation_failed|service_registration_failed|installer_failed|win32_error|rollback_complete") || !label(action, "Prepare|Install|Uninstall") || !validStage(stage) {
		return ErrRefused
	}
	body, e := json.Marshal(struct {
		Time   string `json:"time_utc"`
		Event  string `json:"event"`
		Action string `json:"action"`
		Stage  string `json:"stage"`
		Result int32  `json:"result_code"`
	}{time.Now().UTC().Format(time.RFC3339Nano), event, action, stage, result})
	if e != nil {
		return e
	}
	body = append(body, '\n')
	policy, e := diagnosticPolicy()
	if e != nil {
		return e
	}
	d, e := winfs.OpenDirectory(root)
	if e != nil {
		return e
	}
	defer func() { err = errors.Join(err, d.Close()) }()
	sd, e := d.SecurityDescriptor()
	if e != nil {
		return e
	}
	if e = policyDescriptor(sd, true, policy); e != nil {
		return e
	}
	current, e := d.OpenFile("installer.jsonl", true)
	if e != nil {
		return e
	}
	defer func() { err = errors.Join(err, current.Close()) }()
	previous, e := d.OpenFile("installer.jsonl.previous", true)
	if e != nil {
		return e
	}
	defer func() { err = errors.Join(err, previous.Close()) }()
	for _, f := range []*os.File{current, previous} {
		sd, e = fileDescriptor(f)
		if e != nil {
			return e
		}
		if e = policyDescriptor(sd, false, policy); e != nil {
			return e
		}
	}
	info, e := current.Stat()
	if e != nil {
		return e
	}
	if info.Size() > diagnosticLimit {
		return ErrRefused
	}
	if info.Size()+int64(len(body)) > diagnosticLimit {
		// Preserve the full validated current log before replacing its contents.
		if e = previous.Truncate(0); e != nil {
			return e
		}
		if _, e = io.Copy(previous, io.LimitReader(current, diagnosticLimit+1)); e != nil {
			return e
		}
		if e = previous.Sync(); e != nil {
			return e
		}
		if e = current.Truncate(0); e != nil {
			return e
		}
	}
	if _, e = current.Seek(0, io.SeekEnd); e != nil {
		return e
	}
	if _, e = current.Write(body); e != nil {
		return e
	}
	return current.Sync()
}

// trustedCodeDescriptor matches the existing code mutation policy on the actual
// held object. Read-only users are allowed; unknown/object/callback ACEs are not.
func trustedCodeDescriptor(sd *windows.SECURITY_DESCRIPTOR) error {
	if sd == nil {
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
	trusted := map[string]bool{"S-1-5-18": true, "S-1-5-32-544": true, u.User.Sid.String(): true, "S-1-5-80-956008885-3418522649-1831038044-1853292631-2271478464": true}
	if !trusted[owner.String()] {
		return ErrRefused
	}
	acl, _, e := sd.DACL()
	if e != nil || acl == nil {
		return ErrRefused
	}
	for i := uint32(0); i < uint32(acl.AceCount); i++ {
		var ace *windows.ACCESS_ALLOWED_ACE
		if e = windows.GetAce(acl, i, &ace); e != nil {
			return e
		}
		if ace.Header.AceType != windows.ACCESS_ALLOWED_ACE_TYPE {
			return ErrRefused
		}
		sid := (*windows.SID)(unsafe.Pointer(&ace.SidStart)).String()
		if sid == "S-1-3-0" && ace.Header.AceFlags&8 != 0 {
			continue
		}
		if uint32(ace.Mask)&0x500d0156 != 0 && !trusted[sid] {
			return ErrRefused
		}
	}
	return nil
}

// FailureNote never accepts exception text or paths as record content. Callers
// must keep recovery independent of this optional diagnostic operation.
func FailureNote(app, stage string, result int32) (err error) {
	if !validStage(stage) {
		return ErrRefused
	}
	d, e := winfs.OpenDirectory(app)
	if e != nil {
		return e
	}
	defer func() { err = errors.Join(err, d.Close()) }()
	sd, e := d.SecurityDescriptor()
	if e != nil {
		return e
	}
	if e = trustedCodeDescriptor(sd); e != nil {
		return e
	}
	f, e := d.OpenFile("install-diagnostic.txt", true)
	if e != nil {
		return e
	}
	defer func() { err = errors.Join(err, f.Close()) }()
	sd, e = fileDescriptor(f)
	if e != nil {
		return e
	}
	if e = trustedCodeDescriptor(sd); e != nil {
		return e
	}
	body, e := json.Marshal(struct {
		Stage  string `json:"stage"`
		Result int32  `json:"result_code"`
	}{stage, result})
	if e != nil {
		return e
	}
	if e = f.Truncate(0); e != nil {
		return e
	}
	if _, e = f.Write(append(body, '\n')); e != nil {
		return e
	}
	return f.Sync()
}
