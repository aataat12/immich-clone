package library

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	bolt "go.etcd.io/bbolt"
	"image"
	"image/png"
	"immichclone/internal/media"
	"immichclone/internal/recognition"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
	"unicode/utf8"
)

const analysisBucket = "analysis_v1"
const analysisQueue = "analysis_queue_v1"
const facesBucket = "faces_v1"
const labelsBucket = "face_labels_v1"
const tagsBucket = "custom_tags_v1"

var ErrOrganizationInput = errors.New("invalid organization request")

type Analysis struct {
	Video    *VideoProgress `json:"video,omitempty"`
	AssetID  string         `json:"asset_id"`
	Status   string         `json:"status"`
	Error    string         `json:"error,omitempty"`
	Provider string         `json:"provider"`
	Objects  []string       `json:"objects"`
}
type StoredFace struct {
	SampleUS  *int64    `json:"sample_us,omitempty"`
	ID        string    `json:"id"`
	AssetID   string    `json:"asset_id"`
	Box       []float64 `json:"box"`
	Embedding []float64 `json:"embedding"`
	Provider  string    `json:"provider"`
}
type FaceLabel struct {
	Source         string  `json:"name_source,omitempty"`
	Policy         string  `json:"automatic_policy,omitempty"`
	Score          float64 `json:"automatic_score,omitempty"`
	PreviousSource string  `json:"previous_source,omitempty"`
	PreviousPolicy string  `json:"previous_policy,omitempty"`
	PreviousScore  float64 `json:"previous_score,omitempty"`
	Dismissed      bool    `json:"dismissed,omitempty"`
	Name           string  `json:"name"`
	Previous       string  `json:"previous,omitempty"`
	CanUndo        bool    `json:"can_undo"`
}
type FaceView struct {
	Source     string    `json:"name_source,omitempty"`
	Policy     string    `json:"automatic_policy,omitempty"`
	SampleUS   *int64    `json:"sample_us,omitempty"`
	Dismissed  bool      `json:"dismissed"`
	ID         string    `json:"id"`
	AssetID    string    `json:"asset_id"`
	AssetName  string    `json:"asset_name"`
	Box        []float64 `json:"box"`
	Name       string    `json:"name"`
	CanUndo    bool      `json:"can_undo"`
	Suggestion string    `json:"suggestion,omitempty"`
	Score      float64   `json:"score,omitempty"`
}
type OrganizationAsset struct {
	Reanalysis *ReanalysisView `json:"reanalysis,omitempty"`
	Asset      Asset           `json:"asset"`
	Analysis   Analysis        `json:"analysis"`
	Tags       []string        `json:"tags"`
	Dismissed  []string        `json:"dismissed"`
}
type Organization struct {
	AutomaticNaming bool                `json:"automatic_naming"`
	MediaAvailable  bool                `json:"media_available"`
	Faces           []FaceView          `json:"faces"`
	Names           []string            `json:"names"`
	Assets          []OrganizationAsset `json:"assets"`
	After           string              `json:"after,omitempty"`
	FaceAfter       string              `json:"face_after,omitempty"`
	Available       bool                `json:"available"`
}

