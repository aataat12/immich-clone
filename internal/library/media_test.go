package library

import (
	"bytes"
	"context"
	"encoding/base64"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestCancelledMediaJobSurvivesRestartAndRestore(t *testing.T) {
	encoded, e := os.ReadFile("../../experiments/media-worker/fixtures/synthetic.heic.b64")
	if e != nil {
		t.Fatal(e)
	}
	original, e := base64.StdEncoding.DecodeString(string(encoded))
	if e != nil {
		t.Fatal(e)
	}
	s, root := openTest(t)
	a, _, e := s.Put("synthetic.heic", bytes.NewReader(original))
	if e != nil {
		t.Fatal(e)
	}
	if a.MIME != "image/heic" {
		t.Fatal("HEIC signature not recognized")
	}
	// Missing decoder is a visible job failure, while original preservation works.
	if _, e = s.ProcessOne(); e != nil {
		t.Fatal(e)
	}
	a, _ = s.Get(a.ID)
	if a.Preview != "failed" {
		t.Fatal("missing optional provider reported success")
	}
	if e = s.Verify(); e != nil {
		t.Fatal(e)
	}
	if e = s.Retry(a.ID); e != nil {
		t.Fatal(e)
	}
	entered := make(chan struct{})
	s.SetMediaPreview(func(ctx context.Context, _ string) (image.Image, int, int, error) {
		close(entered)
		<-ctx.Done()
		return nil, 0, 0, ctx.Err()
	})
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { _, e := s.ProcessOneContext(ctx); done <- e }()
	select {
	case <-entered:
	case <-time.After(5 * time.Second):
		t.Fatal("worker did not start")
	}
	cancel()
	if e = <-done; e != context.Canceled {
		t.Fatal(e)
	}
	a, _ = s.Get(a.ID)
	if a.Preview != "queued" {
		t.Fatal("cancelled job lost its durable queue record")
	}
	s.Close()
	s, e = Open(root)
	if e != nil {
		t.Fatal(e)
	}
	defer s.Close()
	render := func(context.Context, string) (image.Image, int, int, error) {
		img := image.NewRGBA(image.Rect(0, 0, 320, 240))
		for y := 0; y < 240; y++ {
			for x := 0; x < 320; x++ {
				img.Set(x, y, color.RGBA{255, 0, 0, 255})
			}
		}
		return img, 4032, 3024, nil
	}
	s.SetMediaPreview(render)
	if _, e = s.ProcessOne(); e != nil {
		t.Fatal(e)
	}
	a, _ = s.Get(a.ID)
	if a.Preview != "ready" || a.Width != 4032 || a.Height != 3024 {
		t.Fatal("resumed external job failed")
	}
	f, e := os.Open(s.PreviewPath(a.ID))
	if e != nil {
		t.Fatal(e)
	}
	preview, e := png.Decode(f)
	f.Close()
	if e != nil {
		t.Fatal(e)
	}
	r, _, _, _ := preview.At(319, 239).RGBA()
	if r != 65535 {
		t.Fatal("source dimensions incorrectly applied to thumbnail pixels")
	}
	var archive bytes.Buffer
	if e = s.Backup(&archive); e != nil {
		t.Fatal(e)
	}
	zip := filepath.Join(t.TempDir(), "backup.zip")
	if e = os.WriteFile(zip, archive.Bytes(), 0600); e != nil {
		t.Fatal(e)
	}
	target := filepath.Join(t.TempDir(), "restore")
	if e = Restore(zip, target); e != nil {
		t.Fatal(e)
	}
	restored, e := Open(target)
	if e != nil {
		t.Fatal(e)
	}
	defer restored.Close()
	restored.SetMediaPreview(render)
	if _, e = restored.ProcessOne(); e != nil {
		t.Fatal(e)
	}
	ra, _ := restored.Get(a.ID)
	if ra.Preview != "ready" || ra.MIME != "image/heic" {
		t.Fatal("restored external job failed")
	}
	if e = restored.Verify(); e != nil {
		t.Fatal(e)
	}
}
