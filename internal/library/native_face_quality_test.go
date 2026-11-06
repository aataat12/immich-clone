package library

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"golang.org/x/image/draw"
	"image"
	"image/color"
	"immichclone/internal/recognition"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// Frozen thresholds and identity split from experiments/face-quality/PROTOCOL.md.
// Go BiLinear 3x interpolation is declared before evaluation; not retuned after it.
// No portraits, embeddings or individual predictions are logged or saved.
func TestActualNativeCrossPhotoQuality(t *testing.T) {
	root := os.Getenv("IMMICH_NATIVE_QUALITY_ROOT")
	if root == "" {
		t.Skip("requires explicit pinned research corpus and native provider")
	}
	archive := filepath.Join(root, "att_faces.zip")
	raw, e := os.ReadFile(archive)
	if e != nil {
		t.Fatal("research archive unavailable")
	}
	digest := sha256.Sum256(raw)
	const archiveSHA = "509d763bd2276aa054d260d7f1000e8be212738134f2f83d8ff13bc2adf1c8ec"
	if hex.EncodeToString(digest[:]) != archiveSHA {
		t.Fatal("research archive integrity failed")
	}
	z, e := zip.NewReader(bytes.NewReader(raw), int64(len(raw)))
	if e != nil {
		t.Fatal("research archive invalid")
	}
	provider := recognition.New(filepath.Join(root, "ai"))
	if !provider.Available() {
		t.Fatal("pinned native provider unavailable")
	}
	scratch := filepath.Join(root, "scratch")
	if e = os.MkdirAll(scratch, 0700); e != nil {
		t.Fatal(e)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 12*time.Minute)
	defer cancel()
	started := time.Now()
	type record = automaticQualityRecord
	all := make([]record, 0, 400)
	detected, calibrationDetected, heldoutDetected := 0, 0, 0
	for subject := 1; subject <= 40; subject++ {
		for index := 1; index <= 10; index++ {
			f, e := z.Open(fmt.Sprintf("s%d/%d.pgm", subject, index))
			if e != nil {
				t.Fatal("expected research entry missing")
			}
			data, e := io.ReadAll(io.LimitReader(f, 12000))
			f.Close()
			if e != nil {
				t.Fatal("research entry read failed")
			}
			header := []byte("P5\n92 112\n255\n")
			if !bytes.HasPrefix(data, header) || len(data) != len(header)+92*112 {
				t.Fatal("research entry dimensions/header mismatch")
			}
			gray := image.NewGray(image.Rect(0, 0, 92, 112))
			copy(gray.Pix, data[len(header):])
			frame := image.NewRGBA(image.Rect(0, 0, 276, 336))
			draw.BiLinear.Scale(frame, frame.Bounds(), gray, gray.Bounds(), draw.Src, nil)
			result, e := provider.Analyze(ctx, frame, scratch)
			if e != nil {
				t.Fatal("actual native research inference failed")
			}
			r := record{subject: subject, index: index}
			if len(result.Faces) == 1 {
				face := result.Faces[0]
				r.face = &StoredFace{Provider: recognition.AnalysisID, Embedding: face.Embedding}
				detected++
				if subject <= 10 {
					calibrationDetected++
				} else {
					heldoutDetected++
				}
			}
			all = append(all, r)
		}
		if subject%10 == 0 {
			t.Logf("Native quality inference completed for %d of 400 images", subject*10)
		}
	}
	score := func(a, b *StoredFace) float64 {
		sum := 0.
		for i, v := range a.Embedding {
			sum += v * b.Embedding[i]
		}
		return sum
	}
	maxNegative := -1.
	calibrationPairs := 0
	for i, a := range all[:100] {
		for _, b := range all[i+1 : 100] {
			if a.subject != b.subject && a.face != nil && b.face != nil {
				v := score(a.face, b.face)
				if v > maxNegative {
					maxNegative = v
				}
				calibrationPairs++
			}
		}
	}
	// The current production threshold is fixed before this run. Never recalibrate.
	calibrationPass := calibrationPairs > 0 && maxNegative+.02 <= .363
	refs := []reviewReference{}
	for _, r := range all {
		if r.subject >= 11 && r.subject <= 30 && r.index == 1 && r.face != nil {
			refs = append(refs, reviewReference{f: *r.face, l: FaceLabel{Name: fmt.Sprintf("Identity %d", r.subject)}})
		}
	}
	knownCorrect, knownWrong, knownUnresolved, unknownAccept, unknownRejected, unknownUnprocessed := 0, 0, 0, 0, 0, 0
	for _, r := range all[100:] {
		if r.subject <= 30 && r.index == 1 {
			continue
		}
		known := r.subject <= 30
		if r.face == nil {
			if known {
				knownUnresolved++
			} else {
				unknownUnprocessed++
			}
			continue
		}
		name, _ := suggestReviewFace(*r.face, refs)
		if known {
			if name == "" {
				knownUnresolved++
			} else if name == fmt.Sprintf("Identity %d", r.subject) {
				knownCorrect++
			} else {
				knownWrong++
			}
		} else {
			if name == "" {
				unknownRejected++
			} else {
				unknownAccept++
			}
		}
	}
	samePairs, differentPairs, sameMisses, differentAccepts, unscoredSame, unscoredDifferent := 0, 0, 0, 0, 0, 0
	heldout := all[100:]
	for i, a := range heldout {
		for _, b := range heldout[i+1:] {
			same := a.subject == b.subject
			if a.face == nil || b.face == nil {
				if same {
					unscoredSame++
				} else {
					unscoredDifferent++
				}
				continue
			}
			v := score(a.face, b.face)
			if same {
				samePairs++
				if v < .363 {
					sameMisses++
				}
			} else {
				differentPairs++
				if v >= .363 {
					differentAccepts++
				}
			}
		}
	}
	noFacePass := 0
	for _, level := range []uint8{0, 127, 255} {
		img := image.NewRGBA(image.Rect(0, 0, 276, 336))
		draw.Draw(img, img.Bounds(), image.NewUniform(color.RGBA{level, level, level, 255}), image.Point{}, draw.Src)
		result, e := provider.Analyze(ctx, img, scratch)
		if e != nil {
			t.Fatal("native no-face control inference failed")
		}
		if len(result.Faces) == 0 {
			noFacePass++
		}
	}
	coffee, e := os.ReadFile(filepath.Join(root, "coffee.png"))
	if e != nil {
		t.Fatal("public natural no-face control missing")
	}
	coffeeHash := sha256.Sum256(coffee)
	if hex.EncodeToString(coffeeHash[:]) != "cc02f8ca188b167c775a7101b5d767d1e71792cf762c33d6fa15a4599b5a8de7" {
		t.Fatal("natural control checksum mismatch")
	}
	coffeeImage, _, e := image.Decode(bytes.NewReader(coffee))
	if e != nil {
		t.Fatal("natural control decode failed")
	}
	coffeeResult, e := provider.Analyze(ctx, coffeeImage, scratch)
	if e != nil {
		t.Fatal("natural no-face control inference failed")
	}
	naturalNoFace := len(coffeeResult.Faces) == 0
	after, e := os.ReadFile(archive)
	if e != nil {
		t.Fatal("research archive unavailable after evaluation")
	}
	digest = sha256.Sum256(after)
	if hex.EncodeToString(digest[:]) != archiveSHA {
		t.Fatal("research archive changed")
	}
	scratchFiles, e := os.ReadDir(scratch)
	if e != nil || len(scratchFiles) != 0 {
		t.Fatal("native analysis scratch retained")
	}
	if os.Getenv("IMMICH_AUTOMATIC_QUALITY") == "1" {
		evaluateAutomaticNativeQuality(t, all)
	}
	pass := calibrationPass && detected >= 360 && knownCorrect >= 144 && knownWrong == 0 && unknownAccept == 0 && noFacePass == 3 && naturalNoFace
	report := map[string]any{"scope": "ORL narrow identity-disjoint native Go/provider screen; not general-library quality", "credit": "AT&T Laboratories Cambridge / Olivetti Research Laboratory", "provider": recognition.AnalysisID, "images_total": 400, "detected_exactly_one": detected, "calibration_detected": calibrationDetected, "heldout_detected": heldoutDetected, "calibration_max_negative": maxNegative, "calibration_negative_pairs": calibrationPairs, "threshold": .363, "margin": .05, "calibration_pass": calibrationPass, "known_total": 180, "known_correct": knownCorrect, "known_wrong": knownWrong, "known_unresolved": knownUnresolved, "unknown_total": 100, "unknown_false_accept": unknownAccept, "unknown_rejected": unknownRejected, "unknown_unprocessed": unknownUnprocessed, "enrollments_available": len(refs), "scored_same_pairs": samePairs, "same_pair_misses": sameMisses, "scored_different_pairs": differentPairs, "different_pair_accepts": differentAccepts, "unscored_same_pairs": unscoredSame, "unscored_different_pairs": unscoredDifferent, "flat_no_face_controls_passed": noFacePass, "natural_no_face_control_passed": naturalNoFace, "elapsed_seconds": time.Since(started).Seconds(), "originals_unchanged": true, "narrow_quality_screen_passed": pass}
	b, e := json.Marshal(report)
	if e != nil {
		t.Fatal(e)
	}
	t.Log(string(b))
	if knownCorrect+knownWrong+knownUnresolved != 180 || unknownAccept+unknownRejected+unknownUnprocessed != 100 {
		t.Fatal("quality denominator accounting invalid")
	}
	if !pass {
		t.Fatal("frozen native quality screen did not meet predeclared gates; no model or threshold changes made")
	}
}
