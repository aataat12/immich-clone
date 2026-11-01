//go:build !windows

package main

import "errors"

func runService(o options) error { return errors.New("native service mode requires Windows") }
