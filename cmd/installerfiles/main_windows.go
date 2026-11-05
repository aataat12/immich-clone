//go:build windows

package main

import (
	"errors"
	"flag"
	"fmt"
	"golang.org/x/sys/windows"
	"immichclone/internal/installerfiles"
	"immichclone/internal/winfs"
	"os"
)

func main() {
	action := flag.String("action", "", "snapshot or restore")
	app := flag.String("app-dir", "", "existing installation directory")
	data := flag.String("data-dir", "", "existing state directory")
	logRoot := flag.String("log-dir", "", "existing diagnostic directory")
	event := flag.String("event", "", "fixed diagnostic event")
	operation := flag.String("operation", "", "installer operation")
	stage := flag.String("stage", "", "fixed installer stage")
	result := flag.Int64("result", 0, "numeric result")
	root := flag.String("root-dir", "", "selected storage root")
	reader := flag.String("reader-sid", "", "storage reader SID")
	service := flag.Bool("service-access", false, "LocalService storage access")
	source := flag.String("source-dir", "", "staged payload directory")
	hadService := flag.Bool("had-service", false, "transaction has a prior service snapshot")
	backup := flag.String("backup-dir", "", "private publication rollback directory")
	flag.Parse()
	if *result < -2147483648 || *result > 2147483647 {
		os.Exit(1)
	}
	var err error
	switch *action {
    case "audit-upgrade":
        err = installerfiles.AuditUpgrade(*app, *data, *root)
    case "begin-upgrade":
        err = installerfiles.BeginUpgrade(*app, *data, *root, *hadService)
    case "commit-upgrade":
        err = installerfiles.CommitUpgrade(*app, *data, *root)
    case "recover-upgrade":
        var kind string
        kind, err = installerfiles.RecoverUpgrade(*app, *data, *root)
        if err == nil { fmt.Println(kind); return }

	case "audit-code":
		err = installerfiles.AuditCode(*app)
	case "audit-cleanup":
		err = installerfiles.AuditCleanup(*app, *data)
	case "cleanup":
		err = installerfiles.Cleanup(*app, *data)
	case "publish-payload":
		err = installerfiles.PublishPayload(*app, *source, *backup)
	case "recover-interrupted-payload":
		err = installerfiles.RecoverInterruptedPayload(*app, *backup)
	case "restore-payload":
		err = installerfiles.RestorePayload(*app, *backup)
	case "grant-code-read":
		err = installerfiles.GrantCodeReadAccess(*app)
	case "normalize-private-tree":
		err = installerfiles.NormalizePrivateTree(*root, *reader, *service)
	case "ensure-private-root":
		err = installerfiles.EnsurePrivateRoot(*root, *reader, *service)
	case "diagnostic":
		err = installerfiles.Diagnostic(*logRoot, *event, *operation, *stage, int32(*result))
	case "failure-note":
		err = installerfiles.FailureNote(*app, *stage, int32(*result))
	case "snapshot":
		err = installerfiles.Snapshot(*app, *data)
	case "restore":
		err = installerfiles.Restore(*app, *data)
	default:
		err = installerfiles.ErrRefused
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "Installer file operation failed; completion not recorded.")
		if *action == "audit-cleanup" {
			if errors.Is(err, windows.STATUS_SHARING_VIOLATION) || errors.Is(err, windows.ERROR_SHARING_VIOLATION) {
				os.Exit(32)
			}
			if errors.Is(err, windows.STATUS_ACCESS_DENIED) || errors.Is(err, windows.ERROR_ACCESS_DENIED) {
				os.Exit(5)
			}
			if errors.Is(err, winfs.ErrUnsafeObject) {
				os.Exit(2)
			}
			if errors.Is(err, installerfiles.ErrRefused) {
				os.Exit(3)
			}
		}
		os.Exit(1)
	}
	fmt.Println("Installer file set verified.")
}
