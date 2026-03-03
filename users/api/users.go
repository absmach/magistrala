// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"regexp"
	"strings"

	"github.com/absmach/supermq"
	grpcTokenV1 "github.com/absmach/supermq/api/grpc/token/v1"
	api "github.com/absmach/supermq/api/http"
	apiutil "github.com/absmach/supermq/api/http/util"
	smqauth "github.com/absmach/supermq/auth"
	smqauthn "github.com/absmach/supermq/pkg/authn"
	"github.com/absmach/supermq/pkg/errors"
	"github.com/absmach/supermq/pkg/oauth2"
	"github.com/absmach/supermq/users"
	"github.com/go-chi/chi/v5"
	kithttp "github.com/go-kit/kit/transport/http"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

var passRegex = regexp.MustCompile("^.{8,}$")

// usersHandler returns a HTTP handler for API endpoints.
func usersHandler(svc users.Service, authn smqauthn.AuthNMiddleware, tokenClient grpcTokenV1.TokenServiceClient, selfRegister bool, r *chi.Mux, logger *slog.Logger, pr *regexp.Regexp, idp supermq.IDProvider, providers ...oauth2.Provider) *chi.Mux {
	passRegex = pr

	opts := []kithttp.ServerOption{
		kithttp.ServerErrorEncoder(apiutil.LoggingErrorEncoder(logger, api.EncodeError)),
	}

	// All endpoints in users service don't required Domain check
	authn = authn.WithOptions(smqauthn.WithDomainCheck(false))
	r.Route("/users", func(r chi.Router) {
		r.Use(api.RequestIDMiddleware(idp))

		switch selfRegister {
		case true:
			r.Post("/", otelhttp.NewHandler(kithttp.NewServer(
				registrationEndpoint(svc, selfRegister),
				decodeCreateUserReq,
				api.EncodeResponse,
				opts...,
			), "register_user").ServeHTTP)
		default:
			r.With(authn.Middleware()).Post("/", otelhttp.NewHandler(kithttp.NewServer(
				registrationEndpoint(svc, selfRegister),
				decodeCreateUserReq,
				api.EncodeResponse,
				opts...,
			), "register_user").ServeHTTP)
		}
		// Endpoints which are allowed for unverified user
		r.Group(func(r chi.Router) {
			r.Use(authn.WithOptions(smqauthn.WithAllowUnverifiedUser(true)).Middleware())
			r.Post("/send-verification", otelhttp.NewHandler(kithttp.NewServer(
				sendVerificationEndpoint(svc),
				decodeSendVerification,
				api.EncodeResponse,
				opts...,
			), "send_verification").ServeHTTP)

			r.Get("/profile", otelhttp.NewHandler(kithttp.NewServer(
				viewProfileEndpoint(svc),
				decodeViewProfile,
				api.EncodeResponse,
				opts...,
			), "view_profile").ServeHTTP)
			r.Post("/tokens/refresh", otelhttp.NewHandler(kithttp.NewServer(
				refreshTokenEndpoint(svc),
				decodeRefreshToken,
				api.EncodeResponse,
				opts...,
			), "refresh_token").ServeHTTP)
			r.Post("/tokens/revoke", otelhttp.NewHandler(kithttp.NewServer(
				revokeRefreshTokenEndpoint(svc),
				decodeRevokeRefreshToken,
				api.EncodeResponse,
				opts...,
			), "revoke_refresh_token").ServeHTTP)
			r.Get("/tokens/refresh-tokens", otelhttp.NewHandler(kithttp.NewServer(
				listActiveRefreshTokensEndpoint(svc),
				decodeListActiveRefreshTokens,
				api.EncodeResponse,
				opts...,
			), "list_active_refresh_tokens").ServeHTTP)
			r.Patch("/{id}/email", otelhttp.NewHandler(kithttp.NewServer(
				updateEmailEndpoint(svc),
				decodeUpdateUserEmail,
				api.EncodeResponse,
				opts...,
			), "update_user_email").ServeHTTP)
		})

		r.Group(func(r chi.Router) {
			r.Use(authn.Middleware())

			r.Get("/{id}", otelhttp.NewHandler(kithttp.NewServer(
				viewEndpoint(svc),
				decodeViewUser,
				api.EncodeResponse,
				opts...,
			), "view_user").ServeHTTP)

			r.Get("/", otelhttp.NewHandler(kithttp.NewServer(
				listUsersEndpoint(svc),
				decodeListUsers,
				api.EncodeResponse,
				opts...,
			), "list_users").ServeHTTP)

			r.Get("/search", otelhttp.NewHandler(kithttp.NewServer(
				searchUsersEndpoint(svc),
				decodeSearchUsers,
				api.EncodeResponse,
				opts...,
			), "search_users").ServeHTTP)

			r.Patch("/secret", otelhttp.NewHandler(kithttp.NewServer(
				updateSecretEndpoint(svc),
				decodeUpdateUserSecret,
				api.EncodeResponse,
				opts...,
			), "update_user_secret").ServeHTTP)

			r.Patch("/{id}", otelhttp.NewHandler(kithttp.NewServer(
				updateEndpoint(svc),
				decodeUpdateUser,
				api.EncodeResponse,
				opts...,
			), "update_user").ServeHTTP)

			r.Patch("/{id}/username", otelhttp.NewHandler(kithttp.NewServer(
				updateUsernameEndpoint(svc),
				decodeUpdateUsername,
				api.EncodeResponse,
				opts...,
			), "update_username").ServeHTTP)

			r.Patch("/{id}/picture", otelhttp.NewHandler(kithttp.NewServer(
				updateProfilePictureEndpoint(svc),
				decodeUpdateUserProfilePicture,
				api.EncodeResponse,
				opts...,
			), "update_profile_picture").ServeHTTP)

			r.Patch("/{id}/tags", otelhttp.NewHandler(kithttp.NewServer(
				updateTagsEndpoint(svc),
				decodeUpdateUserTags,
				api.EncodeResponse,
				opts...,
			), "update_user_tags").ServeHTTP)

			r.Patch("/{id}/role", otelhttp.NewHandler(kithttp.NewServer(
				updateRoleEndpoint(svc),
				decodeUpdateUserRole,
				api.EncodeResponse,
				opts...,
			), "update_user_role").ServeHTTP)

			r.Post("/{id}/enable", otelhttp.NewHandler(kithttp.NewServer(
				enableEndpoint(svc),
				decodeChangeUserStatus,
				api.EncodeResponse,
				opts...,
			), "enable_user").ServeHTTP)

			r.Post("/{id}/disable", otelhttp.NewHandler(kithttp.NewServer(
				disableEndpoint(svc),
				decodeChangeUserStatus,
				api.EncodeResponse,
				opts...,
			), "disable_user").ServeHTTP)

			r.Delete("/{id}", otelhttp.NewHandler(kithttp.NewServer(
				deleteEndpoint(svc),
				decodeChangeUserStatus,
				api.EncodeResponse,
				opts...,
			), "delete_user").ServeHTTP)
		})
	})

	r.Group(func(r chi.Router) {
		r.Use(authn.WithOptions(smqauthn.WithAllowUnverifiedUser(true)).Middleware())
		r.Put("/password/reset", otelhttp.NewHandler(kithttp.NewServer(
			passwordResetEndpoint(svc),
			decodePasswordReset,
			api.EncodeResponse,
			opts...,
		), "password_reset").ServeHTTP)
	})

	r.Post("/users/tokens/issue", otelhttp.NewHandler(kithttp.NewServer(
		issueTokenEndpoint(svc),
		decodeCredentials,
		api.EncodeResponse,
		opts...,
	), "issue_token").ServeHTTP)

	r.Post("/password/reset-request", otelhttp.NewHandler(kithttp.NewServer(
		passwordResetRequestEndpoint(svc),
		decodePasswordResetRequest,
		api.EncodeResponse,
		opts...,
	), "password_reset_req").ServeHTTP)

	r.Get("/verify-email", otelhttp.NewHandler(kithttp.NewServer(
		verifyEmailEndpoint(svc),
		decodeVerifyEmail,
		api.EncodeResponse,
		opts...,
	), "verify_email").ServeHTTP)

	for _, provider := range providers {
		r.HandleFunc("/oauth/callback/"+provider.Name(), oauth2CallbackHandler(provider, svc, tokenClient))
	}

	return r
}

