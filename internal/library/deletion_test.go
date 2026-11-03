package library

import (
	"bytes"
	"context"
	"errors"
	bolt "go.etcd.io/bbolt"
	"image"
	"immichclone/internal/recognition"
	"os"
	"path/filepath"
	"testing"
)

func deleteForTest(s *Store, ctx context.Context, id, confirm string) (bool, error) {
	a, _ := s.Get(id)
	return s.DeleteHidden(ctx, id, confirm, a.Added)
}
func TestDeletionPreservesSourcesBackupAndExplicitReimport(t *testing.T) {
	s, id, face := seedReviewedPhoto(t)
	ctx := context.Background()
	data := fixture(t)
	originalAsset, _ := s.Get(id)
	external := filepath.Join(t.TempDir(), "source.png")
	os.WriteFile(external, data, 0600)
	s.ProcessOne()
	s.ControlReanalysis(id, "start")
	session, e := s.CreateImport([]ImportFile{{Name: "receipt.png", Size: int64(len(data))}})
	if e != nil {
		t.Fatal(e)
	}
	s.ControlImport(session.ID, "start")
	if _, e = s.PutImport(ctx, session.ID, 0, bytes.NewReader(data)); e != nil {
		t.Fatal(e)
	}
	s.ControlImport(session.ID, "pause")
	other, _, e := s.Put("other.txt", bytes.NewBufferString("unrelated managed original"))
	if e != nil {
		t.Fatal(e)
	}
	s.SetHidden(id, true)
	var before bytes.Buffer
	if e = s.Backup(&before); e != nil {
		t.Fatal(e)
	}
	if pending, e := deleteForTest(s, ctx, id, id); e != nil || pending {
		t.Fatal(pending, e)
	}
	for _, path := range []string{s.Original(id), s.PreviewPath(id), filepath.Join(s.root, "faces", face+".png")} {
		if _, e = os.Lstat(path); !os.IsNotExist(e) {
			t.Fatal("deleted managed file remains", path, e)
		}
	}
	if _, e = s.Get(id); !os.IsNotExist(e) {
		t.Fatal("catalog record remains")
	}
	if _, e = s.Get(other.ID); e != nil {
		t.Fatal("unrelated asset lost")
	}
	if e = s.db.View(func(tx *bolt.Tx) error {
		for _, bucket := range []string{analysisBucket, analysisQueue, reanalysisBucket, reanalysisQueue, tagsBucket, "dismissed_ai_tags_v1", batchMembers} {
			if tx.Bucket([]byte(bucket)).Get([]byte(id)) != nil {
				t.Fatal("associated record remains", bucket)
			}
		}
		if tx.Bucket([]byte(facesBucket)).Get([]byte(face)) != nil || tx.Bucket([]byte(labelsBucket)).Get([]byte(face)) != nil {
			t.Fatal("face or label remains")
		}
		return nil
	}); e != nil {
		t.Fatal(e)
	}
	p, _ := s.GalleryPage("name", "hidden", "", 60)
	if p.Total != 1 || p.Hidden != 0 {
		t.Fatal("gallery counts stale", p)
	}
	entries, e := s.ImportPage(session.ID, 0)
	if e != nil || !entries.Entries[0].Deleted || entries.Session.AcceptedFiles != 1 {
		t.Fatal("historical receipt not marked")
	}
	s.ControlImport(session.ID, "start")
	if _, e = s.PutImport(ctx, session.ID, 0, bytes.NewReader(data)); !errors.Is(e, ErrImportConflict) {
		t.Fatal("old receipt resurrected deletion", e)
	}
	s.ControlImport(session.ID, "cancel")
	var after bytes.Buffer
	if e = s.Backup(&after); e != nil {
		t.Fatal(e)
	}
	for _, backup := range []struct {
		name    string
		data    []byte
		present bool
	}{{"before", before.Bytes(), true}, {"after", after.Bytes(), false}} {
		file := filepath.Join(t.TempDir(), backup.name+".zip")
		os.WriteFile(file, backup.data, 0600)
		target := filepath.Join(t.TempDir(), "restored")
		if e = Restore(file, target); e != nil {
			t.Fatal(e)
		}
		r, e := Open(target)
		if e != nil {
			t.Fatal(e)
		}
		a, e := r.Get(id)
		if backup.present {
			if e != nil || !a.Hidden {
				t.Fatal("pre-deletion backup lost item")
			}
			r.SetHidden(id, false)
			o, e := r.OrganizationReviewPage("", "", "", true)
			if e != nil || len(o.Faces) != 1 || o.Faces[0].Name != "Saved name" {
				t.Fatal("pre-deletion backup lost corrections")
			}
		} else if !os.IsNotExist(e) {
			t.Fatal("post-deletion backup resurrected item")
		}
		if e = r.Verify(); e != nil {
			t.Fatal(e)
		}
		r.Close()
	}
	a, duplicate, e := s.Put("new-import.png", bytes.NewReader(data))
	if e != nil || duplicate || a.Hidden {
		t.Fatal("explicit new import failed", e)
	}
	s.SetHidden(id, true)
	if _, err := s.DeleteHidden(ctx, id, id, originalAsset.Added); !errors.Is(err, ErrOrganizationInput) {
		t.Fatal("stale confirmation deleted reimport", err)
	}
	s.SetHidden(id, false)
	o, e := s.OrganizationPage("", "", "")
	if e != nil || len(o.Faces) != 0 {
		t.Fatal("new import inherited old corrections")
	}
	if source, e := os.ReadFile(external); e != nil || !bytes.Equal(source, data) {
		t.Fatal("external source altered")
	}
	if e = s.Verify(); e != nil {
		t.Fatal(e)
	}
}
func pendingDeletionFixture(t *testing.T) (*Store, string, string) {
	t.Helper()
	s, root := openTest(t)
	a, _, e := s.Put("pending.txt", bytes.NewBufferString("original retained until cleanup completes"))
	if e != nil {
		t.Fatal(e)
	}
	s.SetHidden(a.ID, true)
	// A non-file at a cache path forces a real cleanup refusal on every OS.
	if e = os.Mkdir(s.PreviewPath(a.ID), 0700); e != nil {
		t.Fatal(e)
	}
	pending, e := deleteForTest(s, context.Background(), a.ID, a.ID)
	if !pending || !errors.Is(e, ErrDeletionPending) {
		t.Fatal("expected durable pending cleanup", pending, e)
	}
	return s, root, a.ID
}
func TestDeletionInterruptedCleanupRestartAndMissingFiles(t *testing.T) {
	for _, removed := range []bool{false, true} {
		t.Run(map[bool]string{false: "original-remains", true: "original-already-removed"}[removed], func(t *testing.T) {
			s, root, id := pendingDeletionFixture(t)
			if _, _, e := s.Put("same.txt", bytes.NewBufferString("original retained until cleanup completes")); !errors.Is(e, ErrDeletionPending) {
				t.Fatal("pending ID reimport allowed", e)
			}
			var b bytes.Buffer
			if e := s.Backup(&b); !errors.Is(e, ErrDeletionPending) {
				t.Fatal("backup accepted incomplete deletion", e)
			}
			s.Close()
			r, e := Open(root)
			if e != nil {
				t.Fatal("pending cleanup blocked library startup", e)
			}
			jobs, e := r.PendingDeletions()
			if e != nil || len(jobs) != 1 {
				t.Fatal("pending state lost")
			}
			r.Close()
			os.Remove(filepath.Join(root, "previews", id+".png"))
			if removed {
				os.Remove(filepath.Join(root, "originals", id))
			}
			r, e = Open(root)
			if e != nil {
				t.Fatal(e)
			}
			defer r.Close()
			jobs, _ = r.PendingDeletions()
			if len(jobs) != 0 {
				t.Fatal("restart did not finish cleanup")
			}
			if _, e = os.Stat(r.Original(id)); !os.IsNotExist(e) {
				t.Fatal("original remains")
			}
			if e = r.Verify(); e != nil {
				t.Fatal(e)
			}
		})
	}
}
func TestDeletionOldWriterConflictKeepsReappearedCopy(t *testing.T) {
	s, root, id := pendingDeletionFixture(t)
	// Simulate an older executable which does not understand the additive journal.
	a := Asset{ID: id, Name: "reimported.txt", Size: int64(len("original retained until cleanup completes")), MIME: "text/plain", Hidden: true, Preview: "failed"}
	if e := s.save(a); e != nil {
		t.Fatal(e)
	}
	os.Remove(s.PreviewPath(id))
	s.Close()
	r, e := Open(root)
	if e != nil {
		t.Fatal(e)
	}
	defer r.Close()
	jobs, _ := r.PendingDeletions()
	if len(jobs) != 1 || jobs[0].Status != "conflict" {
		t.Fatal("older writer conflict not visible")
	}
	if e = r.RetryDeletion(context.Background(), id); !errors.Is(e, ErrDeletionConflict) {
		t.Fatal("cleanup deleted reappeared copy", e)
	}
	if e = r.KeepDeletionConflict(id); e != nil {
		t.Fatal(e)
	}
	if e = r.Verify(); e != nil {
		t.Fatal(e)
	}
}
func TestDeletionGuardsAndActiveInference(t *testing.T) {
	s, id, _ := seedReviewedPhoto(t)
	ctx := context.Background()
	if _, e := deleteForTest(s, ctx, id, id); !errors.Is(e, ErrOrganizationInput) {
		t.Fatal("visible item deleted")
	}
	s.SetHidden(id, true)
	if _, e := deleteForTest(s, ctx, id, "wrong"); !errors.Is(e, ErrOrganizationInput) {
		t.Fatal("unconfirmed deletion")
	}
	cancelled, cancel := context.WithCancel(ctx)
	cancel()
	if _, e := deleteForTest(s, cancelled, id, id); !errors.Is(e, context.Canceled) {
		t.Fatal("cancelled request changed state")
	}
	for _, lock := range []struct {
		take    func()
		release func()
	}{{s.gate.Lock, s.gate.Unlock}, {s.importMu.Lock, s.importMu.Unlock}, {s.analysisGate.Lock, s.analysisGate.Unlock}, {func() { s.mediaAnalysisGate <- struct{}{} }, func() { <-s.mediaAnalysisGate }}} {
		lock.take()
		_, e := deleteForTest(s, ctx, id, id)
		lock.release()
		if !errors.Is(e, ErrDeletionBusy) {
			t.Fatal("active work not refused", e)
		}
	}
	m, e := s.CreateImport([]ImportFile{{Name: "hold.txt", Size: 1}})
	if e != nil {
		t.Fatal(e)
	}
	s.ControlImport(m.ID, "start")
	if _, e = deleteForTest(s, ctx, id, id); !errors.Is(e, ErrDeletionBusy) {
		t.Fatal("active import lease ignored")
	}
	s.ControlImport(m.ID, "cancel")
	// Start reanalysis before hiding; block actual processing outside the catalog gate.
	s.SetHidden(id, false)
	started, finish := make(chan struct{}), make(chan struct{})
	s.SetRecognizer(func(c context.Context, i image.Image, p string) (recognition.Result, error) {
		close(started)
		<-finish
		return fakeAnalysis(c, i, p)
	}, true)
	s.ControlReanalysis(id, "start")
	s.SetHidden(id, true)
	done := make(chan error, 1)
	go func() { _, e := s.ProcessAnalysis(ctx); done <- e }()
	<-started
	if _, e = deleteForTest(s, ctx, id, id); !errors.Is(e, ErrDeletionBusy) {
		t.Fatal("in-flight inference deletion allowed")
	}
	close(finish)
	if e = <-done; e != nil {
		t.Fatal(e)
	}
	if pending, e := deleteForTest(s, ctx, id, id); e != nil || pending {
		t.Fatal(pending, e)
	}
	if _, e = s.ProcessAnalysis(ctx); e != nil {
		t.Fatal("stale work after deletion", e)
	}
}

