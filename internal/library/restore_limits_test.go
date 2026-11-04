package library

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"hash/crc32"
	"io"
	"math"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"
)

type restoreLimitFixture struct {
	archive             []byte
	source, asset, face string
	declared            uint64
}

func newRestoreLimitFixture(t *testing.T, large bool) restoreLimitFixture {
	t.Helper()
	s, root := openTest(t)
	var input io.Reader = bytes.NewReader(fixture(t))
	name := "synthetic.png"
	if large {
		input = io.LimitReader(restoreZeroReader{}, 16<<20)
		name = "compressible.bin"
	}
	a, _, e := s.Put(name, input)
	if e != nil {
		t.Fatal("fixture original failed")
	}
	f := restoreLimitFixture{source: root, asset: a.ID}
	if !large {
		s.SetRecognizer(fakeAnalysis, true)
		if e = s.QueueAnalysis(a.ID); e != nil {
			t.Fatal("fixture analysis queue failed")
		}
		if _, e = s.ProcessAnalysis(context.Background()); e != nil {
			t.Fatal("fixture analysis failed")
		}
		page, e := s.OrganizationPage("", "", "")
		if e != nil || len(page.Faces) != 1 {
			t.Fatal("fixture face missing")
		}
		f.face = page.Faces[0].ID
		for _, name := range []string{"Prior name", "Corrected name"} {
			if e = s.LabelFaces([]string{f.face}, name); e != nil {
				t.Fatal("fixture correction failed")
			}
		}
		if e = s.DismissFace(f.face, false); e != nil {
			t.Fatal("fixture dismissal failed")
		}
		if e = s.SetTags(a.ID, []string{"Keep tag"}); e != nil {
			t.Fatal("fixture tag failed")
		}
	}
	if e = os.WriteFile(filepath.Join(root, "access-code.txt"), []byte("synthetic-restore-code"), 0600); e != nil {
		t.Fatal("fixture code failed")
	}
	var archive bytes.Buffer
	if e = s.Backup(&archive); e != nil {
		t.Fatal("fixture backup failed")
	}
	f.archive = archive.Bytes()
	zr, e := zip.NewReader(bytes.NewReader(f.archive), int64(len(f.archive)))
	if e != nil {
		t.Fatal("fixture ZIP failed")
	}
	for _, entry := range zr.File {
		f.declared += entry.UncompressedSize64
	}
	if e = s.Close(); e != nil {
		t.Fatal("fixture close failed")
	}
	return f
}

type restoreZeroReader struct{}

func (restoreZeroReader) Read(p []byte) (int, error) { clear(p); return len(p), nil }

func restoreLimitArchive(t *testing.T, data []byte) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "candidate.zip")
	if e := os.WriteFile(p, data, 0600); e != nil {
		t.Fatal("fixture archive write failed")
	}
	return p
}
func assertRestoreRejected(t *testing.T, archive string, limits RestoreLimits) {
	t.Helper()
	parent := t.TempDir()
	sentinel := filepath.Join(parent, "sentinel.bin")
	if e := os.WriteFile(sentinel, []byte("untouched"), 0600); e != nil {
		t.Fatal("fixture sentinel failed")
	}
	target := filepath.Join(parent, "restored")
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	started := time.Now()
	if e := RestoreWithLimits(ctx, archive, target, limits); e == nil {
		t.Fatal("bounded restore accepted hostile/budget-excess archive")
	}
	if time.Since(started) > 7*time.Second {
		t.Fatal("restore failed to honor bounded rejection")
	}
	if _, e := os.Lstat(target); !errors.Is(e, os.ErrNotExist) {
		t.Fatal("failed restore published target")
	}
	entries, e := os.ReadDir(parent)
	if e != nil || len(entries) != 1 || entries[0].Name() != "sentinel.bin" {
		t.Fatal("failed restore retained extraction state")
	}
	data, e := os.ReadFile(sentinel)
	if e != nil || string(data) != "untouched" {
		t.Fatal("restore modified outside sentinel")
	}
}

