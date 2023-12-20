// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package apiutil_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/absmach/magistrala/internal/apiutil"
	mglog "github.com/absmach/magistrala/logger"
	"github.com/absmach/magistrala/pkg/errors"
	"github.com/stretchr/testify/assert"
)

func TestReadStringQuery(t *testing.T) {
	cases := []struct {
		desc string
		url  string
		key  string
		ret  string
		err  error
	}{
		{
			desc: "valid string query",
			url:  "http://localhost:8080/?key=test",
			key:  "key",
			ret:  "test",
			err:  nil,
		},
		{
			desc: "empty string query",
			url:  "http://localhost:8080/",
			key:  "key",
			ret:  "",
			err:  nil,
		},
		{
			desc: "multiple string query",
			url:  "http://localhost:8080/?key=test&key=random",
			key:  "key",
			ret:  "",
			err:  apiutil.ErrInvalidQueryParams,
		},
	}

	for _, c := range cases {
		t.Run(c.desc, func(t *testing.T) {
			parsedURL, err := url.Parse(c.url)
			assert.NoError(t, err)

			r := &http.Request{URL: parsedURL}
			ret, err := apiutil.ReadStringQuery(r, c.key, "")
			assert.Equal(t, c.err, err)
			assert.Equal(t, c.ret, ret)
		})
	}
}

func TestReadMetadataQuery(t *testing.T) {
	cases := []struct {
		desc string
		url  string
		key  string
		ret  map[string]interface{}
		err  error
	}{
		{
			desc: "valid metadata query",
			url:  "http://localhost:8080/?key={\"test\":\"test\"}",
			key:  "key",
			ret:  map[string]interface{}{"test": "test"},
			err:  nil,
		},
		{
			desc: "empty metadata query",
			url:  "http://localhost:8080/",
			key:  "key",
			ret:  nil,
			err:  nil,
		},
		{
			desc: "multiple metadata query",
			url:  "http://localhost:8080/?key={\"test\":\"test\"}&key={\"random\":\"random\"}",
			key:  "key",
			ret:  nil,
			err:  apiutil.ErrInvalidQueryParams,
		},
		{
			desc: "invalid metadata query",
			url:  "http://localhost:8080/?key=abc",
			key:  "key",
			ret:  nil,
			err:  apiutil.ErrInvalidQueryParams,
		},
	}

	for _, c := range cases {
		t.Run(c.desc, func(t *testing.T) {
			parsedURL, err := url.Parse(c.url)
			assert.NoError(t, err)

			r := &http.Request{URL: parsedURL}
			ret, err := apiutil.ReadMetadataQuery(r, c.key, nil)
			assert.True(t, errors.Contains(err, c.err), fmt.Sprintf("expected: %v, got: %v", c.err, err))
			assert.Equal(t, c.ret, ret)
		})
	}
}

func TestReadBoolQuery(t *testing.T) {
	cases := []struct {
		desc string
		url  string
		key  string
		ret  bool
		err  error
	}{
		{
			desc: "valid bool query",
			url:  "http://localhost:8080/?key=true",
			key:  "key",
			ret:  true,
			err:  nil,
		},
		{
			desc: "valid bool query",
			url:  "http://localhost:8080/?key=false",
			key:  "key",
			ret:  false,
			err:  nil,
		},
		{
			desc: "invalid bool query",
			url:  "http://localhost:8080/?key=abc",
			key:  "key",
			ret:  false,
			err:  apiutil.ErrInvalidQueryParams,
		},
		{
			desc: "empty bool query",
			url:  "http://localhost:8080/",
			key:  "key",
			ret:  false,
			err:  nil,
		},
		{
			desc: "multiple bool query",
			url:  "http://localhost:8080/?key=true&key=false",
			key:  "key",
			ret:  false,
			err:  apiutil.ErrInvalidQueryParams,
		},
	}

	for _, c := range cases {
		t.Run(c.desc, func(t *testing.T) {
			parsedURL, err := url.Parse(c.url)
			assert.NoError(t, err)

			r := &http.Request{URL: parsedURL}
			ret, err := apiutil.ReadBoolQuery(r, c.key, false)
			assert.True(t, errors.Contains(err, c.err), fmt.Sprintf("expected: %v, got: %v", c.err, err))
			assert.Equal(t, c.ret, ret)
		})
	}
}

