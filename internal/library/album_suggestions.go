package library

import (
	"context"
	"encoding/json"
	"os"
	"regexp"
	"sort"
	"strings"
	"time"

	bolt "go.etcd.io/bbolt"
)

const (
	suggestionPageSize = 12
	suggestionContext  = 6
	captureTimeLayout  = "2006-01-02T15:04:05"
)

// Accept full camera-style timestamps at the start of the name, optionally
// following a recognized camera prefix. A date alone has no ordering precision.
var cameraFilenameTime = regexp.MustCompile(`(?i)^(?:(?:IMG|VID|PXL|DSC|MVIMG|MOV)[_-])?(\d{4})-?(\d{2})-?(\d{2})[ _T-](\d{2})[.:-]?(\d{2})[.:-]?(\d{2})(?:\d{3})?(?:[ ._()+-]|$)`)

type SuggestionMedia struct {
	ID      string    `json:"id"`
	Added   time.Time `json:"added"`
	Name    string    `json:"name"`
	MIME    string    `json:"mime"`
	Preview string    `json:"preview"`
	Capture string    `json:"capture"`
	Basis   string    `json:"basis"`
}

type AlbumSuggestion struct {
	ID    string `json:"id"`
	Start string `json:"start"`
	End   string `json:"end"`
	Count int    `json:"count"`
}

type AlbumSuggestionPage struct {
	Suggestions []AlbumSuggestion `json:"suggestions"`
	Next        string            `json:"next,omitempty"`
	Undated     int               `json:"undated"`
}

type AlbumSuggestionDetail struct {
	Items []SuggestionMedia `json:"items"`
	Start int               `json:"start"`
	End   int               `json:"end"` // exclusive
}

func suggestionDate(a Asset) (string, string) {
	if stamp, err := time.Parse(captureTimeLayout, a.Photo.Captured); err == nil && stamp.Year() > 0 {
		return a.Photo.Captured, "embedded"
	}
	m := cameraFilenameTime.FindStringSubmatch(a.Name)
	if m == nil {
		return "", ""
	}
	value := m[1] + "-" + m[2] + "-" + m[3] + "T" + m[4] + ":" + m[5] + ":" + m[6]
	if stamp, err := time.Parse(captureTimeLayout, value); err == nil && stamp.Year() >= 1900 {
		return value, "filename"
	}
	return "", ""
}

type suggestionSpan struct{ start, end int }

func suggestionSpans(items []SuggestionMedia, trips bool) []suggestionSpan {
	gap, duration, minimum := 3*time.Hour, 24*time.Hour, 3
	if trips {
		gap, duration, minimum = 18*time.Hour, 7*24*time.Hour, 6
	}
	var spans []suggestionSpan
	for start := 0; start < len(items); {
		first, _ := time.Parse(captureTimeLayout, items[start].Capture)
		previous, end := first, start+1
		for ; end < len(items) && end-start < maxInitialAlbumMembers-2*suggestionContext; end++ {
			current, _ := time.Parse(captureTimeLayout, items[end].Capture)
			if current.Sub(previous) > gap || current.Sub(first) > duration {
				break
			}
			previous = current
		}
		if end-start >= minimum && (!trips || previous.Sub(first) >= 18*time.Hour && previous.Day() != first.Day()) {
			spans = append(spans, suggestionSpan{start, end})
		}
		start = end
	}
	return spans
}

func (s *Store) suggestionTimeline(ctx context.Context) ([]SuggestionMedia, int, error) {
	var items []SuggestionMedia
	undated := 0
	if s.offline != nil {
		return nil, 0, ErrGalleryInput
	}
	err := s.db.View(func(tx *bolt.Tx) error {
		return tx.Bucket([]byte("assets")).ForEach(func(_, raw []byte) error {
			if err := ctx.Err(); err != nil {
				return err
			}
			var a Asset
			if err := json.Unmarshal(raw, &a); err != nil {
				return err
			}
			if a.Hidden || !(strings.HasPrefix(a.MIME, "image/") || strings.HasPrefix(a.MIME, "video/")) {
				return nil
			}
			capture, basis := suggestionDate(a)
			if capture == "" {
				undated++
				return nil
			}
			items = append(items, SuggestionMedia{a.ID, a.Added, a.Name, a.MIME, a.Preview, capture, basis})
			return nil
		})
	})
	if err != nil {
		return nil, 0, err
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].Capture == items[j].Capture {
			return items[i].ID < items[j].ID
		}
		return items[i].Capture < items[j].Capture
	})
	return items, undated, nil
}

func (s *Store) SuggestedAlbums(ctx context.Context, trips bool, after string) (AlbumSuggestionPage, error) {
	out := AlbumSuggestionPage{Suggestions: []AlbumSuggestion{}}
	if after != "" && !validID(after) {
		return out, ErrGalleryInput
	}
	items, undated, err := s.suggestionTimeline(ctx)
	if err != nil {
		return out, err
	}
	out.Undated = undated
	spans := suggestionSpans(items, trips)
	for i := len(spans) - 1; i >= 0; i-- {
		span := spans[i]
		first, last := items[span.start], items[span.end-1]
		if after != "" {
			if first.ID == after {
				after = ""
			}
			continue
		}
		if len(out.Suggestions) == suggestionPageSize {
			out.Next = out.Suggestions[suggestionPageSize-1].ID
			break
		}
		out.Suggestions = append(out.Suggestions, AlbumSuggestion{first.ID, first.Capture, last.Capture, span.end - span.start})
	}
	if after != "" {
		return out, ErrGalleryChanged
	}
	return out, nil
}

func (s *Store) SuggestedAlbum(ctx context.Context, trips bool, id string) (AlbumSuggestionDetail, error) {
	if !validID(id) {
		return AlbumSuggestionDetail{}, ErrGalleryInput
	}
	items, _, err := s.suggestionTimeline(ctx)
	if err != nil {
		return AlbumSuggestionDetail{}, err
	}
	for _, span := range suggestionSpans(items, trips) {
		if items[span.start].ID == id {
			from, until := max(0, span.start-suggestionContext), min(len(items), span.end+suggestionContext)
			return AlbumSuggestionDetail{items[from:until], span.start - from, span.end - from}, nil
		}
	}
	return AlbumSuggestionDetail{}, os.ErrNotExist
}
