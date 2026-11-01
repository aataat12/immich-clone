package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"immichclone/internal/diagnostics"
	"immichclone/internal/library"
	"immichclone/internal/media"
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
	diagnostics.Record("startup", "begin", nil, 0, 0, 0)
	host, _, e := net.SplitHostPort(o.Address)
	if e != nil || net.ParseIP(host) == nil || !net.ParseIP(host).IsLoopback() {
		return nil, errors.New("loopback address required")
	}
	s, e := library.Open(o.Data)
	diagnostics.Record("startup", "catalog_open", e, 0, 0, 0)
	if e != nil {
		return nil, e
	}
	if exe, err := os.Executable(); err == nil {
		s.SetMediaPreview(media.New(filepath.Join(filepath.Dir(exe), "media", media.ProviderID)).Thumbnail)
	}
	code, e := web.AccessCode(o.Data)
	diagnostics.Record("startup", "access_code", e, 0, 0, 0)
	if e != nil {
		s.Close()
		return nil, e
	}
	listener, e := net.Listen("tcp", o.Address)
	diagnostics.Record("startup", "listen", e, 0, 0, 0)
	if e != nil {
		s.Close()
		return nil, e
	}
	diagnostics.Record("startup", "ready", nil, 0, 0, 0)
	server := &http.Server{Handler: web.New(s, o.Data, code, version), ReadHeaderTimeout: 5 * time.Second, IdleTimeout: 60 * time.Second, MaxHeaderBytes: 16 << 10}
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
				if _, e := s.ProcessOneContext(child); e != nil {
					diagnostics.Record("preview", "failed", e, 0, 0, 0)
					log.Print("Preview worker encountered a storage error; job remains persistent")
				}
			}
		}
	}()
	serveDone := make(chan error, 1)
	go func() {
		err := server.Serve(listener)
		logged := err
		if errors.Is(logged, http.ErrServerClosed) {
			logged = nil
		}
		diagnostics.Record("shutdown", "listen", logged, 0, 0, 0)
		serveDone <- err
	}()
	return func() error {
		diagnostics.Record("shutdown", "begin", nil, 0, 0, 0)
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
		diagnostics.Record("shutdown", "complete", closeErr, 0, 0, 0)
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
	logs := flag.String("logs", diagnostics.DefaultDir(), "private diagnostic directory")
	archive := flag.String("archive", "", "backup archive file")
	flag.Parse()
	if *mode == "version" {
		fmt.Println(version)
		return
	}
	channel := "commands"
	if *mode == "service" || *mode == "serve" {
		channel = "application"
	}
	if e := diagnostics.Init(*logs, version, channel); e != nil {
		fmt.Fprintln(os.Stderr, "Diagnostic file unavailable; check logging directory permissions")
	}
	diagnostics.Record("startup", "requested", nil, 0, 0, 0)
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
			diagnostics.Record("restore", "failed", e, 0, 0, 0)
			log.Fatal("Restore refused or failed validation; existing data was not replaced")
		}
		diagnostics.Record("restore", "complete", nil, 0, 0, 0)
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
	diagnostics.Record("startup", "catalog_open", e, 0, 0, 0)
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
			diagnostics.Record("verify", "failed", e, 0, 0, 0)
			log.Fatal("Original integrity verification failed")
		}
		diagnostics.Record("verify", "complete", nil, 0, 0, 0)
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
