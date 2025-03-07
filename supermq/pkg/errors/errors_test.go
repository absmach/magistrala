// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package errors_test

import (
	nerrors "errors"
	"fmt"
	"strconv"
	"testing"

	"github.com/absmach/supermq/pkg/errors"
	"github.com/stretchr/testify/assert"
)

const level = 10

var (
	err0 = errors.New("0")
	err1 = errors.New("1")
	err2 = errors.New("2")
	nat  = nerrors.New("native error")
)

func TestError(t *testing.T) {
	cases := []struct {
		desc     string
		err      error
		msg      string
		bytes    []byte
		bytesErr error
	}{
		{
			desc:     "level 0 wrapped error",
			err:      err0,
			msg:      "0",
			bytes:    []byte(`{"error":"","message":"0"}`),
			bytesErr: nil,
		},
		{
			desc:     "level 1 wrapped error",
			err:      wrap(1),
			msg:      message(1),
			bytes:    []byte(`{"error":"0","message":"1"}`),
			bytesErr: nil,
		},
		{
			desc:     "level 2 wrapped error",
			err:      wrap(2),
			msg:      message(2),
			bytes:    []byte(`{"error":"1","message":"2"}`),
			bytesErr: nil,
		},
		{
			desc:     fmt.Sprintf("level %d wrapped error", level),
			err:      wrap(level),
			msg:      message(level),
			bytes:    []byte(`{"error":"9","message":"` + strconv.Itoa(level) + `"}`),
			bytesErr: nil,
		},
		{
			desc:     "nil error",
			err:      errors.New(""),
			msg:      "",
			bytes:    []byte(`{"error":"","message":""}`),
			bytesErr: nil,
		},
	}

	for _, c := range cases {
		t.Run(c.desc, func(t *testing.T) {
			errMsg := c.err.Error()
			assert.Equal(t, c.msg, errMsg)
			err := c.err.(errors.Error)
			data, derr := err.MarshalJSON()
			assert.Equal(t, c.bytesErr, derr)
			assert.Equal(t, c.bytes, data)
		})
	}
}

func TestContains(t *testing.T) {
	cases := []struct {
		desc      string
		container error
		contained error
		contains  bool
	}{
		{
			desc:      "nil contains nil",
			container: nil,
			contained: nil,
			contains:  true,
		},
		{
			desc:      "nil contains non-nil",
			container: nil,
			contained: err0,
			contains:  false,
		},
		{
			desc:      "non-nil contains nil",
			container: err0,
			contained: nil,
			contains:  false,
		},
		{
			desc:      "non-nil contains non-nil",
			container: err0,
			contained: err1,
			contains:  false,
		},
		{
			desc:      "res of errors.Wrap(err1, err0) contains err0",
			container: errors.Wrap(err1, err0),
			contained: err0,
			contains:  true,
		},
		{
			desc:      "res of errors.Wrap(err1, err0) contains err1",
			container: errors.Wrap(err1, err0),
			contained: err1,
			contains:  true,
		},
		{
			desc:      "res of errors.Wrap(err2, errors.Wrap(err1, err0)) contains err1",
			container: errors.Wrap(err2, errors.Wrap(err1, err0)),
			contained: err1,
			contains:  true,
		},
		{
			desc:      fmt.Sprintf("level %d wrapped error contains", level),
			container: wrap(level),
			contained: errors.New(strconv.Itoa(level / 2)),
			contains:  true,
		},
		{
			desc:      "superset wrapper error contains subset wrapper error",
			container: wrap(level),
			contained: wrap(level / 2),
			contains:  false,
		},
		{
			desc:      "native error contains error",
			container: nat,
			contained: err0,
			contains:  false,
		},
		{
			desc:      "res of errors.Wrap(err1, errors.New('')) contains err1",
			container: errors.Wrap(err1, nat),
			contained: err1,
			contains:  true,
		},
		{
			desc:      "error contains native error",
			container: err0,
			contained: nat,
			contains:  false,
		},
		{
			desc:      "res of errors.Wrap(errors.New(''), err0) contains err0",
			container: errors.Wrap(nat, err0),
			contained: err0,
			contains:  true,
		},
	}
	for _, c := range cases {
		t.Run(c.desc, func(t *testing.T) {
			contains := errors.Contains(c.container, c.contained)
			assert.Equal(t, c.contains, contains)
		})
	}
}

