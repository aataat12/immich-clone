package web

import (
	"archive/zip"
	"bufio"
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func backupDownload(t *testing.T, s *Server, c *http.Cookie, receiveBuffer int) (net.Conn, *http.Response) {
	t.Helper()
	server := httptest.NewServer(s)
	t.Cleanup(server.Close)
	address := strings.TrimPrefix(server.URL, "http://")
	conn, e := net.DialTimeout("tcp", address, time.Second)
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() { conn.Close() })
	if tcp, ok := conn.(*net.TCPConn); ok {
		if e = tcp.SetReadBuffer(receiveBuffer); e != nil {
			t.Fatal(e)
		}
	}
	conn.SetDeadline(time.Now().Add(15 * time.Second))
	if _, e = fmt.Fprintf(conn, "GET /api/backup HTTP/1.1\r\nHost: %s\r\nCookie: session=%s\r\n\r\n", address, c.Value); e != nil {
		t.Fatal(e)
	}
	response, e := http.ReadResponse(bufio.NewReader(conn), nil)
	if e != nil {
		t.Fatal(e)
	}
	if response.StatusCode != 200 {
		t.Fatal("backup did not start", response.StatusCode)
	}
	return conn, response
}
func TestStalledDownloadReleasesExportSlot(t *testing.T) {
	s, c := setup(t)
	s.writeIdle = 200 * time.Millisecond
	data := make([]byte, 16<<20)
	if _, e := rand.Read(data); e != nil {
		t.Fatal(e)
	}
	if _, _, e := s.Store.Put("random-fixture.bin", bytes.NewReader(data)); e != nil {
		t.Fatal(e)
	}
	conn, response := backupDownload(t, s, c, 32<<10)
	defer response.Body.Close()
	defer conn.Close()
	// Leave the TCP receive window full; another export must recover on timeout,
	// before the client closes or drains the original response.
	deadline := time.Now().Add(4 * time.Second)
	for {
		result := request(s, "GET", "/api/export", "", c)
		if result.Code == 200 {
			break
		}
		if result.Code != 429 {
			t.Fatal("unexpected export response", result.Code)
		}
		if time.Now().After(deadline) {
			t.Fatal("slow client stranded export slot")
		}
		time.Sleep(25 * time.Millisecond)
	}
	if e := s.Store.Verify(); e != nil {
		t.Fatal(e)
	}
}
func TestProgressingDownloadExceedsWriteIdleWindow(t *testing.T) {
	s, c := setup(t)
	s.writeIdle = 2 * time.Second
	data := make([]byte, 8<<20)
	if _, e := rand.Read(data); e != nil {
		t.Fatal(e)
	}
	asset, _, e := s.Store.Put("progress-fixture.bin", bytes.NewReader(data))
	if e != nil {
		t.Fatal(e)
	}
	conn, response := backupDownload(t, s, c, 256<<10)
	defer response.Body.Close()
	defer conn.Close()
	started := time.Now()
	var archive bytes.Buffer
	for {
		_, e = io.CopyN(&archive, response.Body, 64<<10)
		if e == io.EOF {
			break
		}
		if e != nil {
			t.Fatal("progressing response truncated", e)
		}
		time.Sleep(35 * time.Millisecond)
	}
	if time.Since(started) <= s.writeIdle {
		t.Fatal("fixture did not exceed idle window")
	}
	if int64(archive.Len()) != response.ContentLength {
		t.Fatal("incomplete archive")
	}
	zr, e := zip.NewReader(bytes.NewReader(archive.Bytes()), int64(archive.Len()))
	if e != nil {
		t.Fatal(e)
	}
	found := false
	for _, file := range zr.File {
		if file.Name == "originals/"+asset.ID {
			r, e := file.Open()
			if e != nil {
				t.Fatal(e)
			}
			h := sha256.New()
			_, e = io.Copy(h, r)
			r.Close()
			if e != nil {
				t.Fatal(e)
			}
			want := sha256.Sum256(data)
			if !bytes.Equal(h.Sum(nil), want[:]) {
				t.Fatal("downloaded original hash changed")
			}
			found = true
		}
	}
	if !found {
		t.Fatal("downloaded original missing")
	}
}