func TestDeletionPreDeleteArchiveRestored(t *testing.T) {
	root := os.Getenv("IMMICH_DELETE_RESTORE")
	if root == "" {
		t.Skip("requires actual service pre-deletion archive")
	}
	s, e := Open(root)
	if e != nil {
		t.Fatal(e)
	}
	defer s.Close()
	id := "88431cd9653ccd539741b555fb0a46b61558b301d4110412b5bc28b5e3ea6cb5"
	a, e := s.Get(id)
	if e != nil || !a.Hidden {
		t.Fatal("pre-deletion archive lost hidden original")
	}
	if e = s.Verify(); e != nil {
		t.Fatal(e)
	}
	s.SetHidden(id, false)
	o, e := s.OrganizationPage("", "", "")
	if e != nil {
		t.Fatal(e)
	}
	found := false
	for _, f := range o.Faces {
		if f.AssetID == id && f.Name == "Deletion Test Person" {
			if _, e = s.FaceCrop(f.ID); e != nil {
				t.Fatal(e)
			}
			found = true
		}
	}
	if !found {
		t.Fatal("pre-deletion archive lost named face")
	}
	for _, record := range o.Assets {
		if record.Asset.ID == id {
			if len(record.Tags) != 1 || record.Tags[0] != "Delete Review" || record.Analysis.Status != "ready" || record.Reanalysis == nil || record.Reanalysis.Status != "paused" {
				t.Fatal("restored review data changed")
			}
		}
	}
}