func decodeSendVerification(_ context.Context, r *http.Request) (any, error) {
	req := sendVerificationReq{}
	return req, nil
}

func decodeVerifyEmail(_ context.Context, r *http.Request) (any, error) {
	token, err := apiutil.ReadStringQuery(r, api.TokenKey, "")
	if err != nil {
		return nil, errors.Wrap(apiutil.ErrValidation, err)
	}

	return verifyEmailReq{
		token: token,
	}, nil
}

func decodeViewUser(_ context.Context, r *http.Request) (any, error) {
	req := viewUserReq{
		id: chi.URLParam(r, "id"),
	}

	return req, nil
}

func decodeViewProfile(_ context.Context, r *http.Request) (any, error) {
	return nil, nil
}

func decodeListUsers(_ context.Context, r *http.Request) (any, error) {
	s, err := apiutil.ReadStringQuery(r, api.StatusKey, api.DefUserStatus)
	if err != nil {
		return nil, errors.Wrap(apiutil.ErrValidation, err)
	}
	o, err := apiutil.ReadNumQuery[uint64](r, api.OffsetKey, api.DefOffset)
	if err != nil {
		return nil, errors.Wrap(apiutil.ErrValidation, err)
	}
	l, err := apiutil.ReadNumQuery[uint64](r, api.LimitKey, api.DefLimit)
	if err != nil {
		return nil, errors.Wrap(apiutil.ErrValidation, err)
	}
	m, err := apiutil.ReadMetadataQuery(r, api.MetadataKey, nil)
	if err != nil {
		return nil, errors.Wrap(apiutil.ErrValidation, err)
	}
	n, err := apiutil.ReadStringQuery(r, api.UsernameKey, "")
	if err != nil {
		return nil, errors.Wrap(apiutil.ErrValidation, err)
	}
	d, err := apiutil.ReadStringQuery(r, api.EmailKey, "")
	if err != nil {
		return nil, errors.Wrap(apiutil.ErrValidation, err)
	}
	i, err := apiutil.ReadStringQuery(r, api.FirstNameKey, "")
	if err != nil {
		return nil, errors.Wrap(apiutil.ErrValidation, err)
	}
	f, err := apiutil.ReadStringQuery(r, api.LastNameKey, "")
	if err != nil {
		return nil, errors.Wrap(apiutil.ErrValidation, err)
	}
	t, err := apiutil.ReadStringQuery(r, api.TagsKey, "")
	if err != nil {
		return nil, errors.Wrap(apiutil.ErrValidation, err)
	}
	var tq users.TagsQuery
	if t != "" {
		tq = users.ToTagsQuery(t)
	}
	order, err := apiutil.ReadStringQuery(r, api.OrderKey, api.DefOrder)
	if err != nil {
		return nil, errors.Wrap(apiutil.ErrValidation, err)
	}
	dir, err := apiutil.ReadStringQuery(r, api.DirKey, api.DefDir)
	if err != nil {
		return nil, errors.Wrap(apiutil.ErrValidation, err)
	}
	id, err := apiutil.ReadStringQuery(r, api.IDOrder, "")
	if err != nil {
		return nil, errors.Wrap(apiutil.ErrValidation, err)
	}

	st, err := users.ToStatus(s)
	if err != nil {
		return nil, errors.Wrap(apiutil.ErrValidation, err)
	}

	ot, err := apiutil.ReadBoolQuery(r, api.OnlyTotal, false)
	if err != nil {
		return nil, errors.Wrap(apiutil.ErrValidation, err)
	}

	req := listUsersReq{
		status:    st,
		offset:    o,
		limit:     l,
		onlyTotal: ot,
		metadata:  m,
		userName:  n,
		firstName: i,
		lastName:  f,
		tags:      tq,
		order:     order,
		dir:       dir,
		id:        id,
		email:     d,
	}

	return req, nil
}

