package library

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	bolt "go.etcd.io/bbolt"
)

func automaticSchemaValue(t *testing.T, s *Store) string {
	t.Helper()
	var schema string
	if e := s.db.View(func(tx *bolt.Tx) error {
		schema = string(tx.Bucket([]byte("meta")).Get([]byte("schema")))
		return nil
	}); e != nil {
		t.Fatal(e)
	}
	return schema
}

func automaticSchemaFixture(t *testing.T, root string) (*Store, StoredFace) {
	t.Helper()
	s, e := Open(root)
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() { s.Close() })
	var faces []StoredFace
	for i := 0; i < 3; i++ {
		data := append(append([]byte(nil), fixture(t)...), byte(i))
		a, _, e := s.Put(fmt.Sprintf("schema-synthetic-%d.png", i), bytes.NewReader(data))
		if e != nil {
			t.Fatal(e)
		}
		faces = append(faces, automaticTestFace(1001+i, a.ID, 0))
	}
	if e = s.db.Update(func(tx *bolt.Tx) error {
		for i, f := range faces {
			if e := writeJSON(tx.Bucket([]byte(facesBucket)), f.ID, f); e != nil {
				return e
			}
			if i < 2 {
				// Legacy labels had no provenance field and are manual references.
				if e := writeJSON(tx.Bucket([]byte(labelsBucket)), f.ID, FaceLabel{Name: "Schema person"}); e != nil {
					return e
				}
			}
		}
		return nil
	}); e != nil {
		t.Fatal(e)
	}
	if e = os.WriteFile(filepath.Join(root, "access-code.txt"), []byte(strings.Repeat("a", 64)+"\n"), 0600); e != nil {
		t.Fatal(e)
	}
	return s, faces[2]
}

func automaticSchemaSnapshot(t *testing.T, root string) map[string][32]byte {
	t.Helper()
	out := map[string][32]byte{}
	if e := filepath.Walk(root, func(path string, info os.FileInfo, e error) error {
		if e != nil || info.IsDir() {
			return e
		}
		b, e := os.ReadFile(path)
		if e != nil {
			return e
		}
		rel, e := filepath.Rel(root, path)
		if e != nil {
			return e
		}
		out[rel] = sha256.Sum256(b)
		return nil
	}); e != nil {
		t.Fatal(e)
	}
	return out
}

func TestAutomaticSchemaManualOnlyAndNoAssignmentRemainOne(t *testing.T) {
	empty, _ := openTest(t)
	if automaticSchemaValue(t, empty) != "1" {
		t.Fatal("empty catalog should retain legacy compatibility")
	}
	for _, scenario := range []string{"disabled", "already manually named", "insufficient references"} {
		t.Run(scenario, func(t *testing.T) {
			s, subject := automaticSchemaFixture(t, filepath.Join(t.TempDir(), "state"))
			if automaticSchemaValue(t, s) != "1" {
				t.Fatal("manual-only fixture promoted its catalog")
			}
			if scenario == "already manually named" {
				if e := s.LabelFaces([]string{subject.ID}, "Explicit correction"); e != nil {
					t.Fatal(e)
				}
			}
			if e := s.db.Update(func(tx *bolt.Tx) error {
				if scenario == "insufficient references" {
					if e := tx.Bucket([]byte(labelsBucket)).Delete([]byte(fmt.Sprintf("%064x", 1002))); e != nil {
						return e
					}
				}
				p := candidateAutomaticNamePolicy()
				if scenario == "disabled" {
					p.enabled = false
				}
				return applyAutomaticNames(tx, []StoredFace{subject}, p)
			}); e != nil {
				t.Fatal(e)
			}
			if automaticSchemaValue(t, s) != "1" {
				t.Fatal("catalog promoted without a new automatic assignment")
			}
		})
	}
}

func TestAutomaticSchemaPromotionIsAtomicAndDoesNotDowngrade(t *testing.T) {
	s, subject := automaticSchemaFixture(t, filepath.Join(t.TempDir(), "state"))
	abort := errors.New("synthetic publication abort")
	if e := s.db.Update(func(tx *bolt.Tx) error {
		if e := applyAutomaticNames(tx, []StoredFace{subject}, candidateAutomaticNamePolicy()); e != nil {
			return e
		}
		if string(tx.Bucket([]byte("meta")).Get([]byte("schema"))) != "2" {
			t.Fatal("automatic label and catalog version were not staged together")
		}
		return abort
	}); !errors.Is(e, abort) {
		t.Fatal("synthetic transaction did not abort", e)
	}
	if automaticSchemaValue(t, s) != "1" {
		t.Fatal("failed automatic publication advanced schema")
	}
	if e := s.db.View(func(tx *bolt.Tx) error {
		if tx.Bucket([]byte(labelsBucket)).Get([]byte(subject.ID)) != nil {
			t.Fatal("failed automatic publication retained label")
		}
		return nil
	}); e != nil {
		t.Fatal(e)
	}
	if e := s.db.Update(func(tx *bolt.Tx) error {
		return applyAutomaticNames(tx, []StoredFace{subject}, candidateAutomaticNamePolicy())
	}); e != nil {
		t.Fatal(e)
	}
	if automaticSchemaValue(t, s) != "2" || automaticLabel(t, s, subject.ID).Source != "automatic" {
		t.Fatal("successful automatic assignment did not promote schema")
	}
	if e := s.UndoFace(subject.ID); e != nil {
		t.Fatal(e)
	}
	if e := s.LabelFaces([]string{subject.ID}, "Manual after undo"); e != nil {
		t.Fatal(e)
	}
	if automaticSchemaValue(t, s) != "2" {
		t.Fatal("correction or undo downgraded catalog version")
	}
}

