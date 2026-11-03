package library

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestImportOriginalsDirectoryUnavailableRecovery(t *testing.T) {
	root := filepath.Join(t.TempDir(), "state")
	s, err := Open(root)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if s != nil {
			s.Close()
		}
	}()
	a, b := []byte("earlier destination outage original"), bytes.Repeat([]byte{79}, 1<<20)
	m, err := s.CreateImport([]ImportFile{{Name: "a.txt", Size: int64(len(a))}, {Name: "b.bin", Size: int64(len(b))}, {Name: "c.txt", Size: int64(len(a))}})
	if err != nil {
		t.Fatal(err)
	}
	if err = s.ControlImport(m.ID, "start"); err != nil {
		t.Fatal(err)
	}
	first, err := s.PutImport(context.Background(), m.ID, 0, bytes.NewReader(a))
	if err != nil {
		t.Fatal(err)
	}
	originals, offline := filepath.Join(root, "originals"), filepath.Join(root, "originals.offline")
	if err = os.Rename(originals, offline); err != nil {
		t.Fatal(err)
	}
	for _, attempt := range []struct {
		index int
		data  []byte
	}{{1, b}, {0, a}} {
		if _, err = s.PutImport(context.Background(), m.ID, attempt.index, bytes.NewReader(attempt.data)); err == nil {
			t.Fatal("unavailable destination falsely succeeded", attempt.index)
		}
	}
	p, err := s.ImportPage(m.ID, 0)
	if err != nil || p.Session.AcceptedFiles != 1 || p.Session.AcceptedBytes != int64(len(a)) || p.Session.NewFiles != 1 || p.Session.Duplicates != 0 || p.Entries[0].AssetID != first.AssetID || p.Entries[1].AssetID != "" || p.Entries[0].Error == "" || p.Entries[1].Error == "" {
		t.Fatal("outage changed receipts or counters", p, err)
	}
	if _, err = os.Stat(originals); !os.IsNotExist(err) {
		t.Fatal("live import recreated unavailable originals directory", err)
	}
	staging, err := os.ReadDir(filepath.Join(root, "staging"))
	if err != nil || len(staging) != 0 {
		t.Fatal("failed copies retained temporary files", err)
	}
	got, err := os.ReadFile(filepath.Join(offline, first.AssetID))
	if err != nil || !bytes.Equal(got, a) {
		t.Fatal("offline original changed", err)
	}
	assets, err := s.List()
	if err != nil || len(assets) != 1 || assets[0].ID != first.AssetID {
		t.Fatal("outage removed catalog item", err)
	}
	if err = os.Rename(offline, originals); err != nil {
		t.Fatal(err)
	}
	if err = s.Close(); err != nil {
		t.Fatal(err)
	}
	s, err = Open(root)
	if err != nil {
		t.Fatal(err)
	}
	p, err = s.ImportPage(m.ID, 0)
	if err != nil || p.Session.Status != "paused" || p.Session.AcceptedFiles != 1 || p.BackgroundPaused {
		t.Fatal("reconnected restart lost durable receipt or stranded lease", p, err)
	}
	if err = s.ControlImport(m.ID, "start"); err != nil {
		t.Fatal(err)
	}
	for i, data := range [][]byte{a, b, a} {
		if _, err = s.PutImport(context.Background(), m.ID, i, bytes.NewReader(data)); err != nil {
			t.Fatal("reconnected resume failed", i, err)
		}
	}
	if err = s.ControlImport(m.ID, "finish"); err != nil {
		t.Fatal(err)
	}
	p, err = s.ImportPage(m.ID, 0)
	if err != nil || p.Session.Status != "completed" || p.Session.AcceptedFiles != 3 || p.Session.NewFiles != 2 || p.Session.Duplicates != 1 || p.Session.AcceptedBytes != int64(2*len(a)+len(b)) || p.BackgroundPaused {
		t.Fatal("recovery double-counted receipts", p, err)
	}
	for _, e := range p.Entries {
		if e.AssetID == "" || e.Error != "" {
			t.Fatal("recovered receipt incomplete", e.Index)
		}
	}
	if err = s.Verify(); err != nil {
		t.Fatal(err)
	}
}
