package library

import (
	"encoding/json"
	bolt "go.etcd.io/bbolt"
	"path"
	"sort"
	"strings"
	"time"
	_ "time/tzdata" // Browser IANA zones must also work on Windows without zoneinfo files.
)

// Provenance is catalog metadata only. None of these operations open, enumerate,
// rename or hydrate an external source. Receipt membership includes duplicates,
// but excludes unaccepted/deleted entries and originals no longer in the catalog.
func (index *gallerySearchIndex) recordImportSources(tx *bolt.Tx) error {
	b := tx.Bucket([]byte(importBucket))
	if b == nil { return nil }
	sourceSeen := map[string]map[string]bool{}
	return b.ForEach(func(k, v []byte) error {
		child := b.Bucket(k)
		if child == nil { return ErrGalleryInput }
		var session ImportSession
		if e := readJSON(child, "meta", &session); e != nil { return e }
		index.batches = append(index.batches, session)
		batchSeen := map[string]map[string]bool{}
		return child.ForEach(func(key, value []byte) error {
			if string(key) == "meta" { return nil }
			var entry ImportEntry
			if e := json.Unmarshal(value, &entry); e != nil { return e }
			row, exists := index.assets[entry.AssetID]
			if !exists || entry.Deleted || entry.AssetID == "" { return nil }
			if row.batches == nil { row.batches = map[string][]string{} }
			// Retain membership even when no relative folder was recorded.
			if _, ok := row.batches[session.ID]; !ok { row.batches[session.ID] = []string{} }
			if entry.Relative != "" {
				if _, e := cleanRelative(entry.Relative); e != nil { return ErrGalleryInput }
				if sourceSeen[entry.AssetID] == nil {
					sourceSeen[entry.AssetID] = map[string]bool{}
					for _, source := range row.sources { sourceSeen[entry.AssetID][source] = true }
				}
				if batchSeen[entry.AssetID] == nil { batchSeen[entry.AssetID] = map[string]bool{} }
				if !batchSeen[entry.AssetID][entry.Relative] {
					row.batches[session.ID] = append(row.batches[session.ID], entry.Relative)
					batchSeen[entry.AssetID][entry.Relative] = true
				}
				if !sourceSeen[entry.AssetID][entry.Relative] {
					row.sources = append(row.sources, entry.Relative)
					sourceSeen[entry.AssetID][entry.Relative] = true
				}
			}
			index.assets[entry.AssetID] = row
			return nil
		})
	})
}

func galleryFolderMatches(sources []string, folder string) bool {
	for _, source := range sources {
		if folder == "." || strings.HasPrefix(source, folder+"/") { return true }
	}
	return false
}

type GalleryFolder struct {
	Path string `json:"path"`
	Count uint64 `json:"count"`
}
type GalleryImportBatch struct {
	ID string `json:"id"`
	Created time.Time `json:"created"`
	Status string `json:"status"`
	Count uint64 `json:"count"`
}
type GallerySources struct {
	Folders []GalleryFolder `json:"folders"`
	Batches []GalleryImportBatch `json:"batches"`
	Next string `json:"next,omitempty"`
	RootCount uint64 `json:"root_count"`
	Count uint64 `json:"count"`
}

