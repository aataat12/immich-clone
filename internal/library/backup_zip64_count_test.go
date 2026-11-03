package library

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"

	bolt "go.etcd.io/bbolt"
)

const zip64CountOriginals = 65536

func zip64CountPayload(i int) []byte {
	b := make([]byte, 32)
	copy(b, "zip64-count-fixture")
	binary.LittleEndian.PutUint64(b[24:], uint64(i))
	return b
}
func zip64CountID(i int) string {
	h := sha256.Sum256(zip64CountPayload(i))
	return hex.EncodeToString(h[:])
}
func zip64CountImportID(i int) string {
	h := sha256.Sum256([]byte(fmt.Sprintf("zip64-count-import-%d", i)))
	return hex.EncodeToString(h[:])
}

// Opt-in: 65k actual filesystem objects and production ZIP/restore operations.
// Fixture seeding is deliberately not a claim of 65k browser/import execution.
func TestActualZIP64EntryCountBackupRestore(t *testing.T) {
	if os.Getenv("IMMICH_ZIP64_COUNT") != "1" {
		t.Skip("requires disposable 3 GiB-free environment and explicit ZIP64 count opt-in")
	}
	parent := os.Getenv("IMMICH_ZIP64_TEMP")
	if parent == "" {
		parent = t.TempDir()
	}
	free, err := AvailableBytes(parent)
	if err != nil || free < 3<<30 {
		t.Fatal("ZIP64 fixture requires 3 GiB disposable free capacity", err)
	}
	root, err := os.MkdirTemp(parent, "immich-zip64-count-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(root)
	ctx, cancel := context.WithTimeout(context.Background(), 14*time.Minute)
	defer cancel()
	started := time.Now()
	// All manifest values have fixed-length SHA256 strings. Bound its exact
	// encoded size before creating expensive filesystem fixtures.
	manifest := Manifest{Schema: 1, Files: map[string]string{"catalog.db": fmt.Sprintf("%064d", 0), "state-format.txt": fmt.Sprintf("%064d", 0), "access-code.txt": fmt.Sprintf("%064d", 0)}}
	expected := make(map[string]int, zip64CountOriginals)
	for i := 0; i < zip64CountOriginals; i++ {
		id := zip64CountID(i)
		expected[id] = i
		manifest.Files["originals/"+id] = id
	}
	encoded, err := json.Marshal(manifest)
	if err != nil || len(encoded)+1 > 16<<20 || len(expected) != zip64CountOriginals {
		t.Fatal("fixture manifest exceeds unchanged default budget", len(encoded), err)
	}
	t.Logf("Preflight: %d distinct 32-byte originals, %d archive entries, %d manifest bytes; default restore limits retained", zip64CountOriginals, zip64CountOriginals+4, len(encoded)+1)
	manifest.Files = nil
	encoded = nil
	state := filepath.Join(root, "state")
	s, err := Open(state)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if s != nil {
			s.Close()
		}
	}()
	secret := []byte("1234567890abcdef1234567890abcdef\n")
	if err = os.WriteFile(filepath.Join(state, "access-code.txt"), secret, 0600); err != nil {
		t.Fatal(err)
	}
	stamp := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	// Two completed receipt sessions respect the 50,000-per-selection limit.
	if err = s.db.Update(func(tx *bolt.Tx) error {
		for session := 0; session < 2; session++ {
			id := zip64CountImportID(session)
			b, e := tx.Bucket([]byte(importBucket)).CreateBucket([]byte(id))
			if e != nil {
				return e
			}
			m := ImportSession{ID: id, Status: "completed", Created: stamp, TotalFiles: 32768, AcceptedFiles: 32768, NewFiles: 32768, TotalBytes: 32768 * 32, AcceptedBytes: 32768 * 32}
			if e = writeJSON(b, "meta", m); e != nil {
				return e
			}
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	for start := 0; start < zip64CountOriginals; start += 1024 {
		if err = ctx.Err(); err != nil {
			t.Fatal(err)
		}
		for i := start; i < start+1024; i++ {
			id := zip64CountID(i)
			file, e := os.OpenFile(filepath.Join(state, "originals", id), os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
			if e != nil {
				t.Fatal(e)
			}
			_, e = file.Write(zip64CountPayload(i))
			closeErr := file.Close()
			if e != nil || closeErr != nil {
				t.Fatal("fixture blob write", e, closeErr)
			}
		}
		if err = s.db.Update(func(tx *bolt.Tx) error {
			for i := start; i < start+1024; i++ {
				id := zip64CountID(i)
				name := fmt.Sprintf("entry-%05d.bin", i)
				a := Asset{ID: id, Name: name, Size: 32, MIME: "application/octet-stream", Added: stamp.Add(time.Duration(i) * time.Millisecond), Hidden: i%1024 == 0, Preview: "failed", PreviewRevision: 3, Error: "Synthetic fixture without preview", Sources: []string{"Zip64Synthetic/" + name}}
				if e := saveAsset(tx, a); e != nil {
					return e
				}
				entry := ImportEntry{ImportFile: ImportFile{Name: name, Relative: "Zip64Synthetic/" + name, Size: 32, Modified: stamp.UnixMilli()}, Index: i % 32768, AssetID: id}
				if e := writeJSON(tx.Bucket([]byte(importBucket)).Bucket([]byte(zip64CountImportID(i/32768))), entryKey(entry.Index), entry); e != nil {
					return e
				}
			}
			return nil
		}); err != nil {
			t.Fatal(err)
		}
		if (start+1024)%16384 == 0 {
			t.Logf("Fixture seeded: %d originals", start+1024)
		}
	}
	t.Logf("Fixture construction: %s", time.Since(started))
	plan, err := s.BackupPlan(ctx)
	if err != nil || !plan.Fits || plan.Originals != zip64CountOriginals || plan.Files != zip64CountOriginals+3 || plan.OriginalBytes != zip64CountOriginals*32 {
		t.Fatal("unchanged backup preflight refused or miscounted fixture", plan, err)
	}
	if err = os.MkdirAll(filepath.Join(state, "backups"), 0700); err != nil {
		t.Fatal(err)
	}
	archive := filepath.Join(state, "backups", "count.zip")
	var progress BackupProgress
	backupStarted := time.Now()
	if err = s.StagedBackup(ctx, archive, func(p BackupProgress) { progress = p }); err != nil {
		t.Fatal("actual staged backup", err)
	}
	if progress.Files != zip64CountOriginals+3 || progress.Bytes != progress.TotalBytes {
		t.Fatal("backup progress incomplete", progress)
	}
	info, err := os.Stat(archive)
	if err != nil || info.Size() >= 4<<30 {
		t.Fatal("fixture is not a bounded sub-4GiB archive", err)
	}
	t.Logf("Actual staged archive: %d bytes; backup duration %s; declared staging bound %d", info.Size(), time.Since(backupStarted), plan.ArchiveUpperBound)
	verifyStarted := time.Now()
	if err = VerifyBackup(ctx, archive, nil); err != nil {
		t.Fatal("actual archive readback", err)
	}
	t.Logf("Full archive checksum readback: %s", time.Since(verifyStarted))
	validateZIP64CountStructure(t, archive, zip64CountOriginals+4)
	if err = s.Close(); err != nil {
		t.Fatal(err)
	}
	s = nil
	standalone := filepath.Join(root, "entry-count.zip")
	if err = os.Rename(archive, standalone); err != nil {
		t.Fatal(err)
	}
	if err = os.Rename(state, state+".offline"); err != nil {
		t.Fatal(err)
	}
	if _, err = os.Stat(state); !os.IsNotExist(err) {
		t.Fatal("source state remains at its original path", err)
	}
	restoredPath := filepath.Join(root, "independent-restored")
	restoreStarted := time.Now()
	if err = RestoreWithLimits(ctx, standalone, restoredPath, RestoreLimits{}); err != nil {
		t.Fatal("independent default-budget restore", err)
	}
	t.Logf("Independent default-budget restore: %s", time.Since(restoreStarted))
	r, err := Open(restoredPath)
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	assets, err := r.List()
	if err != nil || len(assets) != zip64CountOriginals {
		t.Fatal("restored original count", len(assets), err)
	}
	for _, a := range assets {
		i, ok := expected[a.ID]
		if !ok {
			t.Fatal("unexpected restored ID")
		}
		b, e := os.ReadFile(r.Original(a.ID))
		if e != nil || !bytes.Equal(b, zip64CountPayload(i)) {
			t.Fatal("restored original bytes", i, e)
		}
		if a.Name != fmt.Sprintf("entry-%05d.bin", i) || a.Hidden != (i%1024 == 0) || len(a.Sources) != 1 || a.Sources[0] != "Zip64Synthetic/"+a.Name {
			t.Fatal("restored catalog metadata", i)
		}
	}
	if b, e := os.ReadFile(filepath.Join(restoredPath, "access-code.txt")); e != nil || !bytes.Equal(b, secret) {
		t.Fatal("restored required secret", e)
	}
	history, err := r.ImportHistory()
	if err != nil || len(history) != 2 {
		t.Fatal("restored receipt history", err)
	}
	for _, m := range history {
		if m.Status != "completed" || m.AcceptedFiles != 32768 || m.NewFiles != 32768 || m.Duplicates != 0 || m.AcceptedBytes != 32768*32 {
			t.Fatal("restored import counters", m.ID)
		}
		count := 0
		for after := 0; after >= 0; {
			p, e := r.ImportPage(m.ID, after)
			if e != nil {
				t.Fatal(e)
			}
			for _, entry := range p.Entries {
				index, ok := expected[entry.AssetID]
				if !ok || entry.Index != index%32768 || entry.Name != fmt.Sprintf("entry-%05d.bin", index) || entry.Size != 32 || entry.Error != "" || entry.Deleted || entry.Duplicate {
					t.Fatal("restored receipt mismatch", entry.Index)
				}
				count++
			}
			after = p.Next
		}
		if count != 32768 {
			t.Fatal("restored receipt count", count)
		}
	}
	page, err := r.GalleryPage("name", "library", "", 60)
	if err != nil || page.Total != zip64CountOriginals || page.Hidden != 64 || page.Count != zip64CountOriginals-64 {
		t.Fatal("restored gallery indexes", page, err)
	}
	t.Logf("PASS: all %d restored original byte/hash identities and catalog records, two receipt sessions, all receipts, secret and rebuilt gallery counts; total %s", zip64CountOriginals, time.Since(started))
}

func validateZIP64CountStructure(t *testing.T, path string, want uint64) {
	t.Helper()
	f, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil {
		t.Fatal(err)
	}
	end := make([]byte, 42)
	if _, err = f.ReadAt(end, info.Size()-int64(len(end))); err != nil {
		t.Fatal(err)
	}
	locator, eocd := end[:20], end[20:]
	if binary.LittleEndian.Uint32(eocd) != 0x06054b50 || binary.LittleEndian.Uint16(eocd[8:]) != 65535 || binary.LittleEndian.Uint16(eocd[10:]) != 65535 || binary.LittleEndian.Uint16(eocd[20:]) != 0 {
		t.Fatal("legacy EOCD did not mark ZIP64 entry count")
	}
	if binary.LittleEndian.Uint32(locator) != 0x07064b50 || binary.LittleEndian.Uint32(locator[4:]) != 0 || binary.LittleEndian.Uint32(locator[16:]) != 1 {
		t.Fatal("ZIP64 locator missing or invalid")
	}
	offset := binary.LittleEndian.Uint64(locator[8:])
	record := make([]byte, 56)
	if offset > uint64(info.Size()-98) {
		t.Fatal("ZIP64 record points outside archive")
	}
	if _, err = f.ReadAt(record, int64(offset)); err != nil {
		t.Fatal(err)
	}
	if binary.LittleEndian.Uint32(record) != 0x06064b50 || binary.LittleEndian.Uint64(record[4:]) != 44 || binary.LittleEndian.Uint64(record[24:]) != want || binary.LittleEndian.Uint64(record[32:]) != want {
		t.Fatal("ZIP64 EOCD count invalid")
	}
	z, err := zip.OpenReader(path)
	if err != nil {
		t.Fatal(err)
	}
	defer z.Close()
	if uint64(len(z.File)) != want {
		t.Fatal("ZIP64 reader count mismatch", len(z.File))
	}
	var metadata uint64
	manifestSeen := false
	for _, entry := range z.File {
		metadata += 46 + uint64(len(entry.Name)+len(entry.Extra)+len(entry.Comment))
		if entry.Name == "manifest.json" {
			if entry.UncompressedSize64 > 16<<20 {
				t.Fatal("actual manifest exceeds default bound")
			}
			in, e := entry.Open()
			if e != nil {
				t.Fatal(e)
			}
			var m Manifest
			e = json.NewDecoder(io.LimitReader(in, 16<<20)).Decode(&m)
			closeErr := in.Close()
			if e != nil || closeErr != nil || len(m.Files) != int(want)-1 {
				t.Fatal("actual manifest mismatch", e, closeErr)
			}
			manifestSeen = true
		}
	}
	if !manifestSeen || metadata > 64<<20 {
		t.Fatal("actual archive metadata exceeds default restore bound", metadata)
	}
	t.Logf("ZIP64 binary structures validated: %d central entries; metadata %d bytes", want, metadata)
}