func initOrganization(tx *bolt.Tx) error {
	if e := initReanalysis(tx); e != nil {
		return e
	}
	for _, name := range []string{analysisBucket, facesBucket, labelsBucket, tagsBucket, "dismissed_ai_tags_v1", batchMembers, batchMeta} {
		if _, e := tx.CreateBucketIfNotExists([]byte(name)); e != nil {
			return e
		}
	}
	if tx.Bucket([]byte(analysisQueue)) != nil {
		if e := tx.DeleteBucket([]byte(analysisQueue)); e != nil {
			return e
		}
	}
	q, e := tx.CreateBucket([]byte(analysisQueue))
	if e != nil {
		return e
	}
	return tx.Bucket([]byte(analysisBucket)).ForEach(func(k, v []byte) error {
		var a Analysis
		if e := json.Unmarshal(v, &a); e != nil {
			return e
		}
		if a.Status == "queued" {
			return q.Put(k, []byte{1})
		}
		return nil
	})
}
func readJSON(b *bolt.Bucket, key string, v any) error {
	data := b.Get([]byte(key))
	if data == nil {
		return os.ErrNotExist
	}
	if len(data) > 1<<20 {
		return errors.New("organization record too large")
	}
	return json.Unmarshal(data, v)
}
func writeJSON(b *bolt.Bucket, key string, v any) error {
	data, e := json.Marshal(v)
	if e != nil {
		return e
	}
	return b.Put([]byte(key), data)
}
func (s *Store) SetRecognizer(fn func(context.Context, image.Image, string) (recognition.Result, error), available bool) {
	s.gate.Lock()
	defer s.gate.Unlock()
	s.recognizer = fn
	s.recognitionAvailable.Store(available)
}
func (s *Store) QueueAnalysis(id string) error {
	s.gate.Lock()
	defer s.gate.Unlock()
	a, e := s.Get(id)
	if e != nil {
		return e
	}
	if isVideo(a.MIME) {
		return s.queueVideoLocked(a)
	}
	if !supportsStillAI(a.MIME) || a.Size > 64<<20 {
		return errors.New("AI still image format or 64 MiB size limit unsupported")
	}
	if isHEIC(a.MIME) && !s.mediaAnalysisAvailable.Load() {
		return media.ErrUnavailable
	}
	return s.db.Update(func(tx *bolt.Tx) error {
		var current Analysis
		b := tx.Bucket([]byte(analysisBucket))
		e := readJSON(b, id, &current)
		if e != nil && !errors.Is(e, os.ErrNotExist) {
			return e
		}
		if current.Status == "ready" || current.Status == "queued" {
			return nil
		}
		current = Analysis{AssetID: id, Status: "queued", Provider: recognition.AnalysisID, Objects: []string{}}
		if e = writeJSON(b, id, current); e != nil {
			return e
		}
		return tx.Bucket([]byte(analysisQueue)).Put([]byte(id), []byte{1})
	})
}

