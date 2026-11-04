package web

import (
	"bytes"
	"context"
	"immichclone/internal/library"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

// The opt-in boundary test executes the real 4 GiB + 1 body. This smaller
// regression carries the same net/http size error after a partial read so normal
// CI checks the response and receipt contract without multi-GiB disk I/O.
type importSizeErrorReader struct{ read bool }

func (r *importSizeErrorReader) Read(p []byte) (int, error) {
	if !r.read {
		r.read = true
		return copy(p, []byte("partial")), nil
	}
	return 0, &http.MaxBytesError{Limit: library.MaxUpload}
}
func TestImportBodySizeErrorReturns413WithoutAcceptingPartial(t *testing.T) {
	s, c := setup(t)
	old := []byte("synthetic earlier committed original")
	m, e := s.Store.CreateImport([]library.ImportFile{{Name: "old.txt", Size: int64(len(old))}, {Name: "oversized.bin", Size: 100}})
	if e != nil {
		t.Fatal(e)
	}
	if e = s.Store.ControlImport(m.ID, "start"); e != nil {
		t.Fatal(e)
	}
	first, e := s.Store.PutImport(context.Background(), m.ID, 0, bytes.NewReader(old))
	if e != nil {
		t.Fatal(e)
	}
	req := httptest.NewRequest("POST", "http://127.0.0.1:2283/api/imports/"+m.ID+"/files/1", io.NopCloser(&importSizeErrorReader{}))
	req.AddCookie(c)
	req.Header.Set("X-Immich-Request", "1")
	w := httptest.NewRecorder()
	s.ServeHTTP(w, req)
	if w.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("oversized body returned %d instead of 413", w.Code)
	}
	page, e := s.Store.ImportPage(m.ID, 0)
	if e != nil {
		t.Fatal(e)
	}
	if page.Session.AcceptedFiles != 1 || page.Session.AcceptedBytes != int64(len(old)) || page.Entries[0].AssetID != first.AssetID || page.Entries[1].AssetID != "" || page.Entries[1].Error == "" {
		t.Fatal("size failure altered durable receipt/counters")
	}
	got, e := os.ReadFile(s.Store.Original(first.AssetID))
	if e != nil || !bytes.Equal(got, old) {
		t.Fatal("size failure changed earlier original")
	}
	items, e := os.ReadDir(filepath.Join(s.Root, "staging"))
	if e != nil || len(items) != 0 {
		t.Fatal("partial upload retained after size failure", e)
	}
}
