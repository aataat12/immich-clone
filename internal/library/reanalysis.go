package library

import (
	"context"
	"errors"
	bolt "go.etcd.io/bbolt"
	"image"
	"immichclone/internal/diagnostics"
	"immichclone/internal/media"
	"immichclone/internal/recognition"
	"os"
	"path/filepath"
	"sort"
	"time"
)

const reanalysisBucket = "reanalysis_v1"
const reanalysisQueue = "reanalysis_queue_v1"

type proposedFace struct {
	recognition.Face
	Provider string `json:"provider"`
	SampleUS *int64 `json:"sample_us,omitempty"`
}
type reanalysisJob struct {
	Version       int            `json:"version"`
	AssetID       string         `json:"asset_id"`
	Token         string         `json:"token"`
	Status        string         `json:"status"`
	Provider      string         `json:"provider"`
	MediaProvider string         `json:"media_provider"`
	Error         string         `json:"error,omitempty"`
	Faces         []proposedFace `json:"faces"`
	Objects       []string       `json:"objects"`
	Video         *VideoProgress `json:"video,omitempty"`
	Added         int            `json:"added"`
	Retained      int            `json:"retained"`
	Skipped       int            `json:"skipped"`
}
type ReanalysisView struct {
	Status   string   `json:"status"`
	Error    string   `json:"error,omitempty"`
	Faces    int      `json:"faces"`
	Objects  []string `json:"objects"`
	Next     int      `json:"next"`
	Total    int      `json:"total"`
	Added    int      `json:"added"`
	Retained int      `json:"retained"`
	Skipped  int      `json:"skipped"`
}