func decodeSearchUsers(_ context.Context, r *http.Request) (any, error) {
	o, err := apiutil.ReadNumQuery[uint64](r, api.OffsetKey, api.DefOffset)
	if err != nil {
		return nil, errors.Wrap(apiutil.ErrValidation, err)
	}
	l, err := apiutil.ReadNumQuery[uint64](r, api.LimitKey, api.DefLimit)
	if err != nil {
		return nil, errors.Wrap(apiutil.ErrValidation, err)
	}
	n, err := apiutil.ReadStringQuery(r, api.UsernameKey, "")
	if err != nil {
		return nil, errors.Wrap(apiutil.ErrValidation, err)
	}
	f, err := apiutil.ReadStringQuery(r, api.FirstNameKey, "")
	if err != nil {
		return nil, errors.Wrap(apiutil.ErrValidation, err)
	}
	e, err := apiutil.ReadStringQuery(r, api.LastNameKey, "")
	if err != nil {
		return nil, errors.Wrap(apiutil.ErrValidation, err)
	}
	id, err := apiutil.ReadStringQuery(r, api.IDOrder, "")
	if err != nil {
		return nil, errors.Wrap(apiutil.ErrValidation, err)
	}
	order, err := apiutil.ReadStringQuery(r, api.OrderKey, api.DefOrder)
	if err != nil {
		return nil, errors.Wrap(apiutil.ErrValidation, err)
	}
	dir, err := apiutil.ReadStringQuery(r, api.DirKey, api.DefDir)
	if err != nil {
		return nil, errors.Wrap(apiutil.ErrValidation, err)
	}

	req := searchUsersReq{
		Offset:    o,
		Limit:     l,
		Username:  n,
		FirstName: f,
		LastName:  e,
		Id:        id,
		Order:     order,
		Dir:       dir,
	}

	for _, field := range []string{req.Username, req.Id} {
		if field != "" && len(field) < 3 {
			req = searchUsersReq{}
			return req, errors.Wrap(apiutil.ErrLenSearchQuery, apiutil.ErrValidation)
		}
	}

	return req, nil
}

