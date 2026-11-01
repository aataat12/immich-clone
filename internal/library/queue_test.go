package library

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	bolt "go.etcd.io/bbolt"
	"testing"
)

func queueKeys(t *testing.T, s *Store) []string {
	t.Helper()
	var keys []string
	e := s.db.View(func(tx *bolt.Tx) error {
		return tx.Bucket([]byte(queueBucket)).ForEach(func(k, v []byte) error { keys = append(keys, string(k)); return nil })
	})
	if e != nil {
		t.Fatal(e)
	}
	return keys
}
func TestQueueRebuildFromAuthoritativeStates(t *testing.T) {
	s, root := openTest(t)
	data := fixture(t)
	a, _, e := s.Put("synthetic-a.png", bytes.NewReader(data))
	if e != nil {
		t.Fatal(e)
	}
	if _, _, e = s.Put("synthetic-b.png", bytes.NewReader(append(append([]byte{}, data...), 1))); e != nil {
		t.Fatal(e)
	}
	if len(queueKeys(t, s)) != 2 {
		t.Fatal("new uploads not indexed")
	}
	if _, e = s.ProcessOne(); e != nil {
		t.Fatal(e)
	}
	items, _ := s.List()
	queued := ""
	for _, item := range items {
		if item.Preview == "queued" {
			queued = item.ID
		}
	}
	if queued == "" {
		t.Fatal("expected remaining job")
	}
	// Simulate an obsolete/incorrect derived index; source asset states stay intact.
	e = s.db.Update(func(tx *bolt.Tx) error {
		if e := tx.DeleteBucket([]byte(queueBucket)); e != nil {
			return e
		}
		b, e := tx.CreateBucket([]byte(queueBucket))
		if e != nil {
			return e
		}
		orphan := sha256.Sum256([]byte("synthetic orphan index"))
		return b.Put([]byte(hex.EncodeToString(orphan[:])), []byte{1})
	})
	if e != nil {
		t.Fatal(e)
	}
	s.Close()
	s, e = Open(root)
	if e != nil {
		t.Fatal(e)
	}
	defer s.Close()
	keys := queueKeys(t, s)
	if len(keys) != 1 || keys[0] != queued {
		t.Fatal("reopen did not rebuild exact pending jobs")
	}
	if _, e = s.ProcessOne(); e != nil {
		t.Fatal(e)
	}
	if ok, e := s.ProcessOne(); e != nil || ok {
		t.Fatal("idle worker retried completed job")
	}
	if e = s.Retry(a.ID); e != nil {
		t.Fatal(e)
	}
	if e = s.SetHidden(a.ID, true); e != nil {
		t.Fatal(e)
	}
	if len(queueKeys(t, s)) != 1 {
		t.Fatal("visibility change lost queued work")
	}
	if _, e = s.ProcessOne(); e != nil {
		t.Fatal(e)
	}
	if e = s.Verify(); e != nil {
		t.Fatal(e)
	}
}
func TestQueuePublicationRollsBackWithCatalog(t *testing.T) {
	s, root := openTest(t)
	data := fixture(t)
	hash := sha256.Sum256(data)
	id := hex.EncodeToString(hash[:])
	// A bucket/value collision makes queue.Put fail after the asset Put in one TX.
	e := s.db.Update(func(tx *bolt.Tx) error { _, e := tx.Bucket([]byte(queueBucket)).CreateBucket([]byte(id)); return e })
	if e != nil {
		t.Fatal(e)
	}
	if _, _, e = s.Put("synthetic.png", bytes.NewReader(data)); e == nil {
		t.Fatal("failed queue publication reported success")
	}
	items, e := s.List()
	if e != nil || len(items) != 0 {
		t.Fatal("partial asset record survived failed queue transaction")
	}
	s.Close()
	s, e = Open(root)
	if e != nil {
		t.Fatal(e)
	}
	defer s.Close()
	// Orphan original is deliberately retained and reused safely after recovery.
	a, duplicate, e := s.Put("synthetic.png", bytes.NewReader(data))
	if e != nil || duplicate || a.ID != id {
		t.Fatal("safe original reuse failed")
	}
	if _, e = s.ProcessOne(); e != nil {
		t.Fatal(e)
	}
	a, _ = s.Get(id)
	if a.Preview != "ready" {
		t.Fatal("recovered job failed")
	}
	if e = s.Verify(); e != nil {
		t.Fatal(e)
	}
}