// analysisImage decodes the original, not the low-resolution browsing thumbnail.
func (s *Store) analysisImage(ctx context.Context, id string) (image.Image, error) {
	a, e := s.Get(id)
	if e != nil {
		return nil, e
	}
	if !supportsStillAI(a.MIME) || a.Size > 64<<20 {
		return nil, errors.New("AI image format or encoded size unsupported")
	}
	hash, e := fileHashContext(ctx, s.Original(id))
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	if e != nil || hash != id {
		return nil, errors.New("original integrity check failed")
	}
	if isHEIC(a.MIME) {
		return s.stillMediaImage(ctx, s.Original(id))
	}
	f, e := os.Open(s.Original(id))
	if e != nil {
		return nil, errors.New("original unavailable")
	}
	defer f.Close()
	photo := ReadPhotoMetadata(contextReader{ctx, f})
	if _, e = f.Seek(0, 0); e != nil {
		return nil, errors.New("original read failed")
	}
	img, e := decodeParentImage(ctx, f)
	if e != nil {
		return nil, e
	}
	return orientedContext(ctx, img, photo.Orientation)
}
func newFaceID() (string, error) {
	var b [32]byte
	if _, e := rand.Read(b[:]); e != nil {
		return "", e
	}
	return hex.EncodeToString(b[:]), nil
}
func (s *Store) ProcessAnalysis(ctx context.Context) (bool, error) {
	if e := ctx.Err(); e != nil {
		return false, e
	}
	s.analysisGate.Lock()
	defer s.analysisGate.Unlock()
	s.gate.Lock()
	if s.ImportsRunning() {
		s.gate.Unlock()
		return false, nil
	}
	recognizer := s.recognizer
	var id string
	e := s.db.View(func(tx *bolt.Tx) error {
		batch, e := readBatch(tx)
		if e != nil {
			return e
		}
		c := tx.Bucket([]byte(analysisQueue)).Cursor()
		for k, _ := c.First(); k != nil; k, _ = c.Next() {
			if string(tx.Bucket([]byte(batchMembers)).Get(k)) == "pending" && (batch.Status != "running" || !s.recognitionAvailable.Load()) {
				continue
			}
			id = string(k)
			break
		}
		return nil
	})
	if e != nil || id == "" {
		s.gate.Unlock()
		if e == nil {
			return s.processReanalysis(ctx)
		}
		return false, e
	}
	s.activeAnalysis = id
	s.gate.Unlock()
	defer func() { s.gate.Lock(); s.activeAnalysis = ""; s.gate.Unlock() }()
	var a Analysis
	if e := s.db.View(func(tx *bolt.Tx) error { return readJSON(tx.Bucket([]byte(analysisBucket)), id, &a) }); e != nil {
		return true, e
	}
	if a.Status != "queued" || a.AssetID != id {
		return true, errors.New("analysis queue inconsistent")
	}
	asset, err := s.Get(id)
	if err != nil {
		return true, err
	}
	if isVideo(asset.MIME) {
		return true, s.processVideoAnalysis(ctx, asset, a, recognizer)
	}
	parent := ctx
	opCtx, cancel := context.WithTimeout(ctx, 75*time.Second)
	defer cancel()
	ctx = opCtx
	img, e := s.analysisImage(ctx, id)
	var result recognition.Result
	if e == nil {
		if recognizer == nil {
			e = recognition.ErrUnavailable
		} else {
			result, e = recognizer(ctx, img, filepath.Join(s.root, "staging"))
		}
	}
	if parent.Err() != nil {
		return true, parent.Err()
	}
	if ctx.Err() != nil {
		e = errors.New("local AI analysis time limit exceeded")
	}
	if e == nil {
		e = recognition.Validate(result, 1, 1)
		for _, f := range result.Faces {
			if e != nil {
				break
			}
			if f.Box[0] < 0 || f.Box[1] < 0 || f.Box[0]+f.Box[2] > 1.000001 || f.Box[1]+f.Box[3] > 1.000001 {
				e = errors.New("invalid normalized AI geometry")
			}
		}
	}
	var faces []StoredFace
	if e == nil {
		for _, f := range result.Faces {
			key, err := newFaceID()
			if err != nil {
				return true, err
			}
			face := StoredFace{ID: key, AssetID: id, Box: f.Box, Embedding: f.Embedding, Provider: a.Provider}
			if e = s.publishFaceCrop(face, img); e != nil {
				break
			}
			faces = append(faces, face)
		}
	}
	if parent.Err() != nil {
		return true, parent.Err()
	}
	if ctx.Err() != nil {
		e = errors.New("local AI analysis time limit exceeded")
	}
	if e != nil {
		a.Status = "failed"
		a.Error = e.Error()
		if len(a.Error) > 256 {
			a.Error = "AI processing failed"
		}
	} else {
		a.Status = "ready"
		a.Error = ""
		a.Objects = []string{}
		for _, o := range result.Objects {
			name := recognition.Classes[o.Class]
			found := false
			for _, n := range a.Objects {
				if n == name {
					found = true
				}
			}
			if !found {
				a.Objects = append(a.Objects, name)
			}
		}
		sort.Strings(a.Objects)
	}
	s.gate.Lock()
	defer s.gate.Unlock()
	return true, s.db.Update(func(tx *bolt.Tx) error {
		if a.Status == "ready" {
			for _, f := range faces {
				if e := writeJSON(tx.Bucket([]byte(facesBucket)), f.ID, f); e != nil {
					return e
				}
			}
		}
		if a.Status == "ready" {
			if e := applyAutomaticNames(tx, faces, productionAutomaticNamePolicy()); e != nil {
				return e
			}
		}
		if e := writeJSON(tx.Bucket([]byte(analysisBucket)), id, a); e != nil {
			return e
		}
		if e := finishBatchMember(tx, id, a.Status); e != nil {
			return e
		}
		return tx.Bucket([]byte(analysisQueue)).Delete([]byte(id))
	})
}
func (s *Store) publishFaceCrop(f StoredFace, img image.Image) error {
	if !validFace(f, f.ID) {
		return ErrOrganizationInput
	}
	b := img.Bounds()
	x := int(f.Box[0] * float64(b.Dx()))
	y := int(f.Box[1] * float64(b.Dy()))
	w := int(f.Box[2] * float64(b.Dx()))
	h := int(f.Box[3] * float64(b.Dy()))
	if w < 1 {
		w = 1
	}
	if h < 1 {
		h = 1
	}
	width, height := 160, 160
	if w >= h {
		height = h * 160 / w
	} else {
		width = w * 160 / h
	}
	if width < 1 {
		width = 1
	}
	if height < 1 {
		height = 1
	}
	crop := image.NewRGBA(image.Rect(0, 0, width, height))
	for yy := 0; yy < height; yy++ {
		for xx := 0; xx < width; xx++ {
			px := b.Min.X + x + xx*w/width
			py := b.Min.Y + y + yy*h/height
			if px >= b.Max.X {
				px = b.Max.X - 1
			}
			if py >= b.Max.Y {
				py = b.Max.Y - 1
			}
			crop.Set(xx, yy, img.At(px, py))
		}
	}
	out, e := os.CreateTemp(filepath.Join(s.root, "staging"), "face-*.part")
	if e != nil {
		return errors.New("face preview storage unavailable")
	}
	defer os.Remove(out.Name())
	e = png.Encode(out, crop)
	if e == nil {
		e = out.Sync()
	}
	ce := out.Close()
	if e == nil {
		e = ce
	}
	if e != nil {
		return errors.New("face preview write failed")
	}
	if e = os.Rename(out.Name(), filepath.Join(s.root, "faces", f.ID+".png")); e != nil {
		return errors.New("face preview publication failed")
	}
	return nil
}
func (s *Store) FaceCrop(id string) (string, error) {
	if !validID(id) {
		return "", os.ErrNotExist
	}
	var f StoredFace
	if e := s.db.View(func(tx *bolt.Tx) error { return readJSON(tx.Bucket([]byte(facesBucket)), id, &f) }); e != nil {
		return "", e
	}
	if !validFace(f, id) {
		return "", ErrOrganizationInput
	}
	path := filepath.Join(s.root, "faces", id+".png")
	if _, e := os.Stat(path); e == nil {
		return path, nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 75*time.Second)
	defer cancel()
	img, e := s.faceSource(ctx, f)
	if e != nil {
		return "", e
	}
	s.gate.Lock()
	defer s.gate.Unlock()
	if e = s.db.View(func(tx *bolt.Tx) error {
		var current StoredFace
		return readJSON(tx.Bucket([]byte(facesBucket)), id, &current)
	}); e != nil {
		return "", e
	}
	if e = s.publishFaceCrop(f, img); e != nil {
		return "", e
	}
	return path, nil
}