func decodeUpdateUser(_ context.Context, r *http.Request) (any, error) {
	if !strings.Contains(r.Header.Get("Content-Type"), api.ContentType) {
		return nil, errors.Wrap(apiutil.ErrValidation, apiutil.ErrUnsupportedContentType)
	}

	req := updateUserReq{
		id: chi.URLParam(r, "id"),
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, errors.Wrap(apiutil.ErrMalformedRequestBody, err)
	}

	return req, nil
}

func decodeUpdateUserTags(_ context.Context, r *http.Request) (any, error) {
	if !strings.Contains(r.Header.Get("Content-Type"), api.ContentType) {
		return nil, errors.Wrap(apiutil.ErrValidation, apiutil.ErrUnsupportedContentType)
	}

	req := updateUserTagsReq{
		id: chi.URLParam(r, "id"),
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, errors.Wrap(apiutil.ErrMalformedRequestBody, err)
	}

	return req, nil
}

func decodeUpdateUserEmail(_ context.Context, r *http.Request) (any, error) {
	if !strings.Contains(r.Header.Get("Content-Type"), api.ContentType) {
		return nil, errors.Wrap(apiutil.ErrValidation, apiutil.ErrUnsupportedContentType)
	}

	req := updateEmailReq{
		id: chi.URLParam(r, "id"),
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, errors.Wrap(apiutil.ErrMalformedRequestBody, err)
	}

	return req, nil
}

