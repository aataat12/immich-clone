package library

import (
	"encoding/json"
	"errors"
	bolt "go.etcd.io/bbolt"
	"strconv"
)

const maxImportHistoryExportBytes = 64 << 20

var ErrImportHistoryExportLimit = errors.New("retained import history exceeds the 64 MiB JSON export budget; use a complete backup for recovery")

type ImportHistorySessionExport struct {
	Session json.RawMessage            `json:"session"`
	Entries map[string]json.RawMessage `json:"entries"`
}
type ImportHistoryExport struct {
	Version  int                                   `json:"version"`
	Sessions map[string]ImportHistorySessionExport `json:"sessions"`
}

// Export existing receipts without synthesizing batches or altering saved status.
// Every raw JSON value is copied once while its read transaction is alive. The
// 64 MiB aggregate raw-record/key budget is independent of the catalog's write
// limits; oversized interchange export refuses rather than omitting history.
// Complete backup remains the full recovery format and keeps existing limits.
func exportImportHistory(tx *bolt.Tx, maxBytes int) (ImportHistoryExport, error) {
	out := ImportHistoryExport{Version: 1, Sessions: map[string]ImportHistorySessionExport{}}
	remaining := maxBytes
	charge := func(key, value []byte) error {
		if len(value) > 1<<20 {
			return ErrImportHistoryExportLimit
		}
		cost := len(key) + len(value) + 16
		if cost > remaining {
			return ErrImportHistoryExportLimit
		}
		remaining -= cost
		return nil
	}
	b := tx.Bucket([]byte(importBucket))
	if b == nil {
		return out, nil
	}
	e := b.ForEach(func(key, value []byte) error {
		if value != nil || !validID(string(key)) || len(out.Sessions) >= 128 {
			return ErrImportHistoryExportLimit
		}
		child := b.Bucket(key)
		meta := child.Get([]byte("meta"))
		if meta == nil {
			return ErrImportInput
		}
		if e := charge(key, meta); e != nil {
			return e
		}
		if !json.Valid(meta) {
			return ErrImportInput
		}
		item := ImportHistorySessionExport{Session: append(json.RawMessage(nil), meta...), Entries: map[string]json.RawMessage{}}
		if e := child.ForEach(func(entryKeyBytes, raw []byte) error {
			if string(entryKeyBytes) == "meta" {
				return nil
			}
			if len(entryKeyBytes) != 8 {
				return ErrImportInput
			}
			index, e := strconv.Atoi(string(entryKeyBytes))
			if e != nil || index < 0 || index >= MaxImportFiles || string(entryKeyBytes) != entryKey(index) || raw == nil {
				return ErrImportInput
			}
			if len(item.Entries) >= MaxImportFiles {
				return ErrImportHistoryExportLimit
			}
			if e := charge(entryKeyBytes, raw); e != nil {
				return e
			}
			if !json.Valid(raw) {
				return ErrImportInput
			}
			item.Entries[string(entryKeyBytes)] = append(json.RawMessage(nil), raw...)
			return nil
		}); e != nil {
			return e
		}
		out.Sessions[string(key)] = item
		return nil
	})
	if e != nil {
		return ImportHistoryExport{}, e
	}
	return out, nil
}
