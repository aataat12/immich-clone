package media

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"image"
	"image/color"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestActualVideoMetadataCompatibilityMatrix(t *testing.T) {
	root := os.Getenv("IMMICH_COMPAT_FIXTURES")
	provider := os.Getenv("IMMICH_COMPAT_PROVIDER")
	if root == "" || provider == "" {
		t.Skip("requires actual pinned provider and compatibility fixtures")
	}
	p := New(provider)
	if !p.Available() {
		t.Fatal("pinned provider unavailable")
	}
	raw, e := os.ReadFile(filepath.Join(root, "video-hashes.json"))
	if e != nil {
		t.Fatal(e)
	}
	var hashes map[string]string
	if e = json.Unmarshal(raw, &hashes); e != nil {
		t.Fatal(e)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	for _, tc := range []struct {
		name  string
		start int64
		index int
	}{{"audio-first.mp4", 0, 1}, {"nonzero-start.mov", 5000000, 0}} {
		t.Run(tc.name, func(t *testing.T) {
			path := filepath.Join(root, tc.name)
			before, e := os.ReadFile(path)
			if e != nil {
				t.Fatal(e)
			}
			sum := sha256.Sum256(before)
			if hex.EncodeToString(sum[:]) != hashes[tc.name] {
				t.Fatal("fixture hash mismatch")
			}
			started := time.Now()
			clip, metadata, e := p.clip(ctx, path)
			if e != nil || clip.Width != 160 || clip.Height != 120 || clip.StartUS != tc.start || clip.DurationUS < 900000 || clip.DurationUS > 1300000 {
				t.Fatal("actual clip metadata", clip, e)
			}
			var parsed probeInfo
			if e = json.Unmarshal(metadata, &parsed); e != nil {
				t.Fatal(e)
			}
			_, _, selector, e := dimensions(metadata)
			want := "0:0"
			if tc.index == 1 {
				want = "0:1"
			}
			if e != nil || selector != want {
				t.Fatal("actual stream selection", selector, e)
			}
			check := func(img image.Image) {
				t.Helper()
				if img == nil || img.Bounds().Dx() > 960 || img.Bounds().Dy() > 960 {
					t.Fatal("unbounded frame")
				}
				b := img.Bounds()
				c := color.NRGBAModel.Convert(img.At(b.Min.X+b.Dx()/2, b.Min.Y+b.Dy()/2)).(color.NRGBA)
				if c.R < 220 || c.G > 30 || c.B > 30 {
					t.Fatal("selected timestamp did not decode expected red frame")
				}
			}
			thumbnail, w, h, e := p.Thumbnail(ctx, path)
			if e != nil || w != 160 || h != 120 {
				t.Fatal("actual thumbnail", w, h, e)
			}
			check(thumbnail)
			frame, sample, e := p.FrameAt(ctx, path, 0)
			if e != nil || sample.PositionUS < 0 || sample.PositionUS > 50000 || sample.Width != 160 || sample.Height != 120 {
				t.Fatal("source-relative sample", sample, e)
			}
			check(frame)
			after, e := os.ReadFile(path)
			if e != nil {
				t.Fatal(e)
			}
			sum = sha256.Sum256(after)
			if hex.EncodeToString(sum[:]) != hashes[tc.name] {
				t.Fatal("original changed")
			}
			t.Logf("PASS: stream index %d, source start %d us, source-relative frame %d us, bounded thumbnail/frame and unchanged original; %s", tc.index, clip.StartUS, sample.PositionUS, time.Since(started))
		})
	}
}
