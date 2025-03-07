// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package errors_test

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"testing"

	"github.com/absmach/supermq/pkg/errors"
	"github.com/stretchr/testify/assert"
)

var body = []byte(`{"error":"error","message":"message"}`)

func TestNewSDKError(t *testing.T) {
	cases := []struct {
		desc string
		err  error
	}{
		{
			desc: "nil error",
			err:  nil,
		},
		{
			desc: "non nil error",
			err:  err0,
		},
		{
			desc: "non nil error with wrapped error",
			err:  errors.Wrap(err0, err1),
		},
		{
			desc: "native error",
			err:  nat,
		},
	}

	for _, c := range cases {
		t.Run(c.desc, func(t *testing.T) {
			sdk := errors.NewSDKError(c.err)
			if c.err != nil {
				assert.Equal(t, sdk.StatusCode(), 0)
				assert.Equal(t, sdk.Error(), fmt.Sprintf("Status: %s: %s", http.StatusText(0), c.err.Error()))
			}
		})
	}
}

func TestNewSDKErrorWithStatus(t *testing.T) {
	cases := []struct {
		desc string
		err  error
		sc   int
	}{
		{
			desc: "nil error with 0 status code",
			err:  nil,
			sc:   0,
		},
		{
			desc: "nil error with 404 status code",
			err:  nil,
			sc:   404,
		},
		{
			desc: "non nil error with 0 status code",
			err:  err0,
			sc:   0,
		},
		{
			desc: "non nil error with 404 status code",
			err:  err0,
			sc:   404,
		},
		{
			desc: "non nil error with wrapped error and 0 status code",
			err:  errors.Wrap(err0, err1),
			sc:   0,
		},
		{
			desc: "non nil error with wrapped error and 404 status code",
			err:  errors.Wrap(err0, err1),
			sc:   404,
		},
		{
			desc: "native error with 0 status code",
			err:  nat,
			sc:   0,
		},
		{
			desc: "native error with 404 status code",
			err:  nat,
			sc:   404,
		},
	}

	for _, c := range cases {
		t.Run(c.desc, func(t *testing.T) {
			sdk := errors.NewSDKErrorWithStatus(c.err, c.sc)
			if c.err != nil {
				assert.Equal(t, sdk.StatusCode(), c.sc)
				assert.Equal(t, sdk.Error(), fmt.Sprintf("Status: %s: %s", http.StatusText(c.sc), c.err.Error()))
			}
		})
	}
}

func TestCheckError(t *testing.T) {
	cases := []struct {
		desc  string
		resp  *http.Response
		codes []int
		err   errors.SDKError
	}{
		{
			desc:  "nil response",
			resp:  nil,
			codes: []int{http.StatusOK},
			err:   nil,
		},
		{
			desc:  "nil response with 404 status code",
			resp:  nil,
			codes: []int{http.StatusNotFound},
			err:   nil,
		},
		{
			desc: "valid response with 200 status code",
			resp: &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(bytes.NewReader(body)),
			},
			codes: []int{http.StatusOK},
			err:   nil,
		},
		{
			desc: "valid response with 404 status code",
			resp: &http.Response{
				StatusCode: http.StatusNotFound,
				Body:       io.NopCloser(bytes.NewReader(body)),
			},
			codes: []int{http.StatusNotFound},
			err:   nil,
		},
		{
			desc: "invalid response with 200 status code",
			resp: &http.Response{
				StatusCode: http.StatusNotFound,
				Body:       io.NopCloser(bytes.NewReader(body)),
			},
			codes: []int{http.StatusOK},
			err:   errors.NewSDKErrorWithStatus(errors.Wrap(errors.New("message"), errors.New("error")), http.StatusNotFound),
		},
		{
			desc: "invalid response with 404 status code",
			resp: &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(bytes.NewReader(body)),
			},
			codes: []int{http.StatusNotFound},
			err:   errors.NewSDKErrorWithStatus(errors.Wrap(errors.New("message"), errors.New("error")), http.StatusOK),
		},
		{
			desc: "valid response with 200 status code and 404 status code",
			resp: &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(bytes.NewReader(body)),
			},
			codes: []int{http.StatusOK, http.StatusNotFound},
			err:   nil,
		},
		{
			desc: "error in JSON marshalling",
			resp: &http.Response{
				StatusCode: http.StatusNotFound,
				Body:       io.NopCloser(bytes.NewReader([]byte(`"error":`))),
			},
			codes: []int{http.StatusOK},
			err:   errors.NewSDKErrorWithStatus(errors.New("invalid character ':' after top-level value"), http.StatusNotFound),
		},
		{
			desc: "empty error message",
			resp: &http.Response{
				StatusCode: http.StatusNotFound,
				Body:       io.NopCloser(bytes.NewReader([]byte(`{"error":"","message":""}`))),
			},
			codes: []int{http.StatusOK},
			err:   errors.NewSDKErrorWithStatus(errors.New(""), http.StatusNotFound),
		},
	}

	for _, c := range cases {
		t.Run(c.desc, func(t *testing.T) {
			sdk := errors.CheckError(c.resp, c.codes...)
			assert.Equal(t, sdk, c.err)
			if c.err != nil {
				assert.Equal(t, sdk, c.err)
				assert.Equal(t, sdk.StatusCode(), c.resp.StatusCode)
			}
		})
	}
}
