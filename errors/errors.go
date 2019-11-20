// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package errors

import "fmt"

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
	if ce != nil {
		if ce.err != nil {
			return fmt.Sprintf("%s: %s", ce.msg, ce.err.Error())
		}

		return ce.msg
	}
	return ""
}

func (ce *customError) Msg() string {
	return ce.msg
}

func (ce *customError) Err() Error {
	return ce.err
}

// Contains inspects if Error's message is same as error
// in argument. If not it continues further unwrapping
// layers of Error until it founds it or unwrap all layers
func Contains(ce Error, e error) bool {
	if ce == nil || e == nil {
		return ce == nil
	}
	if ce.Msg() == e.Error() {
		return true
	}
	if ce.Err() == nil {
		return false
	}

	return Contains(ce.Err(), e)
}

// Wrap returns an Error that wrap err with wrapper
func Wrap(wrapper Error, err error) Error {
	if wrapper == nil || err == nil {
		return nil
	}
	return &customError{
		msg: wrapper.Msg(),
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
