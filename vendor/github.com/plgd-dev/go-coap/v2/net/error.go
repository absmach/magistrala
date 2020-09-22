package net

import "errors"

var ErrListenerIsClosed = errors.New("listen socket was closed")
