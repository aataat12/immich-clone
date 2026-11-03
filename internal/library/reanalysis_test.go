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
)

func snapshotReview(t *testing.T, s *Store, id string) map[string][]byte {
	t.Helper()
	m := map[string][]byte{}
	if e := s.db.View(func(tx *bolt.Tx) error {
		for _, name := range []string{analysisBucket, facesBucket, labelsBucket, tagsBucket, "dismissed_ai_tags_v1"} {
			b := tx.Bucket([]byte(name))
			if e := b.ForEach(func(k, v []byte) error { m[name+"/"+string(k)] = append([]byte(nil), v...); return nil }); e != nil {
				return e
			}
		}
		return nil
	}); e != nil {
		t.Fatal(e)
	}
	return m
}
func readReanalysisTest(t *testing.T, s *Store, id string) reanalysisJob {
	t.Helper()
	var j reanalysisJob
	if e := s.db.View(func(tx *bolt.Tx) error { return readJSON(tx.Bucket([]byte(reanalysisBucket)), id, &j) }); e != nil {
		t.Fatal(e)
	}
	return j
}
func seedReviewedPhoto(t *testing.T) (*Store, string, string) {
	s, _ := openTest(t)
	s.SetRecognizer(fakeAnalysis, true)
	a, _, e := s.Put("photo.png", bytes.NewReader(fixture(t)))
	if e != nil {
		t.Fatal(e)
	}
	if e = s.QueueAnalysis(a.ID); e != nil {
		t.Fatal(e)
	}
	if _, e = s.ProcessAnalysis(context.Background()); e != nil {
		t.Fatal(e)
	}
	o, _ := s.OrganizationPage("", "", "")
	face := o.Faces[0].ID
	for _, e := range []error{s.LabelFaces([]string{face}, "Saved name"), s.DismissFace(face, false), s.SetTags(a.ID, []string{"Custom tag"}), s.DismissTag(a.ID, "cat", false)} {
		if e != nil {
			t.Fatal(e)
		}
	}
	return s, a.ID, face
}
func proposalAnalysis(ctx context.Context, img image.Image, dir string) (recognition.Result, error) {
	r, e := fakeAnalysis(ctx, img, dir)
	f := r.Faces[0]
	f.Box = []float64{.65, .1, .2, .25}
	r.Faces = append(r.Faces, f)
	r.Objects = []recognition.Object{{Class: 41, Confidence: .9, Box: []float64{.1, .1, .2, .2}}}
	return r, e
}
func TestReanalysisPreviewApplyPreservesCorrectionsAndNoDuplicates(t *testing.T) {
	s, id, old := seedReviewedPhoto(t)
	before := snapshotReview(t, s, id)
	s.SetRecognizer(proposalAnalysis, true)
	if e := s.ControlReanalysis(id, "start"); e != nil {
		t.Fatal(e)
	}
	if _, e := s.ProcessAnalysis(context.Background()); e != nil {
		t.Fatal(e)
	}
	j := readReanalysisTest(t, s, id)
	if j.Status != "ready" || len(j.Faces) != 2 {
		t.Fatal("proposal missing")
	}
	after := snapshotReview(t, s, id)
	for k, v := range before {
		if !bytes.Equal(v, after[k]) {
			t.Fatal("proposal changed existing review", k)
		}
	}
	out, e := s.OrganizationExport()
	if e != nil {
		t.Fatal(e)
	}
	b, _ := json.Marshal(out)
	if bytes.Contains(b, []byte("\"embedding\":[")) {
		t.Fatal("export exposed staged embeddings")
	}
	if e = s.ApplyReanalysis(context.Background(), id); e != nil {
		t.Fatal(e)
	}
	after = snapshotReview(t, s, id)
	for k, v := range before {
		if k != analysisBucket+"/"+id && !bytes.Equal(v, after[k]) {
			t.Fatal("apply changed old correction", k)
		}
	}
	j = readReanalysisTest(t, s, id)
	if j.Status != "applied" || j.Added != 1 || j.Retained != 1 || j.Skipped != 1 {
		t.Fatal("apply accounting", j.Status, j.Added, j.Retained, j.Skipped)
	}
	if e = s.ApplyReanalysis(context.Background(), id); e == nil {
		t.Fatal("proposal applied twice")
	}
	if e = s.ControlReanalysis(id, "start"); e != nil {
		t.Fatal(e)
	}
	s.ProcessAnalysis(context.Background())
	if e = s.ApplyReanalysis(context.Background(), id); e != nil {
		t.Fatal(e)
	}
	j = readReanalysisTest(t, s, id)
	if j.Added != 0 {
		t.Fatal("duplicate observations after repeat")
	}
	o, e := s.OrganizationFilteredPage("", "", "", true, false)
	if e != nil || len(o.Faces) != 1 || o.Faces[0].ID != old || o.Faces[0].Name != "Saved name" {
		t.Fatal("dismissed name/history lost", e)
	}
	if e = s.DismissTag(id, "cat", true); e != nil {
		t.Fatal("cannot undo old tag dismissal", e)
	}
	if e = s.Verify(); e != nil {
		t.Fatal(e)
	}
}
func TestReanalysisFailureDiscardAndRestartKeepReadyData(t *testing.T) {
	s, id, _ := seedReviewedPhoto(t)
	before := snapshotReview(t, s, id)
	s.SetRecognizer(func(context.Context, image.Image, string) (recognition.Result, error) {
		return recognition.Result{}, errors.New("controlled failure")
	}, true)
	s.ControlReanalysis(id, "start")
	s.ProcessAnalysis(context.Background())
	if readReanalysisTest(t, s, id).Status != "failed" {
		t.Fatal("failure not visible")
	}
	after := snapshotReview(t, s, id)
	for k, v := range before {
		if !bytes.Equal(v, after[k]) {
			t.Fatal("failure changed original results")
		}
	}
	s.ControlReanalysis(id, "resume")
	s.ControlReanalysis(id, "pause")
	if p, e := s.ProcessAnalysis(context.Background()); p || e != nil {
		t.Fatal("paused proposal processed", e)
	}
	root := s.root
	s.Close()
	reopened, e := Open(root)
	if e != nil {
		t.Fatal(e)
	}
	defer reopened.Close()
	if readReanalysisTest(t, reopened, id).Status != "paused" {
		t.Fatal("pause lost")
	}
	reopened.ControlReanalysis(id, "discard")
	if readReanalysisTest(t, reopened, id).Status != "cancelled" {
		t.Fatal("discard not persisted")
	}
}
func TestReanalysisDiscardDuringInferenceCannotPublish(t *testing.T) {
	s, id, _ := seedReviewedPhoto(t)
	entered, release := make(chan struct{}), make(chan struct{})
	s.SetRecognizer(func(ctx context.Context, img image.Image, dir string) (recognition.Result, error) {
		close(entered)
		<-release
		return proposalAnalysis(ctx, img, dir)
	}, true)
	s.ControlReanalysis(id, "start")
	done := make(chan error, 1)
	go func() { _, e := s.ProcessAnalysis(context.Background()); done <- e }()
	<-entered
	s.ControlReanalysis(id, "discard")
	s.ControlReanalysis(id, "start")
	token := readReanalysisTest(t, s, id).Token
	close(release)
	if e := <-done; e != nil {
		t.Fatal(e)
	}
	j := readReanalysisTest(t, s, id)
	if j.Token != token || j.Status != "queued" || len(j.Faces) != 0 {
		t.Fatal("stale inference replaced new request")
	}
}
func TestReanalysisVideoCheckpointAndIndependentProposalRestore(t *testing.T) {
	s, root := openTest(t)
	mockAnalysisMedia(s)
	s.SetRecognizer(fakeAnalysis, true)
	a, _, e := s.Put("video.mp4", bytes.NewReader(mediaFixture(t, "hevc.mp4")))
	if e != nil {
		t.Fatal(e)
	}
	s.QueueAnalysis(a.ID)
	for i := 0; i < 12; i++ {
		if _, e = s.ProcessAnalysis(context.Background()); e != nil {
			t.Fatal(e)
		}
	}
	before := snapshotReview(t, s, a.ID)
	s.ControlReanalysis(a.ID, "start")
	for i := 0; i < 3; i++ {
		s.ProcessAnalysis(context.Background())
	}
	s.ControlReanalysis(a.ID, "pause")
	if readReanalysisTest(t, s, a.ID).Video.Next != 3 {
		t.Fatal("video proposal checkpoint missing")
	}
	s.Close()
	s, e = Open(root)
	if e != nil {
		t.Fatal(e)
	}
	mockAnalysisMedia(s)
	s.SetRecognizer(fakeAnalysis, true)
	s.ControlReanalysis(a.ID, "resume")
	for i := 3; i < 12; i++ {
		if _, e = s.ProcessAnalysis(context.Background()); e != nil {
			t.Fatal(e)
		}
	}
	j := readReanalysisTest(t, s, a.ID)
	if j.Status != "ready" || j.Video.Next != 12 || len(j.Video.Samples) != 6 {
		t.Fatal("sample restart or dedup failed")
	}
	after := snapshotReview(t, s, a.ID)
	for k, v := range before {
		if !bytes.Equal(v, after[k]) {
			t.Fatal("video proposal changed ready results")
		}
	}
	var archive bytes.Buffer
	if e = s.Backup(&archive); e != nil {
		t.Fatal(e)
	}
	s.Close()
	file := filepath.Join(t.TempDir(), "backup.zip")
	os.WriteFile(file, archive.Bytes(), 0600)
	target := filepath.Join(t.TempDir(), "restored")
	if e = Restore(file, target); e != nil {
		t.Fatal(e)
	}
	r, e := Open(target)
	if e != nil {
		t.Fatal(e)
	}
	defer r.Close()
	mockAnalysisMedia(r)
	if e = r.ApplyReanalysis(context.Background(), a.ID); e != nil {
		t.Fatal(e)
	}
	j = readReanalysisTest(t, r, a.ID)
	if j.Added != 0 || j.Retained != 6 || j.Skipped != 6 {
		t.Fatal("restored proposal duplicated original observations")
	}
	if e = r.Verify(); e != nil {
		t.Fatal(e)
	}
}

