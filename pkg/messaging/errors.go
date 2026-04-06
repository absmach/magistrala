// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package messaging

// MsgError is an error type for Magistrala SDK.
type Error interface {
	error
	Ack() AckType
}

type msgError struct {
	err error
	ack AckType
}

var _ Error = (*msgError)(nil)

func (ce *msgError) Error() string {
	return ce.err.Error()
}

func (ce *msgError) Ack() AckType {
	return ce.ack
}

// NewError returns an Error setting the acknowledgement type.
func NewError(err error, ack AckType) Error {
	if err == nil {
		return &msgError{ack: NoAck}
	}
	return &msgError{
		ack: ack,
		err: err,
	}
}