func rewriteRestoreCatalog(t *testing.T, original []byte, edit func([]byte) []byte) []byte {
	t.Helper()
	zr, e := zip.NewReader(bytes.NewReader(original), int64(len(original)))
	if e != nil {
		t.Fatal("fixture ZIP read failed")
	}
	entries := map[string][]byte{}
	var order []string
	for _, f := range zr.File {
		r, e := f.Open()
		if e != nil {
			t.Fatal("fixture entry open failed")
		}
		data, e := io.ReadAll(r)
		r.Close()
		if e != nil {
			t.Fatal("fixture entry read failed")
		}
		entries[f.Name] = data
		order = append(order, f.Name)
	}
	entries["catalog.db"] = edit(entries["catalog.db"])
	var manifest Manifest
	if e = json.Unmarshal(entries["manifest.json"], &manifest); e != nil {
		t.Fatal("fixture manifest failed")
	}
	h := sha256.Sum256(entries["catalog.db"])
	manifest.Files["catalog.db"] = hex.EncodeToString(h[:])
	entries["manifest.json"], e = json.Marshal(manifest)
	if e != nil {
		t.Fatal("fixture manifest encoding failed")
	}
	var out bytes.Buffer
	zw := zip.NewWriter(&out)
	for _, name := range order {
		w, e := zw.Create(name)
		if e != nil {
			t.Fatal("fixture ZIP rewrite failed")
		}
		if _, e = w.Write(entries[name]); e != nil {
			t.Fatal("fixture entry write failed")
		}
	}
	if e = zw.Close(); e != nil {
		t.Fatal("fixture ZIP close failed")
	}
	return out.Bytes()
}

func TestRestoreLimitsAggregateCompressionAndIndependentRecovery(t *testing.T) {
	f := newRestoreLimitFixture(t, false)
	archive := restoreLimitArchive(t, f.archive)
	for _, tc := range []struct {
		name   string
		limits RestoreLimits
	}{
		{"aggregate", RestoreLimits{MaxBytes: f.declared - 1}},
		{"entries", RestoreLimits{MaxEntries: 2}},
		{"manifest", RestoreLimits{MaxManifestBytes: 1}},
		{"catalog", RestoreLimits{MaxCatalogBytes: 1024}},
		{"metadata", RestoreLimits{MaxMetadataBytes: 32}},
		{"negative deadline", RestoreLimits{Timeout: -time.Second}},
		{"oversized aggregate setting", RestoreLimits{MaxBytes: math.MaxUint64}},
		{"overflow entry setting", RestoreLimits{MaxEntries: math.MaxUint64}},
		{"overflow manifest setting", RestoreLimits{MaxManifestBytes: math.MaxUint64}},
		{"overflow catalog setting", RestoreLimits{MaxCatalogBytes: math.MaxUint64}},
		{"overflow memory setting", RestoreLimits{MemoryBytes: math.MaxUint64}},
	} {
		t.Run(tc.name, func(t *testing.T) { assertRestoreRejected(t, archive, tc.limits) })
	}
	// Restore must use the archive, never source files or an old live database.
	if e := os.Rename(f.source, f.source+"-unavailable"); e != nil {
		t.Fatal("fixture source isolation failed")
	}
	target := filepath.Join(t.TempDir(), "independent")
	if e := RestoreWithLimits(context.Background(), archive, target, RestoreLimits{MaxBytes: f.declared + (4 << 20)}); e != nil {
		t.Fatal("valid bounded old-format restore failed")
	}
	restored, e := Open(target)
	if e != nil {
		t.Fatal("restored catalog unavailable")
	}
	defer restored.Close()
	if e = restored.Verify(); e != nil {
		t.Fatal("restored original integrity failed")
	}
	page, e := restored.OrganizationReviewPage("", "", "", true)
	if e != nil || len(page.Faces) != 1 || page.Faces[0].Name != "Corrected name" || !page.Faces[0].CanUndo || !page.Faces[0].Dismissed {
		t.Fatal("bounded restore lost name/history/dismissal")
	}
	if len(page.Assets) != 1 || len(page.Assets[0].Tags) != 1 || page.Assets[0].Tags[0] != "Keep tag" || page.Assets[0].Analysis.Status != "ready" {
		t.Fatal("bounded restore lost tags/ready result")
	}
	if e = restored.DismissFace(f.face, true); e != nil {
		t.Fatal("restored face unavailable")
	}
	if e = restored.UndoFace(f.face); e != nil {
		t.Fatal("restored undo unavailable")
	}
	code, e := os.ReadFile(filepath.Join(target, "access-code.txt"))
	if e != nil || string(code) != "synthetic-restore-code" {
		t.Fatal("bounded restore lost access code")
	}
	big := newRestoreLimitFixture(t, true)
	if uint64(len(big.archive))*100 >= big.declared {
		t.Fatal("compressible fixture did not exercise expansion ratio")
	}
	bigArchive := restoreLimitArchive(t, big.archive)
	assertRestoreRejected(t, bigArchive, RestoreLimits{MaxBytes: 1 << 20})
	bigTarget := filepath.Join(t.TempDir(), "expanded-valid")
	if e := RestoreWithLimits(context.Background(), bigArchive, bigTarget, RestoreLimits{MaxBytes: big.declared + (4 << 20)}); e != nil {
		t.Fatal("valid high-expansion backup refused within explicit budget")
	}
	largeRestored, e := Open(bigTarget)
	if e != nil {
		t.Fatal("expanded restore unavailable")
	}
	defer largeRestored.Close()
	if e = largeRestored.Verify(); e != nil {
		t.Fatal("expanded original integrity failed")
	}
}

