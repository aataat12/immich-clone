package library

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	bolt "go.etcd.io/bbolt"
	"image"
	"immichclone/internal/recognition"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func fakeAnalysis(context.Context, image.Image, string) (recognition.Result, error) {
	v := make([]float64, 128)
	v[0] = 1
	return recognition.Result{Protocol: 1, Faces: []recognition.Face{{Box: []float64{.1, .1, .4, .4}, Confidence: .99, Embedding: v}}, Objects: []recognition.Object{{Class: 15, Confidence: .9, Box: []float64{.1, .1, .8, .8}}}}, nil
}
func TestOrganizationOriginalsCorrectionsAndIndependentRestore(t *testing.T) {
	s, root := openTest(t)
	s.SetRecognizer(fakeAnalysis, true)
	data := fixture(t)
	a, _, e := s.Put("synthetic.png", bytes.NewReader(data))
	if e != nil {
		t.Fatal(e)
	}
	if e = s.QueueAnalysis(a.ID); e != nil {
		t.Fatal(e)
	}
	if ok, e := s.ProcessAnalysis(context.Background()); e != nil || !ok {
		t.Fatal(e)
	}
	page, e := s.OrganizationPage("", "", "")
	if e != nil || len(page.Faces) != 1 || page.Assets[0].Analysis.Status != "ready" {
		t.Fatal("analysis publication", e)
	}
	id := page.Faces[0].ID
	if e = s.LabelFaces([]string{id}, "Review Person"); e != nil {
		t.Fatal(e)
	}
	if e = s.SetTags(a.ID, []string{"Birthday", "Trip", "birthday"}); e != nil {
		t.Fatal(e)
	}
	if e = s.LabelFaces([]string{id, "invalid"}, "Must roll back"); e == nil {
		t.Fatal("invalid batch accepted")
	}
	page, _ = s.OrganizationPage("", "", "")
	if page.Faces[0].Name != "Review Person" {
		t.Fatal("batch not atomic")
	}
	if e = s.LabelFaces([]string{id}, "Other Person"); e != nil {
		t.Fatal(e)
	}
	if e = s.UndoFace(id); e != nil {
		t.Fatal(e)
	}
	if e = s.QueueAnalysis(a.ID); e != nil {
		t.Fatal(e)
	}
	if ok, e := s.ProcessAnalysis(context.Background()); ok || e != nil {
		t.Fatal("ready analysis repeated")
	}
	if e = s.ManualFace(context.Background(), a.ID, []float64{.5, .1, .2, .2}, "Manual Person"); e != nil {
		t.Fatal(e)
	}
	exported, e := s.OrganizationExport()
	if e != nil {
		t.Fatal(e)
	}
	encoded, _ := json.Marshal(exported)
	if bytes.Contains(encoded, []byte("[1,0,0,0")) || !bytes.Contains(encoded, []byte("Review Person")) {
		t.Fatal("export omitted corrections or leaked embedding")
	}
	if e = s.SetHidden(a.ID, true); e != nil {
		t.Fatal(e)
	}
	page, _ = s.OrganizationPage("", "", "")
	if len(page.Faces) != 0 || len(page.Assets) != 0 {
		t.Fatal("hidden items in review")
	}
	s.SetHidden(a.ID, false)
	var archive bytes.Buffer
	if e = s.Backup(&archive); e != nil {
		t.Fatal(e)
	}
	zip := filepath.Join(t.TempDir(), "backup.zip")
	os.WriteFile(zip, archive.Bytes(), 0600)
	target := filepath.Join(t.TempDir(), "restored")
	if e = Restore(zip, target); e != nil {
		t.Fatal(e)
	}
	s.Close()
	if e = os.Rename(root, root+"-unavailable"); e != nil {
		t.Fatal(e)
	}
	restored, e := Open(target)
	if e != nil {
		t.Fatal(e)
	}
	defer restored.Close()
	page, e = restored.OrganizationPage("", "", "")
	if e != nil || len(page.Faces) != 2 || len(page.Names) != 2 || len(page.Assets[0].Tags) != 2 || len(page.Assets[0].Analysis.Objects) != 1 {
		t.Fatal("independent restore lost organization", e)
	}
	for _, f := range page.Faces {
		p, e := restored.FaceCrop(f.ID)
		if e != nil {
			t.Fatal("face crop regeneration", e)
		}
		if _, e = os.Stat(p); e != nil {
			t.Fatal(e)
		}
	}
	if e = restored.Verify(); e != nil {
		t.Fatal(e)
	}
}
func TestAnalysisCancellationQueueRebuildAndFailureIsolation(t *testing.T) {
	s, root := openTest(t)
	a, _, e := s.Put("synthetic.png", bytes.NewReader(fixture(t)))
	if e != nil {
		t.Fatal(e)
	}
	s.QueueAnalysis(a.ID)
	ctx, cancel := context.WithCancel(context.Background())
	s.SetRecognizer(func(ctx context.Context, _ image.Image, _ string) (recognition.Result, error) {
		cancel()
		return recognition.Result{}, ctx.Err()
	}, true)
	if _, e = s.ProcessAnalysis(ctx); !errors.Is(e, context.Canceled) {
		t.Fatal(e)
	}
	s.Close()
	s, e = Open(root)
	if e != nil {
		t.Fatal(e)
	}
	defer s.Close()
	s.SetRecognizer(nil, false)
	if _, e = s.ProcessAnalysis(context.Background()); e != nil {
		t.Fatal(e)
	}
	page, _ := s.OrganizationPage("", "", "")
	if page.Assets[0].Analysis.Status != "failed" {
		t.Fatal("missing provider not surfaced")
	}
	if e = s.Verify(); e != nil {
		t.Fatal("AI failure damaged original", e)
	}
	s.SetRecognizer(fakeAnalysis, true)
	s.QueueAnalysis(a.ID)
	if _, e = s.ProcessAnalysis(context.Background()); e != nil {
		t.Fatal(e)
	}

}

