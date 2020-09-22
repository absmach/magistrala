package keepalive

import (
	"context"
	"fmt"
)

// ErrKeepAliveDeadlineExceeded occurs during waiting for pong response
var ErrKeepAliveDeadlineExceeded = fmt.Errorf("keepalive: %w", context.DeadlineExceeded)