func decodeUpdateUserSecret(_ context.Context, r *http.Request) (any, error) {
	if !strings.Contains(r.Header.Get("Content-Type"), api.ContentType) {
		return nil, errors.Wrap(apiutil.ErrValidation, apiutil.ErrUnsupportedContentType)
	}

	req := updateUserSecretReq{}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, errors.Wrap(apiutil.ErrMalformedRequestBody, err)
	}

	return req, nil
}

func decodeUpdateUsername(_ context.Context, r *http.Request) (any, error) {
	if !strings.Contains(r.Header.Get("Content-Type"), api.ContentType) {
		return nil, errors.Wrap(apiutil.ErrValidation, apiutil.ErrUnsupportedContentType)
	}

	req := updateUsernameReq{
		id: chi.URLParam(r, "id"),
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, errors.Wrap(apiutil.ErrMalformedRequestBody, err)
	}

	return req, nil
}

func decodeUpdateUserProfilePicture(_ context.Context, r *http.Request) (any, error) {
	if !strings.Contains(r.Header.Get("Content-Type"), api.ContentType) {
		return nil, errors.Wrap(apiutil.ErrValidation, apiutil.ErrUnsupportedContentType)
	}

	req := updateProfilePictureReq{
		id: chi.URLParam(r, "id"),
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, errors.Wrap(apiutil.ErrMalformedRequestBody, err)
	}

	return req, nil
}

func decodePasswordResetRequest(_ context.Context, r *http.Request) (any, error) {
	if !strings.Contains(r.Header.Get("Content-Type"), api.ContentType) {
		return nil, apiutil.ErrUnsupportedContentType
	}

	var req passResetReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, errors.Wrap(apiutil.ErrMalformedRequestBody, err)
	}

	return req, nil
}

func decodePasswordReset(_ context.Context, r *http.Request) (any, error) {
	if !strings.Contains(r.Header.Get("Content-Type"), api.ContentType) {
		return nil, errors.Wrap(apiutil.ErrValidation, apiutil.ErrUnsupportedContentType)
	}

	var req resetTokenReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, errors.Wrap(apiutil.ErrMalformedRequestBody, err)
	}

	return req, nil
}

func decodeUpdateUserRole(_ context.Context, r *http.Request) (any, error) {
	if !strings.Contains(r.Header.Get("Content-Type"), api.ContentType) {
		return nil, errors.Wrap(apiutil.ErrValidation, apiutil.ErrUnsupportedContentType)
	}

	req := updateUserRoleReq{
		id: chi.URLParam(r, "id"),
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, errors.Wrap(apiutil.ErrMalformedRequestBody, err)
	}
	var err error
	req.role, err = users.ToRole(req.Role)
	return req, err
}

func decodeCredentials(_ context.Context, r *http.Request) (any, error) {
	if !strings.Contains(r.Header.Get("Content-Type"), api.ContentType) {
		return nil, errors.Wrap(apiutil.ErrValidation, apiutil.ErrUnsupportedContentType)
	}

	req := loginUserReq{}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, errors.Wrap(apiutil.ErrMalformedRequestBody, err)
	}

	return req, nil
}

func decodeRefreshToken(_ context.Context, r *http.Request) (any, error) {
	if !strings.Contains(r.Header.Get("Content-Type"), api.ContentType) {
		return nil, errors.Wrap(apiutil.ErrValidation, apiutil.ErrUnsupportedContentType)
	}
	req := tokenReq{RefreshToken: apiutil.ExtractBearerToken(r)}

	return req, nil
}