func TestReanalysisApplyUsesLatestManualCorrections(t *testing.T) {
	s, id, old := seedReviewedPhoto(t)
	s.SetRecognizer(proposalAnalysis, true)
	s.ControlReanalysis(id, "start")
	s.ProcessAnalysis(context.Background())
	if e := s.DismissFace(old, true); e != nil {
		t.Fatal(e)
	}
	if e := s.LabelFaces([]string{old}, "Edited while pending"); e != nil {
		t.Fatal(e)
	}
	if e := s.SetTags(id, []string{"Latest custom tag"}); e != nil {
		t.Fatal(e)
	}
	if e := s.ManualFace(context.Background(), id, []float64{.65, .1, .2, .25}, "New manual face"); e != nil {
		t.Fatal(e)
	}
	before := snapshotReview(t, s, id)
	if e := s.ApplyReanalysis(context.Background(), id); e != nil {
		t.Fatal(e)
	}
	after := snapshotReview(t, s, id)
	for k, v := range before {
		if k != analysisBucket+"/"+id && !bytes.Equal(v, after[k]) {
			t.Fatal("concurrent manual correction overwritten", k)
		}
	}
	j := readReanalysisTest(t, s, id)
	if j.Added != 0 || j.Retained != 2 || j.Skipped != 2 {
		t.Fatal("manual addition not included in duplicate check")
	}
}
