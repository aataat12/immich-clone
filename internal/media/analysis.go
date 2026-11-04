package media

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"math"
	"path/filepath"
	"strconv"
	"time"
)

// Clip times are source-relative microseconds, not assumed constant-rate frame numbers.
type Clip struct {
	DurationUS int64 `json:"duration_us"`
	StartUS    int64 `json:"start_us"`
	Width      int   `json:"width"`
	Height     int   `json:"height"`
}
type Sample struct {
	PositionUS int64 `json:"position_us"`
	Width      int   `json:"width"`
	Height     int   `json:"height"`
}

func parseSeconds(s string) (int64, error) {
	f, e := strconv.ParseFloat(s, 64)
	if e != nil || math.IsNaN(f) || math.IsInf(f, 0) || math.Abs(f) > 7*86400 {
		return 0, errors.New("unsupported media timestamp")
	}
	return int64(math.Round(f * 1e6)), nil
}
func parseClip(b []byte) (Clip, error) {
	var v struct {
		Streams []struct {
			CodecType string `json:"codec_type"`
			Duration  string `json:"duration"`
			Start     string `json:"start_time"`
		}
		Format struct {
			Duration string `json:"duration"`
			Start    string `json:"start_time"`
		}
	}
	var c Clip
	if json.Unmarshal(b, &v) != nil {
		return c, errors.New("video metadata unavailable")
	}
	w, h, selector, e := dimensions(b)
	if e != nil {
		return c, e
	}
	if len(selector) > 0 && selector[0] == '[' {
		return c, errors.New("video stream group unsupported")
	}
	c.Width, c.Height = w, h
	for _, s := range v.Streams {
		if s.CodecType == "video" {
			c.DurationUS, e = parseSeconds(s.Duration)
			if e != nil || c.DurationUS <= 0 {
				c.DurationUS, e = parseSeconds(v.Format.Duration)
			}
			c.StartUS, _ = parseSeconds(s.Start)
			if s.Start == "" || s.Start == "N/A" {
				c.StartUS, _ = parseSeconds(v.Format.Start)
			}
			if e != nil || c.DurationUS <= 0 || c.DurationUS > 86400*1e6 {
				return Clip{}, errors.New("video duration unavailable or exceeds 24-hour analysis limit")
			}
			return c, nil
		}
	}
	return c, errors.New("video stream unavailable")
}
func (p *Provider) clip(ctx context.Context, path string) (Clip, []byte, error) {
	if !p.available {
		return Clip{}, nil, ErrUnavailable
	}
	r, e := Run(ctx, filepath.Join(p.dir, "ffprobe.exe"), []string{"-v", "error", "-max_alloc", "67108864", "-threads", "1", "-probesize", "8388608", "-analyzeduration", "5000000", "-protocol_whitelist", "file,pipe", "-show_streams", "-show_stream_groups", "-show_format", "-of", "json", path}, nil)
	if e != nil {
		return Clip{}, nil, errors.New("video probe failed")
	}
	c, e := parseClip(r.Output)
	return c, r.Output, e
}
func (p *Provider) VideoInfo(parent context.Context, path string) (Clip, error) {
	ctx, cancel := context.WithTimeout(parent, 30*time.Second)
	defer cancel()
	c, _, e := p.clip(ctx, path)
	return c, e
}

// SampleTargets specifies at most twelve requested positions. Actual decoded PTS
// may coincide on short/VFR clips; callers deduplicate returned PositionUS.
func SampleTargets(durationUS int64) []int64 {
	if durationUS <= 0 || durationUS > 86400*1e6 {
		return nil
	}
	n := 12
	if durationUS < int64(n) {
		n = int(durationUS)
	}
	out := make([]int64, n)
	for i := range out {
		out[i] = durationUS * int64(i) / int64(n)
	}
	return out
}
func chooseSample(b []byte, target, duration, start int64) (int64, error) {
	var v struct {
		Frames []struct {
			Time string `json:"best_effort_timestamp_time"`
		} `json:"frames"`
	}
	if json.Unmarshal(b, &v) != nil || len(v.Frames) > 10000 {
		return 0, errors.New("video frame timestamps unavailable")
	}
	bestAfter, bestBefore := int64(math.MaxInt64), int64(math.MinInt64)
	for _, f := range v.Frames {
		t, e := parseSeconds(f.Time)
		if e != nil {
			return 0, e
		}
		t -= start
		if t < 0 || t >= duration {
			continue
		}
		if t >= target && t < bestAfter {
			bestAfter = t
		}
		if t <= target && t > bestBefore {
			bestBefore = t
		}
	}
	if bestAfter != math.MaxInt64 && bestAfter <= target+1e6 {
		return bestAfter, nil
	}
	if bestBefore != math.MinInt64 {
		return bestBefore, nil
	}
	return 0, errors.New("no timestamped video frame available")
}

// FrameAt resolves an actual source PTS first, then accurately seeks that frame.
func (p *Provider) FrameAt(parent context.Context, path string, targetUS int64) (image.Image, Sample, error) {
	ctx, cancel := context.WithTimeout(parent, 45*time.Second)
	defer cancel()
	c, b, e := p.clip(ctx, path)
	if e != nil {
		return nil, Sample{}, e
	}
	if targetUS < 0 || targetUS >= c.DurationUS {
		return nil, Sample{}, errors.New("video sample outside duration")
	}
	var metadata probeInfo
	_ = json.Unmarshal(b, &metadata)
	selector := ""
	for _, s := range metadata.Streams {
		if s.CodecType == "video" {
			selector = strconv.Itoa(s.Index)
			break
		}
	}
	raw := targetUS + c.StartUS
	interval := fmt.Sprintf("%.6f%%%.6f", float64(raw)/1e6, float64(raw+1e6)/1e6)
	r, e := Run(ctx, filepath.Join(p.dir, "ffprobe.exe"), []string{"-v", "error", "-max_alloc", "67108864", "-threads", "1", "-probesize", "8388608", "-analyzeduration", "5000000", "-protocol_whitelist", "file,pipe", "-select_streams", selector, "-read_intervals", interval, "-show_frames", "-show_entries", "frame=best_effort_timestamp_time", "-of", "json", path}, nil)
	if e != nil {
		return nil, Sample{}, errors.New("video timestamp probe failed")
	}
	position, e := chooseSample(r.Output, targetUS, c.DurationUS, c.StartUS)
	if e != nil {
		return nil, Sample{}, e
	}
	seek := position + c.StartUS
	img, w, h, e := p.decodeFrame(ctx, path, 960, &seek, b)
	return img, Sample{PositionUS: position, Width: w, Height: h}, e
}