func TestReadNumQuery(t *testing.T) {
	cases := []struct {
		desc    string
		url     string
		key     string
		numType string
		ret     interface{}
		err     error
	}{
		{
			desc:    "valid int64 query",
			url:     "http://localhost:8080/?key=123",
			key:     "key",
			numType: "int64",
			ret:     int64(123),
			err:     nil,
		},
		{
			desc:    "valid float64 query",
			url:     "http://localhost:8080/?key=1.23",
			key:     "key",
			numType: "float64",
			ret:     float64(1.23),
			err:     nil,
		},
		{
			desc:    "valid uint64 query",
			url:     "http://localhost:8080/?key=123",
			key:     "key",
			numType: "uint64",
			ret:     uint64(123),
			err:     nil,
		},
		{
			desc:    "valid uint16 query",
			url:     "http://localhost:8080/?key=123",
			key:     "key",
			numType: "uint16",
			ret:     uint16(123),
			err:     nil,
		},
		{
			desc:    "invalid int64 query",
			url:     "http://localhost:8080/?key=abc",
			key:     "key",
			numType: "int64",
			ret:     int64(0),
			err:     apiutil.ErrInvalidQueryParams,
		},
		{
			desc:    "invalid float64 query",
			url:     "http://localhost:8080/?key=abc",
			key:     "key",
			numType: "float64",
			ret:     float64(0),
			err:     apiutil.ErrInvalidQueryParams,
		},
		{
			desc:    "invalid uint64 query",
			url:     "http://localhost:8080/?key=abc",
			key:     "key",
			numType: "uint64",
			ret:     uint64(0),
			err:     apiutil.ErrInvalidQueryParams,
		},
		{
			desc:    "invalid uint16 query",
			url:     "http://localhost:8080/?key=abc",
			key:     "key",
			numType: "uint16",
			ret:     uint16(0),
			err:     apiutil.ErrInvalidQueryParams,
		},
		{
			desc:    "empty int64 query",
			url:     "http://localhost:8080/",
			key:     "key",
			numType: "int64",
			ret:     int64(0),
			err:     nil,
		},
		{
			desc:    "empty float64 query",
			url:     "http://localhost:8080/",
			key:     "key",
			numType: "float64",
			ret:     float64(0),
			err:     nil,
		},
		{
			desc:    "empty uint16 query",
			url:     "http://localhost:8080/",
			key:     "key",
			numType: "uint16",
			ret:     uint16(0),
			err:     nil,
		},
		{
			desc:    "empty uint64 query",
			url:     "http://localhost:8080/",
			key:     "key",
			numType: "uint64",
			ret:     uint64(0),
			err:     nil,
		},
		{
			desc:    "multiple int64 query",
			url:     "http://localhost:8080/?key=123&key=456",
			key:     "key",
			numType: "int64",
			ret:     int64(0),
			err:     apiutil.ErrInvalidQueryParams,
		},
		{
			desc:    "multiple float64 query",
			url:     "http://localhost:8080/?key=1.23&key=4.56",
			key:     "key",
			numType: "float64",
			ret:     float64(0),
			err:     apiutil.ErrInvalidQueryParams,
		},
		{
			desc:    "multiple uint16 query",
			url:     "http://localhost:8080/?key=123&key=456",
			key:     "key",
			numType: "uint16",
			ret:     uint16(0),
			err:     apiutil.ErrInvalidQueryParams,
		},
		{
			desc:    "multiple uint64 query",
			url:     "http://localhost:8080/?key=123&key=456",
			key:     "key",
			numType: "uint64",
			ret:     uint64(0),
			err:     apiutil.ErrInvalidQueryParams,
		},
	}

	for _, c := range cases {
		t.Run(c.desc, func(t *testing.T) {
			parsedURL, err := url.Parse(c.url)
			assert.NoError(t, err)

			r := &http.Request{URL: parsedURL}
			var ret interface{}
			switch c.numType {
			case "int64":
				ret, err = apiutil.ReadNumQuery[int64](r, c.key, 0)
			case "float64":
				ret, err = apiutil.ReadNumQuery[float64](r, c.key, 0)
			case "uint64":
				ret, err = apiutil.ReadNumQuery[uint64](r, c.key, 0)
			case "uint16":
				ret, err = apiutil.ReadNumQuery[uint16](r, c.key, 0)
			}
			assert.True(t, errors.Contains(err, c.err), fmt.Sprintf("expected: %v, got: %v", c.err, err))
			assert.Equal(t, c.ret, ret)
		})
	}
}

func TestLoggingErrorEncoder(t *testing.T) {
	logger := mglog.NewMock()

	cases := []struct {
		desc string
		err  error
	}{
		{
			desc: "error contains ErrValidation",
			err:  errors.Wrap(apiutil.ErrValidation, errors.ErrAuthentication),
		},
		{
			desc: "error does not contain ErrValidation",
			err:  errors.ErrAuthentication,
		},
	}

	for _, c := range cases {
		t.Run(c.desc, func(t *testing.T) {
			encCalled := false
			encFunc := func(ctx context.Context, err error, w http.ResponseWriter) {
				encCalled = true
			}

			errorEncoder := apiutil.LoggingErrorEncoder(logger, encFunc)
			errorEncoder(context.Background(), c.err, httptest.NewRecorder())

			assert.True(t, encCalled)
		})
	}
}