func TestAutomaticSchemaTwoReadOnlyRestartExportBackupAndIndependentRestore(t *testing.T) {
	root := filepath.Join(t.TempDir(), "source")
	s, subject := automaticSchemaFixture(t, root)
	if e := s.db.Update(func(tx *bolt.Tx) error {
		return applyAutomaticNames(tx, []StoredFace{subject}, candidateAutomaticNamePolicy())
	}); e != nil {
		t.Fatal(e)
	}
	if e := s.Close(); e != nil {
		t.Fatal(e)
	}
	before := automaticSchemaSnapshot(t, root)
	ro, e := OpenReadOnly(root)
	if e != nil {
		t.Fatal("schema two read-only open failed", e)
	}
	if automaticSchemaValue(t, ro) != "2" {
		t.Fatal("read-only open lost schema version")
	}
	export, e := ro.OrganizationExport()
	if e != nil {
		t.Fatal(e)
	}
	if export["catalog_schema"] != "2" || export["schema"] != 1 {
		t.Fatal("export confused catalog semantics with the JSON container version")
	}
	encoded, e := json.Marshal(export)
	if e != nil || !bytes.Contains(encoded, []byte(`"name_source":"automatic"`)) || bytes.Contains(encoded, []byte(`"embedding":[`)) {
		t.Fatal("read-only export lost provenance or leaked embeddings", e)
	}
	if e = ro.Verify(); e != nil {
		t.Fatal(e)
	}
	var backup bytes.Buffer
	if e = ro.Backup(&backup); e != nil {
		t.Fatal("schema two offline backup failed", e)
	}
	if e = ro.Close(); e != nil {
		t.Fatal(e)
	}
	if !reflect.DeepEqual(before, automaticSchemaSnapshot(t, root)) {
		t.Fatal("read-only schema two operations modified source bytes or files")
	}
	z, e := zip.NewReader(bytes.NewReader(backup.Bytes()), int64(backup.Len()))
	if e != nil {
		t.Fatal(e)
	}
	for _, f := range z.File {
		if f.Name != "manifest.json" && f.Name != "state-format.txt" {
			continue
		}
		r, e := f.Open()
		if e != nil {
			t.Fatal(e)
		}
		b, e := io.ReadAll(r)
		r.Close()
		if e != nil {
			t.Fatal(e)
		}
		if f.Name == "state-format.txt" && string(b) != "immichclone-schema-1\n" {
			t.Fatal("catalog upgrade changed independent state-layout marker")
		}
		if f.Name == "manifest.json" {
			var m Manifest
			if json.Unmarshal(b, &m) != nil || m.Schema != 1 {
				t.Fatal("catalog upgrade changed independent archive container version")
			}
		}
	}
	current, e := Open(root)
	if e != nil {
		t.Fatal(e)
	}
	if automaticSchemaValue(t, current) != "2" || automaticLabel(t, current, subject.ID).Source != "automatic" {
		t.Fatal("normal restart downgraded catalog or automatic provenance")
	}
	current.Close()
	archive := filepath.Join(t.TempDir(), "catalog-two.zip")
	if e = os.WriteFile(archive, backup.Bytes(), 0600); e != nil {
		t.Fatal(e)
	}
	if e = os.Rename(root, root+"-unavailable"); e != nil {
		t.Fatal(e)
	}
	target := filepath.Join(t.TempDir(), "restored")
	if e = Restore(archive, target); e != nil {
		t.Fatal("schema two independent restore failed", e)
	}
	restored, e := Open(target)
	if e != nil {
		t.Fatal(e)
	}
	defer restored.Close()
	if automaticSchemaValue(t, restored) != "2" || automaticLabel(t, restored, subject.ID).Source != "automatic" {
		t.Fatal("independent restore lost schema/provenance")
	}
	if e = restored.Verify(); e != nil {
		t.Fatal(e)
	}
}

