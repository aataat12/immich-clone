package library

import (
	"bytes"
	"context"
	"errors"
	"image"
	"image/color"
	"image/png"
	"immichclone/internal/recognition"
	"os"
	"path/filepath"
	"testing"
)

func batchPhoto(t *testing.T, s *Store, n uint8) Asset {
	t.Helper()
	im := image.NewRGBA(image.Rect(0, 0, 40, 40))
	im.Set(0, 0, color.RGBA{n, 23, 42, 255})
	var buf bytes.Buffer
	png.Encode(&buf, im)
	a, _, e := s.Put("batch.png", bytes.NewReader(buf.Bytes()))
	if e != nil {
		t.Fatal(e)
	}
	return a
}
func batchControl(t *testing.T, s *Store, action string) AnalysisBatch {
	t.Helper()
	if e := s.ControlAnalysisBatch(context.Background(), action); e != nil {
		t.Fatal(action, e)
	}
	b, e := s.AnalysisBatch()
	if e != nil {
		t.Fatal(e)
	}
	return b
}
func TestAnalysisBatchSelectionPauseCancelRestore(t *testing.T) {
	s, root := openTest(t)
	s.SetRecognizer(fakeAnalysis, true)
	ready := batchPhoto(t, s, 1)
	s.QueueAnalysis(ready.ID)
	s.ProcessAnalysis(context.Background())
	page, _ := s.OrganizationPage("", "", "")
	face := page.Faces[0].ID
	s.LabelFaces([]string{face}, "Preserved name")
	s.SetTags(ready.ID, []string{"Preserved tag"})
	hidden := batchPhoto(t, s, 2)
	s.SetHidden(hidden.ID, true)
	failed := batchPhoto(t, s, 3)
	s.SetRecognizer(func(context.Context, image.Image, string) (recognition.Result, error) {
		return recognition.Result{}, errors.New("fixture failure")
	}, true)
	s.QueueAnalysis(failed.ID)
	s.ProcessAnalysis(context.Background())
	s.SetRecognizer(fakeAnalysis, true)
	individual := batchPhoto(t, s, 4)
	s.QueueAnalysis(individual.ID)
	s.Put("unsupported.txt", bytes.NewReader([]byte("unsupported")))
	a := batchPhoto(t, s, 5)
	b := batchPhoto(t, s, 6)
	if job := batchControl(t, s, "start"); job.Total != 2 || job.Pending != 2 {
		t.Fatalf("selection: %+v", job)
	}
	if e := s.ControlAnalysisBatch(context.Background(), "start"); e == nil {
		t.Fatal("duplicate batch")
	}
	batchControl(t, s, "pause")
	if ok, e := s.ProcessAnalysis(context.Background()); !ok || e != nil {
		t.Fatal("individual blocked", e)
	}
	if ok, e := s.ProcessAnalysis(context.Background()); ok || e != nil {
		t.Fatal("paused batch ran", e)
	}
	// A consistent independent restore preserves paused membership and corrections.
	var backup bytes.Buffer
	if e := s.Backup(&backup); e != nil {
		t.Fatal(e)
	}
	zip := filepath.Join(t.TempDir(), "backup.zip")
	os.WriteFile(zip, backup.Bytes(), 0600)
	target := filepath.Join(t.TempDir(), "restored")
	if e := Restore(zip, target); e != nil {
		t.Fatal(e)
	}
	s.Close()
	os.Rename(root, root+"-unavailable")
	r, e := Open(target)
	if e != nil {
		t.Fatal(e)
	}
	defer r.Close()
	r.SetRecognizer(fakeAnalysis, true)
	if job, e := r.AnalysisBatch(); e != nil || job.Status != "paused" || job.Pending != 2 {
		t.Fatal(job, e)
	}
	batchControl(t, r, "resume")
	if ok, e := r.ProcessAnalysis(context.Background()); !ok || e != nil {
		t.Fatal(e)
	}
	if job := batchControl(t, r, "cancel"); job.Pending != 0 || job.Ready != 1 || job.Cancelled != 1 {
		t.Fatal(job)
	}
	for _, asset := range []Asset{a, b, ready, hidden, individual, failed} {
		if hash, e := fileHashContext(context.Background(), r.Original(asset.ID)); e != nil || hash != asset.ID {
			t.Fatal("original changed", e)
		}
	}
	page, _ = r.OrganizationPage("", "", "")
	found := false
	for _, f := range page.Faces {
		if f.ID == face && f.Name == "Preserved name" {
			found = true
		}
	}
	if !found {
		t.Fatal("name lost")
	}
	if job := batchControl(t, r, "start"); job.Total != 1 {
		t.Fatal("cancelled work not eligible or completed/failed repeated", job)
	}
	r.ProcessAnalysis(context.Background())
	if job, _ := r.AnalysisBatch(); job.Status != "complete" || job.Ready != 1 {
		t.Fatal(job)
	}
}
func TestAnalysisBatchCancelInflightAndRestart(t *testing.T) {
	s, root := openTest(t)
	batchPhoto(t, s, 1)
	batchPhoto(t, s, 2)
	entered := make(chan struct{})
	release := make(chan struct{})
	done := make(chan error, 1)
	s.SetRecognizer(func(ctx context.Context, im image.Image, p string) (recognition.Result, error) {
		close(entered)
		<-release
		return fakeAnalysis(ctx, im, p)
	}, true)
	batchControl(t, s, "start")
	go func() { _, e := s.ProcessAnalysis(context.Background()); done <- e }()
	<-entered
	if b := batchControl(t, s, "cancel"); b.Pending != 1 || b.Cancelled != 1 {
		t.Fatal(b)
	}
	if e := s.ControlAnalysisBatch(context.Background(), "start"); e == nil {
		t.Fatal("replaced draining batch")
	}
	close(release)
	if e := <-done; e != nil {
		t.Fatal(e)
	}
	if b, _ := s.AnalysisBatch(); b.Pending != 0 || b.Ready != 1 || b.Status != "cancelled" {
		t.Fatal(b)
	}
	s.SetRecognizer(fakeAnalysis, true)
	batchControl(t, s, "start")
	batchControl(t, s, "pause")
	s.Close()
	r, e := Open(root)
	if e != nil {
		t.Fatal(e)
	}
	defer r.Close()
	r.SetRecognizer(fakeAnalysis, true)
	if ok, e := r.ProcessAnalysis(context.Background()); ok || e != nil {
		t.Fatal("pause lost after restart", e)
	}
	batchControl(t, r, "resume")
	r.ProcessAnalysis(context.Background())
	if b, _ := r.AnalysisBatch(); b.Status != "complete" {
		t.Fatal(b)
	}
}
func TestAnalysisBatchCancelledContextAndUnavailable(t *testing.T) {
	s, _ := openTest(t)
	batchPhoto(t, s, 1)
	if e := s.ControlAnalysisBatch(context.Background(), "start"); e == nil {
		t.Fatal("unavailable AI accepted")
	}
	s.SetRecognizer(fakeAnalysis, true)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if e := s.ControlAnalysisBatch(ctx, "start"); !errors.Is(e, context.Canceled) {
		t.Fatal(e)
	}
	if b, _ := s.AnalysisBatch(); b.Status != "idle" || b.Total != 0 {
		t.Fatal("cancelled transaction committed", b)
	}
}