func TestRestoreLimitsCorruptAndCyclicCatalogCleanup(t *testing.T) {
	f := newRestoreLimitFixture(t, false)
	for _, tc := range []struct {
		name string
		edit func([]byte) []byte
	}{
		{"invalid database", func([]byte) []byte { return []byte("not a database") }},
		{"cyclic root branch", func(data []byte) []byte {
			out := append([]byte(nil), data...)
			if len(out) < 8192 {
				t.Fatal("catalog fixture too small")
			}
			ps := int(binary.LittleEndian.Uint32(out[24:28]))
			if ps < 1024 || ps > len(out)/2 {
				t.Fatal("catalog fixture page size invalid")
			}
			meta := 0
			if binary.LittleEndian.Uint64(out[ps+64:ps+72]) > binary.LittleEndian.Uint64(out[64:72]) {
				meta = ps
			}
			root := binary.LittleEndian.Uint64(out[meta+32 : meta+40])
			offset := int(root) * ps
			if root < 2 || offset < 0 || offset+33 > len(out) {
				t.Fatal("catalog fixture root invalid")
			}
			binary.LittleEndian.PutUint16(out[offset+8:offset+10], 1)
			binary.LittleEndian.PutUint16(out[offset+10:offset+12], 1)
			binary.LittleEndian.PutUint32(out[offset+12:offset+16], 0)
			binary.LittleEndian.PutUint32(out[offset+16:offset+20], 16)
			binary.LittleEndian.PutUint32(out[offset+20:offset+24], 1)
			binary.LittleEndian.PutUint64(out[offset+24:offset+32], root)
			out[offset+32] = 'x'
			return out
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			bad := rewriteRestoreCatalog(t, f.archive, tc.edit)
			assertRestoreRejected(t, restoreLimitArchive(t, bad), RestoreLimits{Timeout: 1500 * time.Millisecond})
		})
	}
	// Extreme ZIP64 size declarations must not wrap a cumulative byte counter.
	var raw bytes.Buffer
	zw := zip.NewWriter(&raw)
	h := zip.FileHeader{Name: "catalog.db", Method: zip.Store, UncompressedSize64: math.MaxUint64, CompressedSize64: 0, CRC32: crc32.ChecksumIEEE(nil)}
	if _, e := zw.CreateRaw(&h); e != nil {
		t.Fatal("ZIP64 fixture creation failed")
	}
	if e := zw.Close(); e != nil {
		t.Fatal("ZIP64 fixture close failed")
	}
	assertRestoreRejected(t, restoreLimitArchive(t, raw.Bytes()), RestoreLimits{})
}

