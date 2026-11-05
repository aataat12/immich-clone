//go:build windows

package installerfiles

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestNamespaceCreateOnlyParentAllowsPrivateChildren(t *testing.T) {
	parent := t.TempDir()
	policy := "D:P(A;OICI;FA;;;SY)(A;OICI;FA;;;BA)(A;;0x1201bf;;;BU)(A;OICIIO;GA;;;CO)"
	setPrivacyDescriptor(t, parent, policy)
	app := filepath.Join(parent, "application")
	data := filepath.Join(parent, "state")
	txn := filepath.Join(parent, "recovery")
	if e := AuditNamespaceParents(app, data, txn); e != nil {
		t.Fatal(e)
	}
	entries, e := os.ReadDir(parent)
	if e != nil || len(entries) != 0 {
		t.Fatal("read-only parent audit changed namespace", e)
	}
	if e := EnsurePrivateRoot(data, "S-1-5-19", true); e != nil {
		t.Fatal(e)
	}
	if e := BeginUpgrade(app, data, txn, false); e != nil {
		t.Fatal(e)
	}
	if e := AuditNamespaceParents(filepath.Join(parent, "missing", "nested")); e != nil {
		t.Fatal(e)
	}
}

func TestNamespaceReplacementRightsRefusedWithoutMutation(t *testing.T) {
	for _, mask := range []uint32{0x10000, 0x40, 0x40000, 0x80000, 0x10000000, 0x40000000} {
		t.Run(fmt.Sprintf("%x", mask), func(t *testing.T) {
			parent := t.TempDir()
			setPrivacyDescriptor(t, parent, fmt.Sprintf("D:P(A;OICI;FA;;;SY)(A;OICI;FA;;;BA)(A;;0x%x;;;BU)", mask))
			app, data, txn := filepath.Join(parent, "app"), filepath.Join(parent, "data"), filepath.Join(parent, "txn")
			if e := AuditNamespaceParents(app, data, txn); e == nil {
				t.Fatal("replacement authority accepted")
			}
			if e := BeginUpgrade(app, data, txn, false); e == nil {
				t.Fatal("replacement authority reached upgrade writes")
			}
			entries, e := os.ReadDir(parent)
			if e != nil || len(entries) != 0 {
				t.Fatal("refusal changed namespace", e)
			}
		})
	}
}

func TestNamespaceExistingHostileAppRefused(t *testing.T) {
	parent := t.TempDir()
	app, data, txn := filepath.Join(parent, "app"), filepath.Join(parent, "data"), filepath.Join(parent, "txn")
	if e := os.Mkdir(app, 0700); e != nil {
		t.Fatal(e)
	}
	setPrivacyDescriptor(t, app, "D:P(A;OICI;FA;;;SY)(A;OICI;FA;;;BA)(A;OICI;FA;;;BU)")
	if e := BeginUpgrade(app, data, txn, false); e == nil {
		t.Fatal("existing hostile application name accepted")
	}
	if _, e := os.Stat(filepath.Join(txn, upgradeRecordName)); !os.IsNotExist(e) {
		t.Fatal("hostile application caused durable transaction write", e)
	}
}
