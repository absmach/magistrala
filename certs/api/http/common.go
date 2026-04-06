// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package http

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/absmach/magistrala/certs"
	"github.com/absmach/magistrala/pkg/errors"
)

const (
	// ContentType represents JSON content type.
	ContentType = "application/json"
	OCSPType    = "application/ocsp-response"
)

// Response contains HTTP response specific methods.
type Response interface {
	// Code returns HTTP response code.
	Code() int

	// Headers returns map of HTTP headers with their values.
	Headers() map[string]string

	// Empty indicates if HTTP response has content.
	Empty() bool
}

// EncodeError encodes an error response.
func EncodeError(_ context.Context, err error, w http.ResponseWriter) {
	var wrapper error
	if errors.Contains(err, ErrValidation) {
		wrapper, err = errors.Unwrap(err)
	}

	w.Header().Set("Content-Type", ContentType)
	switch {
	case errors.Contains(err, certs.ErrCertExpired):
		err = unwrap(err)
		w.WriteHeader(http.StatusForbidden)

	case errors.Contains(err, certs.ErrCertRevoked):
		err = unwrap(err)
		w.WriteHeader(http.StatusUnauthorized)
	case errors.Contains(err, certs.ErrMalformedEntity),
		errors.Contains(err, ErrMissingEntityID),
		errors.Contains(err, ErrEmptySerialNo),
		errors.Contains(err, ErrEmptyToken),
		errors.Contains(err, ErrInvalidQueryParams),
		errors.Contains(err, ErrValidation),
		errors.Contains(err, ErrInvalidRequest):
		err = unwrap(err)
		w.WriteHeader(http.StatusBadRequest)

	case errors.Contains(err, certs.ErrCreateEntity),
		errors.Contains(err, certs.ErrUpdateEntity),
		errors.Contains(err, certs.ErrViewEntity),
		errors.Contains(err, certs.ErrFailedCertCreation):
		err = unwrap(err)
		w.WriteHeader(http.StatusUnprocessableEntity)

	case errors.Contains(err, certs.ErrNotFound),
		errors.Contains(err, certs.ErrRootCANotFound),
		errors.Contains(err, certs.ErrIntermediateCANotFound):
		err = unwrap(err)
		w.WriteHeader(http.StatusNotFound)

	case errors.Contains(err, certs.ErrConflict):
		err = unwrap(err)
		w.WriteHeader(http.StatusConflict)

	default:
		w.WriteHeader(http.StatusInternalServerError)
	}

	if wrapper != nil {
		err = errors.Wrap(wrapper, err)
	}

	if errorVal, ok := err.(errors.Error); ok {
		if err := json.NewEncoder(w).Encode(errorVal); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
		}
	}
}

func unwrap(err error) error {
	wrapper, err := errors.Unwrap(err)
	if wrapper != nil {
		return wrapper
	}
	return err
}