func TestDeletionCorruptOriginalRefusesBeforeIntent(t *testing.T) {
	s, id, _ := seedReviewedPhoto(t)
	s.SetHidden(id, true)
	before := snapshotReview(t, s, id)
	if e := os.WriteFile(s.Original(id), []byte("modified outside the application"), 0600); e != nil {
		t.Fatal(e)
	}
	pending, e := deleteForTest(s, context.Background(), id, id)
	if pending || !errors.Is(e, ErrDeletionOriginal) {
		t.Fatal("corrupt source deletion accepted", pending, e)
	}
	if _, e = s.Get(id); e != nil {
		t.Fatal("refused deletion lost catalog")
	}
	after := snapshotReview(t, s, id)
	for k, v := range before {
		if !bytes.Equal(v, after[k]) {
			t.Fatal("refused deletion altered review", k)
		}
	}
	jobs, _ := s.PendingDeletions()
	if len(jobs) != 0 {
		t.Fatal("refused deletion created recovery intent")
	}
}

type deletionHoldWriter struct {
	b              bytes.Buffer
	start, release chan struct{}
	started        bool
}

func (w *deletionHoldWriter) Write(p []byte) (int, error) {
	if !w.started {
		w.started = true
		close(w.start)
		<-w.release
	}
	return w.b.Write(p)
}
func TestDeletionRefusesActualBackupAndCancelsQueuedBatch(t *testing.T) {
	s, _ := openTest(t)
	s.SetRecognizer(fakeAnalysis, true)
	a, _, e := s.Put("queued.png", bytes.NewReader(fixture(t)))
	if e != nil {
		t.Fatal(e)
	}
	if e = s.ControlAnalysisBatch(context.Background(), "start"); e != nil {
		t.Fatal(e)
	}
	s.SetHidden(a.ID, true)
	w := &deletionHoldWriter{start: make(chan struct{}), release: make(chan struct{})}
	done := make(chan error, 1)
	go func() { done <- s.Backup(w) }()
	<-w.start
	if _, e = deleteForTest(s, context.Background(), a.ID, a.ID); !errors.Is(e, ErrDeletionBusy) {
		t.Fatal("actual backup not protected", e)
	}
	close(w.release)
	if e = <-done; e != nil {
		t.Fatal(e)
	}
	if pending, e := deleteForTest(s, context.Background(), a.ID, a.ID); pending || e != nil {
		t.Fatal(pending, e)
	}
	b, e := s.AnalysisBatch()
	if e != nil || b.Pending != 0 || b.Total != 0 {
		t.Fatal("queued batch bookkeeping retained deleted item", b, e)
	}
	if processed, e := s.ProcessAnalysis(context.Background()); e != nil || processed {
		t.Fatal("deleted analysis job remained", processed, e)
	}
}
