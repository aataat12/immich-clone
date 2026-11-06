package web

import (
	"context"
	"encoding/json"
	"immichclone/internal/library"
	"net/url"
	"strings"
	"testing"
)

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