func TestRestoreLimitsCancellationDoesNotPublish(t *testing.T) {
	f := newRestoreLimitFixture(t, false)
	archive := restoreLimitArchive(t, f.archive)
	parent := t.TempDir()
	target := filepath.Join(parent, "restored")
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if e := RestoreWithLimits(ctx, archive, target, RestoreLimits{}); e == nil {
		t.Fatal("cancelled restore succeeded")
	}
	entries, e := os.ReadDir(parent)
	if e != nil || len(entries) != 0 {
		t.Fatal("cancelled restore retained state or target")
	}
}

func TestRestoreBudgetWriterEnforcesActualCopiedBytes(t *testing.T) {
	var sink bytes.Buffer
	w := &restoreBudgetWriter{Writer: &sink, Remaining: 4096}
	n, e := io.Copy(w, io.LimitReader(restoreZeroReader{}, 4096))
	if e != nil || n != 4096 || sink.Len() != 4096 || w.Remaining != 0 {
		t.Fatal("copy byte guard failed exact-budget positive control")
	}
	n, e = io.Copy(w, io.LimitReader(restoreZeroReader{}, 8192))
	if e == nil || n != 0 || sink.Len() != 4096 || w.Remaining != 0 {
		t.Fatal("copy byte guard wrote beyond aggregate budget")
	}
}

func TestRestoreBoundedHelper(t *testing.T) {
	index := -1
	for i, arg := range os.Args {
		if arg == "restore-test-helper" {
			index = i
			break
		}
	}
	if index < 0 {
		return
	}
	if len(os.Args) < index+3 {
		os.Exit(2)
	}
	mode, root := os.Args[index+1], os.Args[index+2]
	if os.WriteFile(filepath.Join(root, "started"), []byte("started"), 0600) != nil {
		os.Exit(2)
	}
	switch mode {
	case "success":
		os.Stdout.Write([]byte("bounded-control"))
		os.Exit(0)
	case "hang":
		for {
			time.Sleep(time.Second)
		}
	case "flood":
		for {
			if _, e := os.Stdout.Write(bytes.Repeat([]byte("x"), 8192)); e != nil {
				os.Exit(3)
			}
		}
	case "memory-small", "memory-large":
		size := 32 << 20
		if mode == "memory-large" {
			size = 600 << 20
		}
		data := make([]byte, size)
		for i := 0; i < len(data); i += 4096 {
			data[i] = 1
		}
		runtime.KeepAlive(data)
		os.Exit(0)
	default:
		os.Exit(2)
	}
}

