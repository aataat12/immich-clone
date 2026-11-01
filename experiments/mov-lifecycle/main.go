// Synthetic catalog/job lifecycle probe. It deliberately simulates a decoder
// failure; it does not claim to reproduce any private MOV or validate its codec.
package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"image"
	"immichclone/internal/library"
	"os"
	"path/filepath"
)

func check(ok bool) {
	if !ok {
		panic("synthetic MOV lifecycle assertion failed")
	}
}
func main() {
	encoded, e := os.ReadFile("experiments/media-worker/fixtures/rotated.mov.b64")
	if e != nil {
		panic(e)
	}
	data, e := base64.StdEncoding.DecodeString(string(encoded))
	if e != nil {
		panic(e)
	}
	root, e := os.MkdirTemp("", "synthetic-mov-lifecycle-")
	if e != nil {
		panic(e)
	}
	defer os.RemoveAll(root)
	storePath := filepath.Join(root, "state")
	s, e := library.Open(storePath)
	if e != nil {
		panic(e)
	}
	a, duplicate, e := s.Put("synthetic.mov", bytes.NewReader(data))
	check(e == nil && !duplicate && a.MIME == "video/quicktime")
	s.SetMediaPreview(func(context.Context, string) (image.Image, int, int, error) {
		return nil, 0, 0, errors.New("synthetic unsupported decoder result")
	})
	_, e = s.ProcessOne()
	check(e == nil)
	items, e := s.List()
	check(e == nil && len(items) == 1 && items[0].ID == a.ID && items[0].Preview == "failed")
	original, e := os.ReadFile(s.Original(a.ID))
	check(e == nil && bytes.Equal(original, data))
	again, duplicate, e := s.Put("synthetic-again.mov", bytes.NewReader(data))
	check(e == nil && duplicate && again.ID == a.ID)
	items, e = s.List()
	check(e == nil && len(items) == 1)
	check(s.Close() == nil)
	s, e = library.Open(storePath)
	if e != nil {
		panic(e)
	}
	defer s.Close()
	items, e = s.List()
	check(e == nil && len(items) == 1 && items[0].Preview == "failed")
	check(s.Verify() == nil)
	fmt.Println("PASS: synthetic MOV remains listed after simulated decoder rejection, duplicate re-upload and catalog reopen; original bytes/checksum unchanged. This is lifecycle evidence only, not reproduction of the user's MOV.")
}
