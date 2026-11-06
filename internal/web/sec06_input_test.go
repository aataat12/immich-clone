package web

import (
	"net/http/httptest"
	"strings"
	"testing"
)

func TestSEC06BoundedJSONEncodingAndFraming(t *testing.T) {
	for _, body := range []string{"{\"name\":\"" + string([]byte{255}) + "\"}", `{"name":"ok"} {}`, `{"unknown":1}`, `{"name":"ok"}` + strings.Repeat(" ", 128)} {
		r := httptest.NewRequest("POST", "/", strings.NewReader(body))
		w := httptest.NewRecorder()
		var out struct {
			Name string `json:"name"`
		}
		if boundedJSON(w, r, 64, &out) == nil {
			t.Fatal("malformed input accepted")
		}
	}
	r := httptest.NewRequest("POST", "/", strings.NewReader(`{"name":"caf\u00e9 <script>"}`))
	w := httptest.NewRecorder()
	var out struct {
		Name string `json:"name"`
	}
	if err := boundedJSON(w, r, 64, &out); err != nil || out.Name != "caf\u00e9 <script>" {
		t.Fatal("valid Unicode display data", err)
	}
}
