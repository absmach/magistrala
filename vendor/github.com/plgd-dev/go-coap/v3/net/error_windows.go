//go:build windows
// +build windows

package net

import (
	"errors"
	"syscall"
)

func IsConnectionBrokenError(err error) bool {
	return errors.Is(err, syscall.WSAECONNRESET) ||
		errors.Is(err, syscall.WSAECONNABORTED)
}
