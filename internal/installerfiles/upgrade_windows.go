//go:build windows

package installerfiles

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"path/filepath"
	"strings"

	"golang.org/x/sys/windows"
	"immichclone/internal/winfs"
)

const upgradeRecordName = "upgrade-pending.json"
const upgradeTerminalName = "upgrade-terminal.json"

type upgradeRecord struct {
	Version    int
	App, Data  winfs.Identity
	DataKey    string
	HadService bool
	Phase      string
}

func upgradeDataKey(data string) (string, error) {
	if !filepath.IsAbs(data) {
		return "", ErrRefused
	}
	key := sha256.Sum256([]byte(strings.ToLower(filepath.Clean(data))))
	return hex.EncodeToString(key[:]), nil
}
func upgradeRoot(app, data, root string, create bool) (*winfs.Directory, error) {
	if !filepath.IsAbs(app) || !filepath.IsAbs(data) || !filepath.IsAbs(root) || within(app, root) || within(root, app) || within(data, root) || within(root, data) {
		return nil, ErrRefused
	}
	if create {
		if e := EnsurePrivateRoot(root, "", false); e != nil {
			return nil, e
		}
		if e := EnsurePrivateRoot(filepath.Join(root, "payload"), "", false); e != nil {
			return nil, e
		}
	}
	d, e := winfs.OpenDirectory(root)
	if e != nil {
		return nil, e
	}
	if e = privateRollback(d); e != nil {
		d.Close()
		return nil, e
	}
	for _, path := range []string{app, data} {
		other, e := winfs.OpenDirectory(path)
		if missingCleanup(e) {
			continue
		}
		if e != nil {
			d.Close()
			return nil, e
		}
		e = disjointPayload(d, other)
		ce := other.Close()
		if e != nil || ce != nil {
			d.Close()
			return nil, errors.Join(e, ce)
		}
	}
	return d, nil
}
func readUpgradeRecord(d *winfs.Directory, name string) (record upgradeRecord, err error) {
	f, e := d.OpenFile(name, false)
	if e != nil {
		return record, e
	}
	defer func() { err = errors.Join(err, f.Close()) }()
	if e = privateFile(f); e != nil {
		return record, e
	}
	body, e := io.ReadAll(io.LimitReader(f, 4097))
	if e != nil {
		return record, e
	}
	if len(body) > 4096 {
		return record, ErrRefused
	}
	dec := json.NewDecoder(bytes.NewReader(body))
	dec.DisallowUnknownFields()
	if dec.Decode(&record) != nil || dec.Decode(new(any)) != io.EOF || record.Version != 1 {
		return record, ErrRefused
	}
	return record, nil
}
func writeUpgradeRecord(d *winfs.Directory, name string, record upgradeRecord) (err error) {
	body, e := json.Marshal(record)
	if e != nil {
		return e
	}
	sd, e := windows.SecurityDescriptorFromString("D:P(A;;FA;;;SY)(A;;FA;;;BA)")
	if e != nil {
		return e
	}
	f, e := d.CreateExclusiveFile(name, sd)
	if e != nil {
		return e
	}
	defer func() { err = errors.Join(err, f.Close()) }()
	if e = privateFile(f); e != nil {
		return e
	}
	n, e := f.Write(body)
	if e != nil {
		return e
	}
	if n != len(body) {
		return io.ErrShortWrite
	}
	return f.Sync()
}
func validateUpgradeRecord(record upgradeRecord, app, data string) error {
	if record.Phase != "pending" && record.Phase != "committed" && record.Phase != "recovered" {
		return ErrRefused
	}
	key, e := upgradeDataKey(data)
	if e != nil {
		return e
	}
	if key != record.DataKey {
		return ErrRefused
	}
	a, e := winfs.OpenDirectory(app)
	if e != nil {
		return e
	}
	id, e := a.Identity()
	ce := a.Close()
	if e != nil || ce != nil {
		return errors.Join(e, ce)
	}
	if record.App != id {
		return ErrRefused
	}
	if record.Data != (winfs.Identity{}) {
		d, e := winfs.OpenDirectory(data)
		if e != nil {
			return e
		}
		id, e := d.Identity()
		ce := d.Close()
		if e != nil || ce != nil {
			return errors.Join(e, ce)
		}
		if record.Data != id {
			return ErrRefused
		}
	} else if record.HadService {
		return ErrRefused
	}
	return nil
}

