//go:build aix || darwin || dragonfly || freebsd || js || linux || netbsd || openbsd || solaris
// +build aix darwin dragonfly freebsd js linux netbsd openbsd solaris

package net

import (
	"errors"
	"syscall"
)

// Check if error returned by operation on a socket failed because
// the other side has closed the connection.
func IsConnectionBrokenError(err error) bool {
	return errors.Is(err, syscall.EPIPE) ||
		errors.Is(err, syscall.ECONNRESET)
}