func textOK(s string, max int) bool {
	if !utf8.ValidString(s) || len(s) > max {
		return false
	}
	for _, r := range s {
		if r < 32 || r == 127 {
			return false
		}
	}
	return true
}
func (s *Store) LabelFaces(ids []string, name string) error {
	if !textOK(name, 128) {
		return ErrOrganizationInput
	}
	name = strings.TrimSpace(name)
	if len(ids) == 0 || len(ids) > 100 || !textOK(name, 128) {
		return ErrOrganizationInput
	}
	s.gate.Lock()
	defer s.gate.Unlock()
	return s.db.Update(func(tx *bolt.Tx) error {
		seen := map[string]bool{}
		for _, id := range ids {
			if !validID(id) || seen[id] {
				return ErrOrganizationInput
			}
			seen[id] = true
			var f StoredFace
			if e := readJSON(tx.Bucket([]byte(facesBucket)), id, &f); e != nil {
				return e
			}
			if !validFace(f, id) {
				return ErrOrganizationInput
			}
			var label FaceLabel
			e := readJSON(tx.Bucket([]byte(labelsBucket)), id, &label)
			if e != nil && !errors.Is(e, os.ErrNotExist) {
				return e
			}
			if label.Dismissed {
				return ErrOrganizationInput
			}
			if label.Name == name && faceNameSource(label) != "automatic" && name != "" {
				continue
			}
			label = FaceLabel{Name: name, Source: "manual", Previous: label.Name, PreviousSource: faceNameSource(label), PreviousPolicy: label.Policy, PreviousScore: label.Score, CanUndo: true}
			if e = writeJSON(tx.Bucket([]byte(labelsBucket)), id, label); e != nil {
				return e
			}
		}
		return nil
	})
}
func (s *Store) UndoFace(id string) error {
	if !validID(id) {
		return ErrOrganizationInput
	}
	s.gate.Lock()
	defer s.gate.Unlock()
	return s.db.Update(func(tx *bolt.Tx) error {
		var l FaceLabel
		if e := readJSON(tx.Bucket([]byte(labelsBucket)), id, &l); e != nil {
			return e
		}
		if !l.CanUndo {
			return ErrOrganizationInput
		}
		l.Name, l.Source, l.Policy, l.Score = l.Previous, l.PreviousSource, l.PreviousPolicy, l.PreviousScore
		l.Previous, l.PreviousSource, l.PreviousPolicy, l.PreviousScore = "", "", "", 0
		l.CanUndo = false
		return writeJSON(tx.Bucket([]byte(labelsBucket)), id, l)
	})
}
func (s *Store) SetTags(id string, tags []string) error {
	if !validID(id) || len(tags) > 32 {
		return ErrOrganizationInput
	}
	clean := []string{}
	for _, tag := range tags {
		if !textOK(tag, 64) {
			return ErrOrganizationInput
		}
		tag = strings.TrimSpace(tag)
		if tag == "" {
			continue
		}
		if !textOK(tag, 64) || strings.Contains(tag, ",") {
			return ErrOrganizationInput
		}
		found := false
		for _, s := range clean {
			if strings.EqualFold(tag, s) {
				found = true
			}
		}
		if !found {
			clean = append(clean, tag)
		}
	}
	sort.Strings(clean)
	s.gate.Lock()
	defer s.gate.Unlock()
	if _, e := s.Get(id); e != nil {
		return e
	}
	return s.db.Update(func(tx *bolt.Tx) error { return writeJSON(tx.Bucket([]byte(tagsBucket)), id, clean) })
}

