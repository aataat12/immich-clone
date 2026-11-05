package library

import (
	"context"
	"encoding/json"
	"errors"
	"os"

	bolt "go.etcd.io/bbolt"
	"immichclone/internal/recognition"
)

const batchMembers = "analysis_batch_members_v1"
const batchMeta = "analysis_batch_v1"

type AnalysisBatch struct {
	Status        string `json:"status"`
	Total         int    `json:"total"`
	Pending       int    `json:"pending"`
	Ready         int    `json:"ready"`
	Failed        int    `json:"failed"`
	Cancelled     int    `json:"cancelled"`
	Available     bool   `json:"available"`
	ImportWaiting bool   `json:"import_waiting"`
}

func readBatch(tx *bolt.Tx) (AnalysisBatch, error) {
	b := AnalysisBatch{Status: "idle"}
	e := readJSON(tx.Bucket([]byte(batchMeta)), "current", &b)
	if errors.Is(e, os.ErrNotExist) {
		e = nil
	}
	return b, e
}

// Status uses a database read snapshot and atomic runtime flags, not the copy gate.
func (s *Store) AnalysisBatch() (AnalysisBatch, error) {
	var b AnalysisBatch
	e := s.db.View(func(tx *bolt.Tx) error { var e error; b, e = readBatch(tx); return e })
	b.Available = s.recognitionAvailable.Load()
	b.ImportWaiting = s.ImportsRunning()
	return b, e
}
func eligibleAnalysis(a Asset) bool {
	return !a.Hidden && supportsStillAI(a.MIME) && a.Size <= 64<<20 && (a.Width <= 0 || a.Height <= 0 || validImageDimensions(a.Width, a.Height))
}

// ControlAnalysisBatch changes only this batch's pending work. An in-flight
// inference may finish; cancellation never removes manual corrections or results.
func (s *Store) ControlAnalysisBatch(ctx context.Context, action string) error {
	s.gate.Lock()
	defer s.gate.Unlock()
	return s.db.Update(func(tx *bolt.Tx) error {
		b, e := readBatch(tx)
		if e != nil {
			return e
		}
		members := tx.Bucket([]byte(batchMembers))
		analyses := tx.Bucket([]byte(analysisBucket))
		queue := tx.Bucket([]byte(analysisQueue))
		switch action {
		case "start":
			if !s.recognitionAvailable.Load() {
				return recognition.ErrUnavailable
			}
			if b.Status == "running" || b.Status == "paused" || b.Pending > 0 {
				return errors.New("batch already active")
			}
			if e = tx.DeleteBucket([]byte(batchMembers)); e != nil {
				return e
			}
			members, e = tx.CreateBucket([]byte(batchMembers))
			if e != nil {
				return e
			}
			b = AnalysisBatch{Status: "running"}
			e = tx.Bucket([]byte("assets")).ForEach(func(k, v []byte) error {
				if e := ctx.Err(); e != nil {
					return e
				}
				var a Asset
				if e := json.Unmarshal(v, &a); e != nil {
					return e
				}
				if !eligibleAnalysis(a) || (isHEIC(a.MIME) && !s.mediaAnalysisAvailable.Load()) || analyses.Get(k) != nil {
					return nil
				}
				current := Analysis{AssetID: a.ID, Status: "queued", Provider: recognition.AnalysisID, Objects: []string{}}
				if e := writeJSON(analyses, a.ID, current); e != nil {
					return e
				}
				if e := queue.Put(k, []byte{1}); e != nil {
					return e
				}
				if e := members.Put(k, []byte("pending")); e != nil {
					return e
				}
				b.Total++
				b.Pending++
				return nil
			})
			if e != nil {
				return e
			}
			if b.Pending == 0 {
				b.Status = "complete"
			}
		case "pause":
			if b.Status != "running" {
				return errors.New("batch is not running")
			}
			b.Status = "paused"
		case "resume":
			if !s.recognitionAvailable.Load() {
				return recognition.ErrUnavailable
			}
			if b.Status != "paused" {
				return errors.New("batch is not paused")
			}
			b.Status = "running"
		case "cancel":
			if b.Status != "running" && b.Status != "paused" {
				return errors.New("batch is not active")
			}
			var cancelled []string
			e = members.ForEach(func(k, v []byte) error {
				if e := ctx.Err(); e != nil {
					return e
				}
				if string(v) != "pending" || string(k) == s.activeAnalysis {
					return nil
				}
				var a Analysis
				if e := readJSON(analyses, string(k), &a); e != nil {
					return e
				}
				if a.Status != "queued" {
					return errors.New("batch queue inconsistent")
				}
				if e := queue.Delete(k); e != nil {
					return e
				}
				if e := analyses.Delete(k); e != nil {
					return e
				}
				cancelled = append(cancelled, string(k))
				b.Pending--
				b.Cancelled++
				return nil
			})
			if e != nil {
				return e
			}
			for _, id := range cancelled {
				if e := members.Put([]byte(id), []byte("cancelled")); e != nil {
					return e
				}
			}
			b.Status = "cancelled"
		default:
			return ErrOrganizationInput
		}
		if e := ctx.Err(); e != nil {
			return e
		}
		return writeJSON(tx.Bucket([]byte(batchMeta)), "current", b)
	})
}
func finishBatchMember(tx *bolt.Tx, id, status string) error {
	m := tx.Bucket([]byte(batchMembers))
	if string(m.Get([]byte(id))) != "pending" {
		return nil
	}
	b, e := readBatch(tx)
	if e != nil {
		return e
	}
	if e = m.Put([]byte(id), []byte(status)); e != nil {
		return e
	}
	b.Pending--
	if status == "ready" {
		b.Ready++
	} else {
		b.Failed++
	}
	if b.Pending == 0 && b.Status != "cancelled" {
		b.Status = "complete"
	}
	return writeJSON(tx.Bucket([]byte(batchMeta)), "current", b)
}