func TestRestoreSupervisorHardStopAndOutputBound(t *testing.T) {
	exe, e := os.Executable()
	if e != nil {
		t.Fatal("test executable unavailable")
	}
	for _, tc := range []struct {
		mode      string
		timeout   time.Duration
		wantError bool
	}{{"success", 3 * time.Second, false}, {"hang", time.Second, true}, {"flood", 3 * time.Second, true}} {
		t.Run(tc.mode, func(t *testing.T) {
			root := t.TempDir()
			ctx, cancel := context.WithTimeout(context.Background(), tc.timeout)
			defer cancel()
			var pid uint32
			started := time.Now()
			result, e := runRestoreProcess(ctx, exe, []string{"-test.run=^TestRestoreBoundedHelper$", "--", "restore-test-helper", tc.mode, root}, 512<<20, 4096, func(p uint32) { pid = p })
			if (e != nil) != tc.wantError || pid == 0 {
				t.Fatal("supervisor control outcome incorrect")
			}
			if !result.Reaped || result.PID != pid || len(result.Output) > 4096 {
				t.Fatal("supervisor did not confirm reaping or bounded output")
			}
			if tc.mode == "success" && string(result.Output) != "bounded-control" {
				t.Fatal("supervisor positive control did not execute helper")
			}
			if tc.mode == "hang" && !errors.Is(e, context.DeadlineExceeded) {
				t.Fatal("noncooperative helper did not hit hard deadline")
			}
			if tc.mode == "flood" && !errors.Is(e, ErrRestoreOutputLimit) {
				t.Fatal("output flood did not trigger explicit bounded-output failure")
			}
			if time.Since(started) > 5*time.Second {
				t.Fatal("supervisor failed hard-stop time bound")
			}
			if _, e := os.Stat(filepath.Join(root, "started")); e != nil {
				t.Fatal("child did not reach noncooperative helper body")
			}
			if process, e := os.FindProcess(int(pid)); e == nil {
				if e = process.Kill(); e == nil {
					t.Fatal("supervisor returned with child still running")
				}
				process.Release()
			}
		})
	}
}

func TestRestoreWindowsHardMemoryBound(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Windows Job Object memory evidence requires actual Windows execution")
	}
	exe, e := os.Executable()
	if e != nil {
		t.Fatal("test executable unavailable")
	}
	for _, mode := range []string{"memory-small", "memory-large"} {
		t.Run(mode, func(t *testing.T) {
			root := t.TempDir()
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			_, e := runRestoreProcess(ctx, exe, []string{"-test.run=^TestRestoreBoundedHelper$", "--", "restore-test-helper", mode, root}, 512<<20, 4096, nil)
			if mode == "memory-small" && e != nil {
				t.Fatal("small-memory control refused")
			}
			if mode == "memory-large" && e == nil {
				t.Fatal("hard memory budget failed to refuse excess allocation")
			}
			if _, e := os.Stat(filepath.Join(root, "started")); e != nil {
				t.Fatal("memory child never reached allocation body")
			}
		})
	}
}

func TestRestoreSupervisorExplicitCancellationReapsWorker(t *testing.T) {
	exe, e := os.Executable()
	if e != nil {
		t.Fatal("test executable unavailable")
	}
	root := t.TempDir()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	observed := make(chan bool, 1)
	go func() {
		for i := 0; i < 300; i++ {
			if _, e := os.Stat(filepath.Join(root, "started")); e == nil {
				observed <- true
				cancel()
				return
			}
			time.Sleep(10 * time.Millisecond)
		}
		observed <- false
		cancel()
	}()
	var pid uint32
	started := time.Now()
	_, e = runRestoreProcess(ctx, exe, []string{"-test.run=^TestRestoreBoundedHelper$", "--", "restore-test-helper", "hang", root}, 512<<20, 4096, func(p uint32) { pid = p })
	if !errors.Is(e, context.Canceled) || !<-observed || pid == 0 || time.Since(started) > 5*time.Second {
		t.Fatal("explicit cancellation did not terminate started noncooperative worker")
	}
	if process, e := os.FindProcess(int(pid)); e == nil {
		if e = process.Kill(); e == nil {
			t.Fatal("cancelled worker remained running")
		}
		process.Release()
	}
}

func TestRestoreBudgetArithmeticCannotWrap(t *testing.T) {
	if got, e := restoreAddWithin(5, 5, 10); e != nil || got != 10 {
		t.Fatal("budget arithmetic exact-bound positive failed")
	}
	for _, tc := range []struct{ total, n, limit uint64 }{{5, 6, 10}, {math.MaxUint64 - 4, 8, math.MaxUint64}, {0, math.MaxUint64, 1024}} {
		got, e := restoreAddWithin(tc.total, tc.n, tc.limit)
		if !errors.Is(e, ErrRestoreBudget) || got != tc.total {
			t.Fatal("budget arithmetic wrapped or accepted overflow")
		}
	}
}