func validReanalysis(j reanalysisJob, id string) bool {
	if j.Version != 1 || j.AssetID != id || !validID(id) || !validID(j.Token) || len(j.Faces) > 768 || len(j.Objects) > 80 {
		return false
	}
	switch j.Status {
	case "queued", "paused", "ready", "failed", "cancelled", "applied":
	default:
		return false
	}
	if j.Added < 0 || j.Retained < 0 || j.Skipped < 0 {
		return false
	}
	if j.Video != nil && ((j.Status == "ready" && j.Video.Next != len(j.Video.Targets)) || ((j.Status == "queued" || j.Status == "paused") && j.Video.Next >= len(j.Video.Targets))) {
		return false
	}
	if j.Video != nil && !validVideo(j.Video) {
		return false
	}
	for _, f := range j.Faces {
		if (j.Video == nil) != (f.SampleUS == nil) {
			return false
		}
		if recognition.Validate(recognition.Result{Protocol: 1, Faces: []recognition.Face{f.Face}}, 1, 1) != nil {
			return false
		}
		if f.SampleUS != nil && (*f.SampleUS < 0 || *f.SampleUS >= 86400*1e6) {
			return false
		}
	}
	return true
}
func initReanalysis(tx *bolt.Tx) error {
	if _, e := tx.CreateBucketIfNotExists([]byte(reanalysisBucket)); e != nil {
		return e
	}
	if tx.Bucket([]byte(reanalysisQueue)) != nil {
		if e := tx.DeleteBucket([]byte(reanalysisQueue)); e != nil {
			return e
		}
	}
	q, e := tx.CreateBucket([]byte(reanalysisQueue))
	if e != nil {
		return e
	}
	b := tx.Bucket([]byte(reanalysisBucket))

	if e = b.ForEach(func(k, v []byte) error {
		var j reanalysisJob
		if e := readJSON(b, string(k), &j); e != nil {
			return e
		}
		if !validReanalysis(j, string(k)) {
			return ErrOrganizationInput
		}
		if j.Status == "queued" {
			return q.Put(k, []byte{1})
		}
		return nil
	}); e != nil {
		return e
	}
	return nil
}
func reanalysisView(tx *bolt.Tx, id string) (*ReanalysisView, error) {
	var j reanalysisJob
	e := readJSON(tx.Bucket([]byte(reanalysisBucket)), id, &j)
	if errors.Is(e, os.ErrNotExist) {
		return nil, nil
	}
	if e != nil {
		return nil, e
	}
	if !validReanalysis(j, id) {
		return nil, ErrOrganizationInput
	}
	v := &ReanalysisView{Status: j.Status, Error: j.Error, Faces: len(j.Faces), Objects: j.Objects, Total: 1, Added: j.Added, Retained: j.Retained, Skipped: j.Skipped}
	if j.Video != nil {
		v.Next = j.Video.Next
		v.Total = len(j.Video.Targets)
	} else if j.Status == "ready" || j.Status == "applied" {
		v.Next = 1
	}
	if j.Status == "ready" {
		v.Added = 0
		v.Skipped = 0
		var existing []StoredFace
		fb := tx.Bucket([]byte(facesBucket))
		if e := fb.ForEach(func(k, b []byte) error {
			var f StoredFace
			if e := readJSON(fb, string(k), &f); e != nil {
				return e
			}
			if f.AssetID == id {
				existing = append(existing, f)
			}
			return nil
		}); e != nil {
			return nil, e
		}
		v.Retained = len(existing)
		for _, p := range j.Faces {
			skip := false
			for _, f := range existing {
				if sameSample(f.SampleUS, p.SampleUS) && overlaps(f.Box, p.Box) {
					skip = true
					break
				}
			}
			if skip {
				v.Skipped++
			} else {
				v.Added++
				existing = append(existing, StoredFace{Box: p.Box, SampleUS: p.SampleUS})
			}
		}
	}
	return v, nil
}
func (s *Store) ControlReanalysis(id, action string) error {
	s.gate.Lock()
	defer s.gate.Unlock()
	a, e := s.Get(id)
	if e != nil {
		return e
	}
	if !(supportsStillAI(a.MIME) || isVideo(a.MIME)) || supportsStillAI(a.MIME) && a.Size > 64<<20 {
		return ErrOrganizationInput
	}
	return s.db.Update(func(tx *bolt.Tx) error {
		b, q := tx.Bucket([]byte(reanalysisBucket)), tx.Bucket([]byte(reanalysisQueue))
		var j reanalysisJob
		e := readJSON(b, id, &j)
		if e != nil && !errors.Is(e, os.ErrNotExist) {
			return e
		}
		if e == nil && !validReanalysis(j, id) {
			return ErrOrganizationInput
		}
		if action == "start" {
			var prior Analysis
			if e = readJSON(tx.Bucket([]byte(analysisBucket)), id, &prior); e != nil {
				return e
			}
			if prior.Status != "ready" || a.Hidden {
				return ErrOrganizationInput
			}
			if j.Status == "queued" || j.Status == "paused" || j.Status == "ready" {
				return ErrOrganizationInput
			}
			active := 0
			if e = b.ForEach(func(k, v []byte) error {
				var job reanalysisJob
				if e := readJSON(b, string(k), &job); e != nil {
					return e
				}
				if job.Status != "applied" && job.Status != "cancelled" {
					active++
				}
				return nil
			}); e != nil {
				return e
			}
			if active >= 128 && (j.Status == "" || j.Status == "applied" || j.Status == "cancelled") {
				return errors.New("reanalysis proposal limit reached")
			}
			token, e := newFaceID()
			if e != nil {
				return e
			}
			j = reanalysisJob{Version: 1, AssetID: id, Token: token, Status: "queued", Provider: recognition.AnalysisID, MediaProvider: media.ProviderID, Faces: []proposedFace{}, Objects: []string{}}
		} else {
			if e != nil {
				return e
			}
			switch action {
			case "pause":
				if j.Status != "queued" {
					return ErrOrganizationInput
				}
				j.Status = "paused"
			case "resume":
				if j.Status != "paused" && j.Status != "failed" {
					return ErrOrganizationInput
				}
				j.Status = "queued"
				j.Error = ""
			case "discard":
				if j.Status == "applied" {
					return ErrOrganizationInput
				}
				j.Status = "cancelled"
				j.Faces = nil
				j.Objects = nil
				j.Video = nil
				j.Error = ""
			default:
				return ErrOrganizationInput
			}
		}
		if j.Status == "queued" {
			if !s.recognitionAvailable.Load() {
				return recognition.ErrUnavailable
			}
			if (isVideo(a.MIME) || isHEIC(a.MIME)) && !s.mediaAnalysisAvailable.Load() {
				return media.ErrUnavailable
			}
			if e = q.Put([]byte(id), []byte{1}); e != nil {
				return e
			}
		} else {
			if e = q.Delete([]byte(id)); e != nil {
				return e
			}
		}
		return writeJSON(b, id, j)
	})
}

