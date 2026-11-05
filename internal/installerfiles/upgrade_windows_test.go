//go:build windows

package installerfiles

import (
	"os"
	"path/filepath"
	"testing"
)

func upgradeFixture(t *testing.T) (app, source, data, root string) {
	t.Helper()
	app, source, _, _ = publicationFixture(t)
	data = filepath.Join(filepath.Dir(app), "state")
	root = app + "-upgrade-recovery"
	if e := EnsurePrivateRoot(data, "", false); e != nil {
		t.Fatal(e)
	}
	if e := EnsurePrivateRoot(filepath.Join(app, "rollback"), "", false); e != nil {
		t.Fatal(e)
	}
	for _, name := range []string{"catalog.db", "access-code.txt"} {
		if e := os.WriteFile(filepath.Join(data, name), []byte("old-"+name), 0600); e != nil {
			t.Fatal(e)
		}
	}
	if e := Snapshot(app, data); e != nil {
		t.Fatal(e)
	}
	return
}
func TestUpgradePendingRecoveryPrecedesNewSnapshot(t *testing.T) {
	app, source, data, root := upgradeFixture(t)
	if e := BeginUpgrade(app, data, root, true); e != nil {
		t.Fatal(e)
	}
	exitPublication(t, app, source, filepath.Join(root, "payload"), 5)
	if e := os.WriteFile(filepath.Join(data, "catalog.db"), []byte("candidate catalog"), 0600); e != nil {
		t.Fatal(e)
	}
	if e := BeginUpgrade(app, data, root, true); e == nil {
		t.Fatal("pending transaction overwritten")
	}
	kind, e := RecoverUpgrade(app, data, root)
	if e != nil || kind != "prior" {
		t.Fatal("persistent transaction recovery", kind, e)
	}
	publicationAssertSet(t, app, "prior-")
	assertBytes(t, filepath.Join(data, "catalog.db"), "old-catalog.db")
	assertBytes(t, filepath.Join(data, "access-code.txt"), "old-access-code.txt")
	if kind, e = RecoverUpgrade(app, data, root); e != nil || kind != "none" {
		t.Fatal("retirement retry", kind, e)
	}
	if e = BeginUpgrade(app, data, root, true); e != nil {
		t.Fatal("next transaction", e)
	}
}
func TestUpgradeCommittedRecoveryDoesNotRestorePriorState(t *testing.T) {
	app, source, data, root := upgradeFixture(t)
	if e := BeginUpgrade(app, data, root, true); e != nil {
		t.Fatal(e)
	}
	if e := PublishPayload(app, source, filepath.Join(root, "payload")); e != nil {
		t.Fatal(e)
	}
	if e := os.WriteFile(filepath.Join(data, "catalog.db"), []byte("healthy candidate catalog"), 0600); e != nil {
		t.Fatal(e)
	}
	if e := CommitUpgrade(app, data, root); e != nil {
		t.Fatal(e)
	}
	if kind, e := RecoverUpgrade(app, data, root); e != nil || kind != "none" {
		t.Fatal(kind, e)
	}
	publicationAssertSet(t, app, "candidate-")
	assertBytes(t, filepath.Join(data, "catalog.db"), "healthy candidate catalog")
}
func TestUpgradeWrongStateAndUnknownRetirementNamesRefuse(t *testing.T) {
	app, source, data, root := upgradeFixture(t)
	if e := BeginUpgrade(app, data, root, true); e != nil {
		t.Fatal(e)
	}
	if e := PublishPayload(app, source, filepath.Join(root, "payload")); e != nil {
		t.Fatal(e)
	}
	other := filepath.Join(filepath.Dir(data), "other state")
	if e := EnsurePrivateRoot(other, "", false); e != nil {
		t.Fatal(e)
	}
	if _, e := RecoverUpgrade(app, other, root); e == nil {
		t.Fatal("wrong state accepted")
	}
	publicationAssertSet(t, app, "candidate-")
	unknown := filepath.Join(root, "payload", "unrelated.txt")
	if e := os.WriteFile(unknown, []byte("retain me"), 0600); e != nil {
		t.Fatal(e)
	}
	if e := CommitUpgrade(app, data, root); e == nil {
		t.Fatal("unknown retirement file accepted")
	}
	assertBytes(t, unknown, "retain me")
	// A terminal marker survived the refused retirement. Retrying must finish
	// retirement, never roll back the candidate or catalog.
	if e := os.Remove(unknown); e != nil {
		t.Fatal(e)
	}
	if _, e := RecoverUpgrade(app, data, root); e != nil {
		t.Fatal(e)
	}
	publicationAssertSet(t, app, "candidate-")
}

func TestUpgradeCorruptStateSnapshotRefusesBeforeCodeRollback(t *testing.T) {
	app, source, data, root := upgradeFixture(t)
	if e := BeginUpgrade(app, data, root, true); e != nil {
		t.Fatal(e)
	}
	if e := PublishPayload(app, source, filepath.Join(root, "payload")); e != nil {
		t.Fatal(e)
	}
	if e := os.WriteFile(filepath.Join(app, "rollback", "catalog.db"), []byte("corrupt saved catalog"), 0600); e != nil {
		t.Fatal(e)
	}
	if _, e := RecoverUpgrade(app, data, root); e == nil {
		t.Fatal("corrupt prior catalog accepted")
	}
	publicationAssertSet(t, app, "candidate-")
	assertBytes(t, filepath.Join(data, "catalog.db"), "old-catalog.db")
}
func TestUpgradeEmptyRootAndPreparedOnlyRetry(t *testing.T) {
	app, _, data, root := upgradeFixture(t)
	if e := EnsurePrivateRoot(root, "", false); e != nil {
		t.Fatal(e)
	}
	if kind, e := RecoverUpgrade(app, data, root); e != nil || kind != "none" {
		t.Fatal(kind, e)
	}
	if e := BeginUpgrade(app, data, root, true); e != nil {
		t.Fatal(e)
	}
	if kind, e := RecoverUpgrade(app, data, root); e != nil || kind != "prior" {
		t.Fatal(kind, e)
	}
	publicationAssertSet(t, app, "prior-")
	assertBytes(t, filepath.Join(data, "catalog.db"), "old-catalog.db")
}
func TestUpgradeFreshInterruptedRecoveryPreservesNewState(t *testing.T) {
	app, source, _, root := upgradeFixture(t)
	data := filepath.Join(filepath.Dir(app), "new state")
	if e := BeginUpgrade(app, data, root, false); e != nil {
		t.Fatal(e)
	}
	exitPublication(t, app, source, filepath.Join(root, "payload"), 5)
	if e := EnsurePrivateRoot(data, "", false); e != nil {
		t.Fatal(e)
	}
	if e := os.WriteFile(filepath.Join(data, "catalog.db"), []byte("new state remains"), 0600); e != nil {
		t.Fatal(e)
	}
	if kind, e := RecoverUpgrade(app, data, root); e != nil || kind != "fresh" {
		t.Fatal(kind, e)
	}
	publicationAssertSet(t, app, "prior-")
	assertBytes(t, filepath.Join(data, "catalog.db"), "new state remains")
}
