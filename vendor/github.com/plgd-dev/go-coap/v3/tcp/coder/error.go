package coder

import "errors"

var (
	ErrMessageTruncated      = errors.New("message is truncated")
	ErrMessageInvalidVersion = errors.New("message has invalid version")
)
