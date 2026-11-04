package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"immichclone/internal/diagnostics"
	"immichclone/internal/library"
	"immichclone/internal/media"
	"immichclone/internal/recognition"
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
		decoder := media.New(filepath.Join(filepath.Dir(exe), "media", media.ProviderID))
		s.SetMediaPreview(decoder.Thumbnail)
		s.SetAnalysisMedia(decoder.AnalysisFrame, decoder.VideoInfo, decoder.FrameAt, decoder.Available())
		provider := recognition.New(filepath.Join(filepath.Dir(exe), "ai", recognition.ProviderID))
		s.SetRecognizer(provider.Analyze, provider.Available())
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
	listener = web.LimitConnections(listener, web.MaxConnections)
	diagnostics.Record("startup", "ready", nil, 0, 0, 0)
	handler := web.New(s, o.Data, code, version)
	server := &http.Server{Handler: handler, ReadHeaderTimeout: 5 * time.Second, IdleTimeout: 60 * time.Second, MaxHeaderBytes: 16 << 10}
	child, cancel := context.WithCancel(ctx)
	workerDone := make(chan struct{})
	go func() {
		defer close(workerDone)
		runWorker(child, s, time.Second)
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
		handler.Close()
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

// Drain ready previews serially. ProcessOneContext rechecks the import lease and
// cancellation before each item. Idle, AI and storage-error paths retain their
// polling delay so an unavailable store cannot create a busy retry loop.
func runWorker(ctx context.Context, s *library.Store, idle time.Duration) {
	for ctx.Err() == nil {
		if processed, err := s.ProcessOneContext(ctx); err != nil {
			diagnostics.Record("preview", "failed", err, 0, 0, 0)
			log.Print("Preview worker encountered a storage error; job remains persistent")
		} else if processed {
			continue
		} else if _, err := s.ProcessAnalysis(ctx); err != nil {
			log.Print("AI worker encountered a storage error; job remains persistent")
		}
		timer := time.NewTimer(idle)
		select {
		case <-ctx.Done():
			timer.Stop()
			return
		case <-timer.C:
		}
	}
}

func main() {
	data := flag.String("data", "", "application data directory outside cloud sync folders")
	address := flag.String("listen", "127.0.0.1:2283", "loopback listen address")
	mode := flag.String("mode", "serve", "serve, service, init, verify, backup, restore, show-code or version")
	logs := flag.String("logs", diagnostics.DefaultDir(), "private diagnostic directory")
	archive := flag.String("archive", "", "backup archive file")
	restoreBytes := flag.Uint64("restore-max-bytes", 0, "restore aggregate bytes (0: destination free bytes minus reserve)")
	restoreEntries := flag.Uint64("restore-max-entries", 0, "restore ZIP entries (0: 100000)")
	restoreManifest := flag.Uint64("restore-max-manifest-bytes", 0, "restore manifest bytes (0: 16MiB)")
	restoreCatalog := flag.Uint64("restore-max-catalog-bytes", 0, "restore catalog bytes (0: 4GiB)")
	restoreMetadata := flag.Uint64("restore-max-metadata-bytes", 0, "restore ZIP metadata bytes (0: 64MiB)")
	restoreMemory := flag.Uint64("restore-memory-bytes", 0, "restore worker memory budget (0: 512MiB)")
	restoreTimeout := flag.Duration("restore-timeout", 0, "restore wall timeout (0: 30m)")
	flag.Parse()
	if *mode == "version" {
		fmt.Println(version)
		return
	}
	if !validMode(*mode) {
		fmt.Fprintln(os.Stderr, "Unknown application mode refused before opening files.")
		os.Exit(1)
	}
	if e := checkInitializationMode(*mode); e != nil {
		message := "Elevated initialization is refused; use the installer to initialize under LocalService."
		if *mode == "serve" {
			message = "Elevated interactive serving is refused; use the installed LocalService service."
		}
		if *mode == "service" {
			message = "Service mode requires the installed LocalService service and Service Control Manager."
		}
		fmt.Fprintln(os.Stderr, message)
		os.Exit(1)
	}
	if *mode == "show-code" {
		if !filepath.IsAbs(*data) {
			fmt.Fprintln(os.Stderr, "An absolute existing application data directory is required.")
			os.Exit(1)
		}
		code, e := readExistingAccessCode(*data)
		if e != nil {
			fmt.Fprintln(os.Stderr, "Existing access code unavailable; no state was initialized.")
			os.Exit(1)
		}
		fmt.Println(code)
		return
	}
	if *mode == "verify" || *mode == "backup" {
		if e := runReadOnlyCommand(*mode, *data, *archive); e != nil {
			fmt.Fprintln(os.Stderr, "Offline operation refused or failed; source state was not initialized or modified. An incomplete private archive may remain.")
			os.Exit(1)
		}
		if *mode == "verify" {
			fmt.Println("All catalogued originals passed SHA256 verification")
		} else {
			fmt.Println("Consistent backup created")
		}
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
		ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
		defer cancel()
		limits := library.RestoreLimits{MaxBytes: *restoreBytes, MaxEntries: *restoreEntries, MaxManifestBytes: *restoreManifest, MaxCatalogBytes: *restoreCatalog, MaxMetadataBytes: *restoreMetadata, MemoryBytes: *restoreMemory, Timeout: *restoreTimeout}
		if e := library.RestoreWithLimits(ctx, *archive, *data, limits); e != nil {
			diagnostics.Record("restore", "failed", e, 0, 0, 0)
			if errors.Is(e, library.ErrRestoreCleanup) {
				log.Fatal("Restore failed; private staging beside the requested destination could not be fully cleaned. Existing data was not replaced.")
			}
			if errors.Is(e, library.ErrRestoreInvalidArchive) {
				log.Fatal("Restore archive failed structure, checksum, or catalog validation. Existing data was not replaced.")
			}
			if errors.Is(e, library.ErrRestoreWorkerFailed) || errors.Is(e, library.ErrRestoreOutputLimit) {
				log.Fatalf("Restore worker failed or exceeded its memory/process/output budget (detail %s); inspect the archive and selected -restore-memory-bytes budget before retrying. Existing data was not replaced.", library.RestoreFailureDetail(e))
			}
			if errors.Is(e, context.Canceled) {
				log.Fatal("Restore cancelled before publication; existing data was not replaced.")
			}
			if errors.Is(e, library.ErrRestoreBudget) {
				log.Fatal("Restore exceeded a configured byte/entry budget or destination capacity; adjust the restore budget flags or available space before retrying. Existing data was not replaced.")
			}
			if errors.Is(e, context.DeadlineExceeded) {
				log.Fatal("Restore exceeded its hard worker timeout; adjust -restore-timeout for a larger recovery. Existing data was not replaced.")
			}
			log.Fatal("Restore refused, failed validation, or exceeded worker resources; check selected restore limits. Existing data was not replaced.")
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
	default:
		log.Fatal("Unknown mode")
	}
}