func decodeRevokeRefreshToken(_ context.Context, r *http.Request) (any, error) {
	if !strings.Contains(r.Header.Get("Content-Type"), api.ContentType) {
		return nil, errors.Wrap(apiutil.ErrValidation, apiutil.ErrUnsupportedContentType)
	}

	var req revokeTokenReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, errors.Wrap(apiutil.ErrMalformedRequestBody, err)
	}

	return req, nil
}

func decodeListActiveRefreshTokens(_ context.Context, r *http.Request) (any, error) {
	return nil, nil
}

func decodeCreateUserReq(_ context.Context, r *http.Request) (any, error) {
	if !strings.Contains(r.Header.Get("Content-Type"), api.ContentType) {
		return nil, errors.Wrap(apiutil.ErrValidation, apiutil.ErrUnsupportedContentType)
	}

	var req createUserReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, errors.Wrap(apiutil.ErrMalformedRequestBody, err)
	}

	return req, nil
}

func decodeChangeUserStatus(_ context.Context, r *http.Request) (any, error) {
	req := changeUserStatusReq{
		id: chi.URLParam(r, "id"),
	}

	return req, nil
}

// oauth2CallbackHandler is a http.HandlerFunc that handles OAuth2 callbacks.
func oauth2CallbackHandler(oauth oauth2.Provider, svc users.Service, tokenClient grpcTokenV1.TokenServiceClient) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !oauth.IsEnabled() {
			http.Redirect(w, r, oauth.ErrorURL()+"?error=oauth%20provider%20is%20disabled", http.StatusSeeOther)
			return
		}
		state := r.FormValue("state")
		if state != oauth.State() {
			http.Redirect(w, r, oauth.ErrorURL()+"?error=invalid%20state", http.StatusSeeOther)
			return
		}

		if code := r.FormValue("code"); code != "" {
			token, err := oauth.Exchange(r.Context(), code)
			if err != nil {
				http.Redirect(w, r, oauth.ErrorURL()+"?error="+err.Error(), http.StatusSeeOther)
				return
			}

			user, err := oauth.UserInfo(token.AccessToken)
			if err != nil {
				http.Redirect(w, r, oauth.ErrorURL()+"?error="+err.Error(), http.StatusSeeOther)
				return
			}

			user.AuthProvider = oauth.Name()
			if user.AuthProvider == "" {
				user.AuthProvider = "oauth"
			}
			user, err = svc.OAuthCallback(r.Context(), user)
			if err != nil {
				http.Redirect(w, r, oauth.ErrorURL()+"?error="+err.Error(), http.StatusSeeOther)
				return
			}
			if err := svc.OAuthAddUserPolicy(r.Context(), user); err != nil {
				http.Redirect(w, r, oauth.ErrorURL()+"?error="+err.Error(), http.StatusSeeOther)
				return
			}

			jwt, err := tokenClient.Issue(r.Context(), &grpcTokenV1.IssueReq{
				UserId:   user.ID,
				Type:     uint32(smqauth.AccessKey),
				UserRole: uint32(smqauth.UserRole),
				Verified: !user.VerifiedAt.IsZero(),
			})
			if err != nil {
				http.Redirect(w, r, oauth.ErrorURL()+"?error="+err.Error(), http.StatusSeeOther)
				return
			}

			http.SetCookie(w, &http.Cookie{
				Name:     "access_token",
				Value:    jwt.GetAccessToken(),
				Path:     "/",
				HttpOnly: true,
				Secure:   true,
			})
			http.SetCookie(w, &http.Cookie{
				Name:     "refresh_token",
				Value:    jwt.GetRefreshToken(),
				Path:     "/",
				HttpOnly: true,
				Secure:   true,
			})

			http.Redirect(w, r, oauth.RedirectURL(), http.StatusFound)
			return
		}

		http.Redirect(w, r, oauth.ErrorURL()+"?error=empty%20code", http.StatusSeeOther)
	}
}
