package web

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"hash"
	"immichclone/internal/library"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// Opt-in because the advertised boundary needs real multi-GiB disk and TCP I/O.
// No multi-GiB input buffer or retained fixture is created.
type importBoundaryReader struct {
	remaining, offset int64
	failAt            int64
	changedLast       bool
	digest            hash.Hash
	block             [64 << 10]byte
}

func newImportBoundaryReader(size int64) *importBoundaryReader {
	r := &importBoundaryReader{remaining: size, failAt: -1, digest: sha256.New()}
	for i := range r.block {
		r.block[i] = byte((i*29 + 17) % 251)
	}
	return r
}
func (r *importBoundaryReader) Read(p []byte) (int, error) {
	if r.failAt >= 0 && r.offset >= r.failAt {
		return 0, io.ErrUnexpectedEOF
	}
	if r.remaining == 0 {
		return 0, io.EOF
	}
	n := len(p)
	if n > len(r.block) {
		n = len(r.block)
	}
	if int64(n) > r.remaining {
		n = int(r.remaining)
	}
	if r.failAt >= 0 && int64(n) > r.failAt-r.offset {
		n = int(r.failAt - r.offset)
	}
	for written := 0; written < n; {
		at := int((r.offset + int64(written)) % int64(len(r.block)))
		count := copy(p[written:n], r.block[at:])
		written += count
	}
	if r.changedLast && int64(n) == r.remaining {
		p[n-1] ^= 1
	}
	r.digest.Write(p[:n])
	r.offset += int64(n)
	r.remaining -= int64(n)
	return n, nil
}
func TestImportExact4GiBHTTPBoundary(t *testing.T) {
	if os.Getenv("IMMICH_IMPORT_BOUNDARY") != "1" {
		t.Skip("opt-in: requires at least 12 GiB disposable free space and streams actual 4 GiB HTTP bodies")
	}
	if library.MaxUpload != 4<<30 {
		t.Fatal("boundary test needs deliberate update for changed product limit")
	}
	tempRoot := t.TempDir()
	if parent := os.Getenv("IMMICH_IMPORT_BOUNDARY_TEMP"); parent != "" {
		var e error
		tempRoot, e = os.MkdirTemp(parent, "immich-import-boundary-")
		if e != nil {
			t.Fatal(e)
		}
		t.Cleanup(func() { os.RemoveAll(tempRoot) })
	}
	root := filepath.Join(tempRoot, "state")
	store, e := library.Open(root)
	if e != nil {
		t.Fatal(e)
	}
	free, e := library.AvailableBytes(root)
	if e != nil || free < 12<<30 {
		store.Close()
		t.Fatalf("opt-in boundary test requires 12 GiB available, got %d bytes; err=%v", free, e)
	}
	code, e := AccessCode(root)
	if e != nil {
		t.Fatal(e)
	}
	server := httptest.NewServer(New(store, root, code))
	client := &http.Client{Timeout: 3 * time.Minute}
	var cookie *http.Cookie
	t.Cleanup(func() { server.Close(); store.Close() })
	call := func(method, url string, body io.Reader, length int64) (*http.Response, error) {
		req, e := http.NewRequest(method, server.URL+url, body)
		if e != nil {
			return nil, e
		}
		if length >= 0 {
			req.ContentLength = length
		}
		req.Header.Set("X-Immich-Request", "1")
		if cookie != nil {
			req.AddCookie(cookie)
		}
		return client.Do(req)
	}
	jsonCall := func(method, url string, body any, out any) int {
		var input io.Reader
		if body != nil {
			b, e := json.Marshal(body)
			if e != nil {
				t.Fatal(e)
			}
			input = bytes.NewReader(b)
		}
		r, e := call(method, url, input, -1)
		if e != nil {
			t.Fatal(e)
		}
		defer r.Body.Close()
		if out != nil {
			if e = json.NewDecoder(r.Body).Decode(out); e != nil {
				t.Fatal(e)
			}
		} else {
			io.Copy(io.Discard, r.Body)
		}
		return r.StatusCode
	}
	login := func() {
		r, e := call("POST", "/api/login", bytes.NewBufferString(`{"code":"`+code+`"}`), -1)
		if e != nil {
			t.Fatal(e)
		}
		io.Copy(io.Discard, r.Body)
		r.Body.Close()
		if r.StatusCode != 200 || len(r.Cookies()) != 1 {
			t.Fatal("boundary HTTP login failed")
		}
		cookie = r.Cookies()[0]
	}
	login()
	small := []byte("public synthetic accepted receipt before interrupted large body")
	files := []library.ImportFile{{Name: "retained.txt", Relative: "Boundary/retained.txt", Size: int64(len(small))}, {Name: "boundary.bin", Relative: "Boundary/boundary.bin", Size: library.MaxUpload}}
	var manifest library.ImportSession
	if status := jsonCall("POST", "/api/imports", map[string]any{"files": files}, &manifest); status != 200 {
		t.Fatal("manifest", status)
	}
	base := "/api/imports/" + manifest.ID
	if status := jsonCall("POST", base+"/start", nil, nil); status != 200 {
		t.Fatal(status)
	}
	var first library.ImportEntry
	r, e := call("POST", base+"/files/0", bytes.NewReader(small), int64(len(small)))
	if e != nil {
		t.Fatal(e)
	}
	if e = json.NewDecoder(r.Body).Decode(&first); e != nil {
		t.Fatal(e)
	}
	r.Body.Close()
	if r.StatusCode != 200 {
		t.Fatal("small receipt", r.StatusCode)
	}
	interrupted := newImportBoundaryReader(library.MaxUpload)
	interrupted.failAt = 8 << 20
	started := time.Now()
	r, e = call("POST", base+"/files/1", interrupted, library.MaxUpload)
	if r != nil {
		io.Copy(io.Discard, r.Body)
		r.Body.Close()
	}
	if e == nil && r != nil && r.StatusCode == 200 {
		t.Fatal("source read failure falsely accepted")
	}
	var page library.ImportPage
	deadline := time.Now().Add(15 * time.Second)
	for {
		if status := jsonCall("GET", base, nil, &page); status != 200 {
			t.Fatal(status)
		}
		if len(page.Entries) == 2 && page.Entries[1].Error != "" {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("interrupted body cleanup did not finish")
		}
		time.Sleep(20 * time.Millisecond)
	}
	if page.Session.AcceptedFiles != 1 || page.Session.AcceptedBytes != int64(len(small)) || page.Entries[1].AssetID != "" {
		t.Fatal("interrupted body changed accepted counters/receipt")
	}
	assertStagingEmpty := func() {
		items, e := os.ReadDir(filepath.Join(root, "staging"))
		if e != nil || len(items) != 0 {
			t.Fatal("partial temporary upload retained", e)
		}
	}
	assertStagingEmpty()
	t.Logf("interrupted source: sent=%d bytes, accepted large bytes=0, elapsed=%s", interrupted.offset, time.Since(started).Round(time.Millisecond))
	// A real listener/catalog reopen, not an in-memory status reset.
	server.Close()
	if e = store.Close(); e != nil {
		t.Fatal(e)
	}
	store, e = library.Open(root)
	if e != nil {
		t.Fatal(e)
	}
	server = httptest.NewServer(New(store, root, code))
	cookie = nil
	login()
	if status := jsonCall("GET", base, nil, &page); status != 200 {
		t.Fatal(status)
	}
	if page.Session.Status != "paused" || page.Session.AcceptedFiles != 1 || page.BackgroundPaused {
		t.Fatal("HTTP restart stranded copy lease/receipt")
	}
	if status := jsonCall("POST", base+"/start", nil, nil); status != 200 {
		t.Fatal(status)
	}
	send := func(source *importBoundaryReader) (library.ImportEntry, int) {
		r, e := call("POST", base+"/files/1", source, source.remaining)
		if e != nil {
			t.Fatal(e)
		}
		defer r.Body.Close()
		var entry library.ImportEntry
		if r.StatusCode == 200 {
			e = json.NewDecoder(r.Body).Decode(&entry)
		} else {
			_, e = io.Copy(io.Discard, r.Body)
		}
		if e != nil {
			t.Fatal(e)
		}
		return entry, r.StatusCode
	}
	source := newImportBoundaryReader(library.MaxUpload)
	started = time.Now()
	entry, status := send(source)
	if status != 200 {
		t.Fatal("exact boundary rejected", status)
	}
	expected := hex.EncodeToString(source.digest.Sum(nil))
	if entry.AssetID != expected || source.offset != library.MaxUpload {
		t.Fatal("exact-limit receipt/hash mismatch")
	}
	info, e := os.Stat(store.Original(entry.AssetID))
	if e != nil || info.Size() != library.MaxUpload {
		t.Fatal("exact-limit file not physically stored", e)
	}
	assertStagingEmpty()
	t.Logf("exact limit: streamed=%d bytes, stored=%d bytes, elapsed=%s", source.offset, info.Size(), time.Since(started).Round(time.Millisecond))
	verifyDownload := func() {
		r, e := call("GET", "/api/assets/"+entry.AssetID+"/original", nil, -1)
		if e != nil {
			t.Fatal(e)
		}
		defer r.Body.Close()
		h := sha256.New()
		n, e := io.Copy(h, r.Body)
		if e != nil || r.StatusCode != 200 || n != library.MaxUpload || hex.EncodeToString(h.Sum(nil)) != expected {
			t.Fatal("exact-limit original/download integrity", n, e)
		}
	}
	verifyDownload()
	started = time.Now()
	unchanged := newImportBoundaryReader(library.MaxUpload)
	again, status := send(unchanged)
	if status != 200 || again.AssetID != entry.AssetID {
		t.Fatal("exact-limit unchanged receipt replay failed", status)
	}
	assertStagingEmpty()
	t.Logf("unchanged replay: streamed=%d bytes, elapsed=%s", unchanged.offset, time.Since(started).Round(time.Millisecond))
	changed := newImportBoundaryReader(library.MaxUpload)
	changed.changedLast = true
	started = time.Now()
	_, status = send(changed)
	if status != 409 {
		t.Fatal("same-size final-byte source change accepted", status)
	}
	assertStagingEmpty()
	t.Logf("changed replay refused: streamed=%d bytes, elapsed=%s", changed.offset, time.Since(started).Round(time.Millisecond))
	oversize := newImportBoundaryReader(library.MaxUpload + 1)
	started = time.Now()
	_, status = send(oversize)
	if status != http.StatusRequestEntityTooLarge {
		t.Fatalf("actual 4 GiB + 1 HTTP body must report 413, got %d (source read %d bytes)", status, oversize.offset)
	}
	assertStagingEmpty()
	t.Logf("over-limit refused: source_read=%d bytes, elapsed=%s", oversize.offset, time.Since(started).Round(time.Millisecond))
	if status := jsonCall("GET", base, nil, &page); status != 200 {
		t.Fatal(status)
	}
	if page.Session.AcceptedFiles != 2 || page.Session.AcceptedBytes != library.MaxUpload+int64(len(small)) || page.Session.NewFiles != 2 || page.Session.Duplicates != 0 || page.Entries[1].AssetID != entry.AssetID {
		t.Fatal("replay/over-limit altered durable counters or receipt")
	}
	verifyDownload()
	// The manifest boundary is a separate early rejection; it does not replace actual body execution above.
	if status := jsonCall("POST", "/api/imports", map[string]any{"files": []library.ImportFile{{Name: "too-large.bin", Size: library.MaxUpload + 1}}}, nil); status != 400 {
		t.Fatal("over-limit manifest accepted", status)
	}
	if status := jsonCall("POST", base+"/finish", nil, nil); status != 200 {
		t.Fatal(status)
	}
	if e = store.Verify(); e != nil {
		t.Fatal(e)
	}
	t.Logf("PASS: exact 4 GiB actual HTTP storage/readback, interrupted source + listener/catalog restart, unchanged/hash-changed replay and +1-byte rejection; final accepted=%d bytes; no multi-GiB input buffer", page.Session.AcceptedBytes)
}
