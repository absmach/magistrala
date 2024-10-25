// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"context"
	"net/http"

	"github.com/absmach/magistrala/pkg/apiutil"
	mgauthn "github.com/absmach/magistrala/pkg/authn"
	"github.com/go-chi/chi/v5"
)

type sessionKeyType string

const SessionKey = sessionKeyType("session")

func AuthenticateMiddleware(authn mgauthn.Authentication, domainCheck bool) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token := apiutil.ExtractBearerToken(r)
			if token == "" {
				EncodeError(r.Context(), apiutil.ErrBearerToken, w)
				return
			}

			resp, err := authn.Authenticate(r.Context(), token)
			if err != nil {
				EncodeError(r.Context(), err, w)
				return
			}

			if domainCheck {
				domain := chi.URLParam(r, "domainID")
				if domain == "" {
					EncodeError(r.Context(), apiutil.ErrMissingDomainID, w)
					return
				}
				resp.DomainID = domain
				resp.DomainUserID = domain + "_" + resp.UserID
			}

			ctx := context.WithValue(r.Context(), SessionKey, resp)

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
