package library

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"errors"
	"hash/crc32"
	"io"
	"os"
	"path/filepath"
	"testing"
)

type hostileEntry struct {
	header zip.FileHeader
	data   []byte
}

func TestRestoreRejectsHostileArchiveWithoutPublication(t *testing.T) {
	s, _ := openTest(t)
	a, _, err := s.Put("fixture.bin", bytes.NewReader([]byte("immutable synthetic original")))
	if err != nil {
		t.Fatal(err)
	}
	var baseline bytes.Buffer
	if err = s.Backup(&baseline); err != nil {
		t.Fatal(err)
	}
	load := func() []hostileEntry {
		t.Helper()
		zr, e := zip.NewReader(bytes.NewReader(baseline.Bytes()), int64(baseline.Len()))
		if e != nil {
			t.Fatal(e)
		}
		var entries []hostileEntry
		for _, f := range zr.File {
			r, e := f.Open()
			if e != nil {
				t.Fatal(e)
			}
			data, e := io.ReadAll(r)
			r.Close()
			if e != nil {
				t.Fatal(e)
			}
			h := f.FileHeader
			h.Method = zip.Store
			h.Flags = 0
			h.CRC32 = crc32.ChecksumIEEE(data)
			h.CompressedSize64 = uint64(len(data))
			h.UncompressedSize64 = uint64(len(data))
			h.CompressedSize = uint32(len(data))
			h.UncompressedSize = uint32(len(data))
			entries = append(entries, hostileEntry{h, data})
		}
		return entries
	}
	manifest := func(entries []hostileEntry, edit func(*Manifest)) {
		t.Helper()
		for i := range entries {
			if entries[i].header.Name == "manifest.json" {
				var m Manifest
				if e := json.Unmarshal(entries[i].data, &m); e != nil {
					t.Fatal(e)
				}
				edit(&m)
				data, e := json.Marshal(m)
				if e != nil {
					t.Fatal(e)
				}
				entries[i].data = data
				entries[i].header.CRC32 = crc32.ChecksumIEEE(data)
				entries[i].header.CompressedSize64 = uint64(len(data))
				entries[i].header.UncompressedSize64 = uint64(len(data))
				entries[i].header.CompressedSize = uint32(len(data))
				entries[i].header.UncompressedSize = uint32(len(data))
			}
		}
	}
	cases := []struct {
		name string
		edit func([]hostileEntry) []hostileEntry
	}{
		{"trailing JSON", func(es []hostileEntry) []hostileEntry {
			for i := range es {
				if es[i].header.Name == "manifest.json" {
					es[i].data = append(es[i].data, []byte("\n{}")...)
					h := &es[i].header
					h.CRC32 = crc32.ChecksumIEEE(es[i].data)
					h.CompressedSize64 = uint64(len(es[i].data))
					h.UncompressedSize64 = h.CompressedSize64
					h.CompressedSize = uint32(h.CompressedSize64)
					h.UncompressedSize = h.CompressedSize
				}
			}
			return es
		}},
		{"manifest CRC", func(es []hostileEntry) []hostileEntry {
			for i := range es {
				if es[i].header.Name == "manifest.json" {
					es[i].header.CRC32 ^= 1
				}
			}
			return es
		}},
		{"duplicate ZIP entry", func(es []hostileEntry) []hostileEntry { return append(es, es[0]) }},
		{"symlink", func(es []hostileEntry) []hostileEntry { es[0].header.SetMode(os.ModeSymlink | 0600); return es }},
		{"directory", func(es []hostileEntry) []hostileEntry { es[0].header.SetMode(os.ModeDir | 0700); return es }},
		{"device", func(es []hostileEntry) []hostileEntry { es[0].header.SetMode(os.ModeDevice | 0600); return es }},
		{"oversized file", func(es []hostileEntry) []hostileEntry {
			es[0].header.UncompressedSize64 = uint64(MaxUpload) + 1
			es[0].header.UncompressedSize = uint32(es[0].header.UncompressedSize64)
			return es
		}},
	}
	for _, name := range []string{"../escape", "/absolute", "C:/escape", `C:\escape`, `\\server\share\escape`, "originals/../escape", "originals/" + a.ID + ":stream", "originals/" + a.ID + ".", "originals/" + a.ID + " ", "ORIGINALS/" + a.ID} {
		cases = append(cases, struct {
			name string
			edit func([]hostileEntry) []hostileEntry
		}{"unsafe name " + name, func(es []hostileEntry) []hostileEntry {
			old := "originals/" + a.ID
			for i := range es {
				if es[i].header.Name == old {
					es[i].header.Name = name
				}
			}
			manifest(es, func(m *Manifest) { m.Files[name] = m.Files[old]; delete(m.Files, old) })
			return es
		}})
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			es := tc.edit(load())
			var archive bytes.Buffer
			zw := zip.NewWriter(&archive)
			for _, entry := range es {
				w, e := zw.CreateRaw(&entry.header)
				if e != nil {
					t.Fatal(e)
				}
				if _, e = w.Write(entry.data); e != nil {
					t.Fatal(e)
				}
			}
			if e := zw.Close(); e != nil {
				t.Fatal(e)
			}
			parent := t.TempDir()
			sentinel := filepath.Join(parent, "sentinel")
			if e := os.WriteFile(sentinel, []byte("untouched"), 0600); e != nil {
				t.Fatal(e)
			}
			source := filepath.Join(parent, "hostile.zip")
			if e := os.WriteFile(source, archive.Bytes(), 0600); e != nil {
				t.Fatal(e)
			}
			target := filepath.Join(parent, "restored")
			if e := Restore(source, target); e == nil {
				t.Fatal("hostile archive accepted")
			}
			if _, e := os.Lstat(target); !errors.Is(e, os.ErrNotExist) {
				t.Fatal("rejected restore published target")
			}
			data, e := os.ReadFile(sentinel)
			if e != nil || string(data) != "untouched" {
				t.Fatal("outside sentinel changed")
			}
			paths, e := filepath.Glob(filepath.Join(parent, "immich-restore-*"))
			if e != nil || len(paths) != 0 {
				t.Fatal("restore staging not cleaned")
			}
		})
	}
}
