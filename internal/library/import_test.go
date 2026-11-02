package library

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	bolt "go.etcd.io/bbolt"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestImportReceiptsSchedulingReplayAndIndependentRestore(t *testing.T) {
	s, root := openTest(t)
	data := fixture(t)
	files := []ImportFile{{Name: "one.png", Relative: "Trip/one.png", Size: int64(len(data)), Modified: 1}, {Name: "two.png", Relative: "Trip/two.png", Size: int64(len(data)), Modified: 1}}
	m, e := s.CreateImport(files)
	if e != nil {
		t.Fatal(e)
	}
	if e = s.ControlImport(m.ID, "start"); e != nil {
		t.Fatal(e)
	}
	entry, e := s.PutImport(context.Background(), m.ID, 0, bytes.NewReader(data))
	if e != nil || entry.AssetID == "" {
		t.Fatal(entry, e)
	}
	if ok, e := s.ProcessOne(); ok || e != nil {
		t.Fatal("preview ran while importing", e)
	}
	s.SetRecognizer(fakeAnalysis, true)
	s.QueueAnalysis(entry.AssetID)
	if ok, e := s.ProcessAnalysis(context.Background()); ok || e != nil {
		t.Fatal("AI ran while importing", e)
	}
	page, _ := s.ImportPage(m.ID, 0)
	if page.Session.AcceptedFiles != 1 || page.Session.NewFiles != 1 || page.Session.AcceptedBytes != int64(len(data)) {
		t.Fatal(page)
	}
	if _, e = s.PutImport(context.Background(), m.ID, 0, bytes.NewReader(data)); e != nil {
		t.Fatal(e)
	}
	changed := append([]byte(nil), data...)
	changed[len(changed)-1] ^= 1
	if _, e = s.PutImport(context.Background(), m.ID, 0, bytes.NewReader(changed)); !errors.Is(e, ErrImportConflict) {
		t.Fatal("changed replay accepted", e)
	}
	page, _ = s.ImportPage(m.ID, 0)
	if page.Session.AcceptedFiles != 1 || page.Entries[0].AssetID != entry.AssetID {
		t.Fatal("replay changed receipt", page)
	}
	if _, e = s.PutImport(context.Background(), m.ID, 1, bytes.NewReader(data[:10])); !errors.Is(e, ErrImportConflict) {
		t.Fatal("short copy accepted", e)
	}
	if e = s.ControlImport(m.ID, "finish"); e == nil {
		t.Fatal("incomplete import finished")
	}
	var archive bytes.Buffer
	if e = s.Backup(&archive); e != nil {
		t.Fatal(e)
	}
	path := filepath.Join(t.TempDir(), "backup.zip")
	os.WriteFile(path, archive.Bytes(), 0600)
	target := filepath.Join(t.TempDir(), "restored")
	if e = Restore(path, target); e != nil {
		t.Fatal(e)
	}
	s.Close()
	if e = os.Rename(root, root+".offline"); e != nil {
		t.Fatal(e)
	}
	restored, e := Open(target)
	if e != nil {
		t.Fatal(e)
	}
	defer restored.Close()
	page, e = restored.ImportPage(m.ID, 0)
	if e != nil || page.Session.Status != "paused" || page.Session.AcceptedFiles != 1 || restored.ImportsRunning() {
		t.Fatal("restore stranded lease/receipt", page, e)
	}
	if e = restored.ControlImport(m.ID, "start"); e != nil {
		t.Fatal(e)
	}
	second, e := restored.PutImport(context.Background(), m.ID, 1, bytes.NewReader(data))
	if e != nil || !second.Duplicate {
		t.Fatal("duplicate resume", second, e)
	}
	if e = restored.ControlImport(m.ID, "finish"); e != nil {
		t.Fatal(e)
	}
	page, _ = restored.ImportPage(m.ID, 0)
	if page.Session.AcceptedFiles != 2 || page.Session.Duplicates != 1 || page.Session.NewFiles != 1 || restored.ImportsRunning() {
		t.Fatal(page)
	}
	a, _ := restored.Get(entry.AssetID)
	if len(a.Sources) != 2 {
		t.Fatal("relative aliases lost")
	}
	if ok, e := restored.ProcessOne(); !ok || e != nil {
		t.Fatal("preview did not drain", e)
	}
	if e = restored.Verify(); e != nil {
		t.Fatal(e)
	}
}
func TestImportExpiryCancellationAndAtomicReceipt(t *testing.T) {
	s, _ := openTest(t)
	data := fixture(t)
	m, _ := s.CreateImport([]ImportFile{{Name: "x.png", Size: int64(len(data))}})
	other, _ := s.CreateImport([]ImportFile{{Name: "y.png", Size: 1}})
	s.ControlImport(m.ID, "start")
	if e := s.ControlImport(other.ID, "start"); !errors.Is(e, ErrImportConflict) {
		t.Fatal("two active sessions")
	}
	s.importLease.Store(&importLease{ID: m.ID, Until: time.Now().Add(-time.Second)})
	if s.ImportsRunning() {
		t.Fatal("expired pause stranded")
	}
	if e := s.ControlImport(m.ID, "heartbeat"); e == nil {
		t.Fatal("heartbeat revived expired session")
	}
	if _, e := s.PutImport(context.Background(), m.ID, 0, bytes.NewReader(data)); e == nil {
		t.Fatal("expired copy accepted")
	}
	s.ControlImport(m.ID, "start")
	// Force receipt publication to fail after a valid asset write inside the same TX.
	// Exercise the shared transaction at the receipt publication boundary.
	_, _, e := s.putWithPath("x.png", "", bytes.NewReader(data), int64(len(data)), "", func(tx *bolt.Tx, a Asset, d bool) error { return errors.New("injected receipt failure") })
	if e == nil {
		t.Fatal("injected receipt commit succeeded")
	}
	assets, _ := s.List()
	if len(assets) != 0 {
		t.Fatal("asset survived receipt rollback")
	}
}

