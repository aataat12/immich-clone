package library

import (
	"context"
	"errors"
	bolt "go.etcd.io/bbolt"
	"image"
	"immichclone/internal/media"
	"immichclone/internal/recognition"
	"os"
	"path/filepath"
	"sort"
	"time"
)

const videoPolicy = "sampled-12-v1"

type VideoSample struct {
	Provider   string   `json:"provider"`
	Width      int      `json:"width"`
	Height     int      `json:"height"`
	PositionUS int64    `json:"position_us"`
	Objects    []string `json:"objects"`
}
type VideoProgress struct {
	Version    int           `json:"version"`
	Policy     string        `json:"policy"`
	DurationUS int64         `json:"duration_us"`
	Targets    []int64       `json:"targets"`
	Next       int           `json:"next"`
	Samples    []VideoSample `json:"samples"`
}

func isHEIC(m string) bool          { return m == "image/heic" || m == "image/heif" }
func isVideo(m string) bool         { return m == "video/mp4" || m == "video/quicktime" }
func supportsStillAI(m string) bool { return supportsImagePreview(m) || isHEIC(m) }
func (s *Store) SetAnalysisMedia(still func(context.Context, string) (image.Image, int, int, error), info func(context.Context, string) (media.Clip, error), frame func(context.Context, string, int64) (image.Image, media.Sample, error), available bool) {
	s.gate.Lock()
	defer s.gate.Unlock()
	s.analysisStill = still
	s.videoInfo = info
	s.videoFrame = frame
	s.mediaAnalysisAvailable.Store(available)
}
func (s *Store) mediaSlot(ctx context.Context) (func(), error) {
	select {
	case s.mediaAnalysisGate <- struct{}{}:
		return func() { <-s.mediaAnalysisGate }, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}
func (s *Store) stillMediaImage(ctx context.Context, path string) (image.Image, error) {
	s.gate.Lock()
	fn := s.analysisStill
	s.gate.Unlock()
	if fn == nil {
		return nil, media.ErrUnavailable
	}
	release, e := s.mediaSlot(ctx)
	if e != nil {
		return nil, e
	}
	defer release()
	img, _, _, e := fn(ctx, path)
	return img, e
}
func (s *Store) videoSource(ctx context.Context, id string, position int64) (image.Image, media.Sample, error) {
	a, e := s.Get(id)
	if e != nil {
		return nil, media.Sample{}, e
	}
	if !isVideo(a.MIME) {
		return nil, media.Sample{}, ErrOrganizationInput
	}
	s.gate.Lock()
	fn := s.videoFrame
	s.gate.Unlock()
	if fn == nil {
		return nil, media.Sample{}, media.ErrUnavailable
	}
	release, e := s.mediaSlot(ctx)
	if e != nil {
		return nil, media.Sample{}, e
	}
	defer release()
	hash, e := fileHashContext(ctx, s.Original(id))
	if e != nil || hash != id {
		return nil, media.Sample{}, errors.New("original integrity check failed")
	}
	return fn(ctx, s.Original(id), position)
}
func (s *Store) faceSource(ctx context.Context, f StoredFace) (image.Image, error) {
	if f.SampleUS == nil {
		return s.analysisImage(ctx, f.AssetID)
	}
	img, sample, e := s.videoSource(ctx, f.AssetID, *f.SampleUS)
	if e == nil && sample.PositionUS != *f.SampleUS {
		return nil, errors.New("saved video frame timestamp unavailable")
	}
	return img, e
}

// FaceContext reconstructs the saved sampled frame, without running AI again.
func (s *Store) FaceContext(ctx context.Context, id string) (image.Image, error) {
	var f StoredFace
	if !validID(id) {
		return nil, ErrOrganizationInput
	}
	if e := s.db.View(func(tx *bolt.Tx) error { return readJSON(tx.Bucket([]byte(facesBucket)), id, &f) }); e != nil {
		return nil, e
	}
	if !validFace(f, id) || f.SampleUS == nil {
		return nil, ErrOrganizationInput
	}
	bounded, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()
	return s.faceSource(bounded, f)
}
func validVideo(v *VideoProgress) bool {
	if v == nil || v.Version != 1 || v.Policy != videoPolicy || len(v.Targets) > 12 || len(v.Samples) > 12 || v.Next < 0 || v.Next > len(v.Targets) {
		return false
	}
	if len(v.Targets) == 0 {
		return v.Next == 0 && v.DurationUS == 0 && len(v.Samples) == 0
	}
	if v.DurationUS <= 0 || v.DurationUS > 86400*1e6 {
		return false
	}
	for i, t := range v.Targets {
		if t < 0 || t >= v.DurationUS || (i > 0 && t <= v.Targets[i-1]) {
			return false
		}
	}
	seen := map[int64]bool{}
	for _, sample := range v.Samples {
		if sample.PositionUS < 0 || sample.PositionUS >= v.DurationUS || seen[sample.PositionUS] || len(sample.Objects) > 80 {
			return false
		}
		seen[sample.PositionUS] = true
	}
	return true
}
func (s *Store) queueVideoLocked(asset Asset) error {
	if !s.mediaAnalysisAvailable.Load() {
		return media.ErrUnavailable
	}
	return s.db.Update(func(tx *bolt.Tx) error {
		var a Analysis
		e := readJSON(tx.Bucket([]byte(analysisBucket)), asset.ID, &a)
		if e != nil && !errors.Is(e, os.ErrNotExist) {
			return e
		}
		if a.Status == "ready" || a.Status == "queued" {
			return nil
		}
		if a.Video == nil {
			a = Analysis{AssetID: asset.ID, Provider: recognition.AnalysisID, Objects: []string{}, Video: &VideoProgress{Version: 1, Policy: videoPolicy, Targets: []int64{}, Samples: []VideoSample{}}}
		}
		if !validVideo(a.Video) {
			return ErrOrganizationInput
		}
		a.Status = "queued"
		a.Error = ""
		if e = writeJSON(tx.Bucket([]byte(analysisBucket)), asset.ID, a); e != nil {
			return e
		}
		return tx.Bucket([]byte(analysisQueue)).Put([]byte(asset.ID), []byte{1})
	})
}
func (s *Store) ControlVideoAnalysis(id, action string) error {
	s.gate.Lock()
	defer s.gate.Unlock()
	asset, e := s.Get(id)
	if e != nil {
		return e
	}
	if !isVideo(asset.MIME) {
		return ErrOrganizationInput
	}
	if action == "resume" {
		return s.queueVideoLocked(asset)
	}
	if action != "pause" && action != "cancel" {
		return ErrOrganizationInput
	}
	return s.db.Update(func(tx *bolt.Tx) error {
		var a Analysis
		if e := readJSON(tx.Bucket([]byte(analysisBucket)), id, &a); e != nil {
			return e
		}
		if !validVideo(a.Video) {
			return ErrOrganizationInput
		}
		if a.Status == "ready" {
			return nil
		}
		a.Status = "paused"
		if action == "cancel" {
			a.Status = "cancelled"
		}
		a.Error = ""
		if e := writeJSON(tx.Bucket([]byte(analysisBucket)), id, a); e != nil {
			return e
		}
		return tx.Bucket([]byte(analysisQueue)).Delete([]byte(id))
	})
}
func objectNames(result recognition.Result) []string {
	seen := map[string]bool{}
	for _, o := range result.Objects {
		if o.Class >= 0 && o.Class < len(recognition.Classes) {
			seen[recognition.Classes[o.Class]] = true
		}
	}
	out := make([]string, 0, len(seen))
	for n := range seen {
		out = append(out, n)
	}
	sort.Strings(out)
	return out
}
func (s *Store) processVideoAnalysis(parent context.Context, asset Asset, a Analysis, recognizer func(context.Context, image.Image, string) (recognition.Result, error)) error {
	if !validVideo(a.Video) {
		return ErrOrganizationInput
	}
	ctx, cancel := context.WithTimeout(parent, 75*time.Second)
	defer cancel()
	v := a.Video
	baseline := *v
	priorObjects := append([]string(nil), a.Objects...)
	oldNext := v.Next
	var workErr error
	if len(v.Targets) == 0 {
		s.gate.Lock()
		info := s.videoInfo
		s.gate.Unlock()
		if info == nil {
			workErr = media.ErrUnavailable
		} else {
			hash, e := fileHashContext(ctx, s.Original(asset.ID))
			if e != nil || hash != asset.ID {
				workErr = errors.New("original integrity check failed")
			} else {
				clip, e := info(ctx, s.Original(asset.ID))
				workErr = e
				if e == nil {
					v.DurationUS = clip.DurationUS
					v.Targets = media.SampleTargets(clip.DurationUS)
					if len(v.Targets) == 0 {
						workErr = errors.New("video duration unsupported")
					}
				}
			}
		}
	}
	// Keep successfully probed coverage when the first inference fails. Only a
	// completed frame advances Next; an unprobed/invalid clip still has no plan.
	if workErr == nil && len(baseline.Targets) == 0 && len(v.Targets) > 0 {
		baseline = *v
	}
	var faces []StoredFace
	committed := false
	defer func() {
		if !committed {
			for _, f := range faces {
				_ = os.Remove(filepath.Join(s.root, "faces", f.ID+".png"))
			}
		}
	}()
	if workErr == nil && v.Next < len(v.Targets) {
		img, sample, e := s.videoSource(ctx, asset.ID, v.Targets[v.Next])
		workErr = e
		if e == nil && (sample.PositionUS < 0 || sample.PositionUS >= v.DurationUS || img == nil) {
			workErr = errors.New("invalid decoded video sample")
			e = workErr
		}
		if e == nil {
			a.Provider = recognition.AnalysisID
			duplicate := false
			for _, old := range v.Samples {
				if old.PositionUS == sample.PositionUS {
					duplicate = true
					break
				}
			}
			if !duplicate {
				var result recognition.Result
				if recognizer == nil {
					workErr = recognition.ErrUnavailable
				} else {
					result, workErr = recognizer(ctx, img, filepath.Join(s.root, "staging"))
				}
				if workErr == nil {
					workErr = recognition.Validate(result, 1, 1)
				}
				if workErr == nil {
					for _, detected := range result.Faces {
						key, e := newFaceID()
						if e != nil {
							workErr = e
							break
						}
						position := sample.PositionUS
						f := StoredFace{ID: key, AssetID: asset.ID, Box: detected.Box, Embedding: detected.Embedding, Provider: a.Provider, SampleUS: &position}
						if !validFace(f, key) {
							workErr = ErrOrganizationInput
							break
						}
						if e = s.publishFaceCrop(f, img); e != nil {
							workErr = e
							break
						}
						faces = append(faces, f)
					}
				}
				if workErr == nil {
					names := objectNames(result)
					v.Samples = append(v.Samples, VideoSample{PositionUS: sample.PositionUS, Provider: a.Provider, Width: img.Bounds().Dx(), Height: img.Bounds().Dy(), Objects: names})
					all := map[string]bool{}
					for _, name := range a.Objects {
						all[name] = true
					}
					for _, name := range names {
						all[name] = true
					}
					a.Objects = []string{}
					for name := range all {
						a.Objects = append(a.Objects, name)
					}
					sort.Strings(a.Objects)
				}
			}
			if workErr == nil {
				v.Next++
			}
		}
	}
	if parent.Err() != nil {
		return parent.Err()
	}
	if ctx.Err() != nil {
		workErr = errors.New("video sample analysis time limit exceeded")
	}
	if workErr != nil {
		a.Video = &baseline
		a.Objects = priorObjects
		a.Status = "failed"
		a.Error = workErr.Error()
		if len(a.Error) > 256 {
			a.Error = "video analysis failed"
		}
	} else {
		a.Error = ""
		a.Status = "queued"
		if v.Next == len(v.Targets) {
			a.Status = "ready"
		}
	}
	s.gate.Lock()
	defer s.gate.Unlock()
	if parent.Err() != nil {
		return parent.Err()
	}
	e := s.db.Update(func(tx *bolt.Tx) error {
		var current Analysis
		if e := readJSON(tx.Bucket([]byte(analysisBucket)), asset.ID, &current); e != nil {
			return e
		}
		if current.Video == nil || current.Video.Next != oldNext {
			return errors.New("video progress changed during processing")
		}
		if a.Status != "ready" && (current.Status == "paused" || current.Status == "cancelled") {
			a.Status = current.Status
		}
		if workErr == nil {
			for _, f := range faces {
				if e := writeJSON(tx.Bucket([]byte(facesBucket)), f.ID, f); e != nil {
					return e
				}
			}
		}
		if workErr == nil {
			if e := applyAutomaticNames(tx, faces, productionAutomaticNamePolicy()); e != nil { return e }
		}
		if e := writeJSON(tx.Bucket([]byte(analysisBucket)), asset.ID, a); e != nil {
			return e
		}
		if a.Status != "queued" {
			return tx.Bucket([]byte(analysisQueue)).Delete([]byte(asset.ID))
		}
		return nil
	})
	committed = e == nil && workErr == nil
	return e
}