// Reconcile additive batch bookkeeping with authoritative analysis records after
// restart, restore, or an older executable processing the shared queue.
func recoverAnalysisBatch(tx *bolt.Tx) error {
	b, e := readBatch(tx)
	if e != nil {
		return e
	}
	b.Total = 0
	b.Pending = 0
	b.Ready = 0
	b.Failed = 0
	b.Cancelled = 0
	m := tx.Bucket([]byte(batchMembers))
	a := tx.Bucket([]byte(analysisBucket))
	q := tx.Bucket([]byte(analysisQueue))
	updates := map[string]string{}
	e = m.ForEach(func(k, v []byte) error {
		state := string(v)
		if state == "pending" {
			var record Analysis
			if e := readJSON(a, string(k), &record); e != nil {
				return e
			}
			switch record.Status {
			case "ready", "failed":
				state = record.Status
			case "queued":
				if b.Status == "cancelled" {
					if e := a.Delete(k); e != nil {
						return e
					}
					if e := q.Delete(k); e != nil {
						return e
					}
					state = "cancelled"
				}
			default:
				return errors.New("invalid batch analysis status")
			}
			updates[string(k)] = state
		}
		b.Total++
		switch state {
		case "pending":
			b.Pending++
		case "ready":
			b.Ready++
		case "failed":
			b.Failed++
		case "cancelled":
			b.Cancelled++
		default:
			return errors.New("invalid batch member")
		}
		return nil
	})
	if e != nil {
		return e
	}
	for id, state := range updates {
		if e := m.Put([]byte(id), []byte(state)); e != nil {
			return e
		}
	}
	if b.Pending == 0 && (b.Status == "running" || b.Status == "paused") {
		b.Status = "complete"
	}
	return writeJSON(tx.Bucket([]byte(batchMeta)), "current", b)
}