// Folders contain at most limit immediate children of the selected virtual
// directory. Batch choices omit only the active batch filter; switching batches
// therefore preserves the other current filters, including the selected folder.
func (s *Store) GallerySources(query GalleryQuery, after string, limit int) (GallerySources, error) {
	out := GallerySources{Folders: []GalleryFolder{}, Batches: []GalleryImportBatch{}}
	q, e := query.Normalize()
	if e != nil || limit < 1 || limit > 200 { return out, ErrGalleryInput }
	if after != "" { if _, e := cleanRelative(after); e != nil { return out, ErrGalleryInput } }
	s.gate.Lock()
	defer s.gate.Unlock()
	e = s.db.View(func(tx *bolt.Tx) error {
		if q.Album != "" { if _, e := curationAlbum(tx, q.Album); e != nil { return ErrGalleryChanged } }
		index, e := s.gallerySearchFor(tx)
		if e != nil { return e }
		folderCounts, batchCounts := map[string]uint64{}, map[string]uint64{}
		folderQ := q
		folderQ.SourceFolder = ""
		batchQ := q
		batchQ.ImportBatch = ""
		for _, row := range index.assets {
			if q.matches(row) { out.Count++ }
			if batchQ.matches(row) {
				for id, sources := range row.batches {
					if q.SourceFolder == "" || galleryFolderMatches(sources, q.SourceFolder) { batchCounts[id]++ }
				}
			}
			if !folderQ.matches(row) { continue }
			sources := row.sources
			if q.ImportBatch != "" { sources = row.batches[q.ImportBatch] }
			if len(sources) > 0 { out.RootCount++ }
			seen := map[string]bool{}
			for _, source := range sources {
				directory := path.Dir(source)
				prefix := q.SourceFolder
				if prefix == "." { prefix = "" }
				rest := directory
				if prefix != "" {
					if !strings.HasPrefix(directory, prefix+"/") { continue }
					rest = strings.TrimPrefix(directory, prefix+"/")
				}
				if rest == "." || rest == "" { continue }
				child := strings.SplitN(rest, "/", 2)[0]
				if prefix != "" { child = prefix+"/"+child }
				if !seen[child] { folderCounts[child]++; seen[child] = true }
			}
		}
		paths := make([]string, 0, len(folderCounts))
		for name := range folderCounts { if name > after { paths = append(paths, name) } }
		sort.Strings(paths)
		for i, name := range paths {
			if i == limit { out.Next = paths[i-1]; break }
			out.Folders = append(out.Folders, GalleryFolder{Path:name, Count:folderCounts[name]})
		}
		for _, batch := range index.batches {
			status := batch.Status
			if status == "running" && !s.liveImport(batch.ID) { status = "paused" }
			out.Batches = append(out.Batches, GalleryImportBatch{ID:batch.ID, Created:batch.Created, Status:status, Count:batchCounts[batch.ID]})
		}
		sort.Slice(out.Batches, func(i,j int) bool {
			if out.Batches[i].Created.Equal(out.Batches[j].Created) { return out.Batches[i].ID < out.Batches[j].ID }
			return out.Batches[i].Created.After(out.Batches[j].Created)
		})
		return nil
	})
	return out, e
}

type GalleryCalendarBucket struct {
	Year int `json:"year,omitempty"`
	Month int `json:"month,omitempty"`
	Day int `json:"day,omitempty"`
	Count uint64 `json:"count"`
	From string `json:"from"`
	Until string `json:"until"`
}
type GalleryCalendar struct {
	Basis string `json:"basis"`
	Zone string `json:"zone"`
	Years []GalleryCalendarBucket `json:"years"`
	Months []GalleryCalendarBucket `json:"months"`
	Days []GalleryCalendarBucket `json:"days"`
	MissingCapture uint64 `json:"missing_capture"`
	Count uint64 `json:"count"`
}

// A few zones advance at midnight. time.Date may resolve a nonexistent midnight
// into the previous day. Find the first instant of the requested local date so
// calendar links do not include the preceding day's media in those zones.
func galleryCalendarBoundary(year int, month time.Month, day int, loc *time.Location) time.Time {
	value := time.Date(year,month,day,0,0,0,0,loc)
	y,m,d := value.In(loc).Date()
	if y == year && m == month && d == day { return value }
	target := time.Date(year,month,day,12,0,0,0,time.UTC).Format("2006-01-02")
	low, high := value.Add(-48*time.Hour), value.Add(48*time.Hour)
	for high.Sub(low) > time.Nanosecond {
		mid := low.Add(high.Sub(low)/2)
		if mid.In(loc).Format("2006-01-02") >= target { high = mid } else { low = mid }
	}
	return high.In(loc)
}

