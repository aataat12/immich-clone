package web

import (
	"context"
	"encoding/json"
	"fmt"
	"immichclone/internal/library"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestGalleryBrowseExportBudgetResponse(t *testing.T) {
	for _, cause := range []error{library.ErrImportHistoryExportLimit, fmt.Errorf("wrapped export budget: %w", library.ErrImportHistoryExportLimit)} {
		w := httptest.NewRecorder()
		catalogExportError(w, cause)
		if w.Code != 413 || !strings.Contains(w.Body.String(), "Use a full backup") || strings.Contains(w.Body.String(), "catalog unavailable") || w.Header().Get("Content-Disposition") != "" {
			t.Fatal("misleading budget failure", w.Code, w.Body.String())
		}
	}
}

func TestGalleryBrowseHTTPContractsAndAuthentication(t *testing.T) {
	s, c := setup(t)
	for _, path := range []string{"/api/gallery/sources", "/api/gallery/calendar"} {
		if w := request(s, "GET", path, "", nil); w.Code != 401 {
			t.Fatal("unprotected browse", path, w.Code)
		}
	}
	data := "synthetic recorded media"
	m, e := s.Store.CreateImport([]library.ImportFile{{Name: "recorded.txt", Relative: "Trip/Day/recorded.txt", Size: int64(len(data))}})
	if e != nil {
		t.Fatal(e)
	}
	if e := s.Store.ControlImport(m.ID, "start"); e != nil {
		t.Fatal(e)
	}
	entry, e := s.Store.PutImport(context.Background(), m.ID, 0, strings.NewReader(data))
	if e != nil {
		t.Fatal(e)
	}
	loose, _, e := s.Store.Put("loose.txt", strings.NewReader("unfiled synthetic original"))
	if e != nil {
		t.Fatal(e)
	}
	o, e := s.Store.AssetOrganization(entry.AssetID)
	if e != nil {
		t.Fatal(e)
	}
	if _, e := s.Store.EditCuration(library.CurationEdit{Action: "set_asset", AssetID: entry.AssetID, Added: o.Added, Expected: o.Token, Title: "Searchable title", Caption: "Searchable caption", Notes: "Unicode 雪 notes"}); e != nil {
		t.Fatal(e)
	}
	base := "?sort=name&visibility=library&import_batch=" + m.ID
	w := request(s, "GET", "/api/gallery/sources"+base, "", c)
	var sources library.GallerySources
	if w.Code != 200 || json.Unmarshal(w.Body.Bytes(), &sources) != nil || len(sources.Folders) != 1 || sources.Folders[0].Path != "Trip" || sources.Count != 1 || sources.RootCount != 1 || len(sources.Batches) != 1 || sources.Batches[0].Count != 1 {
		t.Fatal(w.Code, w.Body.String())
	}
	if sources.Batches[0].Accepted != 1 || sources.Batches[0].New != 1 || sources.Batches[0].Total != 1 || sources.Batches[0].Pending != 0 {
		t.Fatal("receipt outcome HTTP", sources.Batches[0])
	}
	for _, tc := range []struct {
		suffix string
		count  uint64
	}{
		{"&source_folder=Trip&source_mode=direct", 0},
		{"&source_folder=Trip%2FDay&source_mode=direct", 1},
		{"&source_folder=Trip&source_mode=subtree", 1},
		{"&source_missing=true", 0},
		{"&source_missing=false&source_folder=Trip", 1},
	} {
		w := request(s, "GET", "/api/gallery"+base+tc.suffix, "", c)
		var result library.GalleryPage
		if w.Code != 200 || json.Unmarshal(w.Body.Bytes(), &result) != nil || result.Count != tc.count {
			t.Fatal(tc, w.Code, w.Body.String())
		}
	}
	w = request(s, "GET", "/api/gallery?sort=name&visibility=library&source_missing=true", "", c)
	var missing library.GalleryPage
	if w.Code != 200 || json.Unmarshal(w.Body.Bytes(), &missing) != nil || missing.Count != 1 || missing.Assets[0].ID != loose.ID {
		t.Fatal("unfiled HTTP", w.Code, w.Body.String())
	}
	w = request(s, "GET", "/api/gallery/sources?sort=name&visibility=library&source_missing=true", "", c)
	if w.Code != 200 || json.Unmarshal(w.Body.Bytes(), &sources) != nil || sources.Count != 1 || sources.MissingCount != 1 || sources.RootCount != 1 || len(sources.Folders) != 1 {
		t.Fatal("unfiled facet HTTP", w.Code, w.Body.String())
	}
	w = request(s, "GET", "/api/export", "", c)
	var export struct {
		ImportHistory library.ImportHistoryExport `json:"import_history"`
	}
	if w.Code != 200 || json.Unmarshal(w.Body.Bytes(), &export) != nil || export.ImportHistory.Version != 1 || len(export.ImportHistory.Sessions[m.ID].Entries) != 1 {
		t.Fatal("receipt interchange HTTP", w.Code, w.Body.String())
	}
	w = request(s, "GET", "/api/gallery"+base+"&source_folder=Trip%2FDay&q="+url.QueryEscape("雪 notes"), "", c)
	var page library.GalleryPage
	if w.Code != 200 || json.Unmarshal(w.Body.Bytes(), &page) != nil || page.Count != 1 || page.Assets[0].ID != entry.AssetID || page.Query.SourceFolder != "Trip/Day" {
		t.Fatal(w.Code, w.Body.String())
	}
	w = request(s, "GET", "/api/gallery/calendar"+base+"&calendar_basis=upload&calendar_zone=America%2FChicago", "", c)
	var calendar library.GalleryCalendar
	if w.Code != 200 || json.Unmarshal(w.Body.Bytes(), &calendar) != nil || calendar.Count != 1 || len(calendar.Years) != 1 || calendar.Years[0].From == "" {
		t.Fatal(w.Code, w.Body.String())
	}
	w = request(s, "GET", "/api/gallery/calendar"+base+"&calendar_basis=capture", "", c)
	if w.Code != 200 || json.Unmarshal(w.Body.Bytes(), &calendar) != nil || calendar.MissingCapture != 1 || len(calendar.Years) != 0 || calendar.Months == nil || calendar.Days == nil {
		t.Fatal(w.Code, w.Body.String())
	}
	for _, tc := range []struct{ path, suffix string }{
		{"/api/gallery", "&source_folder=..%2FTrip"},
		{"/api/gallery", "&source_folder=%2FTrip"},
		{"/api/gallery", "&source_folder=Trip&source_folder=Other"},
		{"/api/gallery", "&source_mode=unknown"},
		{"/api/gallery", "&source_missing=maybe"},
		{"/api/gallery", "&source_missing=true&source_folder=Trip"},
		{"/api/gallery", "&source_missing=true&source_missing=false"},
		{"/api/gallery", "&import_batch=invalid"},
		{"/api/gallery", "&calendar_basis=capture"},
		{"/api/gallery/sources", "&after=cursor"},
		{"/api/gallery/sources", "&source_after=.."},
		{"/api/gallery/sources", "&limit=201"},
		{"/api/gallery/sources", "&source_after=a&source_after=b"},
		{"/api/gallery/calendar", "&calendar_basis=capture&month=2"},
		{"/api/gallery/calendar", "&calendar_basis=capture&year=2024&month=13"},
		{"/api/gallery/calendar", "&calendar_basis=capture&year=10000"},
		{"/api/gallery/calendar", "&calendar_basis=capture&year=0"},
		{"/api/gallery/calendar", "&calendar_basis=capture&calendar_zone=Missing%2FZone"},
		{"/api/gallery/calendar", "&calendar_basis=capture&calendar_basis=upload"},
		{"/api/gallery/calendar", "&calendar_basis=capture&limit=60"},
	} {
		if w := request(s, "GET", tc.path+base+tc.suffix, "", c); w.Code != 400 {
			t.Fatal(tc, w.Code, w.Body.String())
		}
	}
	for _, path := range []string{"/api/gallery/sources", "/api/gallery/calendar"} {
		if w := request(s, "POST", path+base, "{}", c); w.Code != 405 {
			t.Fatal("browse method", path, w.Code, w.Body.String())
		}
	}
}
