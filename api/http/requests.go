// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package http

import (
	"context"
	"net/http"

	"github.com/absmach/supermq"
	"github.com/go-chi/chi/v5/middleware"
)

func RequestIDMiddleware(idp supermq.IDProvider) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requestID, err := idp.ID()
			if err != nil {
				EncodeError(r.Context(), err, w)
				return
			}

			ctx := context.WithValue(r.Context(), middleware.RequestIDKey, requestID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
