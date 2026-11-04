//go:build windows

package main

import (
	"flag"
	"fmt"
	"immichclone/internal/installerfiles"
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
	flag.Parse()
	if *result < -2147483648 || *result > 2147483647 {
		os.Exit(1)
	}
	var err error
	switch *action {
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
		os.Exit(1)
	}
	fmt.Println("Installer file set verified.")
}
