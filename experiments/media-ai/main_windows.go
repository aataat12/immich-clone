// Actual dependency proof only; not linked into the app and no fixtures redistributed.
package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"golang.org/x/sys/windows/svc"
	"image"
	_ "image/png"
	"immichclone/internal/media"
	"immichclone/internal/recognition"
	"math"
	"os"
	"path/filepath"
	"time"
)

func fail(s string) error { return errors.New(s) }
func run(root string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	var hashes map[string]string
	b, e := os.ReadFile(filepath.Join(root, "fixture-hashes.json"))
	if e != nil {
		return nil, fail("fixture manifest unavailable")
	}
	if json.Unmarshal(b, &hashes) != nil {
		return nil, fail("fixture manifest invalid")
	}
	verify := func() error {
		for n, h := range hashes {
			if filepath.Base(n) != n {
				return fail("fixture name invalid")
			}
			b, e := os.ReadFile(filepath.Join(root, n))
			if e != nil {
				return fail("fixture unavailable")
			}
			sum := sha256.Sum256(b)
			if hex.EncodeToString(sum[:]) != h {
				return fail("fixture changed")
			}
		}
		return nil
	}
	if e = verify(); e != nil {
		return nil, e
	}
	decoder := media.New(filepath.Join(root, "media"))
	ai := recognition.New(filepath.Join(root, "ai"))
	if !ai.Available() {
		return nil, fail("AI provider unavailable")
	}
	analyze := func(img image.Image) (recognition.Result, error) {
		return ai.Analyze(ctx, img, filepath.Join(root, "scratch"))
	}
	reference := func(name string) (recognition.Result, error) {
		f, e := os.Open(filepath.Join(root, name))
		if e != nil {
			return recognition.Result{}, fail("reference unavailable")
		}
		defer f.Close()
		img, _, e := image.Decode(f)
		if e != nil {
			return recognition.Result{}, fail("reference decode failed")
		}
		return analyze(img)
	}
	minimum := 1.
	comparisons := 0
	for _, tc := range []struct {
		heic, png string
		w, h      int
	}{{"face.heic", "astronaut.png", 512, 512}, {"face-grid.heic", "face-grid-reference.png", 4032, 3024}} {
		baseline, e := reference(tc.png)
		if e != nil || len(baseline.Faces) != 1 {
			return nil, fail("reference face unavailable")
		}
		img, w, h, e := decoder.AnalysisFrame(ctx, filepath.Join(root, tc.heic))
		if e != nil {
			return nil, fail("HEIC analysis decode failed")
		}
		if w != tc.w || h != tc.h || img.Bounds().Dx() > 960 || img.Bounds().Dy() > 960 || img.Bounds().Dx() <= 320 {
			return nil, fail("HEIC analysis dimensions wrong")
		}
		result, e := analyze(img)
		if e != nil || len(result.Faces) != 1 {
			return nil, fail("HEIC face inference failed")
		}
		dot := 0.
		for i, v := range baseline.Faces[0].Embedding {
			dot += v * result.Faces[0].Embedding[i]
		}
		if dot < .95 {
			return nil, fail("HEIC embedding fidelity failed")
		}
		if dot < minimum {
			minimum = dot
		}
		for i, v := range baseline.Faces[0].Box {
			if math.Abs(v-result.Faces[0].Box[i]) > .03 {
				return nil, fail("HEIC face geometry drift")
			}
		}
		comparisons++
	}
	clip, e := decoder.VideoInfo(ctx, filepath.Join(root, "video.mp4"))
	if e != nil || clip.DurationUS < 3999000 || len(media.SampleTargets(clip.DurationUS)) != 12 {
		return nil, fail("video sampling plan failed")
	}
	for _, tc := range []struct {
		at, expect int64
		face       bool
	}{{0, 0, true}, {800000, 875000, true}, {2500000, 2500000, false}} {
		img, s, e := decoder.FrameAt(ctx, filepath.Join(root, "video.mp4"), tc.at)
		if e != nil {
			return nil, fail("video frame decode failed")
		}
		if s.PositionUS != tc.expect {
			return nil, fail("actual video PTS mismatch")
		}
		result, e := analyze(img)
		if e != nil {
			return nil, fail("video inference failed")
		}
		if tc.face {
			if len(result.Faces) < 1 {
				return nil, fail("video face not detected")
			}
		} else {
			cup := false
			for _, o := range result.Objects {
				if o.Class == 41 {
					cup = true
				}
			}
			if !cup {
				return nil, fail("video cup not detected")
			}
		}
	}
	for _, n := range []string{"hevc.mp4", "rotated-face.mov", "vfr.mp4", "short.mp4"} {
		path := filepath.Join(root, n)
		c, e := decoder.VideoInfo(ctx, path)
		if e != nil {
			return nil, fail("additional clip probe failed")
		}
		target := c.DurationUS / 3
		if n == "short.mp4" {
			target = c.DurationUS - 1
		}
		img, s, e := decoder.FrameAt(ctx, path, target)
		if e != nil || s.PositionUS < 0 || s.PositionUS >= c.DurationUS {
			return nil, fail("additional timestamped decode failed")
		}
		if _, e = analyze(img); e != nil {
			return nil, fail("additional clip inference failed")
		}
	}
	if _, e = decoder.VideoInfo(ctx, filepath.Join(root, "audio.mp4")); e == nil {
		return nil, fail("audio-only clip accepted")
	}
	if _, _, _, e = decoder.AnalysisFrame(ctx, filepath.Join(root, "bad.heic")); e == nil {
		return nil, fail("malformed HEIC accepted")
	}
	cancelled, cancelNow := context.WithCancel(ctx)
	cancelNow()
	if _, _, e = decoder.FrameAt(cancelled, filepath.Join(root, "video.mp4"), 0); e == nil {
		return nil, fail("cancelled frame work accepted")
	}
	if e = verify(); e != nil {
		return nil, e
	}
	return json.Marshal(map[string]any{"heic_reference_comparisons": comparisons, "minimum_heic_cosine": minimum, "known_video_pts_and_content": true, "hevc_rotated_vfr_short": true, "malformed_and_no_video_rejected": true, "cancellation": true, "originals_unchanged": true})
}

