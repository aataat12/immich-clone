package library

import (
	"bytes"
	"context"
	"crypto/sha256"
	"fmt"
	"immichclone/internal/recognition"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"testing"
	"time"

	bolt "go.etcd.io/bbolt"
)

func TestOrganizationIndexRefreshesCorrectionsAndRestoredState(t *testing.T) {
	s, root := openTest(t)
	s.SetRecognizer(fakeAnalysis, true)
	a, _, err := s.Put("index-original.png", bytes.NewReader(fixture(t)))
	if err != nil {
		t.Fatal(err)
	}
	if err = s.QueueAnalysis(a.ID); err != nil {
		t.Fatal(err)
	}
	if _, err = s.ProcessAnalysis(context.Background()); err != nil {
		t.Fatal(err)
	}
	page, err := s.OrganizationPage("", "", "")
	if err != nil || len(page.Faces) != 1 {
		t.Fatal("first page", err)
	}
	face := page.Faces[0].ID
	for _, name := range []string{"First person", "Second person"} {
		if err = s.LabelFaces([]string{face}, name); err != nil {
			t.Fatal(err)
		}
		for repeat := 0; repeat < 2; repeat++ {
			page, err = s.OrganizationPage("", "", name)
			if err != nil || len(page.Faces) != 1 || len(page.Assets) != 1 || len(page.Names) != 1 || page.Names[0] != name {
				t.Fatal("name search stale", err)
			}
		}
	}
	if err = s.UndoFace(face); err != nil {
		t.Fatal(err)
	}
	if page, err = s.OrganizationPage("", "", "Second person"); err != nil || len(page.Faces) != 0 || len(page.Assets) != 0 {
		t.Fatal("undo search stale", err)
	}
	if err = s.SetTags(a.ID, []string{"Résumé trip"}); err != nil {
		t.Fatal(err)
	}
	if page, err = s.OrganizationPage("", "", "RÉSUMÉ"); err != nil || len(page.Assets) != 1 {
		t.Fatal("case-insensitive tag search stale", err)
	}
	if err = s.DismissTag(a.ID, "cat", false); err != nil {
		t.Fatal(err)
	}
	if page, err = s.OrganizationPage("", "", "cat"); err != nil || len(page.Assets) != 0 {
		t.Fatal("dismissed AI tag still searchable", err)
	}
	if err = s.DismissFace(face, false); err != nil {
		t.Fatal(err)
	}
	if page, err = s.OrganizationReviewPage("", "", "First person", true); err != nil || len(page.Faces) != 1 || len(page.Assets) != 0 || len(page.Names) != 0 {
		t.Fatal("dismissed reference cached", err)
	}
	if err = s.DismissFace(face, true); err != nil {
		t.Fatal(err)
	}
	if err = s.SetHidden(a.ID, true); err != nil {
		t.Fatal(err)
	}
	if page, err = s.OrganizationPage("", "", ""); err != nil || len(page.Faces) != 0 || len(page.Assets) != 0 || len(page.Names) != 0 {
		t.Fatal("hidden records cached", err)
	}
	if err = s.SetHidden(a.ID, false); err != nil {
		t.Fatal(err)
	}
	// Caller edits to a returned view must not change later persisted views.
	page, err = s.OrganizationPage("", "", "")
	if err != nil {
		t.Fatal(err)
	}
	page.Faces[0].Box[0] = .99
	page.Names[0] = "caller mutation"
	page, err = s.OrganizationPage("", "", "")
	if err != nil || page.Faces[0].Box[0] == .99 || page.Names[0] == "caller mutation" {
		t.Fatal("returned view aliases derived cache", err)
	}
	archive := filepath.Join(t.TempDir(), "independent.zip")
	f, err := os.Create(archive)
	if err != nil {
		t.Fatal(err)
	}
	err = s.Backup(f)
	ce := f.Close()
	if err != nil || ce != nil {
		t.Fatal(err, ce)
	}
	if err = s.Close(); err != nil {
		t.Fatal(err)
	}
	if err = os.Rename(root, root+".offline"); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(t.TempDir(), "restored")
	if err = Restore(archive, target); err != nil {
		t.Fatal(err)
	}
	restored, err := Open(target)
	if err != nil {
		t.Fatal(err)
	}
	defer restored.Close()
	if page, err = restored.OrganizationPage("", "", "First person"); err != nil || len(page.Faces) != 1 || len(page.Assets) != 1 || page.Names[0] != "First person" {
		t.Fatal("restored derived state stale", err)
	}
}

