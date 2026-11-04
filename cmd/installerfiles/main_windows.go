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
	flag.Parse()
	var err error
	switch *action {
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
