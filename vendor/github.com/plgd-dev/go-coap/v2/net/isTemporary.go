package net

import (
	"net"
	"strings"
	"time"
)

// https://github.com/golang/go/blob/958e212db799e609b2a8df51cdd85c9341e7a404/src/internal/poll/fd.go#L43
const ioTimeout = "i/o timeout"

func isTemporary(err error, deadline time.Time) bool {
	netErr, ok := err.(net.Error)
	if ok {
		if netErr.Timeout() {
			// when connection is closed during TLS handshake, it returns i/o timeout
			// so we need to validate if timeout real occurs by set deadline otherwise infinite loop occurs.
			return deadline.Before(time.Now())
		}
		if netErr.Temporary() {
			return true
		}
	}

	if strings.Contains(err.Error(), ioTimeout) {
		// when connection is closed during TLS handshake, it returns i/o timeout
		// so we need to validate if timeout real occurs by set deadline otherwise infinite loop occurs.
		return deadline.Before(time.Now())
	}
	return false
}
