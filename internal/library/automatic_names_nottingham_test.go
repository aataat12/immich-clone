package library

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"immichclone/internal/recognition"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"
)

const nottinghamArchiveSHA = "b97fecf7c3e8250518a190aa265e617fbd916e14a2bbbcd79a58c65bf1da190e"

type nottinghamQualityRecord struct {
	subject, configuration int
	face                   *StoredFace
	missing                bool
}

type nottinghamQualityCounts struct {
	Total      int `json:"total"`
	Missing    int `json:"missing_original"`
	Detected   int `json:"detected_exactly_one"`
	Correct    int `json:"correct"`
	Wrong      int `json:"wrong"`
	Rejected   int `json:"rejected"`
	Unresolved int `json:"unresolved"`
}

var nottinghamKnownIDs = []int{3,8,11,23,34,37,39,44,48,50,51,52,53,56,58,59,60,62,63,64,66,67,70,71,73,75,76,81,85,90,91,92,93,95,99,100}
var nottinghamUnknownIDs = []int{1,2,4,5,10,13,17,25,30,32,36,38,41,42,43,45,47,49,54,55,57,65,68,69,72,74,77,78,80,83,84,86,88,94,96}

func nottinghamConfigurationGroup(configuration int) string {
	switch configuration {
	case 3,4:
		return "frontal_expressions_3_4"
	case 1,2:
		return "frontal_1_2"
	case 5,6:
		return "three_quarter_5_6"
	case 7:
		return "cap_7"
	default:
		return "additional_8"
	}
}