// Opt-in scale fixture: real tiny originals plus directly seeded valid catalog
// records. This measures review/search, not import throughput or AI inference.
func TestOrganizationIndexLargeLibraryAcceptance(t *testing.T) {
	if os.Getenv("IMMICH_PEOPLE_SCALE") != "1" {
		t.Skip("explicit 10000-original/30000-face scale acceptance")
	}
	s, root := openTest(t)
	const assets = 10000
	const faceCount = assets * 3
	stamp := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	for start := 0; start < assets; start += 500 {
		if err := s.db.Update(func(tx *bolt.Tx) error {
			for i := start; i < start+500; i++ {
				body := []byte(fmt.Sprintf("public synthetic review scale %08d", i))
				id := fmt.Sprintf("%x", sha256.Sum256(body))
				if err := os.WriteFile(s.Original(id), body, 0600); err != nil {
					return err
				}
				a := Asset{ID: id, Name: fmt.Sprintf("review-%05d.bin", i), Size: int64(len(body)), MIME: "application/octet-stream", Added: stamp, Preview: "failed", Sources: []string{fmt.Sprintf("SyntheticReview/%05d.bin", i)}}
				if err := saveAsset(tx, a); err != nil {
					return err
				}
				if err := writeJSON(tx.Bucket([]byte(tagsBucket)), id, []string{fmt.Sprintf("category-%02d", i%100)}); err != nil {
					return err
				}
				for j := 0; j < 3; j++ {
					face := StoredFace{ID: fmt.Sprintf("%x", sha256.Sum256([]byte(fmt.Sprintf("scale-face-%d-%d", i, j)))), AssetID: id, Box: []float64{.1, .1, .3, .3}, Provider: "manual"}
					if j == 0 {
						face.Provider = recognition.AnalysisID
						face.Embedding = make([]float64, 128)
						face.Embedding[i%128] = 1
					}
					if err := writeJSON(tx.Bucket([]byte(facesBucket)), face.ID, face); err != nil {
						return err
					}
					if i%100 == 0 && j == 0 {
						if err := writeJSON(tx.Bucket([]byte(labelsBucket)), face.ID, FaceLabel{Name: fmt.Sprintf("Review person %03d", i/100)}); err != nil {
							return err
						}
					}
				}
			}
			return nil
		}); err != nil {
			t.Fatal(err)
		}
	}
	started := time.Now()
	page, err := s.OrganizationPage("", "", "")
	if err != nil || len(page.Faces) != 100 || len(page.Assets) != 50 || len(page.Names) != 100 || page.After == "" || page.FaceAfter == "" {
		t.Fatal("scale initial page", err)
	}
	cold := time.Since(started)
	var before, after runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&before)
	durations := []time.Duration{}
	for i := 0; i < 8; i++ {
		start := time.Now()
		p, e := s.OrganizationFilteredPage(page.After, page.FaceAfter, "", false, true)
		if e != nil || len(p.Assets) != 50 || len(p.Faces) != 100 {
			t.Fatal("warm independent page", e)
		}
		durations = append(durations, time.Since(start))
	}
	runtime.ReadMemStats(&after)
	perQuery := (after.TotalAlloc - before.TotalAlloc) / 8
	sort.Slice(durations, func(i, j int) bool { return durations[i] < durations[j] })
	if perQuery > 8<<20 || durations[7] > time.Second || cold > 20*time.Second {
		t.Fatal("review scale budget exceeded", perQuery, durations[7], cold)
	}
	p, err := s.OrganizationPage("", "", "CATEGORY-99")
	if err != nil || len(p.Assets) != 50 || p.After == "" {
		t.Fatal("scale tag first page", err)
	}
	next, err := s.OrganizationPage(p.After, "", "CATEGORY-99")
	if err != nil || len(next.Assets) != 50 || next.After != "" {
		t.Fatal("scale exact final search page", err)
	}
	seen := map[string]bool{}
	for _, a := range append(p.Assets, next.Assets...) {
		if seen[a.Asset.ID] {
			t.Fatal("duplicate search result")
		}
		seen[a.Asset.ID] = true
	}
	if p, err = s.OrganizationPage("", "", "no such result"); err != nil || len(p.Assets) != 0 || len(p.Faces) != 0 || p.After != "" || p.FaceAfter != "" {
		t.Fatal("scale empty search", err)
	}
	// Independent file identity check for every source avoids treating metadata
	// seeding as proof that original bytes exist or remain unchanged.
	for i := 0; i < assets; i++ {
		body := []byte(fmt.Sprintf("public synthetic review scale %08d", i))
		id := fmt.Sprintf("%x", sha256.Sum256(body))
		actual, e := os.ReadFile(filepath.Join(root, "originals", id))
		if e != nil || !bytes.Equal(actual, body) {
			t.Fatal("scale source changed", e)
		}
	}
	t.Logf("PASS: %d actual tiny originals, %d seeded faces, 100 name references; cold %s, warm max %s, allocated bytes/query %d; independent pagination, case-insensitive indexed tag search and source identities", assets, faceCount, cold, durations[7], perQuery)
}
