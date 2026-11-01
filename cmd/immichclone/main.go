package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"immichclone/internal/library"
	"immichclone/internal/web"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"time"
)

var version = "prototype"

type options struct{ Data, Address string }

func start(ctx context.Context, o options) (func() error, error) {
	host, _, e := net.SplitHostPort(o.Address)
	if e != nil || net.ParseIP(host) == nil || !net.ParseIP(host).IsLoopback() {
		return nil, errors.New("loopback address required")
	}
	s, e := library.Open(o.Data)
	if e != nil {
		return nil, e
	}
	code, e := web.AccessCode(o.Data)
	if e != nil {
		s.Close()
		return nil, e
	}
	listener, e := net.Listen("tcp", o.Address)
	if e != nil {
		s.Close()
		return nil, e
	}
	server := &http.Server{Handler: web.New(s, o.Data, code), ReadHeaderTimeout: 5 * time.Second, IdleTimeout: 60 * time.Second, MaxHeaderBytes: 16 << 10}
	child, cancel := context.WithCancel(ctx)
	workerDone := make(chan struct{})
	go func() {
		defer close(workerDone)
		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-child.Done():
				return
			case <-ticker.C:
				if _, e := s.ProcessOne(); e != nil {
					log.Print("Preview worker encountered a storage error; job remains persistent")
				}
			}
		}
	}()
	serveDone := make(chan error, 1)
	go func() { serveDone <- server.Serve(listener) }()
	return func() error {
		cancel()
		shutdown, c := context.WithTimeout(context.Background(), 15*time.Second)
		defer c()
		e := server.Shutdown(shutdown)
		if e != nil {
			server.Close()
		}
		<-serveDone
		<-workerDone
		closeErr := s.Close()
		if e != nil {
			return e
		}
		return closeErr
	}, nil
}
func main() {
	data := flag.String("data", "", "application data directory outside cloud sync folders")
	address := flag.String("listen", "127.0.0.1:2283", "loopback listen address")
	mode := flag.String("mode", "serve", "serve, service, init, verify, backup, restore, show-code or version")
	archive := flag.String("archive", "", "backup archive file")
	flag.Parse()
	if *mode == "version" {
		fmt.Println(version)
		return
	}
	if !filepath.IsAbs(*data) {
		log.Fatal("An absolute application data directory is required")
	}
	for _, component := range strings.FieldsFunc(filepath.Clean(*data), func(r rune) bool { return r == '/' || r == '\\' }) {
		if strings.EqualFold(component, "Dropbox") {
			log.Fatal("Application state must be outside cloud sync folders")
		}
	}
	o := options{*data, *address}
	if *mode == "service" {
		if e := runService(o); e != nil {
			log.Fatal("Windows service failed; verify permissions and configuration")
		}
		return
	}
	if *mode == "restore" {
		if e := library.Restore(*archive, *data); e != nil {
			log.Fatal("Restore refused or failed validation; existing data was not replaced")
		}
		fmt.Println("Restore verified and published; previews queued for regeneration")
		return
	}
	if *mode == "serve" {
		ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
		defer cancel()
		close, e := start(ctx, o)
		if e != nil {
			log.Fatal("Application startup failed; check permissions, port availability and schema")
		}
		log.Print("Prototype ready on loopback; no telemetry or remote listener")
		<-ctx.Done()
		if e = close(); e != nil {
			log.Fatal("Shutdown reported a storage error")
		}
		return
	}
	s, e := library.Open(*data)
	if e != nil {
		log.Fatal("Cannot open application state; stop the service before using offline commands")
	}
	defer s.Close()
	switch *mode {
	case "init":
		if _, e = web.AccessCode(*data); e != nil {
			log.Fatal("Access-code initialization failed")
		}
		fmt.Println("Application state initialized")
	case "show-code":
		code, e := web.AccessCode(*data)
		if e != nil {
			log.Fatal("Access code unavailable")
		}
		fmt.Println(code)
	case "verify":
		if e = s.Verify(); e != nil {
			log.Fatal("Original integrity verification failed")
		}
		fmt.Println("All catalogued originals passed SHA256 verification")
	case "backup":
		f, e := os.OpenFile(*archive, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
		if e != nil {
			log.Fatal("Backup destination must be a new writable file")
		}
		e = s.Backup(f)
		if e == nil {
			e = f.Sync()
		}
		ce := f.Close()
		if e == nil {
			e = ce
		}
		if e != nil {
			os.Remove(*archive)
			log.Fatal("Backup failed; incomplete archive removed")
		}
		fmt.Println("Consistent backup created")
	default:
		log.Fatal("Unknown mode")
	}
}
