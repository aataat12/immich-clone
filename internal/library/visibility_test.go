package library

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestVisibilityPreservesOriginalJobsAndRestore(t *testing.T) {
	s, root := openTest(t)
	data := fixture(t)
	a, _, e := s.Put("synthetic.png", bytes.NewReader(data))
	if e != nil {
		t.Fatal(e)
	}
	if e = s.SetHidden(a.ID, true); e != nil {
		t.Fatal(e)
	}
	if _, e = s.ProcessOne(); e != nil {
		t.Fatal(e)
	}
	a, _ = s.Get(a.ID)
	if !a.Hidden || a.Preview != "ready" {
		t.Fatal("preview job overwrote hidden state")
	}
	again, duplicate, e := s.Put("duplicate.png", bytes.NewReader(data))
	if e != nil || !duplicate || !again.Hidden {
		t.Fatal("duplicate upload should preserve visibility")
	}
	if e = s.Close(); e != nil {
		t.Fatal(e)
	}
	s, e = Open(root)
	if e != nil {
		t.Fatal(e)
	}
	defer s.Close()
	a, _ = s.Get(a.ID)
	if !a.Hidden {
		t.Fatal("hidden state lost across restart")
	}
	original, e := os.ReadFile(s.Original(a.ID))
	if e != nil || !bytes.Equal(original, data) {
		t.Fatal("hide changed original")
	}
	var archive bytes.Buffer
	if e = s.Backup(&archive); e != nil {
		t.Fatal(e)
	}
	zip := filepath.Join(t.TempDir(), "backup.zip")
	if e = os.WriteFile(zip, archive.Bytes(), 0600); e != nil {
		t.Fatal(e)
	}
	target := filepath.Join(t.TempDir(), "restore")
	if e = Restore(zip, target); e != nil {
		t.Fatal(e)
	}
	restored, e := Open(target)
	if e != nil {
		t.Fatal(e)
	}
	defer restored.Close()
	a, _ = restored.Get(a.ID)
	if !a.Hidden {
		t.Fatal("backup lost visibility")
	}
	if _, e = restored.ProcessOne(); e != nil {
		t.Fatal(e)
	}
	if e = restored.SetHidden(a.ID, false); e != nil {
		t.Fatal(e)
	}
	a, _ = restored.Get(a.ID)
	if a.Hidden || a.Preview != "ready" {
		t.Fatal("restore-to-library failed")
	}
	if e = restored.Verify(); e != nil {
		t.Fatal(e)
	}
}