// OrganizationPage is a bounded pilot review view; hidden records remain stored.
func (s *Store) OrganizationPage(after, faceAfter, query string) (Organization, error) {
	return s.OrganizationReviewPage(after, faceAfter, query, false)
}

// OrganizationReviewPage separates dismissed corrections from active references.
func (s *Store) OrganizationReviewPage(after, faceAfter, query string, dismissed bool) (Organization, error) {
	return s.OrganizationFilteredPage(after, faceAfter, query, dismissed, false)
}

// OrganizationFilteredPage applies review filters before taking a bounded page.
func (s *Store) OrganizationFilteredPage(after, faceAfter, query string, dismissed, unnamed bool) (Organization, error) {
	out := Organization{Faces: []FaceView{}, Assets: []OrganizationAsset{}, Names: []string{}}
	if (after != "" && !validID(after)) || (faceAfter != "" && !validID(faceAfter)) {
		return out, ErrOrganizationInput
	}
	if !textOK(query, 128) {
		return out, ErrOrganizationInput
	}
	s.gate.Lock()
	defer s.gate.Unlock()
	out.Available = s.recognitionAvailable.Load()
	out.MediaAvailable = s.mediaAnalysisAvailable.Load()
	out.AutomaticNaming = automaticNamesValidated
	e := s.db.View(func(tx *bolt.Tx) error {
		index, err := s.organizationIndexFor(tx)
		if err != nil {
			return err
		}
		assets := index.assets
		out.Names = append(out.Names, index.names...)
		lowerQuery := strings.ToLower(query)
		start := sort.Search(len(index.faces), func(i int) bool { return index.faces[i].f.ID > faceAfter })
		for _, r := range index.faces[start:] {
			if r.l.Dismissed != dismissed {
				continue
			}
			if r.f.ID <= faceAfter {
				continue
			}
			if unnamed && r.l.Name != "" {
				continue
			}
			view := FaceView{Source: faceNameSource(r.l), Policy: r.l.Policy, SampleUS: r.f.SampleUS, Dismissed: r.l.Dismissed, ID: r.f.ID, AssetID: r.f.AssetID, AssetName: assets[r.f.AssetID].Name, Box: append([]float64(nil), r.f.Box...), Name: r.l.Name, CanUndo: r.l.CanUndo}
			if view.SampleUS != nil {
				position := *view.SampleUS
				view.SampleUS = &position
			}
			if query != "" && !strings.Contains(strings.ToLower(view.Name), lowerQuery) && !strings.Contains(strings.ToLower(view.AssetName), lowerQuery) {
				continue
			}
			if len(out.Faces) >= 100 {
				out.FaceAfter = out.Faces[len(out.Faces)-1].ID
				break
			}
			if view.Name == "" && !view.Dismissed {
				var subject StoredFace
				if err := readJSON(tx.Bucket([]byte(facesBucket)), r.f.ID, &subject); err != nil {
					return err
				}
				view.Suggestion, view.Score = suggestReviewFace(subject, index.references)
			}
			out.Faces = append(out.Faces, view)
		}
		cursor := tx.Bucket([]byte("assets")).Cursor()
		k, v := cursor.First()
		if after != "" {
			k, v = cursor.Seek([]byte(after))
			if string(k) == after {
				k, v = cursor.Next()
			}
		}
		for ; k != nil; k, v = cursor.Next() {
			if lowerQuery != "" && !strings.Contains(index.searchByAsset[string(k)], lowerQuery) {
				continue
			}
			var a Asset
			if e := json.Unmarshal(v, &a); e != nil {
				return e
			}
			if a.Hidden {
				continue
			}
			record := OrganizationAsset{Asset: a, Analysis: Analysis{AssetID: a.ID, Status: "not_requested", Objects: []string{}}, Tags: []string{}}
			var jobErr error
			record.Reanalysis, jobErr = reanalysisView(tx, a.ID)
			if jobErr != nil {
				return jobErr
			}
			if e := readJSON(tx.Bucket([]byte(analysisBucket)), a.ID, &record.Analysis); e != nil && !errors.Is(e, os.ErrNotExist) {
				return e
			}
			if e := readJSON(tx.Bucket([]byte(tagsBucket)), a.ID, &record.Tags); e != nil && !errors.Is(e, os.ErrNotExist) {
				return e
			}
			if e := readJSON(tx.Bucket([]byte("dismissed_ai_tags_v1")), a.ID, &record.Dismissed); e != nil && !errors.Is(e, os.ErrNotExist) {
				return e
			}
			visible := []string{}
			for _, tag := range record.Analysis.Objects {
				blocked := false
				for _, d := range record.Dismissed {
					if tag == d {
						blocked = true
					}
				}
				if !blocked {
					visible = append(visible, tag)
				}
			}
			record.Analysis.Objects = visible
			if len(out.Assets) >= 50 {
				out.After = out.Assets[len(out.Assets)-1].Asset.ID
				break
			}
			out.Assets = append(out.Assets, record)
		}
		return nil
	})
	return out, e
}