// BeginUpgrade runs after the prior state snapshot but before Inno publication.
// An existing pending transaction is never overwritten by a new snapshot.
func BeginUpgrade(app, data, root string, hadService bool) (err error) {
	sd, e := codeCreationDescriptor()
	if e != nil {
		return e
	}
	a, e := winfs.CreateDirectories(app, sd, trustedDirectory)
	if e != nil {
		return e
	}
	id, e := a.Identity()
	ce := a.Close()
	if e != nil || ce != nil {
		return errors.Join(e, ce)
	}
	d, e := upgradeRoot(app, data, root, true)
	if e != nil {
		return e
	}
	defer func() { err = errors.Join(err, d.Close()) }()
	names, e := d.Names()
	if e != nil {
		return e
	}
	if len(names) != 1 || names[0] != "payload" {
		return ErrRefused
	}
	payload, e := d.Child("payload", nil)
	if e != nil {
		return e
	}
	defer func() { err = errors.Join(err, payload.Close()) }()
	if e = privateRollback(payload); e != nil {
		return e
	}
	names, e = payload.Names()
	if e != nil {
		return e
	}
	if len(names) != 0 {
		return ErrRefused
	}
	record := upgradeRecord{Version: 1, App: id, HadService: hadService, Phase: "pending"}
	if record.DataKey, e = upgradeDataKey(data); e != nil {
		return e
	}
	state, e := winfs.OpenDirectory(data)
	if e == nil {
		record.Data, e = state.Identity()
		ce = state.Close()
		if e != nil || ce != nil {
			return errors.Join(e, ce)
		}
	} else if !missingCleanup(e) || hadService {
		return e
	}
	return writeUpgradeRecord(d, upgradeRecordName, record)
}

// retireUpgrade admits only fixed metadata and numbered flat backup leaves.
// All surviving objects are held and audited before the first deletion. A valid
// terminal record remains until all payload/other metadata deletion completes.
func retireUpgrade(d *winfs.Directory) error {
	return retireUpgradeWithHook(d, nil)
}

// The nil production hook leaves the retirement protocol unchanged. Package
// tests can stop at held-object deletion boundaries without a public bypass.
func retireUpgradeWithHook(d *winfs.Directory, beforeDelete func(int) error) (err error) {
	names, e := d.Names()
	if e != nil {
		return e
	}
	for _, name := range names {
		if name != "payload" && name != upgradeRecordName && name != upgradeTerminalName {
			return ErrRefused
		}
	}
	payload, e := d.Child("payload", nil)
	if e != nil {
		return e
	}
	defer func() { err = errors.Join(err, payload.Close()) }()
	if e = privateRollback(payload); e != nil {
		return e
	}
	leaves, e := payload.Names()
	if e != nil {
		return e
	}
	type held struct {
		name string
		n    *winfs.Removal
	}
	var files []held
	defer func() {
		for _, f := range files {
			err = errors.Join(err, f.n.Close())
		}
	}()
	add := func(parent *winfs.Directory, name string) error {
		n, e := parent.OpenRemoval(name)
		if e != nil {
			return e
		}
		files = append(files, held{name, n})
		if n.Directory {
			return ErrRefused
		}
		sd, e := n.SecurityDescriptor()
		if e != nil {
			return e
		}
		return privateDescriptor(sd, false)
	}
	for _, name := range leaves {
		valid := name == publicationReceipt || name == publicationIntent
		if len(name) == 15 && strings.HasPrefix(name, "file-") && strings.HasSuffix(name, ".bin") {
			valid = true
			for _, c := range name[5:11] {
				if c < '0' || c > '9' {
					valid = false
				}
			}
		}
		if !valid {
			return ErrRefused
		}
		if e = add(payload, name); e != nil {
			return e
		}
	}
	// Pending first and terminal last, even if only the terminal survived a kill.
	for _, wanted := range []string{upgradeRecordName, upgradeTerminalName} {
		for _, name := range names {
			if name == wanted {
				if e = add(d, name); e != nil {
					return e
				}
			}
		}
	}
	again, e := payload.Names()
	if e != nil {
		return e
	}
	if len(again) != len(leaves) {
		return ErrRefused
	}
	for _, f := range files {
		if e = f.n.Validate(); e != nil {
			return e
		}
	}
	for i, f := range files {
		if beforeDelete != nil {
			if e = beforeDelete(i); e != nil {
				return e
			}
		}
		if e = f.n.Delete(); e != nil {
			return e
		}
	}
	return nil
}

