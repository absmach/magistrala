package blockwise

import "errors"

var (
	// ErrBlockNumberExceedLimit block number exceeded the limit 1,048,576
	ErrBlockNumberExceedLimit = errors.New("block number exceeded the limit 1,048,576")

	// ErrBlockInvalidSize block has invalid size
	ErrBlockInvalidSize = errors.New("block has invalid size")

	// ErrInvalidOptionBlock2 message has invalid value of Block2
	ErrInvalidOptionBlock2 = errors.New("message has invalid value of Block2")

	// ErrInvalidOptionBlock1 message has invalid value of Block1
	ErrInvalidOptionBlock1 = errors.New("message has invalid value of Block1")

	// ErrInvalidResponseCode response code has invalid value
	ErrInvalidResponseCode = errors.New("response code has invalid value")

	// ErrInvalidPayloadSize invalid payload size
	ErrInvalidPayloadSize = errors.New("invalid payload size")

	// ErrInvalidSZX invalid block-wise transfer szx
	ErrInvalidSZX = errors.New("invalid block-wise transfer szx")
)
