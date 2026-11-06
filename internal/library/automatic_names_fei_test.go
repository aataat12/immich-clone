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

const feiArchiveSHA = "1d0b0ebe94278a5677edc450251f91450dadb821d9774edc111ad1be0e7475ed"

type feiQualityRecord struct {
	subject, configuration int
	face                   *StoredFace
}

type feiQualityCounts struct {
	Total      int `json:"total"`
	Detected   int `json:"detected_exactly_one"`
	Correct    int `json:"correct"`
	Wrong      int `json:"wrong"`
	Rejected   int `json:"rejected"`
	Unresolved int `json:"unresolved"`
}

func feiConfigurationGroup(configuration int) string {
	switch {
	case configuration >= 4 && configuration <= 7:
		return "central_sweep_04_07"
	case configuration == 11 || configuration == 12:
		return "frontal_11_12"
	case configuration >= 13:
		return "additional_13_14"
	default:
		return "outer_sweep_01_03_08_10"
	}
}

// Frozen source, split, score policy and gates are declared in the FEI protocol.
// No portraits, raw embeddings, individual predictions or subject metrics leave
// the process. Production automatic naming stays disabled during this screen.
func TestActualNativeIndependentFEIAutomaticQuality(t *testing.T) {
	root := os.Getenv("IMMICH_FEI_QUALITY_ROOT")
	if root == "" {
		t.Skip("requires explicit pinned independent FEI research corpus and native provider")
	}
	started := time.Now()
	ctx, cancel := context.WithTimeout(context.Background(), 24*time.Minute)
	defer cancel()
	archivePath := filepath.Join(root, "fei-part1.zip")
	f, e := os.Open(archivePath)
	if e != nil {
		t.Fatal("FEI research archive unavailable")
	}
	raw, e := io.ReadAll(io.LimitReader(f, 96<<20+1))
	closeErr := f.Close()
	if e != nil || closeErr != nil || len(raw) != 86857988 {
		t.Fatal("FEI research archive size/read mismatch")
	}
	digest := sha256.Sum256(raw)
	if hex.EncodeToString(digest[:]) != feiArchiveSHA {
		t.Fatal("FEI research archive integrity mismatch")
	}
	z, e := zip.NewReader(bytes.NewReader(raw), int64(len(raw)))
	if e != nil || len(z.File) != 700 {
		t.Fatal("FEI research archive inventory invalid")
	}
	entries := map[string]*zip.File{}
	var totalBytes uint64
	for _, entry := range z.File {
		if !entry.Mode().IsRegular() || entry.UncompressedSize64 > 512<<10 || entries[entry.Name] != nil {
			t.Fatal("FEI research archive entry invalid")
		}
		entries[entry.Name] = entry
		totalBytes += entry.UncompressedSize64
	}
	if totalBytes != 88178712 {
		t.Fatal("FEI research archive content size mismatch")
	}
	provider := recognition.New(filepath.Join(root, "ai"))
	if !provider.Available() {
		t.Fatal("pinned native provider unavailable")
	}
	scratch := filepath.Join(root, "scratch")
	if e = os.MkdirAll(scratch, 0700); e != nil {
		t.Fatal("native evaluation scratch unavailable")
	}
	all := make([]feiQualityRecord, 0, 700)
	detected, noFace, multipleFaces := 0, 0, 0
	for subject := 1; subject <= 50; subject++ {
		for configuration := 1; configuration <= 14; configuration++ {
			entry := entries[fmt.Sprintf("%d-%02d.jpg", subject, configuration)]
			if entry == nil {
				t.Fatal("FEI research expected entry missing")
			}
			reader, err := entry.Open()
			if err != nil {
				t.Fatal("FEI research entry unavailable")
			}
			data, err := io.ReadAll(io.LimitReader(reader, 512<<10+1))
			closeErr := reader.Close()
			if err != nil || closeErr != nil || len(data) > 512<<10 {
				t.Fatal("FEI research entry read mismatch")
			}
			// Match the application's Go original decode and EXIF orientation path.
			photo := ReadPhotoMetadata(bytes.NewReader(data))
			img, err := decodeParentImage(ctx, bytes.NewReader(data))
			if err != nil || img.Bounds().Dx() != 640 || img.Bounds().Dy() != 480 {
				t.Fatal("FEI research original decode/dimensions mismatch")
			}
			img, err = orientedContext(ctx, img, photo.Orientation)
			if err != nil {
				t.Fatal("FEI research original orientation failed")
			}
			result, err := provider.Analyze(ctx, img, scratch)
			if err != nil {
				t.Fatal("FEI actual native inference failed")
			}
			record := feiQualityRecord{subject: subject, configuration: configuration}
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
		}
		if subject%10 == 0 {
			t.Logf("Independent native FEI inference completed for %d of 700 originals", subject*14)
		}
	}
	refs := []reviewReference{}
	for _, record := range all {
		if record.subject <= 30 && (record.configuration == 11 || record.configuration == 12) && record.face != nil {
			refs = append(refs, reviewReference{f: *record.face, l: FaceLabel{Name: fmt.Sprintf("Research identity %d", record.subject), Source: "manual"}})
		}
	}
	known, unknown := feiQualityCounts{}, feiQualityCounts{}
	knownGroups, unknownGroups := map[string]feiQualityCounts{}, map[string]feiQualityCounts{}
	policy := candidateAutomaticNamePolicy()
	for _, record := range all {
		isKnown := record.subject <= 30
		if isKnown && (record.configuration == 11 || record.configuration == 12) {
			continue
		}
		outcome := feiQualityCounts{Total: 1}
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
		add := func(a feiQualityCounts, b feiQualityCounts) feiQualityCounts {
			return feiQualityCounts{Total: a.Total+b.Total, Detected: a.Detected+b.Detected, Correct: a.Correct+b.Correct, Wrong: a.Wrong+b.Wrong, Rejected: a.Rejected+b.Rejected, Unresolved: a.Unresolved+b.Unresolved}
		}
		group := feiConfigurationGroup(record.configuration)
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
		if err != nil { t.Fatal("FEI native flat control inference failed") }
		if len(result.Faces) == 0 { flatPass++ }
	}
	coffee, e := os.ReadFile(filepath.Join(root, "coffee.png"))
	if e != nil { t.Fatal("FEI natural no-face control unavailable") }
	coffeeHash := sha256.Sum256(coffee)
	if hex.EncodeToString(coffeeHash[:]) != "cc02f8ca188b167c775a7101b5d767d1e71792cf762c33d6fa15a4599b5a8de7" { t.Fatal("FEI natural control integrity mismatch") }
	coffeeImage, _, e := image.Decode(bytes.NewReader(coffee))
	if e != nil { t.Fatal("FEI natural control decode failed") }
	coffeeResult, e := provider.Analyze(ctx, coffeeImage, scratch)
	if e != nil { t.Fatal("FEI natural control native inference failed") }
	naturalPass := len(coffeeResult.Faces) == 0
	afterHash, e := fileHashContext(ctx, archivePath)
	if e != nil || afterHash != feiArchiveSHA { t.Fatal("FEI original archive changed/unavailable after evaluation") }
	scratchEntries, e := os.ReadDir(scratch)
	if e != nil || len(scratchEntries) != 0 { t.Fatal("FEI native scratch retained") }
	central := knownGroups["central_sweep_04_07"]
	pass := len(refs)==60 && detected>=420 && central.Detected>=108 && central.Correct>=90 && known.Correct>=144 && known.Wrong==0 && unknown.Detected>=160 && unknown.Wrong==0 && flatPass==3 && naturalPass
	report := map[string]any{
		"scope":"independent fixed FEI controlled-adult color/pose/configuration evaluation; not universal accuracy",
		"credit":"Centro Universitario da FEI; Carlos Eduardo Thomaz and Gilson Antonio Giraldi",
		"archive_sha256":feiArchiveSHA, "provider":recognition.AnalysisID,
		"policy":automaticNamePolicyID, "threshold":policy.threshold, "margin":policy.margin,
		"images_total":len(all), "detected_exactly_one":detected, "no_face":noFace, "multiple_faces":multipleFaces,
		"fixed_references_available":len(refs), "known":known, "unknown":unknown,
		"known_configuration_groups":knownGroups, "unknown_configuration_groups":unknownGroups,
		"flat_no_face_controls_passed":flatPass, "natural_no_face_control_passed":naturalPass,
		"original_archive_unchanged":true, "native_scratch_empty":true,
		"independent_frozen_fei_gates_passed":pass, "production_automatic_naming_enabled":automaticNamesValidated,
		"elapsed_seconds":time.Since(started).Seconds(),
	}
	encoded, e := json.Marshal(report)
	if e != nil { t.Fatal("FEI aggregate report encoding failed") }
	t.Log(string(encoded))
	if known.Total!=360 || unknown.Total!=280 || central.Total!=120 || known.Correct+known.Wrong+known.Unresolved!=360 || unknown.Wrong+unknown.Rejected+unknown.Unresolved!=280 || detected+noFace+multipleFaces!=700 { t.Fatal("FEI denominator accounting invalid") }
	if !pass { t.Fatal("independent frozen FEI automatic naming gates failed; policy and references left unchanged") }
}
