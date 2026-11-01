package media

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestMetadataRoutingAndBounds(t *testing.T) {
	for _, c := range []struct {
		json     string
		w, h     int
		selector string
		fails    bool
	}{
		{`{"streams":[{"index":0,"codec_type":"audio","codec_name":"aac"},{"index":1,"codec_type":"video","codec_name":"h264","width":160,"height":120,"side_data_list":[{"rotation":90}]}]}`, 120, 160, "0:1", false},
		{`{"streams":[{"index":0,"codec_type":"video","codec_name":"hevc","width":512,"height":512}],"stream_groups":[{"index":0,"type":"Tile Grid","components":[{"width":4032,"height":3024,"nb_tiles":48}]}]}`, 4032, 3024, "[0:g:0]", false},
		{`{"streams":[{"codec_type":"video","codec_name":"hevc","width":160,"height":120,"pix_fmt":"yuv420p10le"}]}`, 0, 0, "", true},
		{`{"streams":[{"codec_type":"video","codec_name":"hevc","width":999999999,"height":999999999}]}`, 0, 0, "", true},
		{`{"streams":[{"codec_type":"video","codec_name":"av1","width":32,"height":24}]}`, 0, 0, "", true},
		{`{"streams":[]}`, 0, 0, "", true},
	} {
		w, h, selector, e := dimensions([]byte(c.json))
		if (e != nil) != c.fails || (!c.fails && (w != c.w || h != c.h || selector != c.selector)) {
			t.Fatalf("wrong routing: %d %d %s %v", w, h, selector, e)
		}
	}
}
func TestMissingOrTamperedProviderIsOptional(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{"ffmpeg.exe", "ffprobe.exe"} {
		if e := os.WriteFile(filepath.Join(dir, name), []byte("untrusted synthetic executable"), 0600); e != nil {
			t.Fatal(e)
		}
	}
	p := New(dir)
	if p.available {
		t.Fatal("unverified executable accepted")
	}
	if _, _, _, e := p.Thumbnail(context.Background(), "unused"); e != ErrUnavailable {
		t.Fatal("provider absence should be explicit")
	}
}
