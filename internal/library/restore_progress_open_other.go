//go:build !windows

package library

import "os"

func openRestoreProgress(path string) (*os.File, error) { return os.Open(path) }
