// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package http_test

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/absmach/magistrala"
	api "github.com/absmach/magistrala/api/http"
	apiutil "github.com/absmach/magistrala/api/http/util"
	"github.com/absmach/magistrala/internal/testsutil"
	"github.com/absmach/magistrala/pkg/errors"
	svcerr "github.com/absmach/magistrala/pkg/errors/service"
	"github.com/stretchr/testify/assert"
)

var _ magistrala.Response = (*response)(nil)

var validUUID = testsutil.GenerateUUID(&testing.T{})

type responseWriter struct {
	body       []byte
	statusCode int
	header     http.Header
}

func newResponseWriter() *responseWriter {
	return &responseWriter{
		header: http.Header{},
	}
}

func (w *responseWriter) Header() http.Header {
	return w.header
}

func (w *responseWriter) Write(b []byte) (int, error) {
	w.body = b
	return 0, nil
}

func (w *responseWriter) WriteHeader(statusCode int) {
	w.statusCode = statusCode
}

func (w *responseWriter) StatusCode() int {
	return w.statusCode
}

func (w *responseWriter) Body() []byte {
	return w.body
}

type response struct {
	code    int
	headers map[string]string
	empty   bool

	ID        string    `json:"id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
}

func (res response) Code() int {
	return res.code
}

func (res response) Headers() map[string]string {
	return res.headers
}

func (res response) Empty() bool {
	return res.empty
}

type body struct {
	Error   string `json:"error,omitempty"`
	Message string `json:"message"`
}

func TestValidateUUID(t *testing.T) {
	cases := []struct {
		desc string
		uuid string
		err  error
	}{
		{
			desc: "valid uuid",
			uuid: validUUID,
			err:  nil,
		},
		{
			desc: "invalid uuid",
			uuid: "invalid",
			err:  apiutil.ErrInvalidIDFormat,
		},
	}

	for _, c := range cases {
		t.Run(c.desc, func(t *testing.T) {
			err := api.ValidateUUID(c.uuid)
			assert.Equal(t, c.err, err)
		})
	}
}

func TestEncodeResponse(t *testing.T) {
	now := time.Now()
	validBody := []byte(`{"id":"` + validUUID + `","name":"test","created_at":"` + now.Format(time.RFC3339Nano) + `"}` + "\n" + ``)

	cases := []struct {
		desc   string
		resp   any
		header http.Header
		code   int
		body   []byte
		err    error
	}{
		{
			desc: "valid response",
			resp: response{
				code: http.StatusOK,
				headers: map[string]string{
					"Location": "/groups/" + validUUID,
				},
				ID:        validUUID,
				Name:      "test",
				CreatedAt: now,
			},
			header: http.Header{
				"Content-Type": []string{"application/json"},
				"Location":     []string{"/groups/" + validUUID},
			},
			code: http.StatusOK,
			body: validBody,
			err:  nil,
		},
		{
			desc: "valid response with no headers",
			resp: response{
				code:      http.StatusOK,
				ID:        validUUID,
				Name:      "test",
				CreatedAt: now,
			},
			header: http.Header{
				"Content-Type": []string{"application/json"},
			},
			code: http.StatusOK,
			body: validBody,
			err:  nil,
		},
		{
			desc: "valid response with many headers",
			resp: response{
				code: http.StatusOK,
				headers: map[string]string{
					"X-Test":  "test",
					"X-Test2": "test2",
				},
				ID:        validUUID,
				Name:      "test",
				CreatedAt: now,
			},
			header: http.Header{
				"Content-Type": []string{"application/json"},
				"X-Test":       []string{"test"},
				"X-Test2":      []string{"test2"},
			},
			code: http.StatusOK,
			body: validBody,
			err:  nil,
		},
		{
			desc: "valid response with empty body",
			resp: response{
				code:  http.StatusOK,
				empty: true,
				ID:    validUUID,
			},
			header: http.Header{
				"Content-Type": []string{"application/json"},
			},
			code: http.StatusOK,
			body: []byte(``),
			err:  nil,
		},
		{
			desc: "invalid response",
			resp: struct {
				ID string `json:"id"`
			}{
				ID: validUUID,
			},
			header: http.Header{},
			code:   0,
			body:   []byte(`{"id":"` + validUUID + `"}` + "\n" + ``),
			err:    nil,
		},
	}

	for _, c := range cases {
		t.Run(c.desc, func(t *testing.T) {
			responseWriter := newResponseWriter()
			err := api.EncodeResponse(context.Background(), responseWriter, c.resp)
			assert.Equal(t, c.err, err)
			assert.Equal(t, c.header, responseWriter.Header())
			assert.Equal(t, c.code, responseWriter.StatusCode())
			assert.Equal(t, string(c.body), string(responseWriter.Body()))
		})
	}
}

func TestEncodeError(t *testing.T) {
	cases := []struct {
		desc       string
		err        error
		code       int
		hasBody    bool
		checkError bool
	}{
		{
			desc:    "RequestError - Missing Secret",
			err:     apiutil.ErrMissingSecret,
			code:    http.StatusBadRequest,
			hasBody: true,
		},
		{
			desc:    "RequestError - Missing ID",
			err:     apiutil.ErrMissingID,
			code:    http.StatusBadRequest,
			hasBody: true,
		},
		{
			desc:    "RequestError - Empty List",
			err:     apiutil.ErrEmptyList,
			code:    http.StatusBadRequest,
			hasBody: true,
		},
		{
			desc:    "RequestError - Conflict",
			err:     svcerr.ErrConflict,
			code:    http.StatusBadRequest,
			hasBody: true,
		},
		{
			desc:    "NotFoundError - Not Found",
			err:     svcerr.ErrNotFound,
			code:    http.StatusNotFound,
			hasBody: true,
		},
		{
			desc:    "AuthNError - Authentication Failed",
			err:     svcerr.ErrAuthentication,
			code:    http.StatusUnauthorized,
			hasBody: true,
		},
		{
			desc:    "AuthZError - Authorization Failed",
			err:     svcerr.ErrAuthorization,
			code:    http.StatusForbidden,
			hasBody: true,
		},
		{
			desc:    "AuthZError - Domain Authorization Failed",
			err:     svcerr.ErrDomainAuthorization,
			code:    http.StatusForbidden,
			hasBody: true,
		},
		{
			desc:    "MediaTypeError - Unsupported Content Type",
			err:     apiutil.ErrUnsupportedContentType,
			code:    http.StatusUnsupportedMediaType,
			hasBody: true,
		},
		{
			desc:    "ServiceError - Create Entity Failed",
			err:     svcerr.ErrCreateEntity,
			code:    http.StatusUnprocessableEntity,
			hasBody: true,
		},
		{
			desc:    "ServiceError - Update Entity Failed",
			err:     svcerr.ErrUpdateEntity,
			code:    http.StatusUnprocessableEntity,
			hasBody: true,
		},
		{
			desc:    "ServiceError - Remove Entity Failed",
			err:     svcerr.ErrRemoveEntity,
			code:    http.StatusUnprocessableEntity,
			hasBody: true,
		},
		{
			desc:    "InternalError",
			err:     errors.NewInternalError(),
			code:    http.StatusInternalServerError,
			hasBody: false,
		},
	}

	for _, c := range cases {
		t.Run(c.desc, func(t *testing.T) {
			responseWriter := newResponseWriter()
			api.EncodeError(context.Background(), c.err, responseWriter)
			assert.Equal(t, c.code, responseWriter.StatusCode())
			if !c.hasBody {
				return
			}
			message := body{}
			jerr := json.Unmarshal(responseWriter.Body(), &message)
			assert.NoError(t, jerr)
			assert.NotEmpty(t, message.Message)
		})
	}
}
