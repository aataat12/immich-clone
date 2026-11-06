package library

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	bolt "go.etcd.io/bbolt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func jsonCuration(out map[string]any) ([]byte, error) { return json.Marshal(out["curation"]) }

func TestCurationCrashChild(t *testing.T) {
	root := os.Getenv("IMMICH_CURATION_CRASH_ROOT")
	if root == "" {
		t.Skip("isolated child only")
	}
	s, e := Open(root)
	if e != nil {
		t.Fatal(e)
	}
	assets, _ := s.List()
	a := assets[0]
	if os.Getenv("IMMICH_CURATION_CRASH_STAGE") == "before" {
		s.db.Update(func(tx *bolt.Tx) error {
			if e := ensureCuration(tx); e != nil {
				return e
			}
			if e := curationWrite(tx, assetOrganizationBucket, a.ID, AssetOrganization{AssetID: a.ID, Added: a.Added, Revision: 1, Favorite: true}); e != nil {
				return e
			}
			os.Exit(41)
			return nil
		})
		t.Fatal("child did not exit")
	}
	m, _ := s.AssetOrganization(a.ID)
	if _, e = s.EditCuration(CurationEdit{Action: "set_asset", AssetID: a.ID, Added: a.Added, Expected: m.Token, Favorite: true, Caption: "committed before process exit"}); e != nil {
		t.Fatal(e)
	}
	os.Exit(42)
}
func TestCurationProcessCommitBoundary(t *testing.T) {
	s, root := openTest(t)
	a, _, e := s.Put("synthetic.txt", strings.NewReader("preserved original"))
	if e != nil {
		t.Fatal(e)
	}
	s.Close()
	exe, e := os.Executable()
	if e != nil {
		t.Fatal(e)
	}
	for _, stage := range []string{"before", "after"} {
		cmd := exec.Command(exe, "-test.run=^TestCurationCrashChild$")
		cmd.Env = append(os.Environ(), "IMMICH_CURATION_CRASH_ROOT="+root, "IMMICH_CURATION_CRASH_STAGE="+stage)
		out, e := cmd.CombinedOutput()
		var exit *exec.ExitError
		if !errors.As(e, &exit) || exit.ExitCode() != map[string]int{"before": 41, "after": 42}[stage] {
			t.Fatalf("child boundary failed: %v %s", e, out)
		}
		r, e := Open(root)
		if e != nil {
			t.Fatal(e)
		}
		m, _ := r.AssetOrganization(a.ID)
		if stage == "before" && (m.Favorite || automaticSchemaValue(t, r) != "1") {
			t.Fatal("uncommitted metadata survived")
		}
		if stage == "after" && (!m.Favorite || automaticSchemaValue(t, r) != "3") {
			t.Fatal("committed metadata lost")
		}
		if e = r.Verify(); e != nil {
			t.Fatal(e)
		}
		r.Close()
	}
}

func TestActualCurationDiskFull(t *testing.T) {
	volume := os.Getenv("IMMICH_SMALL_VOLUME")
	if volume == "" {
		t.Skip("requires disposable bounded NTFS volume")
	}
	root, e := os.MkdirTemp(volume, "curation-full-")
	if e != nil {
		t.Fatal(e)
	}
	defer os.RemoveAll(root)
	s, e := Open(filepath.Join(root, "state"))
	if e != nil {
		t.Fatal(e)
	}
	defer s.Close()
	var assets []Asset
	for i := 0; i < 100; i++ {
		a, _, e := s.Put(fmt.Sprintf("synthetic-%d.txt", i), strings.NewReader(fmt.Sprint("immutable synthetic original ", i)))
		if e != nil {
			t.Fatal(e)
		}
		assets = append(assets, a)
	}
	batch := prepareBatchFixture(t, s, "favorite_on", "", 30)
	filler := filepath.Join(root, "filler.bin")
	f, e := os.Create(filler)
	if e != nil {
		t.Fatal(e)
	}
	full := false
	block := make([]byte, 64<<10)
	for n := 0; n < 128<<20; n += len(block) {
		if _, e = f.Write(block); e != nil {
			full = true
			break
		}
	}
	f.Close()
	if !full {
		t.Fatal("bounded fixture volume did not fill")
	}
	beforeBatch, _ := s.OrganizationBatch(batch.ID)
	if _, e = s.ContinueOrganizationBatch(batch.ID, 0); e == nil {
		t.Fatal("full disk accepted a batch")
	}
	afterBatch, _ := s.OrganizationBatch(batch.ID)
	rawBefore, _ := json.Marshal(beforeBatch)
	rawAfter, _ := json.Marshal(afterBatch)
	if !bytes.Equal(rawBefore, rawAfter) {
		t.Fatal("full disk changed receipt")
	}
	var failed *Asset
	for _, a := range assets {
		m, _ := s.AssetOrganization(a.ID)
		before, exportErr := s.OrganizationExport()
		if exportErr != nil {
			t.Fatal("pre-failure export unavailable", exportErr)
		}
		b, _ := jsonCuration(before)
		_, e = s.EditCuration(CurationEdit{Action: "set_asset", AssetID: a.ID, Added: a.Added, Expected: m.Token, Notes: strings.Repeat("x", 64<<10)})
		if e != nil {
			after, exportErr := s.OrganizationExport()
			if exportErr != nil {
				t.Fatal("post-failure export unavailable", exportErr)
			}
			v, _ := jsonCuration(after)
			if !bytes.Equal(b, v) || before["catalog_schema"] != after["catalog_schema"] {
				t.Fatal("failed full-disk mutation changed organization")
			}
			copy := a
			failed = &copy
			break
		}
	}
	if failed == nil {
		t.Fatal("full-disk metadata mutation never failed")
	}
	if e = os.Remove(filler); e != nil {
		t.Fatal(e)
	}
	completeBatchFixture(t, s, batch)
	m, _ := s.AssetOrganization(failed.ID)
	if _, e = s.EditCuration(CurationEdit{Action: "set_asset", AssetID: failed.ID, Added: failed.Added, Expected: m.Token, Favorite: true}); e != nil {
		t.Fatal("retry after space recovery", e)
	}
	if e = s.Verify(); e != nil {
		t.Fatal(e)
	}
	if e = s.db.View(validateCatalogInput); e != nil {
		t.Fatal(e)
	}
	t.Log("PASS: actual NTFS-full metadata failure rolls back schema/organization and recovers after space is released")
}
