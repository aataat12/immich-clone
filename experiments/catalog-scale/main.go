// Metadata-only catalog benchmark. Seed rows intentionally have no originals;
// these measurements do not simulate bulk media ingestion or complete libraries.
package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	bolt "go.etcd.io/bbolt"
	"image"
	"image/png"
	"immichclone/internal/library"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"time"
)

type metric struct {
	Name         string `json:"operation"`
	Rows         int    `json:"rows"`
	Microseconds int64  `json:"median_us"`
	Allocated    uint64 `json:"allocated_bytes_at_median"`
	Samples      int    `json:"samples"`
}

func measure(name string, rows int, fn func() error) metric {
	type sample struct {
		us        int64
		allocated uint64
	}
	var samples []sample
	for i := 0; i < 3; i++ {
		runtime.GC()
		var before, after runtime.MemStats
		runtime.ReadMemStats(&before)
		start := time.Now()
		if e := fn(); e != nil {
			panic(e)
		}
		elapsed := time.Since(start).Microseconds()
		runtime.ReadMemStats(&after)
		samples = append(samples, sample{elapsed, after.TotalAlloc - before.TotalAlloc})
	}
	sort.Slice(samples, func(i, j int) bool { return samples[i].us < samples[j].us })
	return metric{name, rows, samples[1].us, samples[1].allocated, 3}
}
func run(n int) []metric {
	root, e := os.MkdirTemp("", "synthetic-catalog-scale-")
	if e != nil {
		panic(e)
	}
	defer os.RemoveAll(root)
	s, e := library.Open(root)
	if e != nil {
		panic(e)
	}
	if e = s.Close(); e != nil {
		panic(e)
	}
	db, e := bolt.Open(filepath.Join(root, "catalog.db"), 0600, nil)
	if e != nil {
		panic(e)
	}
	e = db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("assets"))
		for i := 0; i < n; i++ {
			h := sha256.Sum256([]byte(fmt.Sprintf("synthetic-catalog-row-%d", i)))
			id := hex.EncodeToString(h[:])
			a := library.Asset{ID: id, Name: fmt.Sprintf("synthetic-%06d.jpg", i), MIME: "image/jpeg", Added: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), Size: 1, Preview: "failed", PreviewRevision: 3, Error: "catalog-only synthetic row; no media fixture"}
			v, e := json.Marshal(a)
			if e != nil {
				return e
			}
			if e = b.Put([]byte(id), v); e != nil {
				return e
			}
		}
		return nil
	})
	if e != nil {
		panic(e)
	}
	db.Close()
	s, e = library.Open(root)
	if e != nil {
		panic(e)
	}
	defer s.Close()
	var results []metric
	results = append(results, measure("catalog_list", n, func() error {
		a, e := s.List()
		if e == nil && len(a) != n {
			panic("catalog count")
		}
		return e
	}))
	results = append(results, measure("idle_job_poll", n, func() error {
		processed, e := s.ProcessOne()
		if processed {
			panic("unexpected synthetic job")
		}
		return e
	}))
	results = append(results, measure("list_plus_json", n, func() error {
		a, e := s.List()
		if e != nil {
			return e
		}
		_, e = json.Marshal(a)
		return e
	}))
	// Independently execute a real native PNG job amongst the metadata-only rows.
	var b bytes.Buffer
	if e = png.Encode(&b, image.NewRGBA(image.Rect(0, 0, 64, 48))); e != nil {
		panic(e)
	}
	a, duplicate, e := s.Put("real-synthetic.png", bytes.NewReader(b.Bytes()))
	if e != nil || duplicate {
		panic("PNG import")
	}
	start := time.Now()
	processed, e := s.ProcessOne()
	if e != nil || !processed {
		panic("PNG job")
	}
	elapsed := time.Since(start).Microseconds()
	a, e = s.Get(a.ID)
	if e != nil || a.Preview != "ready" || a.Width != 64 || a.Height != 48 {
		panic("PNG decode")
	}
	original, e := os.ReadFile(s.Original(a.ID))
	if e != nil || !bytes.Equal(original, b.Bytes()) {
		panic("PNG original changed")
	}
	results = append(results, metric{"actual_png_job_one_sample", n, elapsed, 0, 1})
	return results
}
func main() {
	max := flag.Int("max", 50000, "bounded catalog-only rows, <=50000")
	flag.Parse()
	if *max < 1000 || *max > 50000 {
		panic("invalid bounded row count")
	}
	var out []metric
	for _, n := range []int{1000, 10000, 50000} {
		if n <= *max {
			out = append(out, run(n)...)
		}
	}
	if e := json.NewEncoder(os.Stdout).Encode(map[string]any{"scope": "synthetic metadata-only catalog, no bulk ingestion claim; Go allocation bytes, not RSS; actual PNG job verified separately", "goos": runtime.GOOS, "metrics": out}); e != nil {
		panic(e)
	}
}
