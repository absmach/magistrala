package net

import (
	"net"
	"strings"
)

// https://github.com/golang/go/blob/958e212db799e609b2a8df51cdd85c9341e7a404/src/internal/poll/fd.go#L43
const ioTimeout = "i/o timeout"

func isTemporary(err error) bool {
	if netErr, ok := err.(net.Error); ok && (netErr.Temporary() || netErr.Timeout()) {
		return true
	}

	if strings.Contains(err.Error(), ioTimeout) {
		return true
	}
	return false
}