func TestAnalysisBatchFailureImportPriorityAndShutdownRecovery(t *testing.T) {
	s, root := openTest(t)
	batchPhoto(t, s, 1)
	batchPhoto(t, s, 2)
	s.SetRecognizer(fakeAnalysis, true)
	batchControl(t, s, "start")
	s.renewImport("synthetic-lease")
	if ok, e := s.ProcessAnalysis(context.Background()); ok || e != nil {
		t.Fatal("import priority ignored", e)
	}
	s.importLease.Store(nil)
	s.SetRecognizer(func(context.Context, image.Image, string) (recognition.Result, error) {
		return recognition.Result{}, errors.New("synthetic failure")
	}, true)
	s.ProcessAnalysis(context.Background())
	if b, _ := s.AnalysisBatch(); b.Failed != 1 || b.Pending != 1 {
		t.Fatal(b)
	}
	ctx, cancel := context.WithCancel(context.Background())
	s.SetRecognizer(func(context.Context, image.Image, string) (recognition.Result, error) {
		// Model shutdown after cancel is persisted must never strand pending work.
		batchControl(t, s, "cancel")
		cancel()
		return recognition.Result{}, context.Canceled
	}, true)
	if _, e := s.ProcessAnalysis(ctx); !errors.Is(e, context.Canceled) {
		t.Fatal(e)
	}
	s.Close()
	r, e := Open(root)
	if e != nil {
		t.Fatal(e)
	}
	defer r.Close()
	r.SetRecognizer(fakeAnalysis, true)
	b, _ := r.AnalysisBatch()
	if b.Pending != 0 || b.Failed != 1 || b.Cancelled != 1 || b.Status != "cancelled" {
		t.Fatal("shutdown cancellation recovery", b)
	}
	if job := batchControl(t, r, "start"); job.Total != 1 {
		t.Fatal(job)
	}
}