func TestRecognitionDoesNotBlockCatalogAndBackup(t *testing.T) {
	s, _ := openTest(t)
	a, _, _ := s.Put("synthetic.png", bytes.NewReader(fixture(t)))
	entered := make(chan struct{})
	release := make(chan struct{})
	done := make(chan error, 1)
	s.SetRecognizer(func(ctx context.Context, img image.Image, path string) (recognition.Result, error) {
		close(entered)
		<-release
		return fakeAnalysis(ctx, img, path)
	}, true)
	s.QueueAnalysis(a.ID)
	go func() { _, e := s.ProcessAnalysis(context.Background()); done <- e }()
	<-entered
	checked := make(chan error, 1)
	go func() {
		if e := s.SetTags(a.ID, []string{"While AI runs"}); e != nil {
			checked <- e
			return
		}
		var archive bytes.Buffer
		checked <- s.Backup(&archive)
	}()
	select {
	case e := <-checked:
		if e != nil {
			t.Fatal(e)
		}
	case <-time.After(2 * time.Second):
		close(release)
		<-done
		t.Fatal("AI blocked catalog/backup")
	}
	close(release)
	if e := <-done; e != nil {
		t.Fatal(e)
	}
	page, _ := s.OrganizationPage("", "", "")
	if e := s.DismissTag(a.ID, "cat", false); e != nil {
		t.Fatal(e)
	}
	page, _ = s.OrganizationPage("", "", "")
	if len(page.Assets[0].Analysis.Objects) != 0 || len(page.Assets[0].Dismissed) != 1 {
		t.Fatal("AI tag dismissal")
	}
	if e := s.DismissTag(a.ID, "cat", true); e != nil {
		t.Fatal(e)
	}
}

func TestOrganizationRestoredCatalogCheck(t *testing.T) {
	root := os.Getenv("IMMICH_ORGANIZATION_RESTORE")
	if root == "" {
		t.Skip("cloud restore checkpoint only")
	}
	s, e := Open(root)
	if e != nil {
		t.Fatal(e)
	}
	defer s.Close()
	page, e := s.OrganizationPage("", "", "")
	if e != nil {
		t.Fatal(e)
	}
	names := map[string]bool{}
	for _, f := range page.Faces {
		names[f.Name] = true
		if _, e = s.FaceCrop(f.ID); e != nil {
			t.Fatal("independent crop rebuild", e)
		}
	}
	if !names["Test Person"] || !names["Manual Label"] {
		t.Fatal("restored names missing")
	}
	found := false
	for _, a := range page.Assets {
		for _, tag := range a.Tags {
			if tag == "Birthday" {
				found = true
			}
		}
	}
	if !found {
		t.Fatal("restored custom tags missing")
	}
	if e = s.Verify(); e != nil {
		t.Fatal(e)
	}
}

func TestForgedFaceCannotEscapeDerivedDirectory(t *testing.T) {
	s, _ := openTest(t)
	a, _, _ := s.Put("synthetic.png", bytes.NewReader(fixture(t)))
	s.SetRecognizer(fakeAnalysis, true)
	s.QueueAnalysis(a.ID)
	s.ProcessAnalysis(context.Background())
	page, _ := s.OrganizationPage("", "", "")
	id := page.Faces[0].ID
	e := s.db.Update(func(tx *bolt.Tx) error {
		var f StoredFace
		if e := readJSON(tx.Bucket([]byte(facesBucket)), id, &f); e != nil {
			return e
		}
		f.ID = "../../outside"
		return writeJSON(tx.Bucket([]byte(facesBucket)), id, f)
	})
	if e != nil {
		t.Fatal(e)
	}
	if _, e = s.FaceCrop(id); e == nil {
		t.Fatal("forged metadata accepted")
	}
	if e = s.Verify(); e != nil {
		t.Fatal(e)
	}
}
