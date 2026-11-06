package library

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"sort"
	"strings"
	"time"
)

// GalleryQuery is the single membership/order contract for pages, counts and
// viewer neighbors. New durable organization fields belong here as they land.
type GalleryQuery struct {
	Favorite string `json:"favorite,omitempty"`
 Album string `json:"album,omitempty"`
 Sort       string        `json:"sort"`
	Visibility string        `json:"visibility"`
	Text       string        `json:"q,omitempty"`
	Filename   string        `json:"filename,omitempty"`
	Media      string        `json:"media,omitempty"`
	People     GalleryLabels `json:"people,omitempty"`
	Tags       GalleryLabels `json:"tags,omitempty"`
	DateBasis  string        `json:"date_basis,omitempty"`
	From       string        `json:"from,omitempty"`
	Until      string        `json:"until,omitempty"`
	NoCapture  bool          `json:"no_capture,omitempty"`
}

// Exact case-insensitive labels; groups AND together. Exclude rejects any hit.
type GalleryLabels struct {
	Any     []string `json:"any,omitempty"`
	All     []string `json:"all,omitempty"`
	Exclude []string `json:"exclude,omitempty"`
}

func normalizeGalleryText(value string, max int) (string, error) {
	if !textOK(value, max) {
		return "", ErrGalleryInput
	}
	return strings.ToLower(strings.TrimSpace(value)), nil
}

func normalizeGalleryLabels(group GalleryLabels) (GalleryLabels, error) {
	out := GalleryLabels{}
	for _, pair := range []struct {
		in  []string
		out *[]string
	}{{group.Any, &out.Any}, {group.All, &out.All}, {group.Exclude, &out.Exclude}} {
		if len(pair.in) > 32 {
			return out, ErrGalleryInput
		}
		seen := map[string]bool{}
		for _, label := range pair.in {
			label, e := normalizeGalleryText(label, 128)
			if e != nil || label == "" {
				return out, ErrGalleryInput
			}
			if !seen[label] {
				*pair.out = append(*pair.out, label)
				seen[label] = true
			}
		}
		sort.Strings(*pair.out)
	}
	return out, nil
}

// Capture boundaries are recorded calendar times, without an invented zone.
// Import boundaries are explicit instants, resolved by the caller's display zone.
// Both use inclusive From / exclusive Until. Missing capture dates never fall back.
func (q GalleryQuery) Normalize() (GalleryQuery, error) {
	if q.Favorite!="" && q.Favorite!="true" && q.Favorite!="false" {return q,ErrGalleryInput}
 if q.Album!="" && !validID(q.Album){return q,ErrGalleryInput}
 if q.Sort=="manual" && q.Album==""{return q,ErrGalleryInput}
 if q.Sort != "capture" && q.Sort != "upload" && q.Sort != "name" && q.Sort!="manual" {
		return q, ErrGalleryInput
	}
	if q.Visibility != "library" && q.Visibility != "hidden" {
		return q, ErrGalleryInput
	}
	var e error
	if q.Text, e = normalizeGalleryText(q.Text, 128); e != nil {
		return q, e
	}
	if q.Filename, e = normalizeGalleryText(q.Filename, 128); e != nil {
		return q, e
	}
	if q.Media == "all" {
		q.Media = ""
	}
	if q.Media != "" && q.Media != "photo" && q.Media != "video" {
		return q, ErrGalleryInput
	}
	if q.People, e = normalizeGalleryLabels(q.People); e != nil {
		return q, e
	}
	if q.Tags, e = normalizeGalleryLabels(q.Tags); e != nil {
		return q, e
	}
	if q.DateBasis != "" && q.DateBasis != "capture" && q.DateBasis != "upload" {
		return q, ErrGalleryInput
	}
	if (q.From != "" || q.Until != "") && q.DateBasis == "" {
		return q, ErrGalleryInput
	}
	if q.NoCapture && (q.DateBasis != "" || q.From != "" || q.Until != "") {
		return q, ErrGalleryInput
	}
	var from, until time.Time
	for _, pair := range []struct {
		value  *string
		parsed *time.Time
	}{{&q.From, &from}, {&q.Until, &until}} {
		if *pair.value == "" {
			continue
		}
		layout := "2006-01-02T15:04:05"
		if q.DateBasis == "upload" {
			layout = time.RFC3339Nano
		}
		parsed, err := time.Parse(layout, *pair.value)
		if err != nil || parsed.Year() < 1 || parsed.Year() > 9999 {
			return q, ErrGalleryInput
		}
		if q.DateBasis == "capture" && parsed.Format(layout) != *pair.value {
			return q, ErrGalleryInput
		}
		if q.DateBasis == "upload" {
			parsed = parsed.UTC()
			if parsed.Year() < 1 || parsed.Year() > 9999 {
				return q, ErrGalleryInput
			}
			*pair.value = parsed.Format(time.RFC3339Nano)
		}
		*pair.parsed = parsed
	}
	if q.From != "" && q.Until != "" && !from.Before(until) {
		return q, ErrGalleryInput
	}
	// A bare basis adds no membership restriction.
	if q.From == "" && q.Until == "" {
		q.DateBasis = ""
	}
	return q, nil
}

func (q GalleryQuery) key() string {
	b, _ := json.Marshal(q)
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}
func (q GalleryQuery) filtered() bool {
	return q.Favorite!="" || q.Album!="" || q.Text != "" || q.Filename != "" || q.Media != "" || q.From != "" || q.Until != "" || q.NoCapture || len(q.People.Any)+len(q.People.All)+len(q.People.Exclude)+len(q.Tags.Any)+len(q.Tags.All)+len(q.Tags.Exclude) > 0
}

func (g GalleryLabels) matches(values []string) bool {
	contains := func(want string) bool {
		for _, v := range values {
			if v == want {
				return true
			}
		}
		return false
	}
	for _, v := range g.Exclude {
		if contains(v) {
			return false
		}
	}
	for _, v := range g.All {
		if !contains(v) {
			return false
		}
	}
	if len(g.Any) > 0 {
		for _, v := range g.Any {
			if contains(v) {
				return true
			}
		}
		return false
	}
	return true
}

func (q GalleryQuery) matches(row gallerySearchRow) bool {
	if q.Favorite!="" && row.favorite!=(q.Favorite=="true"){return false}
 if q.Album!="" {if _,ok:=row.albums[q.Album];!ok{return false}}
 if row.hidden != (q.Visibility == "hidden") || !strings.Contains(row.terms, q.Text) || !strings.Contains(row.filename, q.Filename) || !q.People.matches(row.people) || !q.Tags.matches(row.tags) {
		return false
	}
	if q.Media == "photo" && !strings.HasPrefix(row.mime, "image/") {
		return false
	}
	if q.Media == "video" && !strings.HasPrefix(row.mime, "video/") {
		return false
	}
	captured, e := time.Parse("2006-01-02T15:04:05", row.captured)
	validCapture := e == nil && captured.Format("2006-01-02T15:04:05") == row.captured
	if q.NoCapture {
		return !validCapture
	}
	if q.DateBasis == "capture" {
		if !validCapture {
			return false
		}
		value := captured.Format("2006-01-02T15:04:05")
		return (q.From == "" || value >= q.From) && (q.Until == "" || value < q.Until)
	}
	if q.DateBasis == "upload" {
		if q.From != "" {
			bound, _ := time.Parse(time.RFC3339Nano, q.From)
			if row.added.Before(bound) {
				return false
			}
		}
		if q.Until != "" {
			bound, _ := time.Parse(time.RFC3339Nano, q.Until)
			if !row.added.Before(bound) {
				return false
			}
		}
	}
	return true
}
