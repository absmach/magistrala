// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package authn

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"strconv"

	apiutil "github.com/absmach/supermq/api/http/util"
	"github.com/absmach/supermq/pkg/errors"
	"github.com/absmach/supermq/pkg/policies"
	"github.com/go-chi/chi/v5"
)

type sessionKeyType string

const (
	allowUnverifiedUserEnv = "MG_ALLOW_UNVERIFIED_USER"
	jsonContentType        = "application/json"

	SessionKey = sessionKeyType("session")
)

// middlewareOptions contains configuration for authentication middleware.
type middlewareOptions struct {
	domainCheck         bool
	allowUnverifiedUser bool
}

// defaultMiddlewareOptions returns the default middleware configuration.
func defaultMiddlewareOptions() *middlewareOptions {
	return &middlewareOptions{
		domainCheck:         true,
		allowUnverifiedUser: false,
	}
}

// MiddlewareOption is a function that modifies middleware options.
type MiddlewareOption func(*middlewareOptions)

// WithDomainCheck sets whether domain checking is enabled.
func WithDomainCheck(enabled bool) MiddlewareOption {
	return func(opts *middlewareOptions) {
		opts.domainCheck = enabled
	}
}

// WithAllowUnverifiedUser sets whether unverified users are allowed.
func WithAllowUnverifiedUser(allowed bool) MiddlewareOption {
	return func(opts *middlewareOptions) {
		opts.allowUnverifiedUser = allowed
	}
}

// WithDefaultMiddlewareOptions resets options to default values.
func WithDefaultMiddlewareOptions() MiddlewareOption {
	return func(opts *middlewareOptions) {
		defaults := defaultMiddlewareOptions()
		opts.domainCheck = defaults.domainCheck
		opts.allowUnverifiedUser = defaults.allowUnverifiedUser
	}
}

// AuthNMiddleware defines the interface for authenticated services with middleware.
type AuthNMiddleware interface {
	Authentication
	WithOptions(options ...MiddlewareOption) AuthNMiddleware
	Middleware() func(http.Handler) http.Handler
}

// authnMiddleware wraps Authentication with middleware functionality.
type authnMiddleware struct {
	Authentication
	options []MiddlewareOption
}

// NewAuthNMiddleware creates a new authenticated service with middleware support.
// The order of precedence for options is as follows, with later options overriding earlier ones:
// 1. Default options (lowest precedence).
// 2. Options from environment variables (e.g., MG_ALLOW_UNVERIFIED_USER).
// 3. Options passed as arguments to this function (highest precedence).
//
// For example, consider the 'allowUnverifiedUser' option:
//   - By default, it is 'false'.
//   - If the MG_ALLOW_UNVERIFIED_USER environment variable is set to "true",
//     it becomes 'true'.
//   - If NewAuthNMiddleware is called with WithAllowUnverifiedUser(false), it will be 'false',
//     regardless of the environment variable, as function arguments have the highest precedence.
func NewAuthNMiddleware(authnSvc Authentication, options ...MiddlewareOption) AuthNMiddleware {
	allOptions := []MiddlewareOption{WithDefaultMiddlewareOptions()}
	if val, ok := os.LookupEnv(allowUnverifiedUserEnv); ok {
		allowUnverifiedUser, err := strconv.ParseBool(val)
		if err == nil && allowUnverifiedUser {
			allOptions = append(allOptions, WithAllowUnverifiedUser(true))
		}
	}
	allOptions = append(allOptions, options...)
	return &authnMiddleware{
		Authentication: authnSvc,
		options:        allOptions,
	}
}

// WithOptions returns a new service with additional options.
func (a *authnMiddleware) WithOptions(options ...MiddlewareOption) AuthNMiddleware {
	return &authnMiddleware{
		Authentication: a.Authentication,
		options:        append(a.options, options...),
	}
}

// getMiddlewareOptions returns the configured middleware options.
func (a *authnMiddleware) getMiddlewareOptions() *middlewareOptions {
	opts := defaultMiddlewareOptions()
	for _, option := range a.options {
		option(opts)
	}
	return opts
}

// Middleware returns an HTTP middleware function that handles authentication.
func (a *authnMiddleware) Middleware() func(http.Handler) http.Handler {
	opts := a.getMiddlewareOptions()
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token := apiutil.ExtractBearerToken(r)
			if token == "" {
				encodeError(w, apiutil.ErrBearerToken, http.StatusUnauthorized)
				return
			}
			resp, err := a.Authenticate(r.Context(), token)
			if err != nil {
				encodeError(w, err, http.StatusUnauthorized)
				return
			}

			if resp.Type == AccessToken && !opts.allowUnverifiedUser && resp.Role != SuperAdminRole && !resp.Verified {
				encodeError(w, apiutil.ErrEmailNotVerified, http.StatusUnauthorized)
				return
			}

			if opts.domainCheck {
				domain := chi.URLParam(r, "domainID")
				if domain == "" {
					encodeError(w, apiutil.ErrMissingDomainID, http.StatusBadRequest)
					return
				}
				resp.DomainID = domain
				switch resp.Role {
				case SuperAdminRole:
					resp.DomainUserID = resp.UserID
				case UserRole:
					resp.DomainUserID = policies.EncodeDomainUserID(domain, resp.UserID)
				}
			}

			ctx := context.WithValue(r.Context(), SessionKey, resp)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func encodeError(w http.ResponseWriter, err error, statusCode int) {
	if errorVal, ok := err.(errors.Error); ok {
		w.Header().Set("Content-Type", jsonContentType)
		w.WriteHeader(statusCode)
		if err := json.NewEncoder(w).Encode(errorVal); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
		}
		return
	}
	http.Error(w, err.Error(), statusCode)
}
