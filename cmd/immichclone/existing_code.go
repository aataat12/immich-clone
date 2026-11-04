package main

import (
	"encoding/hex"
	"errors"
	"io"
	"strings"
)

func existingCode(r io.Reader) (string, error) {
	b, e := io.ReadAll(io.LimitReader(r, 257))
	if e != nil {
		return "", e
	}
	if len(b) > 256 {
		return "", errors.New("invalid access code")
	}
	code := strings.TrimSpace(string(b))
	if len(code) != 64 {
		return "", errors.New("invalid access code")
	}
	if _, e = hex.DecodeString(code); e != nil {
		return "", e
	}
	return code, nil
}
func validMode(mode string) bool {
	switch mode {
	case "version", "init", "show-code", "verify", "backup", "restore", "serve", "service":
		return true
	}
	return false
}
