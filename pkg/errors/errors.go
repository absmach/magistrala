// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package errors

// Error specifies an API that must be fullfiled by error type
type Error interface {

	// Error implements the error interface.
	Error() string

	// Msg returns error message
	Msg() string

	// Err returns wrapped error
	Err() Error
}

var _ Error = (*customError)(nil)

// customError struct represents a Mainflux error
type customError struct {
	msg string
	err Error
}

func (ce *customError) Error() string {
	if ce == nil {
		return ""
	}
	if ce.err == nil {
		return ce.msg
	}
	return ce.msg + " : " + ce.err.Error()
}

func (ce *customError) Msg() string {
	return ce.msg
}

func (ce *customError) Err() Error {
	return ce.err
}

// Contains inspects if e2 error is contained in any layer of e1 error
func Contains(e1 error, e2 error) bool {
	if e1 == nil || e2 == nil {
		return e2 == e1
	}
	ce, ok := e1.(Error)
	if ok {
		if ce.Msg() == e2.Error() {
			return true
		}
		return Contains(ce.Err(), e2)
	}
	return e1.Error() == e2.Error()
}

// Wrap returns an Error that wrap err with wrapper
func Wrap(wrapper error, err error) error {
	if wrapper == nil || err == nil {
		return wrapper
	}
	if w, ok := wrapper.(Error); ok {
		return &customError{
			msg: w.Msg(),
			err: cast(err),
		}
	}
	return &customError{
		msg: wrapper.Error(),
		err: cast(err),
	}
}

func cast(err error) Error {
	if err == nil {
		return nil
	}
	if e, ok := err.(Error); ok {
		return e
	}
	return &customError{
		msg: err.Error(),
		err: nil,
	}
}

// New returns an Error that formats as the given text.
func New(text string) Error {
	return &customError{
		msg: text,
		err: nil,
	}
}