// OrganizationExport excludes embeddings; consistent backup retains the full private catalog.
func (s *Store) OrganizationExport() (map[string]any, error) {
	s.gate.Lock()
	defer s.gate.Unlock()
	out := map[string]any{"schema": 1}
	organization := map[string]any{"version": 1}
	e := s.db.View(func(tx *bolt.Tx) error {
		out["catalog_schema"] = string(tx.Bucket([]byte("meta")).Get([]byte("schema")))
 curation,e:=exportCuration(tx);if e!=nil{return e};out["curation"]=curation
		for _, name := range []string{analysisBucket, facesBucket, labelsBucket, tagsBucket, "dismissed_ai_tags_v1", batchMembers, batchMeta, reanalysisBucket} {
			items := map[string]json.RawMessage{}
			e := tx.Bucket([]byte(name)).ForEach(func(k, v []byte) error {
				if name == facesBucket {
					var f StoredFace
					if e := json.Unmarshal(v, &f); e != nil {
						return e
					}
					f.Embedding = nil
					b, e := json.Marshal(f)
					if e != nil {
						return e
					}
					items[string(k)] = b
				} else if name == reanalysisBucket {
					var job reanalysisJob
					if e := json.Unmarshal(v, &job); e != nil {
						return e
					}
					for i := range job.Faces {
						job.Faces[i].Embedding = nil
					}
					data, e := json.Marshal(job)
					if e != nil {
						return e
					}
					items[string(k)] = data
				} else if name == batchMembers {
					// Membership values are raw internal states, not JSON records.
					b, e := json.Marshal(string(v))
					if e != nil {
						return e
					}
					items[string(k)] = b
				} else {
					items[string(k)] = append([]byte(nil), v...)
				}
				return nil
			})
			if e != nil {
				return e
			}
			organization[name] = items
		}
		assets := []Asset{}
		if e := tx.Bucket([]byte("assets")).ForEach(func(k, v []byte) error {
			var a Asset
			if e := json.Unmarshal(v, &a); e != nil {
				return e
			}
			assets = append(assets, a)
			return nil
		}); e != nil {
			return e
		}
		out["assets"] = assets
		out["organization"] = organization
		return nil
	})
	return out, e
}

