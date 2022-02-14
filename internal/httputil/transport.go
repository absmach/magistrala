// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package httputil

import (
	"context"
	"errors"
	"net/http"

	kithttp "github.com/go-kit/kit/transport/http"
	"github.com/mainflux/mainflux/logger"
)

var (
	// ErrMissingToken indicates missing user token.
	ErrMissingToken = errors.New("missing user token")

	// ErrMissingID indicates missing entity ID.
	ErrMissingID = errors.New("missing entity id")
)

// Middleware is an ErrorEncoder middleware
type Middleware func(kithttp.ErrorEncoder) kithttp.ErrorEncoder

// LoggingErrorEncoder is a go-kit error encoder logging decorator.
func LoggingErrorEncoder(logger logger.Logger, enc kithttp.ErrorEncoder) kithttp.ErrorEncoder {
	return func(ctx context.Context, err error, w http.ResponseWriter) {
		logger.Error(err.Error())
		enc(ctx, err, w)
	}
}
