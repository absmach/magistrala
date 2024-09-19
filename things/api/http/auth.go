// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package http

import (
	"context"
	"net/http"

	"github.com/absmach/magistrala"
	"github.com/absmach/magistrala/internal/api"
	"github.com/absmach/magistrala/pkg/apiutil"
	"github.com/absmach/magistrala/pkg/auth"
	"github.com/absmach/magistrala/pkg/errors"
	svcerr "github.com/absmach/magistrala/pkg/errors/service"
	"github.com/absmach/magistrala/pkg/policies"
	"github.com/go-kit/kit/endpoint"
)

type sessionKeyType string

const sessionKey = sessionKeyType("session")

type authEndpointFunc func(context.Context, interface{}) (*magistrala.AuthorizeReq, error)

func identifyMiddleware(authClient auth.AuthClient) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token := apiutil.ExtractBearerToken(r)
			if token == "" {
				api.EncodeError(r.Context(), apiutil.ErrBearerToken, w)
				return
			}

			resp, err := authClient.Identify(r.Context(), &magistrala.IdentityReq{Token: token})
			if err != nil {
				api.EncodeError(r.Context(), err, w)
				return
			}

			ctx := context.WithValue(r.Context(), sessionKey, auth.Session{
				DomainUserID: resp.GetId(),
				UserID:       resp.GetUserId(),
				DomainID:     resp.GetDomainId(),
			})

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func authorizeMiddleware(authClient auth.AuthClient, getAuthReq authEndpointFunc) endpoint.Middleware {
	return func(next endpoint.Endpoint) endpoint.Endpoint {
		return func(ctx context.Context, request interface{}) (interface{}, error) {
			pr, err := getAuthReq(ctx, request)
			if err != nil {
				return nil, errors.Wrap(apiutil.ErrValidation, err)
			}

			res, err := authClient.Authorize(ctx, pr)
			if err != nil || !res.Authorized {
				return nil, errors.Wrap(svcerr.ErrAuthorization, err)
			}
			return next(ctx, request)
		}
	}
}

func checkSuperAdminMiddleware(authClient auth.AuthClient) endpoint.Middleware {
	return func(next endpoint.Endpoint) endpoint.Endpoint {
		return func(ctx context.Context, request interface{}) (interface{}, error) {
			session, ok := ctx.Value(sessionKey).(auth.Session)
			if !ok {
				return nil, svcerr.ErrAuthorization
			}
			var superAdmin bool
			_, err := authClient.Authorize(ctx, &magistrala.AuthorizeReq{
				SubjectType: policies.UserType,
				SubjectKind: policies.UsersKind,
				Subject:     session.UserID,
				Permission:  policies.AdminPermission,
				ObjectType:  policies.PlatformType,
				Object:      policies.MagistralaObject,
			})
			if err == nil {
				superAdmin = true
			}

			ctx = context.WithValue(ctx, sessionKey, auth.Session{
				DomainUserID: session.DomainUserID,
				UserID:       session.UserID,
				DomainID:     session.DomainID,
				SuperAdmin:   superAdmin,
			})

			return next(ctx, request)
		}
	}
}
