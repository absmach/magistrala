// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/absmach/magistrala/bootstrap"
	api "github.com/absmach/supermq/api/http"
	"github.com/absmach/supermq/pkg/errors"
)

// EncodeError encodes an error response.
func EncodeError(ctx context.Context, err error, w http.ResponseWriter) {
	w.Header().Set("Content-Type", api.ContentType)

	status, nerr := toStatus(err)
	if nerr != nil {
		w.WriteHeader(status)
		if errorVal, ok := err.(errors.Error); ok {
			if err := json.NewEncoder(w).Encode(errorVal); err != nil {
				w.WriteHeader(http.StatusInternalServerError)
			}
		}
		return
	}

	api.EncodeError(ctx, err, w)
}

func toStatus(err error) (int, error) {
	switch {
	case errors.Contains(err, bootstrap.ErrExternalKey),
		errors.Contains(err, bootstrap.ErrExternalKeySecure):
		return http.StatusForbidden, err

	case errors.Contains(err, bootstrap.ErrBootstrapState),
		errors.Contains(err, bootstrap.ErrAddBootstrap):
		return http.StatusBadRequest, err

	case errors.Contains(err, bootstrap.ErrBootstrap):
		return http.StatusNotFound, err

	default:
		return 0, nil
	}
}
