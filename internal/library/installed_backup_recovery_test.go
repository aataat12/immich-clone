package library

// Opt-in metadata oracle after the actual installed CLI independently restores
// and verifies the archive, with the original state volume absent. No fixture
// bytes, access codes, archive contents or catalog values are printed.
import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"testing"
)

func TestInstalledBackupRecoveryMetadata(t *testing.T) {
	root := os.Getenv("IMMICH_BACKUP_RECOVERY_ROOT")
	if root == "" {
		t.Skip("disposable installed backup acceptance only")
	}
	var saved struct {
		Files []struct {
			Name, Relative, ID string
			Size               int64
			JPEG               bool
		}
		Bytes                        int64
		ImportID, SecretHash, FaceID string
	}
	b, err := os.ReadFile(filepath.Join(root, "baseline.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err = json.Unmarshal(b, &saved); err != nil {
		t.Fatal(err)
	}
	if len(saved.Files) != 18 || saved.Bytes <= 4<<30 || saved.FaceID == "" {
		t.Fatal("incomplete acceptance baseline")
	}
	target := filepath.Join(root, "restored")
	s, err := Open(target)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	assets, err := s.List()
	if err != nil {
		t.Fatal(err)
	}
	if len(assets) != len(saved.Files) {
		t.Fatal("restored asset count")
	}
	for i, f := range saved.Files {
		a, err := s.Get(f.ID)
		if err != nil {
			t.Fatal(err)
		}
		if a.Name != f.Name || a.Size != f.Size || !slices.Contains(a.Sources, f.Relative) || a.Hidden != (i == 3) {
			t.Fatalf("restored identity/size/source/visibility fixture %d", i)
		}
		if f.JPEG && (a.Photo.Orientation != 6 || a.Photo.Captured != "2024-03-02T10:11:12" || a.Photo.Make != "Demo!" || a.Width != 10 || a.Height != 20) {
			t.Fatalf("restored JPEG metadata fixture %d", i)
		}
		stat, err := os.Stat(s.Original(f.ID))
		if err != nil || stat.Size() != f.Size {
			t.Fatalf("restored original size fixture %d", i)
		}
	}
	receipt, err := s.ImportPage(saved.ImportID, 0)
	if err != nil {
		t.Fatal(err)
	}
	m := receipt.Session
	if m.Status != "completed" || m.TotalFiles != 18 || m.AcceptedFiles != 18 || m.NewFiles != 18 || m.Duplicates != 0 || m.TotalBytes != saved.Bytes || m.AcceptedBytes != saved.Bytes || len(receipt.Entries) != 18 || receipt.Next != -1 {
		t.Fatal("restored completed import receipt totals")
	}
	for i, e := range receipt.Entries {
		if e.AssetID != saved.Files[i].ID || e.Duplicate || e.Deleted || e.Error != "" || e.Relative != saved.Files[i].Relative {
			t.Fatalf("restored receipt fixture %d", i)
		}
	}
	org, err := s.OrganizationPage("", "", "")
	if err != nil {
		t.Fatal(err)
	}
	faceOK, tagOK := false, false
	for _, f := range org.Faces {
		if f.ID == saved.FaceID && f.AssetID == saved.Files[2].ID && f.Name == "Backup preserved person" && !f.Dismissed {
			faceOK = true
		}
	}
	for _, a := range org.Assets {
		if a.Asset.ID == saved.Files[2].ID && slices.Contains(a.Tags, "Backup preservation tag") {
			tagOK = true
		}
	}
	if !faceOK || !tagOK {
		t.Fatal("restored named manual face/tag")
	}
	secret, err := os.ReadFile(filepath.Join(target, "access-code.txt"))
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(secret)
	if hex.EncodeToString(sum[:]) != saved.SecretHash {
		t.Fatal("restored access code changed")
	}
	t.Log("PASS: independently restored 18 originals preserve EXIF/geometry, aliases, hidden state, named face, tag, complete receipts and access code; original state volume is absent")
}
