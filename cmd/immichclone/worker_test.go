package main

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"immichclone/internal/library"
	"testing"
	"time"
)

func previewFixtureStore(t *testing.T) *library.Store {
	t.Helper()
	s, err := library.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { s.Close() })
	for i := 0; i < 10; i++ {
		im := image.NewRGBA(image.Rect(0, 0, 32, 24))
		im.SetRGBA(i, 0, color.RGBA{R: uint8(i + 1), A: 255})
		var b bytes.Buffer
		if err := png.Encode(&b, im); err != nil {
			t.Fatal(err)
		}
		if _, _, err := s.Put(fmt.Sprintf("worker-%02d.png", i), &b); err != nil {
			t.Fatal(err)
		}
	}
	return s
}

func startFixtureWorker(t *testing.T, s *library.Store, idle time.Duration) context.CancelFunc {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() { defer close(done); runWorker(ctx, s, idle) }()
	t.Cleanup(func() {
		cancel()
		select {
		case <-done:
		case <-time.After(time.Second):
			t.Error("worker did not stop promptly")
		}
	})
	return cancel
}

func waitFixturePreviews(t *testing.T, s *library.Store) {
	t.Helper()
	end := time.Now().Add(5 * time.Second)
	for time.Now().Before(end) {
		assets, err := s.List()
		if err != nil {
			t.Fatal(err)
		}
		ready := 0
		for _, a := range assets {
			if a.Preview == "failed" {
				t.Fatal("valid fixture preview failed")
			}
			if a.Preview == "ready" {
				ready++
			}
		}
		if ready == 10 {
			if err := s.Verify(); err != nil {
				t.Fatal(err)
			}
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatal("serial preview queue did not drain without a per-item polling delay")
}

func TestWorkerDrainsRealPreviewQueueBeforeIdle(t *testing.T) {
	s := previewFixtureStore(t)
	// No second idle wake is available: all ten actual PNG jobs must drain first.
	startFixtureWorker(t, s, time.Hour)
	waitFixturePreviews(t, s)
}

func TestWorkerHonorsCopyFirstAndResumesWithoutLosingJobs(t *testing.T) {
	s := previewFixtureStore(t)
	m, err := s.CreateImport([]library.ImportFile{{Name: "pending.bin", Size: 10}})
	if err != nil {
		t.Fatal(err)
	}
	if err = s.ControlImport(m.ID, "start"); err != nil {
		t.Fatal(err)
	}
	startFixtureWorker(t, s, 10*time.Millisecond)
	time.Sleep(50 * time.Millisecond)
	assets, err := s.List()
	if err != nil {
		t.Fatal(err)
	}
	for _, a := range assets {
		if a.Preview != "queued" {
			t.Fatal("preview started during active copy-first lease")
		}
	}
	if err = s.ControlImport(m.ID, "pause"); err != nil {
		t.Fatal(err)
	}
	waitFixturePreviews(t, s)
}