type service struct{ root string }

func (p service) Execute(_ []string, requests <-chan svc.ChangeRequest, status chan<- svc.Status) (bool, uint32) {
	status <- svc.Status{State: svc.StartPending, WaitHint: 240000}
	done := make(chan error, 1)
	go func() {
		b, e := run(p.root)
		if e == nil {
			e = os.WriteFile(filepath.Join(p.root, "passed.json"), b, 0600)
		}
		done <- e
	}()
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	checkpoint := uint32(1)
	for {
		select {
		case e := <-done:
			if e != nil {
				_ = os.WriteFile(filepath.Join(p.root, "failed.txt"), []byte(e.Error()), 0600)
				return false, 1
			}
			status <- svc.Status{State: svc.Running, Accepts: svc.AcceptStop | svc.AcceptShutdown}
			for r := range requests {
				switch r.Cmd {
				case svc.Interrogate:
					status <- r.CurrentStatus
				case svc.Stop, svc.Shutdown:
					return false, 0
				}
			}
			return false, 0
		case <-ticker.C:
			checkpoint++
			status <- svc.Status{State: svc.StartPending, CheckPoint: checkpoint, WaitHint: 240000}
		}
	}
}
func main() {
	root := flag.String("root", "", "disposable public fixture directory")
	flag.Parse()
	if !filepath.IsAbs(*root) {
		os.Exit(2)
	}
	if svc.Run("ImmichCloneMediaAIProbe", service{*root}) != nil {
		os.Exit(1)
	}
}