// RecoverUpgrade must be called with the service stopped and disabled, before
// any new snapshot. Returns prior/fresh for recovered pending work, none when
// there is no pending work. Committed retirement never rolls back a healthy app.
func RecoverUpgrade(app, data, root string) (kind string, err error) {
	d, e := upgradeRoot(app, data, root, false)
	if missingCleanup(e) {
		return "none", nil
	}
	if e != nil {
		return "", e
	}
	defer func() { err = errors.Join(err, d.Close()) }()
	terminal, e := readUpgradeRecord(d, upgradeTerminalName)
	if e == nil {
		if terminal.Phase == "pending" {
			return "", ErrRefused
		}
		if e = validateUpgradeRecord(terminal, app, data); e != nil {
			return "", e
		}
		return "none", retireUpgrade(d)
	}
	if !missingCleanup(e) {
		return "", e
	}
	record, e := readUpgradeRecord(d, upgradeRecordName)
	if missingCleanup(e) {
		names, e := d.Names()
		if e != nil {
			return "", e
		}
		if len(names) == 0 {
			return "none", nil
		}
		if len(names) != 1 || names[0] != "payload" {
			return "", ErrRefused
		}
		payload, e := d.Child("payload", nil)
		if e != nil {
			return "", e
		}
		defer func() { err = errors.Join(err, payload.Close()) }()
		if e = privateRollback(payload); e != nil {
			return "", e
		}
		names, e = payload.Names()
		if e != nil {
			return "", e
		}
		if len(names) != 0 {
			return "", ErrRefused
		}
		return "none", nil
	}
	if e != nil {
		return "", e
	}
	if record.Phase != "pending" {
		return "", ErrRefused
	}
	if e = validateUpgradeRecord(record, app, data); e != nil {
		return "", e
	}
	payload, e := d.Child("payload", nil)
	if e != nil {
		return "", e
	}
	if e = privateRollback(payload); e != nil {
		payload.Close()
		return "", e
	}
	intent, e := payload.OpenFile(publicationIntent, false)
	hasIntent := e == nil
	if intent != nil {
		e = errors.Join(privateFile(intent), intent.Close())
	}
	ce := payload.Close()
	if e != nil && !missingCleanup(e) {
		return "", errors.Join(e, ce)
	}
	if ce != nil {
		return "", ce
	}
	if hasIntent {
		recoverPayload := func() error { return RecoverInterruptedPayload(app, filepath.Join(root, "payload")) }
		if record.HadService {
			e = restoreSnapshot(app, data, recoverPayload)
		} else {
			e = recoverPayload()
		}
		if e != nil {
			return "", e
		}
	}
	record.Phase = "recovered"
	if e = writeUpgradeRecord(d, upgradeTerminalName, record); e != nil {
		return "", e
	}
	if e = retireUpgrade(d); e != nil {
		return "", e
	}
	if record.HadService {
		return "prior", nil
	}
	return "fresh", nil
}
func CommitUpgrade(app, data, root string) (err error) {
	d, e := upgradeRoot(app, data, root, false)
	if e != nil {
		return e
	}
	defer func() { err = errors.Join(err, d.Close()) }()
	record, e := readUpgradeRecord(d, upgradeRecordName)
	if e != nil {
		return e
	}
	if record.Phase != "pending" {
		return ErrRefused
	}
	if e = validateUpgradeRecord(record, app, data); e != nil {
		return e
	}
	record.Phase = "committed"
	if e = writeUpgradeRecord(d, upgradeTerminalName, record); e != nil {
		return e
	}
	return retireUpgrade(d)
}

func AuditUpgrade(app, data, root string) (err error) {
	d, e := upgradeRoot(app, data, root, false)
	if e != nil {
		return e
	}
	defer func() { err = errors.Join(err, d.Close()) }()
	record, e := readUpgradeRecord(d, upgradeRecordName)
	if e != nil {
		return e
	}
	if record.Phase != "pending" {
		return ErrRefused
	}
	return validateUpgradeRecord(record, app, data)
}
