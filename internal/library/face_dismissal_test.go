package library

import (
	"bytes"
	"context"
	"encoding/json"
	"image"
	"immichclone/internal/recognition"
	"os"
	"path/filepath"
	"testing"
)

func TestFaceDismissalPreservesCorrectionsRestartAndRestore(t *testing.T) {
	s, root := openTest(t)
	s.SetRecognizer(func(ctx context.Context, img image.Image, dir string) (recognition.Result, error) {
		r, e := fakeAnalysis(ctx, img, dir)
		f := r.Faces[0]
		f.Box = []float64{.6, .1, .2, .4}
		r.Faces = append(r.Faces, f)
		return r, e
	}, true)
	a, _, e := s.Put("fixture.png", bytes.NewReader(fixture(t)))
	if e != nil {
		t.Fatal(e)
	}
	if e = s.QueueAnalysis(a.ID); e != nil {
		t.Fatal(e)
	}
	if _, e = s.ProcessAnalysis(context.Background()); e != nil {
		t.Fatal(e)
	}
	page, e := s.OrganizationPage("", "", "")
	if e != nil || len(page.Faces) != 2 {
		t.Fatal("fixture faces", e)
	}
	id := page.Faces[0].ID
	for _, name := range []string{"First name", "Corrected name"} {
		if e = s.LabelFaces([]string{id}, name); e != nil {
			t.Fatal(e)
		}
	}
	if e = s.SetTags(a.ID, []string{"Keep tag"}); e != nil {
		t.Fatal(e)
	}
	page, _ = s.OrganizationPage("", "", "")
	found := false
	for _, f := range page.Faces {
		if f.ID != id && f.Suggestion == "Corrected name" {
			found = true
		}
	}
	if !found {
		t.Fatal("positive suggestion control")
	}
	if e = s.DismissFace(id, false); e != nil {
		t.Fatal(e)
	}
	page, e = s.OrganizationPage("", "", "")
	if e != nil || len(page.Faces) != 1 || len(page.Names) != 0 || page.Faces[0].Suggestion != "" {
		t.Fatal("dismissed reference remained active", e)
	}
	if e = s.LabelFaces([]string{id}, "Unexpected overwrite"); e == nil {
		t.Fatal("dismissed face renamed")
	}
	if e = s.DismissFace("invalid", false); e == nil {
		t.Fatal("invalid face accepted")
	}
	exported, e := s.OrganizationExport()
	if e != nil {
		t.Fatal(e)
	}
	data, _ := json.Marshal(exported)
	if !bytes.Contains(data, []byte(`"dismissed":true`)) {
		t.Fatal("dismissal missing from export")
	}
	if e = s.Close(); e != nil {
		t.Fatal(e)
	}
	s, e = Open(root)
	if e != nil {
		t.Fatal(e)
	}
	defer s.Close()
	dismissed, e := s.OrganizationReviewPage("", "", "", true)
	if e != nil || len(dismissed.Faces) != 1 || dismissed.Faces[0].Name != "Corrected name" || !dismissed.Faces[0].CanUndo {
		t.Fatal("restart lost correction", e)
	}
	var archive bytes.Buffer
	if e = s.Backup(&archive); e != nil {
		t.Fatal(e)
	}
	path := filepath.Join(t.TempDir(), "backup.zip")
	if e = os.WriteFile(path, archive.Bytes(), 0600); e != nil {
		t.Fatal(e)
	}
	target := filepath.Join(t.TempDir(), "restored")
	if e = Restore(path, target); e != nil {
		t.Fatal(e)
	}
	restored, e := Open(target)
	if e != nil {
		t.Fatal(e)
	}
	defer restored.Close()
	dismissed, e = restored.OrganizationReviewPage("", "", "", true)
	if e != nil || len(dismissed.Faces) != 1 || !dismissed.Faces[0].Dismissed {
		t.Fatal("restore lost dismissal", e)
	}
	if e = restored.DismissFace(id, true); e != nil {
		t.Fatal(e)
	}
	if e = restored.UndoFace(id); e != nil {
		t.Fatal(e)
	}
	page, e = restored.OrganizationPage("", "", "")
	found = false
	for _, f := range page.Faces {
		if f.ID == id && f.Name == "First name" && !f.Dismissed {
			found = true
		}
	}
	if e != nil || !found || page.Assets[0].Tags[0] != "Keep tag" || page.Assets[0].Analysis.Status != "ready" {
		t.Fatal("restore/undo changed unrelated results", e)
	}
	if e = restored.Verify(); e != nil {
		t.Fatal(e)
	}
}
