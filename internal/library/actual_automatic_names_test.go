package library

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"image"
	"image/color"
	"image/png"
	"immichclone/internal/recognition"
	"os"
	"path/filepath"
	"testing"
	"time"

	"golang.org/x/image/draw"
)

// These are declared transformations of one already approved public capture.
// Distinct managed originals exercise the production publication policy, but
// they are NOT independent real captures or recognition-quality evidence.
func actualAutomaticVariants(t *testing.T, original []byte) [][]byte {
	t.Helper()
	source, e := png.Decode(bytes.NewReader(original))
	if e != nil {
		t.Fatal("public functional fixture decode failed")
	}
	bounds := source.Bounds()
	darker := image.NewRGBA(image.Rect(0, 0, bounds.Dx(), bounds.Dy()))
	for y := 0; y < bounds.Dy(); y++ {
		for x := 0; x < bounds.Dx(); x++ {
			r, g, b, a := source.At(bounds.Min.X+x, bounds.Min.Y+y).RGBA()
			darker.SetRGBA(x, y, color.RGBA{uint8((r >> 8) * 95 / 100), uint8((g >> 8) * 95 / 100), uint8((b >> 8) * 95 / 100), uint8(a >> 8)})
		}
	}
	resized := image.NewRGBA(image.Rect(0, 0, 480, 480))
	draw.ApproxBiLinear.Scale(resized, resized.Bounds(), source, bounds, draw.Src, nil)
	variants := [][]byte{original}
	for _, img := range []image.Image{darker, resized} {
		var out bytes.Buffer
		if e := png.Encode(&out, img); e != nil {
			t.Fatal("functional fixture transformation failed")
		}
		variants = append(variants, out.Bytes())
	}
	return variants
}

func actualAutomaticFace(t *testing.T, s *Store, asset string) FaceView {
	t.Helper()
	page, e := s.OrganizationPage("", "", "")
	if e != nil {
		t.Fatal("functional publication organization read failed")
	}
	var faces []FaceView
	for _, face := range page.Faces {
		if face.AssetID == asset {
			faces = append(faces, face)
		}
	}
	if len(faces) != 1 {
		t.Fatal("functional native fixture must publish exactly one face")
	}
	for _, record := range page.Assets {
		if record.Asset.ID == asset && record.Analysis.Status != "ready" {
			t.Fatal("functional native analysis did not complete successfully")
		}
	}
	return faces[0]
}