// Frozen source, split, score policy and gates are declared in the Nottingham protocol.
// No portraits, raw embeddings, individual predictions or subject metrics leave
// the process. Production automatic naming stays disabled during this screen.
func TestActualNativeIndependentNottinghamAutomaticQuality(t *testing.T) {
	root := os.Getenv("IMMICH_NOTTINGHAM_QUALITY_ROOT")
	if root == "" {
		t.Skip("requires explicit pinned independent Nottingham research corpus and native provider")
	}
	started := time.Now()
	ctx, cancel := context.WithTimeout(context.Background(), 18*time.Minute)
	defer cancel()
	archivePath := filepath.Join(root, "nottingham-originals.zip")
	f, e := os.Open(archivePath)
	if e != nil {
		t.Fatal("Nottingham research archive unavailable")
	}
	raw, e := io.ReadAll(io.LimitReader(f, 64<<20+1))
	closeErr := f.Close()
	if e != nil || closeErr != nil || len(raw) != 38347802 {
		t.Fatal("Nottingham research archive size/read mismatch")
	}
	digest := sha256.Sum256(raw)
	if hex.EncodeToString(digest[:]) != nottinghamArchiveSHA {
		t.Fatal("Nottingham research archive integrity mismatch")
	}
	z, e := zip.NewReader(bytes.NewReader(raw), int64(len(raw)))
	if e != nil || len(z.File) != 495 {
		t.Fatal("Nottingham research archive inventory invalid")
	}
	entries := map[string]*zip.File{}
	var totalBytes uint64
	for _, entry := range z.File {
		if !entry.Mode().IsRegular() || entry.UncompressedSize64 > 512<<10 || entries[entry.Name] != nil {
			t.Fatal("Nottingham research archive entry invalid")
		}
		entries[entry.Name] = entry
		totalBytes += entry.UncompressedSize64
	}
	if totalBytes != 38832533 {
		t.Fatal("Nottingham research archive content size mismatch")
	}
	provider := recognition.New(filepath.Join(root, "ai"))
	if !provider.Available() {
		t.Fatal("pinned native provider unavailable")
	}
	scratch := filepath.Join(root, "scratch")
	if e = os.MkdirAll(scratch, 0700); e != nil {
		t.Fatal("native evaluation scratch unavailable")
	}
	all := make([]nottinghamQualityRecord, 0, 498)
	detected, noFace, multipleFaces, missing, processed := 0, 0, 0, 0, 0
	knownIDs := map[int]bool{}
	for _, id := range nottinghamKnownIDs { knownIDs[id] = true }
	ids := append(append([]int(nil), nottinghamKnownIDs...), nottinghamUnknownIDs...)
	for _, subject := range ids {
		last := 7
		if subject == 39 { last = 8 }
		for configuration := 1; configuration <= last; configuration++ {
			entry := entries[fmt.Sprintf("m%d-%d.gif", subject, configuration)]
			if entry == nil {
				if !(subject == 5 && (configuration == 4 || configuration == 5) || subject == 13 && configuration == 4) { t.Fatal("Nottingham research unexpected missing entry") }
				missing++
				all = append(all, nottinghamQualityRecord{subject:subject, configuration:configuration, missing:true})
				continue
			}
			reader, err := entry.Open()
			if err != nil {
				t.Fatal("Nottingham research entry unavailable")
			}
			data, err := io.ReadAll(io.LimitReader(reader, 512<<10+1))
			closeErr := reader.Close()
			if err != nil || closeErr != nil || len(data) > 512<<10 {
				t.Fatal("Nottingham research entry read mismatch")
			}
			// Match the application's Go GIF first-frame decode and orientation path.
			photo := ReadPhotoMetadata(bytes.NewReader(data))
			img, err := decodeParentImage(ctx, bytes.NewReader(data))
			if err != nil || img.Bounds().Dx() != 288 || img.Bounds().Dy() != 384 {
				t.Fatal("Nottingham research original decode/dimensions mismatch")
			}
			img, err = orientedContext(ctx, img, photo.Orientation)
			if err != nil {
				t.Fatal("Nottingham research original orientation failed")
			}
			result, err := provider.Analyze(ctx, img, scratch)
			if err != nil {
				t.Fatal("Nottingham actual native inference failed")
			}
			record := nottinghamQualityRecord{subject: subject, configuration: configuration}
			switch len(result.Faces) {
			case 0:
				noFace++
			case 1:
				detected++
				hash := sha256.Sum256(data)
				id := hex.EncodeToString(hash[:])
				face := result.Faces[0]
				record.face = &StoredFace{ID: id, AssetID: id, Provider: recognition.AnalysisID, Box: face.Box, Embedding: face.Embedding}
			default:
				multipleFaces++
			}
			all = append(all, record)
			processed++
			if processed%100 == 0 { t.Logf("Independent native Nottingham inference completed for %d of 495 originals",processed) }
		}

	}
	refs := []reviewReference{}
	for _, record := range all {
		if knownIDs[record.subject] && (record.configuration == 1 || record.configuration == 2) && record.face != nil {
			refs = append(refs, reviewReference{f: *record.face, l: FaceLabel{Name: fmt.Sprintf("Research identity %d", record.subject), Source: "manual"}})
		}
	}
	known, unknown := nottinghamQualityCounts{}, nottinghamQualityCounts{}
	knownGroups, unknownGroups := map[string]nottinghamQualityCounts{}, map[string]nottinghamQualityCounts{}
	policy := candidateAutomaticNamePolicy()
	for _, record := range all {
		isKnown := knownIDs[record.subject]
		if isKnown && (record.configuration == 1 || record.configuration == 2) {
			continue
		}
		outcome := nottinghamQualityCounts{Total: 1}
		if record.missing { outcome.Missing = 1 }
		if record.face == nil {
			outcome.Unresolved = 1
		} else {
			outcome.Detected = 1
			name, _ := matchAutomaticFace(*record.face, refs, policy)
			if isKnown {
				if name == "" {
					outcome.Unresolved = 1
				} else if name == fmt.Sprintf("Research identity %d", record.subject) {
					outcome.Correct = 1
				} else {
					outcome.Wrong = 1
				}
			} else if name == "" {
				outcome.Rejected = 1
			} else {
				outcome.Wrong = 1
			}
		}
		add := func(a nottinghamQualityCounts, b nottinghamQualityCounts) nottinghamQualityCounts {
			return nottinghamQualityCounts{Total: a.Total + b.Total, Missing: a.Missing+b.Missing, Detected: a.Detected + b.Detected, Correct: a.Correct + b.Correct, Wrong: a.Wrong + b.Wrong, Rejected: a.Rejected + b.Rejected, Unresolved: a.Unresolved + b.Unresolved}
		}
		group := nottinghamConfigurationGroup(record.configuration)
		if isKnown {
			known = add(known, outcome)
			knownGroups[group] = add(knownGroups[group], outcome)
		} else {
			unknown = add(unknown, outcome)
			unknownGroups[group] = add(unknownGroups[group], outcome)
		}
	}
	flatPass := 0
	for _, level := range []uint8{0, 127, 255} {
		img := image.NewRGBA(image.Rect(0, 0, 640, 480))
		draw.Draw(img, img.Bounds(), image.NewUniform(color.RGBA{level, level, level, 255}), image.Point{}, draw.Src)
		result, err := provider.Analyze(ctx, img, scratch)
		if err != nil {
			t.Fatal("Nottingham native flat control inference failed")
		}
		if len(result.Faces) == 0 {
			flatPass++
		}
	}
	coffee, e := os.ReadFile(filepath.Join(root, "coffee.png"))
	if e != nil {
		t.Fatal("Nottingham natural no-face control unavailable")
	}
	coffeeHash := sha256.Sum256(coffee)
	if hex.EncodeToString(coffeeHash[:]) != "cc02f8ca188b167c775a7101b5d767d1e71792cf762c33d6fa15a4599b5a8de7" {
		t.Fatal("Nottingham natural control integrity mismatch")
	}
	coffeeImage, _, e := image.Decode(bytes.NewReader(coffee))
	if e != nil {
		t.Fatal("Nottingham natural control decode failed")
	}
	coffeeResult, e := provider.Analyze(ctx, coffeeImage, scratch)
	if e != nil {
		t.Fatal("Nottingham natural control native inference failed")
	}
	naturalPass := len(coffeeResult.Faces) == 0
	afterHash, e := fileHashContext(ctx, archivePath)
	if e != nil || afterHash != nottinghamArchiveSHA {
		t.Fatal("Nottingham original archive changed/unavailable after evaluation")
	}
	scratchEntries, e := os.ReadDir(scratch)
	if e != nil || len(scratchEntries) != 0 {
		t.Fatal("Nottingham native scratch retained")
	}
	central := knownGroups["frontal_expressions_3_4"]
	pass := len(refs)==72 && detected>=420 && central.Detected>=66 && central.Correct>=54 && known.Correct>=90 && known.Wrong==0 && unknown.Detected>=210 && unknown.Wrong==0 && flatPass==3 && naturalPass
	report := map[string]any{
		"scope":          "independent fixed PICS Nottingham monochrome expression/pose/cap evaluation; not universal accuracy",
		"credit":         "Psychological Image Collection at Stirling / University of Stirling; Nottingham original face set",
		"archive_sha256": nottinghamArchiveSHA, "provider": recognition.AnalysisID,
		"policy": automaticNamePolicyID, "threshold": policy.threshold, "margin": policy.margin,
		"images_total": processed, "scheduled_observations":len(all), "missing_originals":missing, "detected_exactly_one": detected, "no_face": noFace, "multiple_faces": multipleFaces,
		"fixed_references_available": len(refs), "known": known, "unknown": unknown,
		"known_configuration_groups": knownGroups, "unknown_configuration_groups": unknownGroups,
		"flat_no_face_controls_passed": flatPass, "natural_no_face_control_passed": naturalPass,
		"original_archive_unchanged": true, "native_scratch_empty": true,
		"independent_frozen_nottingham_gates_passed": pass, "production_automatic_naming_enabled": automaticNamesValidated,
		"elapsed_seconds": time.Since(started).Seconds(),
	}
	encoded, e := json.Marshal(report)
	if e != nil {
		t.Fatal("Nottingham aggregate report encoding failed")
	}
	t.Log(string(encoded))
	if known.Total != 181 || unknown.Total != 245 || central.Total != 72 || known.Correct+known.Wrong+known.Unresolved != 181 || unknown.Wrong+unknown.Rejected+unknown.Unresolved != 245 || detected+noFace+multipleFaces != 495 || missing != 3 || processed != 495 || len(all) != 498 {
		t.Fatal("Nottingham denominator accounting invalid")
	}
	if !pass {
		t.Fatal("independent frozen Nottingham automatic naming gates failed; policy and references left unchanged")
	}
}
