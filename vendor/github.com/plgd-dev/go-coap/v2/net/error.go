package net

import (
	"context"
	"errors"
	"io"
	"net"
)

var ErrListenerIsClosed = io.EOF
var ErrConnectionIsClosed = io.EOF
var ErrWriteInterrupted = errors.New("only part data was written to socket")

func IsCancelOrCloseError(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, io.EOF) || errors.Is(err, net.ErrClosed) {
		// this error was produced by cancellation context or closing connection.
		return true
	}
	return false
}