func TestWrap(t *testing.T) {
	cases := []struct {
		desc      string
		wrapper   error
		wrapped   error
		contained error
		contains  bool
	}{
		{
			desc:      "err 1 wraps err 2",
			wrapper:   err1,
			wrapped:   err0,
			contained: err0,
			contains:  true,
		},
		{
			desc:      "err2 wraps err1 wraps err0 and contains err0",
			wrapper:   err2,
			wrapped:   errors.Wrap(err1, err0),
			contained: err0,
			contains:  true,
		},
		{
			desc:      "err2 wraps err1 wraps err0 and contains err1",
			wrapper:   err2,
			wrapped:   errors.Wrap(err1, err0),
			contained: err1,
			contains:  true,
		},
		{
			desc:      "nil wraps nil",
			wrapper:   nil,
			wrapped:   nil,
			contained: nil,
			contains:  true,
		},
		{
			desc:      "err0 wraps nil",
			wrapper:   err0,
			wrapped:   nil,
			contained: nil,
			contains:  false,
		},
		{
			desc:      "nil wraps err0",
			wrapper:   nil,
			wrapped:   err0,
			contained: err0,
			contains:  false,
		},
		{
			desc:      "err0 wraps native error",
			wrapper:   err0,
			wrapped:   nat,
			contained: nat,
			contains:  true,
		},
		{
			desc:      "nil wraps native error",
			wrapper:   nil,
			wrapped:   nat,
			contained: nat,
			contains:  false,
		},
		{
			desc:      "native error wraps err0",
			wrapper:   nat,
			wrapped:   err0,
			contained: err0,
			contains:  true,
		},
		{
			desc:      "native error wraps nil",
			wrapper:   nat,
			wrapped:   nil,
			contained: nil,
			contains:  false,
		},
		{
			desc:      "err0 wraps err1 wraps native error",
			wrapper:   err0,
			wrapped:   errors.Wrap(err1, nat),
			contained: nat,
			contains:  true,
		},
		{
			desc:      "native error wraps err1 wraps err0",
			wrapper:   nat,
			wrapped:   errors.Wrap(err1, err0),
			contained: err0,
			contains:  true,
		},
	}

	for _, c := range cases {
		t.Run(c.desc, func(t *testing.T) {
			err := errors.Wrap(c.wrapper, c.wrapped)
			contains := errors.Contains(err, c.contained)
			assert.Equal(t, c.contains, contains)
		})
	}
}

func TestUnwrap(t *testing.T) {
	cases := []struct {
		desc    string
		err     error
		wrapper error
		wrapped error
	}{
		{
			desc:    "err 1 wraped err 2",
			err:     errors.Wrap(err1, err2),
			wrapper: err1,
			wrapped: err2,
		},
		{
			desc:    "err2 wraps err1 wraps err0",
			err:     errors.Wrap(err2, errors.Wrap(err1, err0)),
			wrapper: err2,
			wrapped: errors.Wrap(err1, err0),
		},
		{
			desc:    "nil wraps nil",
			err:     errors.Wrap(nil, nil),
			wrapper: nil,
			wrapped: nil,
		},
		{
			desc:    "err0 wraps nil",
			err:     errors.Wrap(err0, nil),
			wrapper: nil,
			wrapped: err0,
		},
		{
			desc:    "nil wraps err0",
			err:     errors.Wrap(nil, err0),
			wrapper: nil,
			wrapped: nil,
		},
		{
			desc:    "nil wraps native error",
			err:     errors.Wrap(nil, nat),
			wrapper: nil,
			wrapped: nil,
		},
		{
			desc:    "native error wraps nil",
			err:     errors.Wrap(nat, nil),
			wrapper: nil,
			wrapped: nat,
		},
	}

	for _, c := range cases {
		t.Run(c.desc, func(t *testing.T) {
			wrapper, wrapped := errors.Unwrap(c.err)
			assert.Equal(t, c.wrapper, wrapper)
			assert.Equal(t, c.wrapped, wrapped)
		})
	}
}

func wrap(level int) error {
	if level == 0 {
		return errors.New(strconv.Itoa(level))
	}
	return errors.Wrap(errors.New(strconv.Itoa(level)), wrap(level-1))
}

// message generates error message of wrap() generated wrapper error.
func message(level int) string {
	if level == 0 {
		return "0"
	}
	return strconv.Itoa(level) + " : " + message(level-1)
}
