package library

import (
	"encoding/json"
	"errors"
	bolt "go.etcd.io/bbolt"
)

// The state directory and backup container keep their version-1 layout.
// Catalog version 2 protects automatic-assignment provenance from old writers,
// which explicitly reject any catalog version other than 1 before committing.
func supportedCatalogSchema(version []byte) bool {
	return string(version) == "1" || string(version) == "2"
}

func automaticProvenance(l FaceLabel) bool {
	return l.Source == "automatic" || l.PreviousSource == "automatic"
}

func requireAutomaticCatalog(tx *bolt.Tx) error {
	meta := tx.Bucket([]byte("meta"))
	if meta == nil || !supportedCatalogSchema(meta.Get([]byte("schema"))) {
		return errors.New("unsupported catalog schema")
	}
	if string(meta.Get([]byte("schema"))) == "2" {
		return nil
	}
	return meta.Put([]byte("schema"), []byte("2"))
}

// Earlier development-only catalogs could contain automatic labels in version
// 1. A writable open upgrades those intact labels before exposing any view.
// Read-only commands refuse inconsistent version-1 automatic provenance.
func protectAutomaticCatalog(tx *bolt.Tx, writable bool) error {
	meta := tx.Bucket([]byte("meta"))
	if meta == nil || !supportedCatalogSchema(meta.Get([]byte("schema"))) {
		return errors.New("unsupported catalog schema")
	}
	if string(meta.Get([]byte("schema"))) == "2" {
		return nil
	}
	labels := tx.Bucket([]byte(labelsBucket))
	if labels == nil {
		return nil
	}
	requires := false
	if e := labels.ForEach(func(_, data []byte) error {
		var label FaceLabel
		if e := json.Unmarshal(data, &label); e != nil {
			return e
		}
		if automaticProvenance(label) {
			requires = true
		}
		return nil
	}); e != nil {
		return e
	}
	if !requires {
		return nil
	}
	if !writable {
		return errors.New("automatic provenance requires catalog upgrade")
	}
	return requireAutomaticCatalog(tx)
}
