package library

import (
	"bytes"
	"context"
	bolt "go.etcd.io/bbolt"
	"image"
	"immichclone/internal/recognition"
	"testing"
)

func pipelineTwoPeople(context.Context, image.Image, string) (recognition.Result, error) {
	a, b := make([]float64, 128), make([]float64, 128)
	a[0], b[1] = 1, 1
	return recognition.Result{Protocol: 1, Faces: []recognition.Face{
		{Box: []float64{.1, .1, .2, .3}, Embedding: a, Confidence: .99},
		{Box: []float64{.6, .1, .2, .3}, Embedding: b, Confidence: .99},
	}}, nil
}

// These exercise the actual committed publication paths when production naming
// is enabled. No environment variable or test-only production bypass exists.
func TestAutomaticNamesEnabledAnalysisPublication(t *testing.T) {
	if !automaticNamesValidated {
		t.Skip("production assignment awaits independent quality acceptance")
	}
	s, _ := openTest(t)
	s.SetRecognizer(pipelineTwoPeople, true)
	for _, n := range []uint8{1, 2} {
		a := batchPhoto(t, s, n)
		if e := s.QueueAnalysis(a.ID); e != nil {
			t.Fatal(e)
		}
		if worked, e := s.ProcessAnalysis(context.Background()); !worked || e != nil {
			t.Fatal(e)
		}
		page, e := s.OrganizationPage("", "", "")
		if e != nil {
			t.Fatal(e)
		}
		for _, f := range page.Faces {
			if f.AssetID != a.ID {
				continue
			}
			var stored StoredFace
			if e = s.db.View(func(tx *bolt.Tx) error { return readJSON(tx.Bucket([]byte(facesBucket)), f.ID, &stored) }); e != nil {
				t.Fatal(e)
			}
			name := "Alice"
			if stored.Embedding[1] == 1 {
				name = "Bob"
			}
			if e = s.LabelFaces([]string{f.ID}, name); e != nil {
				t.Fatal(e)
			}
		}
	}
	newPhoto := batchPhoto(t, s, 3)
	if e := s.QueueAnalysis(newPhoto.ID); e != nil {
		t.Fatal(e)
	}
	if worked, e := s.ProcessAnalysis(context.Background()); !worked || e != nil {
		t.Fatal(e)
	}
	page, e := s.OrganizationPage("", "", "")
	if e != nil {
		t.Fatal(e)
	}
	assigned := map[string]string{}
	for _, f := range page.Faces {
		if f.AssetID == newPhoto.ID {
			if f.Source != "automatic" {
				t.Fatal("new photo not automatically named")
			}
			assigned[f.Name] = f.ID
		}
	}
	if len(assigned) != 2 || assigned["Alice"] == "" || assigned["Bob"] == "" {
		t.Fatal("multi-person assignment incomplete")
	}
	for _, name := range []string{"Alice", "Bob"} {
		p, e := s.SearchGalleryPage("name", "library", "", name, 100)
		if e != nil || p.Count != 3 {
			t.Fatal("automatic name missing from gallery", e, p.Count)
		}
	}
	if e = s.LabelFaces([]string{assigned["Alice"]}, "Corrected"); e != nil {
		t.Fatal(e)
	}
	if e = s.ControlReanalysis(newPhoto.ID, "start"); e != nil {
		t.Fatal(e)
	}
	if worked, e := s.ProcessAnalysis(context.Background()); !worked || e != nil {
		t.Fatal(e)
	}
	if e = s.ApplyReanalysis(context.Background(), newPhoto.ID); e != nil {
		t.Fatal(e)
	}
	if label := automaticLabel(t, s, assigned["Alice"]); label.Name != "Corrected" || label.Source != "manual" {
		t.Fatal("reanalysis overwrote correction")
	}
	// Additive reanalysis must name a newly found face while preserving an
	// explicitly cleared existing face on the same original.
	s.SetRecognizer(func(ctx context.Context, img image.Image, dir string) (recognition.Result, error) {
		result, e := pipelineTwoPeople(ctx, img, dir)
		result.Faces = result.Faces[:1]
		return result, e
	}, true)
	partial := batchPhoto(t, s, 4)
	if e = s.QueueAnalysis(partial.ID); e != nil {
		t.Fatal(e)
	}
	if worked, e := s.ProcessAnalysis(context.Background()); !worked || e != nil {
		t.Fatal(e)
	}
	page, e = s.OrganizationPage("", "", "")
	if e != nil {
		t.Fatal(e)
	}
	clearedID := ""
	for _, f := range page.Faces {
		if f.AssetID == partial.ID {
			clearedID = f.ID
		}
	}
	if clearedID == "" {
		t.Fatal("partial analysis face missing")
	}
	if e = s.LabelFaces([]string{clearedID}, ""); e != nil {
		t.Fatal(e)
	}
	s.SetRecognizer(pipelineTwoPeople, true)
	if e = s.ControlReanalysis(partial.ID, "start"); e != nil {
		t.Fatal(e)
	}
	if worked, e := s.ProcessAnalysis(context.Background()); !worked || e != nil {
		t.Fatal(e)
	}
	if e = s.ApplyReanalysis(context.Background(), partial.ID); e != nil {
		t.Fatal(e)
	}
	page, e = s.OrganizationPage("", "", "")
	if e != nil {
		t.Fatal(e)
	}
	addedBob := false
	for _, f := range page.Faces {
		if f.AssetID != partial.ID {
			continue
		}
		if f.ID == clearedID {
			if f.Name != "" {
				t.Fatal("reanalysis renamed explicitly cleared face")
			}
			continue
		}
		if f.Name == "Bob" && f.Source == "automatic" {
			addedBob = true
		}
	}
	if !addedBob {
		t.Fatal("additive reanalysis did not name newly found Bob")
	}
	// A sampled video is a new original, but its multiple frames cannot create
	// more independent references for later media. The prior Alice correction is
	// a competing manually named identity with the same vector; Alice must now
	// remain in review while Bob still matches unambiguously.
	mockAnalysisMedia(s)
	video, _, e := s.Put("two-people.mp4", bytes.NewReader(mediaFixture(t, "hevc.mp4")))
	if e != nil {
		t.Fatal(e)
	}
	if e = s.QueueAnalysis(video.ID); e != nil {
		t.Fatal(e)
	}
	if worked, e := s.ProcessAnalysis(context.Background()); !worked || e != nil {
		t.Fatal(e)
	}
	page, e = s.OrganizationPage("", "", "")
	if e != nil {
		t.Fatal(e)
	}
	videoNames := map[string]bool{}
	uncertain := 0
	for _, f := range page.Faces {
		if f.AssetID == video.ID {
			if f.SampleUS == nil {
				t.Fatal("sampled video position missing")
			}
			if f.Name == "" {
				uncertain++
				continue
			}
			if f.Source != "automatic" {
				t.Fatal("sampled video assignment provenance missing")
			}
			videoNames[f.Name] = true
		}
	}
	if videoNames["Alice"] || !videoNames["Bob"] || uncertain != 1 {
		t.Fatal("video must retain ambiguous corrected identity for review and assign unambiguous Bob")
	}
}