// Calendar counts intentionally remove only date restrictions. Choosing another
// date remains possible while every other filter and source scope is retained.
// Capture times remain recorded naive calendar values. Upload buckets use exact
// zone-local midnight instants, including 23/25-hour daylight-saving days.
func (s *Store) GalleryCalendar(query GalleryQuery, basis, zone string, year, month int) (GalleryCalendar, error) {
	out := GalleryCalendar{Basis:basis, Zone:zone, Years:[]GalleryCalendarBucket{}, Months:[]GalleryCalendarBucket{}, Days:[]GalleryCalendarBucket{}}
	q, e := query.Normalize()
	if e != nil || (basis != "capture" && basis != "upload") || year < 0 || year > 9999 || month < 0 || month > 12 || (month > 0 && year == 0) || len(zone) > 128 { return out, ErrGalleryInput }
	if zone == "" { zone = "UTC"; out.Zone = zone }
	loc, e := time.LoadLocation(zone)
	if e != nil { return out, ErrGalleryInput }
	if basis == "capture" { loc = time.UTC }
	q.DateBasis, q.From, q.Until, q.NoCapture = "", "", "", false
	s.gate.Lock()
	defer s.gate.Unlock()
	e = s.db.View(func(tx *bolt.Tx) error {
		if q.Album != "" { if _, e := curationAlbum(tx, q.Album); e != nil { return ErrGalleryChanged } }
		index, e := s.gallerySearchFor(tx)
		if e != nil { return e }
		years, months, days := map[int]uint64{}, map[int]uint64{}, map[int]uint64{}
		for _, row := range index.assets {
			if !q.matches(row) { continue }
			out.Count++
			capture, err := time.Parse("2006-01-02T15:04:05", row.captured)
			validCapture := err == nil && capture.Format("2006-01-02T15:04:05") == row.captured && capture.Year() >= 1
			if !validCapture { out.MissingCapture++ }
			value := row.added.In(loc)
			if basis == "capture" { if !validCapture { continue }; value = capture }
			if value.Year() < 1 || value.Year() > 9999 { continue }
			years[value.Year()]++
			if value.Year() == year {
				months[int(value.Month())]++
				if int(value.Month()) == month { days[value.Day()]++ }
			}
		}
		bucket := func(y,m,d, unit int, count uint64) GalleryCalendarBucket {
			from := galleryCalendarBoundary(y,time.Month(m),d,loc)
			// Advance calendar coordinates, rather than a possibly shifted
			// 01:00 start on a date whose midnight did not exist.
			next := time.Date(y,time.Month(m),d+1,0,0,0,0,time.UTC)
			if unit == 1 { next = time.Date(y+1,1,1,0,0,0,0,time.UTC) }
			if unit == 2 { next = time.Date(y,time.Month(m)+1,1,0,0,0,0,time.UTC) }
			until := galleryCalendarBoundary(next.Year(),next.Month(),next.Day(),loc)
			layout := "2006-01-02T15:04:05"
			if basis == "upload" {
				layout = time.RFC3339
				if from.UTC().Year() < 1 { from = time.Date(1,1,1,0,0,0,0,time.UTC) }
			}
			b := GalleryCalendarBucket{Count:count, From:from.Format(layout), Until:until.Format(layout)}
			// The normalized query cannot represent year 10000; leave the upper
			// boundary open for the last representable bucket.
			if until.Year() > 9999 || (basis == "upload" && until.UTC().Year() > 9999) { b.Until = "" }
			if unit == 1 { b.Year = y }
			if unit == 2 { b.Month = m }
			if unit == 3 { b.Day = d }
			return b
		}
		for y := 9999; y >= 1; y-- { if years[y] > 0 { out.Years = append(out.Years, bucket(y,1,1,1,years[y])) } }
		for m := 1; m <= 12; m++ { if months[m] > 0 { out.Months = append(out.Months, bucket(year,m,1,2,months[m])) } }
		for d := 1; d <= 31; d++ { if days[d] > 0 { out.Days = append(out.Days, bucket(year,month,d,3,days[d])) } }
		return nil
	})
	return out, e
}