type brokenImportReader struct{}

func (brokenImportReader) Read([]byte) (int, error) { return 0, io.ErrUnexpectedEOF }
func TestImportInterruptedBodyAndCancelPreserveAccepted(t *testing.T) {
	s, root := openTest(t)
	data := fixture(t)
	m, _ := s.CreateImport([]ImportFile{{Name: "x.png", Size: int64(len(data))}, {Name: "y.png", Size: int64(len(data))}})
	s.ControlImport(m.ID, "start")
	first, e := s.PutImport(context.Background(), m.ID, 0, bytes.NewReader(data))
	if e != nil {
		t.Fatal(e)
	}
	if _, e = s.PutImport(context.Background(), m.ID, 1, brokenImportReader{}); e == nil {
		t.Fatal("failed read accepted")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, e = s.PutImport(ctx, m.ID, 1, bytes.NewReader(data)); e == nil {
		t.Fatal("cancelled body accepted")
	}
	page, _ := s.ImportPage(m.ID, 0)
	if page.Session.AcceptedFiles != 1 || page.Entries[1].AssetID != "" || page.Entries[1].Error == "" {
		t.Fatal(page)
	}
	s.ControlImport(m.ID, "cancel")
	if s.ImportsRunning() {
		t.Fatal("cancel stranded pause")
	}
	if _, e = s.PutImport(context.Background(), m.ID, 1, bytes.NewReader(data)); e == nil {
		t.Fatal("cancelled session accepted")
	}
	if e = s.Verify(); e != nil {
		t.Fatal(e)
	}
	s.Close()
	again, e := Open(root)
	if e != nil {
		t.Fatal(e)
	}
	defer again.Close()
	page, _ = again.ImportPage(m.ID, 0)
	if page.Session.Status != "cancelled" || page.Entries[0].AssetID != first.AssetID {
		t.Fatal("restart lost receipts")
	}
}
func TestImportManifestGuardsAndPaging(t *testing.T) {
	s, _ := openTest(t)
	for _, files := range [][]ImportFile{nil, {{Name: "empty", Size: 0}}, {{Name: "huge", Size: MaxUpload + 1}}, {{Name: "x", Relative: "../x", Size: 1}}, {{Name: "x", Size: 1}, {Name: "x", Size: 1}}} {
		if _, e := s.CreateImport(files); e == nil {
			t.Fatal("bad manifest accepted")
		}
	}
	files := []ImportFile{}
	for i := 0; i < 270; i++ {
		files = append(files, ImportFile{Name: fmt.Sprintf("synthetic-%04d.txt", i), Size: 1})
	}
	m, e := s.CreateImport(files)
	if e != nil {
		t.Fatal(e)
	}
	p, e := s.ImportPage(m.ID, 0)
	if e != nil || len(p.Entries) != 250 || p.Next != 250 {
		t.Fatal(p, e)
	}
	p, e = s.ImportPage(m.ID, 250)
	if e != nil || len(p.Entries) != 20 || p.Next != -1 {
		t.Fatal(p, e)
	}
}

func TestImportRestoredCatalogCheck(t *testing.T) {
	root := os.Getenv("IMMICH_ORGANIZATION_RESTORE")
	if root == "" {
		t.Skip("cloud restored library checkpoint")
	}
	s, e := Open(root)
	if e != nil {
		t.Fatal(e)
	}
	defer s.Close()
	items, e := s.ImportHistory()
	if e != nil {
		t.Fatal(e)
	}
	completed, cancelled := false, false
	for _, m := range items {
		if m.Status == "completed" && m.AcceptedFiles == 4 && m.NewFiles == 3 && m.Duplicates == 1 {
			completed = true
		}
		if m.Status == "cancelled" && m.AcceptedFiles == 1 {
			cancelled = true
		}
	}
	if !completed || !cancelled || s.ImportsRunning() {
		t.Fatal("independent restore lost import receipt history or retained lease")
	}
	if e = s.Verify(); e != nil {
		t.Fatal(e)
	}
	t.Log("completed/cancelled import receipts restored without old data; no processing pause retained")
}
