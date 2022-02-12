package net

import (
	"errors"
	"io"
)

var ErrListenerIsClosed = io.EOF
var ErrConnectionIsClosed = io.EOF
var ErrWriteInterrupted = errors.New("only part data was written to socket")