func TestRestoreLimitsLiveGrowthCancellation(t *testing.T) {
	stage := t.TempDir()
	if e := writeRestoreProgress(stage, "synthetic-nonce", 100); e != nil {
		t.Fatal("progress fixture failed")
	}
	catalog := filepath.Join(stage, "catalog.db")
	if e := os.WriteFile(catalog, make([]byte, 100), 0600); e != nil {
		t.Fatal("catalog fixture failed")
	}
	limits := RestoreLimits{MaxBytes: 1024, MaxCatalogBytes: 512}
	if e := checkRestoreCatalogProgress(stage, "synthetic-nonce", limits); e != nil {
		t.Fatal("within-budget growth control refused")
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan error, 1)
	go func() { done <- monitorRestoreGrowth(ctx, stage, "synthetic-nonce", limits, cancel) }()
	if e := os.Truncate(catalog, 513); e != nil {
		t.Fatal("catalog growth fixture failed")
	}
	select {
	case e := <-done:
		if !errors.Is(e, ErrRestoreBudget) || ctx.Err() != context.Canceled {
			t.Fatal("live catalog excess did not cancel worker")
		}
	case <-time.After(time.Second):
		t.Fatal("live growth cancellation exceeded bounded interval")
	}
}

func TestRestoreLimitsCancellationAfterStageCleanup(t *testing.T) {
	f := newRestoreLimitFixture(t, true)
	archive := restoreLimitArchive(t, f.archive)
	parent := t.TempDir()
	target := filepath.Join(parent, "restore")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	observed := make(chan bool, 1)
	go func() {
		for i := 0; i < 3000; i++ {
			entries, _ := os.ReadDir(parent)
			if len(entries) > 0 {
				observed <- true
				cancel()
				return
			}
			time.Sleep(time.Millisecond)
		}
		observed <- false
		cancel()
	}()
	e := RestoreWithLimits(ctx, archive, target, RestoreLimits{})
	if !<-observed || !errors.Is(e, context.Canceled) {
		t.Fatal("post-stage cancellation was not exercised")
	}
	entries, err := os.ReadDir(parent)
	if err != nil || len(entries) != 0 {
		t.Fatal("cancelled restore left stage or published target")
	}
	data, err := os.ReadFile(archive)
	if err != nil || !bytes.Equal(data, f.archive) {
		t.Fatal("cancelled restore changed source archive")
	}
}

func TestRestoreLimitsFixtureEmitter(t *testing.T) {
	out := os.Getenv("IMMICH_RESTORE_LIMIT_FIXTURE")
	if out == "" {
		return
	}
	f := newRestoreLimitFixture(t, false)
	if e := os.WriteFile(filepath.Join(out, "fixture.zip"), f.archive, 0600); e != nil {
		t.Fatal("CLI fixture write failed")
	}
}

func TestRestoreLimitsFixtureInspector(t *testing.T) {
	target := os.Getenv("IMMICH_RESTORE_LIMIT_TARGET")
	if target == "" {
		return
	}
	s, e := Open(target)
	if e != nil {
		t.Fatal("CLI restored catalog unavailable")
	}
	defer s.Close()
	if e = s.Verify(); e != nil {
		t.Fatal("CLI restored originals failed verification")
	}
	page, e := s.OrganizationReviewPage("", "", "", true)
	if e != nil || len(page.Faces) != 1 || page.Faces[0].Name != "Corrected name" || !page.Faces[0].CanUndo || !page.Faces[0].Dismissed {
		t.Fatal("CLI restore lost correction/history/dismissal")
	}
	if len(page.Assets) != 1 || len(page.Assets[0].Tags) != 1 || page.Assets[0].Tags[0] != "Keep tag" || page.Assets[0].Analysis.Status != "ready" {
		t.Fatal("CLI restore lost tags/results")
	}
	code, e := os.ReadFile(filepath.Join(target, "access-code.txt"))
	if e != nil || string(code) != "synthetic-restore-code" {
		t.Fatal("CLI restore lost synthetic code")
	}
}
