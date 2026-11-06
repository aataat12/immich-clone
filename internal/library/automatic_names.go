package library

import (
	"errors"
	"immichclone/internal/recognition"
	"math"
	"os"

	bolt "go.etcd.io/bbolt"
)

// Frozen policy passed separate ORL and independent Nottingham native screens.
// See docs/release-0.1.43-catalog-compatibility.md for evidence and limits.
// Reversible prototype assignments still require review; there is no runtime
// switch that bypasses the trusted-reference or competitor-margin rules.
const automaticNamesValidated = true
const automaticNamePolicyID = "known-person-two-sources-065-015-v1"

type automaticNamePolicy struct {
	enabled           bool
	threshold, margin float64
}

func candidateAutomaticNamePolicy() automaticNamePolicy {
	return automaticNamePolicy{enabled: true, threshold: .65, margin: .15}
}

func productionAutomaticNamePolicy() automaticNamePolicy {
	p := candidateAutomaticNamePolicy()
	p.enabled = automaticNamesValidated
	return p
}

func faceNameSource(l FaceLabel) string {
	if l.Name == "" {
		return ""
	}
	// Labels written before provenance existed were all explicit user actions.
	if l.Source == "" {
		return "manual"
	}
	return l.Source
}

func validFaceLabel(l FaceLabel) bool {
	validSource := func(s string) bool { return s == "" || s == "manual" || s == "automatic" }
	return textOK(l.Name, 128) && textOK(l.Previous, 128) && validSource(l.Source) && validSource(l.PreviousSource) && textOK(l.Policy, 128) && textOK(l.PreviousPolicy, 128) && !math.IsNaN(l.Score) && !math.IsInf(l.Score, 0) && l.Score >= 0 && l.Score <= 1 && !math.IsNaN(l.PreviousScore) && !math.IsInf(l.PreviousScore, 0) && l.PreviousScore >= 0 && l.PreviousScore <= 1
}

func trustedFaceLabel(l FaceLabel) bool {
	return l.Name != "" && !l.Dismissed && faceNameSource(l) == "manual"
}

func unitFaceEmbedding(v []float64) bool {
	if len(v) != 128 {
		return false
	}
	norm := 0.
	for _, x := range v {
		if math.IsNaN(x) || math.IsInf(x, 0) {
			return false
		}
		norm += x * x
	}
	return math.Abs(norm-1) <= .01
}

// A match requires two manually named, independent source originals. A lone
// strong competing identity can veto it. Same-original video frames cannot
// impersonate independent references; automatic assignments never reinforce it.
func matchAutomaticFace(subject StoredFace, refs []reviewReference, p automaticNamePolicy) (string, float64) {
	if !p.enabled || !unitFaceEmbedding(subject.Embedding) {
		return "", 0
	}
	byName := map[string]map[string]float64{}
	for _, ref := range refs {
		if !trustedFaceLabel(ref.l) || ref.f.AssetID == subject.AssetID || ref.f.ID == subject.ID || !recognition.CompatibleEmbeddings(ref.f.Provider, subject.Provider) || !unitFaceEmbedding(ref.f.Embedding) {
			continue
		}
		score := 0.
		for i, v := range subject.Embedding {
			score += v * ref.f.Embedding[i]
		}
		if byName[ref.l.Name] == nil {
			byName[ref.l.Name] = map[string]float64{}
		}
		if score > byName[ref.l.Name][ref.f.AssetID] {
			byName[ref.l.Name][ref.f.AssetID] = score
		}
	}
	name, best := "", 0.
	for candidate, sources := range byName {
		first, second := 0., 0.
		for _, score := range sources {
			if score > first {
				second, first = first, score
			} else if score > second {
				second = score
			}
		}
		if second >= p.threshold && second > best {
			name, best = candidate, second
		}
	}
	if name == "" {
		return "", 0
	}
	runner := 0.
	for candidate, sources := range byName {
		if candidate == name {
			continue
		}
		for _, score := range sources {
			if score > runner {
				runner = score
			}
		}
	}
	if best-runner < p.margin {
		return "", 0
	}
	return name, math.Min(1, best)
}

// Caller holds Store.gate and the analysis publication transaction. Only the
// newly published faces are considered. Any existing label, including an empty
// correction or undone assignment, is authoritative and is never overwritten.
func applyAutomaticNames(tx *bolt.Tx, faces []StoredFace, p automaticNamePolicy) error {
	if !p.enabled || len(faces) == 0 {
		return nil
	}
	refs := []reviewReference{}
	fb, lb := tx.Bucket([]byte(facesBucket)), tx.Bucket([]byte(labelsBucket))
	if e := fb.ForEach(func(k, _ []byte) error {
		var l FaceLabel
		if e := readJSON(lb, string(k), &l); e != nil {
			if errors.Is(e, os.ErrNotExist) {
				return nil
			}
			return e
		}
		if !trustedFaceLabel(l) {
			return nil
		}
		var f StoredFace
		if e := readJSON(fb, string(k), &f); e != nil {
			return e
		}
		var a Asset
		if e := readJSON(tx.Bucket([]byte("assets")), f.AssetID, &a); e != nil {
			return e
		}
		if !a.Hidden {
			refs = append(refs, reviewReference{f: f, l: l})
		}
		return nil
	}); e != nil {
		return e
	}
	for _, f := range faces {
		if lb.Get([]byte(f.ID)) != nil {
			continue
		}
		name, score := matchAutomaticFace(f, refs, p)
		if name == "" {
			continue
		}
		l := FaceLabel{Name: name, Source: "automatic", Policy: automaticNamePolicyID, Score: score, CanUndo: true}
		if e := writeJSON(lb, f.ID, l); e != nil {
			return e
		}
		if e := requireAutomaticCatalog(tx); e != nil {
			return e
		}
	}
	return nil
}
