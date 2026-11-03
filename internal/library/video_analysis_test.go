package library

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"image"
	"immichclone/internal/media"
	"immichclone/internal/recognition"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
)

func mediaFixture(t *testing.T, name string) []byte {
	t.Helper()
	b, e := os.ReadFile("../../experiments/media-worker/fixtures/" + name + ".b64")
	if e != nil {
		t.Fatal(e)
	}
	b, e = base64.StdEncoding.DecodeString(string(b))
	if e != nil {
		t.Fatal(e)
	}
	return b
}
func mockAnalysisMedia(s *Store) {
	s.SetAnalysisMedia(func(context.Context, string) (image.Image, int, int, error) {
		return image.NewRGBA(image.Rect(0, 0, 100, 100)), 100, 100, nil
	}, func(context.Context, string) (media.Clip, error) {
		return media.Clip{DurationUS: 12000000, Width: 100, Height: 100}, nil
	}, func(_ context.Context, _ string, at int64) (image.Image, media.Sample, error) {
		return image.NewRGBA(image.Rect(0, 0, 100, 100)), media.Sample{PositionUS: at / 2000000 * 2000000, Width: 100, Height: 100}, nil
	}, true)
}
func TestHEICAnalysisUsesOriginalDecoderAndBulkEligibility(t *testing.T) {
	s, _ := openTest(t)
	mockAnalysisMedia(s)
	s.SetRecognizer(fakeAnalysis, true)
	a, _, e := s.Put("face.heic", bytes.NewReader(mediaFixture(t, "synthetic.heic")))
	if e != nil {
		t.Fatal(e)
	}
	if e = s.ControlAnalysisBatch(context.Background(), "start"); e != nil {
		t.Fatal(e)
	}
	b, e := s.AnalysisBatch()
	if e != nil || b.Total != 1 {
		t.Fatal("HEIC not selected", e)
	}
	if ok, e := s.ProcessAnalysis(context.Background()); !ok || e != nil {
		t.Fatal(e)
	}
	p, e := s.OrganizationPage("", "", "")
	if e != nil || len(p.Faces) != 1 || p.Assets[0].Analysis.Status != "ready" || !p.MediaAvailable {
		t.Fatal("HEIC processing", e)
	}
	if e = s.Verify(); e != nil {
		t.Fatal(e)
	}
	if e = s.ManualFace(context.Background(), a.ID, []float64{.1, .1, .2, .2}, "Manual HEIC"); e != nil {
		t.Fatal(e)
	}
}