func TestActualAutomaticNamesPublicationLifecycle(t *testing.T) {
	inputs := os.Getenv("IMMICH_ACTUAL_AUTOMATIC_ROOT")
	if inputs == "" || !automaticNamesValidated {
		t.Skip("requires explicit pinned native inputs and enabled production automatic naming")
	}
	original, e := os.ReadFile(filepath.Join(inputs, "astronaut.png"))
	if e != nil {
		t.Fatal("pinned public functional fixture unavailable")
	}
	digest := sha256.Sum256(original)
	if hex.EncodeToString(digest[:]) != "88431cd9653ccd539741b555fb0a46b61558b301d4110412b5bc28b5e3ea6cb5" {
		t.Fatal("public functional fixture checksum mismatch")
	}
	provider := recognition.New(filepath.Join(inputs, "ai"))
	if !provider.Available() {
		t.Fatal("pinned native provider unavailable")
	}
	s, root := openTest(t)
	s.SetRecognizer(provider.Analyze, true)
	if e = os.WriteFile(filepath.Join(root, "access-code.txt"), bytes.Repeat([]byte("a"), 64), 0600); e != nil {
		t.Fatal(e)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	variants := actualAutomaticVariants(t, original)
	names := []string{"public-original.png", "synthetic-brightness.png", "synthetic-resize.png"}
	assets := make([]Asset, 0, len(variants))
	seen := map[string]bool{}
	var target FaceView
	for i, data := range variants {
		a, duplicate, e := s.Put(names[i], bytes.NewReader(data))
		if e != nil || duplicate || seen[a.ID] {
			t.Fatal("functional fixtures must be distinct preserved managed originals")
		}
		seen[a.ID] = true
		assets = append(assets, a)
		if e = s.QueueAnalysis(a.ID); e != nil {
			t.Fatal("actual native analysis queue failed")
		}
		if worked, e := s.ProcessAnalysis(ctx); e != nil || !worked {
			t.Fatal("actual native publication did not execute")
		}
		face := actualAutomaticFace(t, s, a.ID)
		if i < 2 {
			if face.Name != "" || automaticSchemaValue(t, s) != "1" {
				t.Fatal("automatic assignment occurred before two explicit reference originals")
			}
			if e = s.LabelFaces([]string{face.ID}, "Functional fixture person"); e != nil {
				t.Fatal("explicit native reference naming failed")
			}
			if !trustedFaceLabel(automaticLabel(t, s, face.ID)) {
				t.Fatal("explicit native reference was not trusted")
			}
		} else {
			target = face
		}
	}
	label := automaticLabel(t, s, target.ID)
	if target.Name != "Functional fixture person" || target.Source != "automatic" || label.Policy != automaticNamePolicyID || !label.CanUndo || trustedFaceLabel(label) || automaticSchemaValue(t, s) != "2" {
		t.Fatal("actual native analysis did not publish a protected automatic assignment")
	}
	page, e := s.SearchGalleryPage("name", "library", "", "Functional fixture person", 100)
	if e != nil || page.Count != 3 {
		t.Fatal("actual automatic name not searchable in the gallery")
	}
	exported, e := s.OrganizationExport()
	if e != nil {
		t.Fatal(e)
	}
	encoded, e := json.Marshal(exported)
	if e != nil || !bytes.Contains(encoded, []byte(`"name_source":"automatic"`)) || bytes.Contains(encoded, []byte(`"embedding":[`)) {
		t.Fatal("actual automatic export provenance/embedding contract failed")
	}
	var automaticBackup bytes.Buffer
	if e = s.Backup(&automaticBackup); e != nil {
		t.Fatal("actual automatic assignment backup failed")
	}
	if e = s.LabelFaces([]string{target.ID}, target.Name); e != nil {
		t.Fatal(e)
	}
	if !trustedFaceLabel(automaticLabel(t, s, target.ID)) {
		t.Fatal("explicit same-name confirmation did not promote trust")
	}
	if e = s.UndoFace(target.ID); e != nil {
		t.Fatal(e)
	}
	if restored := automaticLabel(t, s, target.ID); restored.Source != "automatic" || trustedFaceLabel(restored) || restored.Policy != automaticNamePolicyID {
		t.Fatal("confirmation undo did not restore untrusted automatic provenance")
	}
	if e = s.LabelFaces([]string{target.ID}, "Explicit corrected fixture"); e != nil {
		t.Fatal(e)
	}
	corrected := automaticLabel(t, s, target.ID)
	if e = s.ControlReanalysis(assets[2].ID, "start"); e != nil {
		t.Fatal(e)
	}
	if worked, e := s.ProcessAnalysis(ctx); e != nil || !worked {
		t.Fatal("actual native reanalysis did not execute")
	}
	if e = s.ApplyReanalysis(ctx, assets[2].ID); e != nil {
		t.Fatal("actual native reanalysis application failed")
	}
	if got := automaticLabel(t, s, target.ID); got != corrected || actualAutomaticFace(t, s, assets[2].ID).ID != target.ID {
		t.Fatal("actual native reanalysis replaced the face or its explicit correction")
	}
	if e = s.Verify(); e != nil {
		t.Fatal("actual functional originals changed")
	}
	s.Close()
	s, e = Open(root)
	if e != nil {
		t.Fatal("actual automatic lifecycle restart failed")
	}
	if automaticSchemaValue(t, s) != "2" || automaticLabel(t, s, target.ID) != corrected {
		t.Fatal("restart lost protected correction or automatic history")
	}
	var correctedBackup bytes.Buffer
	if e = s.Backup(&correctedBackup); e != nil {
		t.Fatal("actual correction backup failed")
	}
	s.Close()
	if e = os.Rename(root, root+"-unavailable"); e != nil {
		t.Fatal(e)
	}
	for _, checkpoint := range []struct {
		name  string
		data  []byte
		label FaceLabel
	}{{"automatic", automaticBackup.Bytes(), label}, {"corrected", correctedBackup.Bytes(), corrected}} {
		t.Run(checkpoint.name+" independent restore", func(t *testing.T) {
			archive := filepath.Join(t.TempDir(), "recovery.zip")
			if e := os.WriteFile(archive, checkpoint.data, 0600); e != nil {
				t.Fatal(e)
			}
			destination := filepath.Join(t.TempDir(), "restored")
			if e := Restore(archive, destination); e != nil {
				t.Fatal("actual lifecycle independent restore failed")
			}
			restored, e := Open(destination)
			if e != nil {
				t.Fatal(e)
			}
			defer restored.Close()
			if automaticSchemaValue(t, restored) != "2" || automaticLabel(t, restored, target.ID) != checkpoint.label {
				t.Fatal("independent restore lost actual native assignment/correction history")
			}
			for i, a := range assets {
				got, e := os.ReadFile(restored.Original(a.ID))
				if e != nil || !bytes.Equal(got, variants[i]) {
					t.Fatal("independent restore changed an actual functional original")
				}
			}
			if e = restored.Verify(); e != nil {
				t.Fatal(e)
			}
		})
	}
	t.Log("PASS: actual pinned CPU native analysis and automatic publication, confirmation/undo/correction/reanalysis, protected schema, restart and two independent restores; one public capture plus synthetic transformations, functional evidence only")
}