// Called with analysisGate held, only after normal analysis/import priority.
func (s *Store) processReanalysis(parent context.Context) (bool, error) {
	s.gate.Lock()
	var j reanalysisJob
	e := s.db.View(func(tx *bolt.Tx) error {
		k, _ := tx.Bucket([]byte(reanalysisQueue)).Cursor().First()
		if k == nil {
			return nil
		}
		return readJSON(tx.Bucket([]byte(reanalysisBucket)), string(k), &j)
	})
	baseline := j
	baseline.Faces = append([]proposedFace(nil), j.Faces...)
	baseline.Objects = append([]string(nil), j.Objects...)
	if j.Video != nil {
		v := *j.Video
		v.Targets = append([]int64(nil), v.Targets...)
		v.Samples = append([]VideoSample(nil), v.Samples...)
		baseline.Video = &v
	}
	fn := s.recognizer
	available := s.recognitionAvailable.Load()
	s.gate.Unlock()
	if e != nil || j.AssetID == "" {
		return false, e
	}
	if !available {
		return false, nil
	}
	if !validReanalysis(j, j.AssetID) || j.Status != "queued" {
		return true, ErrOrganizationInput
	}
	a, e := s.Get(j.AssetID)
	if e != nil {
		return true, e
	}
	ctx, cancel := context.WithTimeout(parent, 75*time.Second)
	defer cancel()
	var img image.Image
	var sample media.Sample
	oldNext := 0
	if isVideo(a.MIME) {
		if j.Video == nil {
			s.gate.Lock()
			info := s.videoInfo
			s.gate.Unlock()
			if info == nil {
				e = media.ErrUnavailable
			} else {
				var c media.Clip
				c, e = info(ctx, s.Original(a.ID))
				if e == nil {
					j.Video = &VideoProgress{Version: 1, Policy: videoPolicy, DurationUS: c.DurationUS, Targets: media.SampleTargets(c.DurationUS), Samples: []VideoSample{}}
					if !validVideo(j.Video) || len(j.Video.Targets) == 0 {
						e = ErrOrganizationInput
					}
				}
			}
		}
		if j.Video != nil {
			oldNext = j.Video.Next
		}
		if e == nil && j.Video.Next < len(j.Video.Targets) {
			img, sample, e = s.videoSource(ctx, a.ID, j.Video.Targets[j.Video.Next])
		}
	} else {
		img, e = s.analysisImage(ctx, a.ID)
	}
	if e == nil && (img == nil || j.Video != nil && (sample.PositionUS < 0 || sample.PositionUS >= j.Video.DurationUS)) {
		e = ErrOrganizationInput
	}
	duplicate := false
	if j.Video != nil {
		for _, v := range j.Video.Samples {
			if v.PositionUS == sample.PositionUS {
				duplicate = true
			}
		}
	}
	if e == nil && (!isVideo(a.MIME) || !duplicate) {
		var result recognition.Result
		if fn == nil {
			e = recognition.ErrUnavailable
		} else {
			result, e = fn(ctx, img, filepath.Join(s.root, "staging"))
		}
		if e == nil {
			e = recognition.Validate(result, 1, 1)
		}
		if e == nil {
			j.Provider = recognition.AnalysisID
			for _, f := range result.Faces {
				p := proposedFace{Face: f, Provider: j.Provider}
				if j.Video != nil {
					t := sample.PositionUS
					p.SampleUS = &t
				}
				j.Faces = append(j.Faces, p)
			}
			names := objectNames(result)
			set := map[string]bool{}
			for _, n := range append(j.Objects, names...) {
				set[n] = true
			}
			j.Objects = []string{}
			for n := range set {
				j.Objects = append(j.Objects, n)
			}
			sort.Strings(j.Objects)
			if j.Video != nil {
				j.Video.Samples = append(j.Video.Samples, VideoSample{Provider: j.Provider, Width: img.Bounds().Dx(), Height: img.Bounds().Dy(), PositionUS: sample.PositionUS, Objects: names})
			}
		}
	}
	if parent.Err() != nil {
		return true, parent.Err()
	}
	if ctx.Err() != nil {
		e = ctx.Err()
	}
	if e == nil {
		if j.Video != nil {
			j.Video.Next++
			if j.Video.Next == len(j.Video.Targets) {
				j.Status = "ready"
			}
		} else {
			j.Status = "ready"
		}
		j.Error = ""
	} else {
		if baseline.Video == nil && j.Video != nil && validVideo(j.Video) && len(j.Video.Targets) > 0 {
			v := *j.Video
			v.Next = 0
			v.Samples = []VideoSample{}
			baseline.Video = &v
		}
		j = baseline
		diagnostics.Record("recognition", "ai_reanalysis", e, 0, 0, 0)
		j.Status = "failed"
		j.Error = "Reanalysis failed; existing results retained. Check diagnostics before retrying."
	}
	if !validReanalysis(j, j.AssetID) {
		return true, ErrOrganizationInput
	}
	s.gate.Lock()
	defer s.gate.Unlock()
	return true, s.db.Update(func(tx *bolt.Tx) error {
		b, q := tx.Bucket([]byte(reanalysisBucket)), tx.Bucket([]byte(reanalysisQueue))
		var current reanalysisJob
		if e := readJSON(b, j.AssetID, &current); e != nil {
			return e
		}
		if current.Token != j.Token || current.Status == "cancelled" {
			return nil
		}
		if current.Video != nil && current.Video.Next != oldNext {
			return ErrOrganizationInput
		}
		if current.Status == "paused" && j.Status != "ready" {
			j.Status = "paused"
		}
		if parent.Err() != nil {
			return parent.Err()
		}
		if e := writeJSON(b, j.AssetID, j); e != nil {
			return e
		}
		if j.Status != "queued" {
			return q.Delete([]byte(j.AssetID))
		}
		return nil
	})
}
func sameSample(a, b *int64) bool { return a == nil && b == nil || a != nil && b != nil && *a == *b }
func overlaps(a, b []float64) bool {
	return len(a) == 4 && len(b) == 4 && a[0] < b[0]+b[2] && b[0] < a[0]+a[2] && a[1] < b[1]+b[3] && b[1] < a[1]+a[3]
}
func (s *Store) ApplyReanalysis(parent context.Context, id string) error {
	var requested reanalysisJob
	if e := s.db.View(func(tx *bolt.Tx) error { return readJSON(tx.Bucket([]byte(reanalysisBucket)), id, &requested) }); e != nil {
		return e
	}
	// Hash the immutable source before the short atomic metadata transaction.
	ctx, cancel := context.WithTimeout(parent, 75*time.Second)
	defer cancel()
	hash, e := fileHashContext(ctx, s.Original(id))
	if e != nil || hash != id {
		return errors.New("original integrity check failed")
	}
	s.gate.Lock()
	defer s.gate.Unlock()
	return s.db.Update(func(tx *bolt.Tx) error {
		if e := ctx.Err(); e != nil {
			return e
		}
		var j reanalysisJob
		b := tx.Bucket([]byte(reanalysisBucket))
		if e := readJSON(b, id, &j); e != nil {
			return e
		}
		if !validReanalysis(j, id) || j.Status != "ready" || j.Token != requested.Token {
			return ErrOrganizationInput
		}
		if j.MediaProvider != media.ProviderID {
			return errors.New("decoder changed; discard and prepare a new proposal")
		}
		var asset Asset
		if e := readJSON(tx.Bucket([]byte("assets")), id, &asset); e != nil {
			return e
		}
		if isVideo(asset.MIME) != (j.Video != nil) {
			return ErrOrganizationInput
		}
		j.Added = 0
		j.Skipped = 0
		var a Analysis
		if e := readJSON(tx.Bucket([]byte(analysisBucket)), id, &a); e != nil {
			return e
		}
		if a.Status != "ready" {
			return ErrOrganizationInput
		}
		var old []StoredFace
		fb := tx.Bucket([]byte(facesBucket))
		if e := fb.ForEach(func(k, v []byte) error {
			var f StoredFace
			if e := readJSON(fb, string(k), &f); e != nil {
				return e
			}
			if f.AssetID == id {
				old = append(old, f)
			}
			return nil
		}); e != nil {
			return e
		}
		j.Retained = len(old)
		for _, p := range j.Faces {
			overlap := false
			for _, f := range old {
				if sameSample(f.SampleUS, p.SampleUS) && overlaps(f.Box, p.Box) {
					overlap = true
					break
				}
			}
			if overlap {
				j.Skipped++
				continue
			}
			key, e := newFaceID()
			if e != nil {
				return e
			}
			f := StoredFace{ID: key, AssetID: id, Box: p.Box, Embedding: p.Embedding, Provider: p.Provider, SampleUS: p.SampleUS}
			if !validFace(f, key) {
				return ErrOrganizationInput
			}
			if e = writeJSON(fb, key, f); e != nil {
				return e
			}
			old = append(old, f)
			j.Added++
		}
		a.Objects = j.Objects
		a.Provider = j.Provider
		a.Video = j.Video
		a.Error = ""
		if e := writeJSON(tx.Bucket([]byte(analysisBucket)), id, a); e != nil {
			return e
		}
		if e := ctx.Err(); e != nil {
			return e
		}
		j.Status = "applied"
		j.Faces = nil
		j.Objects = nil
		j.Video = nil
		return writeJSON(b, id, j)
	})
}
