package web

import (
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"
)

func TestConnectionLimitRejectsExcessAndReusesReleasedSlot(t *testing.T) {
	raw, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	listener := LimitConnections(raw, 2)
	defer listener.Close()
	accepted := make(chan net.Conn, 8)
	done := make(chan error, 1)
	go func() {
		for {
			conn, e := listener.Accept()
			if e != nil {
				done <- e
				return
			}
			accepted <- conn
		}
	}()
	dial := func() net.Conn {
		t.Helper()
		c, e := net.DialTimeout("tcp", raw.Addr().String(), time.Second)
		if e != nil {
			t.Fatal(e)
		}
		t.Cleanup(func() { c.Close() })
		return c
	}
	receive := func() net.Conn {
		t.Helper()
		select {
		case c := <-accepted:
			t.Cleanup(func() { c.Close() })
			return c
		case <-time.After(time.Second):
			t.Fatal("admitted connection not accepted")
			return nil
		}
	}
	rejected := func() {
		t.Helper()
		c := dial()
		c.SetReadDeadline(time.Now().Add(time.Second))
		var b [1]byte
		_, e := c.Read(b[:])
		if e == nil {
			t.Fatal("excess connection accepted")
		}
		if n, ok := e.(net.Error); ok && n.Timeout() {
			t.Fatal("excess connection queued instead of closed")
		}
	}
	dial()
	first := receive()
	dial()
	receive()
	rejected()
	var wg sync.WaitGroup
	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func() { defer wg.Done(); first.Close() }()
	}
	wg.Wait()
	dial()
	receive()
	rejected() // repeated Close must not release another slot
	listener.Close()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("listener close did not unblock Accept")
	}
}
func TestLimitedListenerHTTPRecovery(t *testing.T) {
	server := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.Write([]byte("ok")) }))
	server.Listener = LimitConnections(server.Listener, 2)
	server.Config.ReadHeaderTimeout = 5 * time.Second
	server.Start()
	defer server.Close()
	clients := []net.Conn{}
	for i := 0; i < 2; i++ {
		c, e := net.DialTimeout("tcp", server.Listener.Addr().String(), time.Second)
		if e != nil {
			t.Fatal(e)
		}
		clients = append(clients, c)
		defer c.Close()
	}
	third, e := net.DialTimeout("tcp", server.Listener.Addr().String(), time.Second)
	if e != nil {
		t.Fatal(e)
	}
	third.SetReadDeadline(time.Now().Add(time.Second))
	var b [1]byte
	_, e = third.Read(b[:])
	third.Close()
	if e == nil {
		t.Fatal("idle sockets did not count toward limit")
	}
	if n, ok := e.(net.Error); ok && n.Timeout() {
		t.Fatal("overflow socket not closed")
	}
	clients[0].Close()
	clients[1].Close()
	transport := &http.Transport{DisableKeepAlives: true}
	defer transport.CloseIdleConnections()
	client := &http.Client{Timeout: time.Second, Transport: transport}
	deadline := time.Now().Add(2 * time.Second)
	for {
		response, e := client.Get(server.URL)
		if e == nil {
			data, _ := io.ReadAll(response.Body)
			response.Body.Close()
			if response.StatusCode == 200 && string(data) == "ok" {
				break
			}
		}
		if time.Now().After(deadline) {
			t.Fatal("normal HTTP did not recover after closing idle peers")
		}
		time.Sleep(10 * time.Millisecond)
	}
}
