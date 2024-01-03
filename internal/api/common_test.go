// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package api_test

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/absmach/magistrala"
	"github.com/absmach/magistrala/internal/api"
	"github.com/absmach/magistrala/internal/apiutil"
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
		resp   interface{}
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
		desc string
		errs []error
		code int
	}{
		{
			desc: "BadRequest",
			errs: []error{
				apiutil.ErrInvalidSecret,
				svcerr.ErrMalformedEntity,
				errors.ErrMalformedEntity,
				apiutil.ErrMissingID,
				apiutil.ErrEmptyList,
				apiutil.ErrMissingMemberType,
				apiutil.ErrMissingMemberKind,
				apiutil.ErrLimitSize,
				apiutil.ErrNameSize,
			},
			code: http.StatusBadRequest,
		},
		{
			desc: "BadRequest with validation error",
			errs: []error{
				errors.Wrap(apiutil.ErrValidation, apiutil.ErrInvalidSecret),
				errors.Wrap(apiutil.ErrValidation, svcerr.ErrMalformedEntity),
				errors.Wrap(apiutil.ErrValidation, errors.ErrMalformedEntity),
				errors.Wrap(apiutil.ErrValidation, apiutil.ErrMissingID),
				errors.Wrap(apiutil.ErrValidation, apiutil.ErrEmptyList),
				errors.Wrap(apiutil.ErrValidation, apiutil.ErrMissingMemberType),
				errors.Wrap(apiutil.ErrValidation, apiutil.ErrMissingMemberKind),
				errors.Wrap(apiutil.ErrValidation, apiutil.ErrLimitSize),
				errors.Wrap(apiutil.ErrValidation, apiutil.ErrNameSize),
			},
			code: http.StatusBadRequest,
		},
		{
			desc: "Unauthorized",
			errs: []error{
				svcerr.ErrAuthentication,
				errors.ErrAuthentication,
				apiutil.ErrBearerToken,
			},
			code: http.StatusUnauthorized,
		},

		{
			desc: "NotFound",
			errs: []error{
				svcerr.ErrNotFound,
			},
			code: http.StatusNotFound,
		},
		{
			desc: "Conflict",
			errs: []error{
				svcerr.ErrConflict,
				errors.ErrConflict,
			},
			code: http.StatusConflict,
		},
		{
			desc: "Forbidden",
			errs: []error{
				svcerr.ErrAuthorization,
				errors.ErrAuthorization,
				errors.ErrDomainAuthorization,
			},
			code: http.StatusForbidden,
		},
		{
			desc: "UnsupportedMediaType",
			errs: []error{
				apiutil.ErrUnsupportedContentType,
			},
			code: http.StatusUnsupportedMediaType,
		},
		{
			desc: "InternalServerError",
			errs: []error{
				svcerr.ErrCreateEntity,
				svcerr.ErrUpdateEntity,
				svcerr.ErrViewEntity,
				svcerr.ErrRemoveEntity,
			},
			code: http.StatusInternalServerError,
		},
		{
			desc: "InternalServerError",
			errs: []error{
				errors.New("test"),
			},
			code: http.StatusInternalServerError,
		},
	}

	for _, c := range cases {
		t.Run(c.desc, func(t *testing.T) {
			responseWriter := newResponseWriter()
			for _, err := range c.errs {
				api.EncodeError(context.Background(), err, responseWriter)
				assert.Equal(t, c.code, responseWriter.StatusCode())

				message := body{}
				jerr := json.Unmarshal(responseWriter.Body(), &message)
				assert.NoError(t, jerr)

				var wrapper error
				switch errors.Contains(err, apiutil.ErrValidation) {
				case true:
					wrapper, err = errors.Unwrap(err)
					assert.Equal(t, err.Error(), message.Error)
					assert.Equal(t, wrapper.Error(), message.Message)
				case false:
					assert.Equal(t, err.Error(), message.Message)
				}
			}
		})
	}
}