func TestAutomaticSchemaUnsupportedThreeRefusedWithoutMutation(t *testing.T) {
	s, root := openTest(t)
	if e := s.db.Update(func(tx *bolt.Tx) error {
		return tx.Bucket([]byte("meta")).Put([]byte("schema"), []byte("3"))
	}); e != nil {
		t.Fatal(e)
	}
	s.Close()
	before := automaticSchemaSnapshot(t, root)
	for _, open := range []func(string) (*Store, error){OpenReadOnly, Open} {
		unexpected, e := open(root)
		if e == nil {
			unexpected.Close()
			t.Fatal("unsupported schema three accepted")
		}
		if !reflect.DeepEqual(before, automaticSchemaSnapshot(t, root)) {
			t.Fatal("unsupported schema refusal mutated source state")
		}
	}
}

func TestAutomaticSchemaLegacyDevelopmentProvenanceRequiresPromotion(t *testing.T) {
	for _, provenance := range []string{"current automatic", "previous automatic"} {
		t.Run(provenance, func(t *testing.T) {
			root := filepath.Join(t.TempDir(), "state")
			s, subject := automaticSchemaFixture(t, root)
			if e := s.db.Update(func(tx *bolt.Tx) error {
				return applyAutomaticNames(tx, []StoredFace{subject}, candidateAutomaticNamePolicy())
			}); e != nil {
				t.Fatal(e)
			}
			if provenance == "previous automatic" {
				if e := s.LabelFaces([]string{subject.ID}, "Explicit correction"); e != nil {
					t.Fatal(e)
				}
			}
			// Model an unpublished pre-gate development catalog, not a supported
			// downgrade of a released version-two catalog.
			if e := s.db.Update(func(tx *bolt.Tx) error {
				return tx.Bucket([]byte("meta")).Put([]byte("schema"), []byte("1"))
			}); e != nil {
				t.Fatal(e)
			}
			s.Close()
			before := automaticSchemaSnapshot(t, root)
			unexpected, e := OpenReadOnly(root)
			if e == nil {
				unexpected.Close()
				t.Fatal("read-only open accepted inconsistent legacy automatic provenance")
			}
			if !reflect.DeepEqual(before, automaticSchemaSnapshot(t, root)) {
				t.Fatal("legacy read-only refusal modified source")
			}
			promoted, e := Open(root)
			if e != nil {
				t.Fatal("development catalog promotion failed", e)
			}
			defer promoted.Close()
			if automaticSchemaValue(t, promoted) != "2" {
				t.Fatal("existing automatic history was not promoted on write-capable open")
			}
			l := automaticLabel(t, promoted, subject.ID)
			if provenance == "current automatic" && l.Source != "automatic" || provenance == "previous automatic" && (l.Source != "manual" || l.PreviousSource != "automatic") {
				t.Fatal("development promotion changed current or previous provenance")
			}
		})
	}
}

// This fixture is requested only by the disposable Windows published-writer
// test. It never consumes a user catalog or enables automatic production names.
func TestAutomaticSchemaPublishedWriterFixture(t *testing.T) {
	root := os.Getenv("IMMICH_R43_SCHEMA_FIXTURE_ROOT")
	if root == "" {
		t.Skip("requires explicit disposable published-writer fixture")
	}
	if os.Getenv("GITHUB_ACTIONS") != "true" || !filepath.IsAbs(root) {
		t.Fatal("published-writer fixture requires an absolute disposable CI path")
	}
	stage := os.Getenv("IMMICH_R43_SCHEMA_FIXTURE_STAGE")
	if stage == "create" {
		s, _ := automaticSchemaFixture(t, root)
		if automaticSchemaValue(t, s) != "1" {
			t.Fatal("old-writer positive control did not start as schema one")
		}
		return
	}
	var s *Store
	var e error
	if stage == "inspect" {
		s, e = OpenReadOnly(root)
	} else if stage == "promote" {
		s, e = Open(root)
	} else {
		t.Fatal("unknown published-writer fixture stage")
	}
	if e != nil {
		t.Fatal(e)
	}
	defer s.Close()
	id := fmt.Sprintf("%064x", 1003)
	if stage == "promote" {
		if automaticSchemaValue(t, s) != "1" {
			t.Fatal("schema one positive control was changed before promotion")
		}
		if e = s.db.Update(func(tx *bolt.Tx) error {
			var subject StoredFace
			if e := readJSON(tx.Bucket([]byte(facesBucket)), id, &subject); e != nil {
				return e
			}
			return applyAutomaticNames(tx, []StoredFace{subject}, candidateAutomaticNamePolicy())
		}); e != nil {
			t.Fatal(e)
		}
	}
	if automaticSchemaValue(t, s) != "2" || automaticLabel(t, s, id).Source != "automatic" {
		t.Fatal("published-writer fixture lost version two/provenance")
	}
	if e = s.Verify(); e != nil {
		t.Fatal(e)
	}
}
