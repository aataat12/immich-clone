package library

import (
	"crypto/sha256"
	"fmt"
	"os"
	"testing"
)

func TestInstalledDeletionArchiveRestored(t *testing.T) {
	root := os.Getenv("IMMICH_INSTALLED_DELETE_RESTORE")
	if root == "" {
		t.Skip("requires actual installed pre-deletion archive")
	}
	s, err := Open(root)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	id := fmt.Sprintf("%x", sha256.Sum256([]byte("disposable deletion permission fixture")))
	a, err := s.Get(id)
	if err != nil || !a.Hidden {
		t.Fatal("restored pre-deletion hidden original missing", err)
	}
	if err = s.Verify(); err != nil {
		t.Fatal(err)
	}
	jobs, err := s.PendingDeletions()
	if err != nil || len(jobs) != 0 {
		t.Fatal("pre-deletion archive unexpectedly contains an intent", jobs, err)
	}
	o, err := s.OrganizationPage("", "", "")
	if err != nil {
		t.Fatal(err)
	}
	names := map[string]bool{}
	for _, face := range o.Faces {
		names[face.Name] = true
	}
	for _, name := range []string{"Before upgrade person", "Installer Person", "Video Test Person"} {
		if !names[name] {
			t.Fatal("independent pre-deletion restore lost preexisting person", name)
		}
	}
}