// ManualFace records user-supplied geometry, without claiming model recognition.
func (s *Store) ManualFace(ctx context.Context, id string, box []float64, name string) error {
	if !validID(id) || len(box) != 4 || !textOK(name, 128) {
		return ErrOrganizationInput
	}
	for _, v := range box {
		if math.IsNaN(v) || math.IsInf(v, 0) || v < 0 || v > 1 {
			return ErrOrganizationInput
		}
	}
	if box[2] <= 0 || box[3] <= 0 || box[0]+box[2] > 1 || box[1]+box[3] > 1 {
		return ErrOrganizationInput
	}
	initial, e := s.Get(id)
	if e != nil {
		return e
	}
	img, e := s.analysisImage(ctx, id)
	if e != nil {
		return e
	}
	s.gate.Lock()
	defer s.gate.Unlock()
	current, e := s.Get(id)
	if e != nil {
		return e
	}
	if !current.Added.Equal(initial.Added) {
		return ErrOrganizationInput
	}
	key, e := newFaceID()
	if e != nil {
		return e
	}
	f := StoredFace{ID: key, AssetID: id, Box: box, Provider: "manual"}
	if e = s.publishFaceCrop(f, img); e != nil {
		return e
	}
	return s.db.Update(func(tx *bolt.Tx) error {
		if e := writeJSON(tx.Bucket([]byte(facesBucket)), key, f); e != nil {
			return e
		}
		return writeJSON(tx.Bucket([]byte(labelsBucket)), key, FaceLabel{Name: strings.TrimSpace(name), Source: "manual"})
	})
}

func (s *Store) DismissTag(id, tag string, restore bool) error {
	if !validID(id) || !textOK(tag, 64) {
		return ErrOrganizationInput
	}
	s.gate.Lock()
	defer s.gate.Unlock()
	return s.db.Update(func(tx *bolt.Tx) error {
		var analysis Analysis
		if e := readJSON(tx.Bucket([]byte(analysisBucket)), id, &analysis); e != nil {
			return e
		}
		found := false
		for _, t := range analysis.Objects {
			if t == tag {
				found = true
			}
		}
		if !found && !restore {
			return ErrOrganizationInput
		}
		var tags []string
		e := readJSON(tx.Bucket([]byte("dismissed_ai_tags_v1")), id, &tags)
		if e != nil && !errors.Is(e, os.ErrNotExist) {
			return e
		}
		next := []string{}
		for _, t := range tags {
			if t != tag {
				next = append(next, t)
			}
		}
		if !restore {
			next = append(next, tag)
		}
		return writeJSON(tx.Bucket([]byte("dismissed_ai_tags_v1")), id, next)
	})
}

func validFace(f StoredFace, key string) bool {
	if f.SampleUS != nil && (*f.SampleUS < 0 || *f.SampleUS >= 86400*1e6) {
		return false
	}
	if !validID(f.ID) || f.ID != key || !validID(f.AssetID) || len(f.Box) != 4 || !textOK(f.Provider, 128) {
		return false
	}
	for _, v := range f.Box {
		if math.IsNaN(v) || math.IsInf(v, 0) || v < 0 || v > 1 {
			return false
		}
	}
	if f.Box[2] <= 0 || f.Box[3] <= 0 || f.Box[0]+f.Box[2] > 1.000001 || f.Box[1]+f.Box[3] > 1.000001 {
		return false
	}
	if f.Provider == "manual" {
		return len(f.Embedding) == 0
	}
	if len(f.Embedding) != 128 {
		return false
	}
	norm := 0.
	for _, v := range f.Embedding {
		if math.IsNaN(v) || math.IsInf(v, 0) {
			return false
		}
		norm += v * v
	}
	return math.Abs(norm-1) <= .02
}

// DismissFace preserves the detection, name/undo history and original bytes.
func (s *Store) DismissFace(id string, restore bool) error {
	if !validID(id) {
		return ErrOrganizationInput
	}
	s.gate.Lock()
	defer s.gate.Unlock()
	return s.db.Update(func(tx *bolt.Tx) error {
		var f StoredFace
		if e := readJSON(tx.Bucket([]byte(facesBucket)), id, &f); e != nil {
			return e
		}
		if !validFace(f, id) {
			return ErrOrganizationInput
		}
		var label FaceLabel
		e := readJSON(tx.Bucket([]byte(labelsBucket)), id, &label)
		if e != nil && !errors.Is(e, os.ErrNotExist) {
			return e
		}
		label.Dismissed = !restore
		return writeJSON(tx.Bucket([]byte(labelsBucket)), id, label)
	})
}