func TestVideoSamplesPauseCancelResumeAndIndependentRestore(t *testing.T) {
	s, root := openTest(t)
	mockAnalysisMedia(s)
	var calls atomic.Int32
	s.SetRecognizer(func(ctx context.Context, img image.Image, path string) (recognition.Result, error) {
		calls.Add(1)
		return fakeAnalysis(ctx, img, path)
	}, true)
	data := mediaFixture(t, "hevc.mp4")
	a, _, e := s.Put("sample.mp4", bytes.NewReader(data))
	if e != nil {
		t.Fatal(e)
	}
	if e = s.QueueAnalysis(a.ID); e != nil {
		t.Fatal(e)
	}
	if ok, e := s.ProcessAnalysis(context.Background()); !ok || e != nil {
		t.Fatal(e)
	}
	p, _ := s.OrganizationPage("", "", "")
	if len(p.Faces) != 1 || p.Faces[0].SampleUS == nil || *p.Faces[0].SampleUS != 0 {
		t.Fatal("timestamped face absent")
	}
	first := p.Faces[0].ID
	if e = s.LabelFaces([]string{first}, "Video person"); e != nil {
		t.Fatal(e)
	}
	if e = s.SetTags(a.ID, []string{"Video custom"}); e != nil {
		t.Fatal(e)
	}
	if e = s.ControlVideoAnalysis(a.ID, "pause"); e != nil {
		t.Fatal(e)
	}
	if ok, e := s.ProcessAnalysis(context.Background()); ok || e != nil {
		t.Fatal("paused work ran", e)
	}
	if e = s.Close(); e != nil {
		t.Fatal(e)
	}
	s, e = Open(root)
	if e != nil {
		t.Fatal(e)
	}
	mockAnalysisMedia(s)
	s.SetRecognizer(func(ctx context.Context, img image.Image, path string) (recognition.Result, error) {
		calls.Add(1)
		return fakeAnalysis(ctx, img, path)
	}, true)
	p, _ = s.OrganizationPage("", "", "")
	if p.Assets[0].Analysis.Status != "paused" || p.Assets[0].Analysis.Video.Next != 1 {
		t.Fatal("pause did not persist")
	}
	if e = s.ControlVideoAnalysis(a.ID, "resume"); e != nil {
		t.Fatal(e)
	}
	if _, e = s.ProcessAnalysis(context.Background()); e != nil {
		t.Fatal(e)
	}
	if calls.Load() != 1 {
		t.Fatal("duplicate actual PTS was analyzed twice")
	}
	if e = s.ControlVideoAnalysis(a.ID, "cancel"); e != nil {
		t.Fatal(e)
	}
	if ok, e := s.ProcessAnalysis(context.Background()); ok || e != nil {
		t.Fatal("cancelled work ran")
	}
	if e = s.ControlVideoAnalysis(a.ID, "resume"); e != nil {
		t.Fatal(e)
	}
	for i := 0; i < 20; i++ {
		ok, e := s.ProcessAnalysis(context.Background())
		if e != nil {
			t.Fatal(e)
		}
		if !ok {
			break
		}
	}
	p, _ = s.OrganizationPage("", "", "")
	v := p.Assets[0].Analysis.Video
	if p.Assets[0].Analysis.Status != "ready" || v.Next != 12 || len(v.Samples) != 6 || len(p.Faces) != 6 || calls.Load() != 6 {
		t.Fatal("sample accounting or duplication")
	}
	named := false
	for _, f := range p.Faces {
		if f.ID == first && f.Name == "Video person" {
			named = true
		}
	}
	if !named {
		t.Fatal("name lost")
	}
	export, e := s.OrganizationExport()
	if e != nil {
		t.Fatal(e)
	}
	encoded, _ := json.Marshal(export)
	if !bytes.Contains(encoded, []byte("position_us")) || bytes.Contains(encoded, []byte("\"embedding\":[")) {
		t.Fatal("video export incorrect")
	}
	var backup bytes.Buffer
	if e = s.Backup(&backup); e != nil {
		t.Fatal(e)
	}
	archive := filepath.Join(t.TempDir(), "backup.zip")
	if e = os.WriteFile(archive, backup.Bytes(), 0600); e != nil {
		t.Fatal(e)
	}
	target := filepath.Join(t.TempDir(), "independent")
	if e = Restore(archive, target); e != nil {
		t.Fatal(e)
	}
	s.Close()
	if e = os.Rename(root, root+"-offline"); e != nil {
		t.Fatal(e)
	}
	restored, e := Open(target)
	if e != nil {
		t.Fatal(e)
	}
	defer restored.Close()
	mockAnalysisMedia(restored)
	p, e = restored.OrganizationPage("", "", "")
	if e != nil || len(p.Faces) != 6 || p.Assets[0].Analysis.Video.Next != 12 || len(p.Assets[0].Tags) != 1 {
		t.Fatal("restored video metadata lost", e)
	}
	for _, f := range p.Faces {
		if _, e = restored.FaceCrop(f.ID); e != nil {
			t.Fatal("video crop regeneration", e)
		}
	}
	if _, e = restored.FaceContext(context.Background(), first); e != nil {
		t.Fatal("saved frame context", e)
	}
	if e = restored.Verify(); e != nil {
		t.Fatal(e)
	}
}
func TestVideoCancellationDoesNotAdvanceCheckpoint(t *testing.T) {
	s, _ := openTest(t)
	mockAnalysisMedia(s)
	s.SetRecognizer(fakeAnalysis, true)
	a, _, e := s.Put("sample.mp4", bytes.NewReader(mediaFixture(t, "hevc.mp4")))
	if e != nil {
		t.Fatal(e)
	}
	entered := make(chan struct{})
	s.SetAnalysisMedia(nil, func(context.Context, string) (media.Clip, error) { return media.Clip{DurationUS: 12000000}, nil }, func(ctx context.Context, _ string, _ int64) (image.Image, media.Sample, error) {
		close(entered)
		<-ctx.Done()
		return nil, media.Sample{}, ctx.Err()
	}, true)
	if e = s.QueueAnalysis(a.ID); e != nil {
		t.Fatal(e)
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { _, e := s.ProcessAnalysis(ctx); done <- e }()
	<-entered
	cancel()
	if e = <-done; e != context.Canceled {
		t.Fatal(e)
	}
	p, _ := s.OrganizationPage("", "", "")
	if p.Assets[0].Analysis.Status != "queued" || p.Assets[0].Analysis.Video.Next != 0 || len(p.Faces) != 0 {
		t.Fatal("cancelled sample committed")
	}
}

// Cloud checkpoint uses the real decoder, but no AI model or old state, to rebuild
// derived crops and full sampled-frame context from the independent archive.
func TestMediaAIApplicationRestoredCheckpoint(t *testing.T) {
	root := os.Getenv("IMMICH_6A_RESTORE")
	if root == "" {
		t.Skip("cloud application checkpoint only")
	}
	s, e := Open(root)
	if e != nil {
		t.Fatal(e)
	}
	defer s.Close()
	decoder := media.New(os.Getenv("IMMICH_6A_MEDIA"))
	s.SetAnalysisMedia(decoder.AnalysisFrame, decoder.VideoInfo, decoder.FrameAt, decoder.Available())
	o, e := s.OrganizationPage("", "", "")
	if e != nil {
		t.Fatal(e)
	}
	videoCount, heicCount := 0, 0
	for _, a := range o.Assets {
		if a.Analysis.Status != "ready" {
			t.Fatal("restored analysis not ready")
		}
		if isHEIC(a.Asset.MIME) {
			heicCount++
		}
		if isVideo(a.Asset.MIME) {
			videoCount++
			v := a.Analysis.Video
			if !validVideo(v) || v.Next != 12 || len(v.Samples) == 0 {
				t.Fatal("restored sample coverage")
			}
		}
	}
	if videoCount != 1 || heicCount != 2 {
		t.Fatal("restored assets missing")
	}
	named := false
	for _, f := range o.Faces {
		if _, e = s.FaceCrop(f.ID); e != nil {
			t.Fatal("independent crop", e)
		}
		if f.Name == "Video Test Person" {
			named = true
			if f.SampleUS == nil {
				t.Fatal("timestamp lost")
			}
			if _, e = s.FaceContext(context.Background(), f.ID); e != nil {
				t.Fatal("independent context", e)
			}
		}
	}
	if !named {
		t.Fatal("restored video correction missing")
	}
	if e = s.Verify(); e != nil {
		t.Fatal(e)
	}
}

func TestVideoPauseDuringInferenceCommitsOnlyCurrentFrame(t *testing.T) {
	s, _ := openTest(t)
	mockAnalysisMedia(s)
	entered, release := make(chan struct{}), make(chan struct{})
	s.SetRecognizer(func(ctx context.Context, img image.Image, dir string) (recognition.Result, error) {
		close(entered)
		<-release
		return fakeAnalysis(ctx, img, dir)
	}, true)
	a, _, e := s.Put("clip.mp4", bytes.NewReader(mediaFixture(t, "hevc.mp4")))
	if e != nil {
		t.Fatal(e)
	}
	if e = s.QueueAnalysis(a.ID); e != nil {
		t.Fatal(e)
	}
	done := make(chan error, 1)
	go func() { _, e := s.ProcessAnalysis(context.Background()); done <- e }()
	<-entered
	if e = s.ControlVideoAnalysis(a.ID, "pause"); e != nil {
		t.Fatal(e)
	}
	close(release)
	if e = <-done; e != nil {
		t.Fatal(e)
	}
	o, e := s.OrganizationPage("", "", "")
	if e != nil {
		t.Fatal(e)
	}
	v := o.Assets[0].Analysis
	if v.Status != "paused" || v.Video.Next != 1 || len(v.Video.Samples) != 1 || len(o.Faces) != 1 {
		t.Fatal("in-flight pause did not preserve exactly the completed frame")
	}
	if processed, e := s.ProcessAnalysis(context.Background()); processed || e != nil {
		t.Fatal("paused queue processed another frame", e)
	}
}
