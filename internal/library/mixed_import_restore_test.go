package library

import (
	"context"
	"immichclone/internal/media"
	"os"
	"testing"
)

func TestMixedImportArchiveRestored(t *testing.T) {
	root := os.Getenv("IMMICH_MIXED_RESTORE")
	if root == "" {
		t.Skip("requires actual mixed-import service archive")
	}
	s, err := Open(root)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	if err = s.Verify(); err != nil {
		t.Fatal(err)
	}
	history, err := s.ImportHistory()
	if err != nil || len(history) != 1 {
		t.Fatal("restored import history", history, err)
	}
	m := history[0]
	if m.Status != "completed" || m.AcceptedFiles != 240 || m.NewFiles != 192 || m.Duplicates != 48 || m.AcceptedBytes != m.TotalBytes {
		t.Fatal("restored counters", m)
	}
	var accepted int64
	for after := 0; after >= 0; {
		p, err := s.ImportPage(m.ID, after)
		if err != nil {
			t.Fatal(err)
		}
		for _, e := range p.Entries {
			if e.AssetID == "" || e.Error != "" || e.Duplicate != (e.Index >= 192) {
				t.Fatal("restored receipt", e.Index)
			}
			accepted += e.Size
		}
		after = p.Next
	}
	if accepted != m.TotalBytes {
		t.Fatal("restored receipt byte total")
	}
	decoder := media.New(os.Getenv("IMMICH_MIXED_MEDIA"))
	if !decoder.Available() {
		t.Fatal("accepted media provider unavailable")
	}
	s.SetMediaPreview(decoder.Thumbnail)
	for i := 0; i < 193; i++ {
		processed, err := s.ProcessOneContext(context.Background())
		if err != nil {
			t.Fatal(err)
		}
		if !processed {
			break
		}
	}
	assets, err := s.List()
	if err != nil || len(assets) != 192 {
		t.Fatal("restored original count", len(assets), err)
	}
	counts := map[string]int{}
	for _, a := range assets {
		if a.Preview != "ready" || a.Width <= 0 || a.Height <= 0 || len(a.Sources) < 1 {
			t.Fatal("independent restored preview/alias", a.ID, a.Preview)
		}
		counts[a.MIME]++
	}
	for _, mime := range []string{"image/png", "image/heic", "video/mp4", "video/quicktime"} {
		if counts[mime] != 48 {
			t.Fatal("restored media mix", counts)
		}
	}
}
