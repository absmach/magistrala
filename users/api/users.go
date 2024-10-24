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

	mgauth "github.com/absmach/magistrala/auth"
	"github.com/absmach/magistrala/internal/api"
	grpcTokenV1 "github.com/absmach/magistrala/internal/grpc/token/v1"
	"github.com/absmach/magistrala/pkg/apiutil"
	mgauthn "github.com/absmach/magistrala/pkg/authn"
	"github.com/absmach/magistrala/pkg/errors"
	"github.com/absmach/magistrala/pkg/oauth2"
	"github.com/absmach/magistrala/pkg/policies"
	"github.com/absmach/magistrala/users"
	"github.com/go-chi/chi/v5"
	kithttp "github.com/go-kit/kit/transport/http"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

var passRegex = regexp.MustCompile("^.{8,}$")

// usersHandler returns a HTTP handler for API endpoints.
func usersHandler(svc users.Service, authn mgauthn.Authentication, tokenClient grpcTokenV1.TokenServiceClient, selfRegister bool, r *chi.Mux, logger *slog.Logger, pr *regexp.Regexp, providers ...oauth2.Provider) *chi.Mux {
	passRegex = pr

	opts := []kithttp.ServerOption{
		kithttp.ServerErrorEncoder(apiutil.LoggingErrorEncoder(logger, api.EncodeError)),
	}

	r.Route("/users", func(r chi.Router) {
		switch selfRegister {
		case true:
			r.Post("/", otelhttp.NewHandler(kithttp.NewServer(
				registrationEndpoint(svc, selfRegister),
				decodeCreateUserReq,
				api.EncodeResponse,
				opts...,
			), "register_user").ServeHTTP)
		default:
			r.With(api.AuthenticateMiddleware(authn, false)).Post("/", otelhttp.NewHandler(kithttp.NewServer(
				registrationEndpoint(svc, selfRegister),
				decodeCreateUserReq,
				api.EncodeResponse,
				opts...,
			), "register_user").ServeHTTP)
		}

		r.Group(func(r chi.Router) {
			r.Use(api.AuthenticateMiddleware(authn, false))

			r.Get("/profile", otelhttp.NewHandler(kithttp.NewServer(
				viewProfileEndpoint(svc),
				decodeViewProfile,
				api.EncodeResponse,
				opts...,
			), "view_profile").ServeHTTP)

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

			r.Patch("/{id}/email", otelhttp.NewHandler(kithttp.NewServer(
				updateEmailEndpoint(svc),
				decodeUpdateUserEmail,
				api.EncodeResponse,
				opts...,
			), "update_user_email").ServeHTTP)

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

			r.Post("/tokens/refresh", otelhttp.NewHandler(kithttp.NewServer(
				refreshTokenEndpoint(svc),
				decodeRefreshToken,
				api.EncodeResponse,
				opts...,
			), "refresh_token").ServeHTTP)
		})
	})

	r.Group(func(r chi.Router) {
		r.Use(api.AuthenticateMiddleware(authn, false))
		r.Put("/password/reset", otelhttp.NewHandler(kithttp.NewServer(
			passwordResetEndpoint(svc),
			decodePasswordReset,
			api.EncodeResponse,
			opts...,
		), "password_reset").ServeHTTP)
	})

	r.Group(func(r chi.Router) {
		r.Use(api.AuthenticateMiddleware(authn, true))

		// Ideal location: users service, groups endpoint.
		// Reason for placing here :
		// SpiceDB provides list of user ids in given user_group_id
		// and users service can access spiceDB and get the user list with user_group_id.
		// Request to get list of users present in the user_group_id {groupID}
		r.Get("/{domainID}/groups/{groupID}/users", otelhttp.NewHandler(kithttp.NewServer(
			listMembersByGroupEndpoint(svc),
			decodeListMembersByGroup,
			api.EncodeResponse,
			opts...,
		), "list_users_by_user_group_id").ServeHTTP)

		// Ideal location: clients service, channels endpoint.
		// Reason for placing here :
		// SpiceDB provides list of user ids in given channel_id
		// and users service can access spiceDB and get the user list with channel_id.
		// Request to get list of users present in the user_group_id {channelID}
		r.Get("/{domainID}/channels/{channelID}/users", otelhttp.NewHandler(kithttp.NewServer(
			listMembersByChannelEndpoint(svc),
			decodeListMembersByChannel,
			api.EncodeResponse,
			opts...,
		), "list_users_by_channel_id").ServeHTTP)

		r.Get("/{domainID}/clients/{clientID}/users", otelhttp.NewHandler(kithttp.NewServer(
			listMembersByClientEndpoint(svc),
			decodeListMembersByClient,
			api.EncodeResponse,
			opts...,
		), "list_users_by_client_id").ServeHTTP)

		r.Get("/{domainID}/users", otelhttp.NewHandler(kithttp.NewServer(
			listMembersByDomainEndpoint(svc),
			decodeListMembersByDomain,
			api.EncodeResponse,
			opts...,
		), "list_users_by_domain_id").ServeHTTP)
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

	for _, provider := range providers {
		r.HandleFunc("/oauth/callback/"+provider.Name(), oauth2CallbackHandler(provider, svc, tokenClient))
	}

	return r
}

func decodeViewUser(_ context.Context, r *http.Request) (interface{}, error) {
	req := viewUserReq{
		id: chi.URLParam(r, "id"),
	}

	return req, nil
}

func decodeViewProfile(_ context.Context, r *http.Request) (interface{}, error) {
	return nil, nil
}

func decodeListUsers(_ context.Context, r *http.Request) (interface{}, error) {
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
	t, err := apiutil.ReadStringQuery(r, api.TagKey, "")
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
	id, err := apiutil.ReadStringQuery(r, api.IDOrder, "")
	if err != nil {
		return nil, errors.Wrap(apiutil.ErrValidation, err)
	}

	st, err := users.ToStatus(s)
	if err != nil {
		return nil, errors.Wrap(apiutil.ErrValidation, err)
	}
	req := listUsersReq{
		status:    st,
		offset:    o,
		limit:     l,
		metadata:  m,
		userName:  n,
		firstName: i,
		lastName:  f,
		tag:       t,
		order:     order,
		dir:       dir,
		id:        id,
		email:     d,
	}

	return req, nil
}

func decodeSearchUsers(_ context.Context, r *http.Request) (interface{}, error) {
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

func decodeUpdateUser(_ context.Context, r *http.Request) (interface{}, error) {
	if !strings.Contains(r.Header.Get("Content-Type"), api.ContentType) {
		return nil, errors.Wrap(apiutil.ErrValidation, apiutil.ErrUnsupportedContentType)
	}

	req := updateUserReq{
		id: chi.URLParam(r, "id"),
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, errors.Wrap(apiutil.ErrValidation, errors.Wrap(err, errors.ErrMalformedEntity))
	}

	return req, nil
}

func decodeUpdateUserTags(_ context.Context, r *http.Request) (interface{}, error) {
	if !strings.Contains(r.Header.Get("Content-Type"), api.ContentType) {
		return nil, errors.Wrap(apiutil.ErrValidation, apiutil.ErrUnsupportedContentType)
	}

	req := updateUserTagsReq{
		id: chi.URLParam(r, "id"),
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, errors.Wrap(apiutil.ErrValidation, errors.Wrap(err, errors.ErrMalformedEntity))
	}

	return req, nil
}

func decodeUpdateUserEmail(_ context.Context, r *http.Request) (interface{}, error) {
	if !strings.Contains(r.Header.Get("Content-Type"), api.ContentType) {
		return nil, errors.Wrap(apiutil.ErrValidation, apiutil.ErrUnsupportedContentType)
	}

	req := updateEmailReq{
		id: chi.URLParam(r, "id"),
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, errors.Wrap(apiutil.ErrValidation, errors.Wrap(err, errors.ErrMalformedEntity))
	}

	return req, nil
}

func decodeUpdateUserSecret(_ context.Context, r *http.Request) (interface{}, error) {
	if !strings.Contains(r.Header.Get("Content-Type"), api.ContentType) {
		return nil, errors.Wrap(apiutil.ErrValidation, apiutil.ErrUnsupportedContentType)
	}

	req := updateUserSecretReq{}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, errors.Wrap(apiutil.ErrValidation, errors.Wrap(err, errors.ErrMalformedEntity))
	}

	return req, nil
}

func decodeUpdateUsername(_ context.Context, r *http.Request) (interface{}, error) {
	if !strings.Contains(r.Header.Get("Content-Type"), api.ContentType) {
		return nil, errors.Wrap(apiutil.ErrValidation, apiutil.ErrUnsupportedContentType)
	}

	req := updateUsernameReq{
		id: chi.URLParam(r, "id"),
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, errors.Wrap(apiutil.ErrValidation, errors.Wrap(err, errors.ErrMalformedEntity))
	}

	return req, nil
}

func decodeUpdateUserProfilePicture(_ context.Context, r *http.Request) (interface{}, error) {
	if !strings.Contains(r.Header.Get("Content-Type"), api.ContentType) {
		return nil, errors.Wrap(apiutil.ErrValidation, apiutil.ErrUnsupportedContentType)
	}

	req := updateProfilePictureReq{
		id: chi.URLParam(r, "id"),
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, errors.Wrap(apiutil.ErrValidation, errors.Wrap(err, errors.ErrMalformedEntity))
	}

	return req, nil
}

func decodePasswordResetRequest(_ context.Context, r *http.Request) (interface{}, error) {
	if !strings.Contains(r.Header.Get("Content-Type"), api.ContentType) {
		return nil, apiutil.ErrUnsupportedContentType
	}

	var req passwResetReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, errors.Wrap(apiutil.ErrValidation, errors.Wrap(err, errors.ErrMalformedEntity))
	}

	req.Host = r.Header.Get("Referer")
	return req, nil
}

func decodePasswordReset(_ context.Context, r *http.Request) (interface{}, error) {
	if !strings.Contains(r.Header.Get("Content-Type"), api.ContentType) {
		return nil, errors.Wrap(apiutil.ErrValidation, apiutil.ErrUnsupportedContentType)
	}

	var req resetTokenReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, errors.Wrap(apiutil.ErrValidation, errors.Wrap(err, errors.ErrMalformedEntity))
	}

	return req, nil
}

func decodeUpdateUserRole(_ context.Context, r *http.Request) (interface{}, error) {
	if !strings.Contains(r.Header.Get("Content-Type"), api.ContentType) {
		return nil, errors.Wrap(apiutil.ErrValidation, apiutil.ErrUnsupportedContentType)
	}

	req := updateUserRoleReq{
		id: chi.URLParam(r, "id"),
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, errors.Wrap(apiutil.ErrValidation, errors.Wrap(err, errors.ErrMalformedEntity))
	}
	var err error
	req.role, err = users.ToRole(req.Role)
	return req, err
}

func decodeCredentials(_ context.Context, r *http.Request) (interface{}, error) {
	if !strings.Contains(r.Header.Get("Content-Type"), api.ContentType) {
		return nil, errors.Wrap(apiutil.ErrValidation, apiutil.ErrUnsupportedContentType)
	}

	req := loginUserReq{}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, errors.Wrap(apiutil.ErrValidation, errors.Wrap(err, errors.ErrMalformedEntity))
	}

	return req, nil
}

func decodeRefreshToken(_ context.Context, r *http.Request) (interface{}, error) {
	if !strings.Contains(r.Header.Get("Content-Type"), api.ContentType) {
		return nil, errors.Wrap(apiutil.ErrValidation, apiutil.ErrUnsupportedContentType)
	}
	req := tokenReq{RefreshToken: apiutil.ExtractBearerToken(r)}

	return req, nil
}

func decodeCreateUserReq(_ context.Context, r *http.Request) (interface{}, error) {
	if !strings.Contains(r.Header.Get("Content-Type"), api.ContentType) {
		return nil, errors.Wrap(apiutil.ErrValidation, apiutil.ErrUnsupportedContentType)
	}

	var req createUserReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, errors.Wrap(apiutil.ErrValidation, errors.Wrap(err, errors.ErrMalformedEntity))
	}

	return req, nil
}

func decodeChangeUserStatus(_ context.Context, r *http.Request) (interface{}, error) {
	req := changeUserStatusReq{
		id: chi.URLParam(r, "id"),
	}

	return req, nil
}

func decodeListMembersByGroup(_ context.Context, r *http.Request) (interface{}, error) {
	page, err := queryPageParams(r, api.DefPermission)
	if err != nil {
		return nil, err
	}
	req := listMembersByObjectReq{
		Page:     page,
		objectID: chi.URLParam(r, "groupID"),
	}

	return req, nil
}

func decodeListMembersByChannel(_ context.Context, r *http.Request) (interface{}, error) {
	page, err := queryPageParams(r, api.DefPermission)
	if err != nil {
		return nil, err
	}
	req := listMembersByObjectReq{
		Page:     page,
		objectID: chi.URLParam(r, "channelID"),
	}

	return req, nil
}

func decodeListMembersByClient(_ context.Context, r *http.Request) (interface{}, error) {
	page, err := queryPageParams(r, api.DefPermission)
	if err != nil {
		return nil, err
	}
	req := listMembersByObjectReq{
		Page:     page,
		objectID: chi.URLParam(r, "clientID"),
	}

	return req, nil
}

func decodeListMembersByDomain(_ context.Context, r *http.Request) (interface{}, error) {
	page, err := queryPageParams(r, policies.MembershipPermission)
	if err != nil {
		return nil, err
	}

	req := listMembersByObjectReq{
		Page:     page,
		objectID: chi.URLParam(r, "domainID"),
	}

	return req, nil
}

func queryPageParams(r *http.Request, defPermission string) (users.Page, error) {
	s, err := apiutil.ReadStringQuery(r, api.StatusKey, api.DefClientStatus)
	if err != nil {
		return users.Page{}, errors.Wrap(apiutil.ErrValidation, err)
	}
	o, err := apiutil.ReadNumQuery[uint64](r, api.OffsetKey, api.DefOffset)
	if err != nil {
		return users.Page{}, errors.Wrap(apiutil.ErrValidation, err)
	}
	l, err := apiutil.ReadNumQuery[uint64](r, api.LimitKey, api.DefLimit)
	if err != nil {
		return users.Page{}, errors.Wrap(apiutil.ErrValidation, err)
	}
	m, err := apiutil.ReadMetadataQuery(r, api.MetadataKey, nil)
	if err != nil {
		return users.Page{}, errors.Wrap(apiutil.ErrValidation, err)
	}
	n, err := apiutil.ReadStringQuery(r, api.UsernameKey, "")
	if err != nil {
		return users.Page{}, errors.Wrap(apiutil.ErrValidation, err)
	}
	f, err := apiutil.ReadStringQuery(r, api.FirstNameKey, "")
	if err != nil {
		return users.Page{}, errors.Wrap(apiutil.ErrValidation, err)
	}
	a, err := apiutil.ReadStringQuery(r, api.LastNameKey, "")
	if err != nil {
		return users.Page{}, errors.Wrap(apiutil.ErrValidation, err)
	}
	i, err := apiutil.ReadStringQuery(r, api.EmailKey, "")
	if err != nil {
		return users.Page{}, errors.Wrap(apiutil.ErrValidation, err)
	}
	t, err := apiutil.ReadStringQuery(r, api.TagKey, "")
	if err != nil {
		return users.Page{}, errors.Wrap(apiutil.ErrValidation, err)
	}
	st, err := users.ToStatus(s)
	if err != nil {
		return users.Page{}, errors.Wrap(apiutil.ErrValidation, err)
	}
	p, err := apiutil.ReadStringQuery(r, api.PermissionKey, defPermission)
	if err != nil {
		return users.Page{}, errors.Wrap(apiutil.ErrValidation, err)
	}
	lp, err := apiutil.ReadBoolQuery(r, api.ListPerms, api.DefListPerms)
	if err != nil {
		return users.Page{}, errors.Wrap(apiutil.ErrValidation, err)
	}
	return users.Page{
		Status:     st,
		Offset:     o,
		Limit:      l,
		Metadata:   m,
		FirstName:  f,
		Username:   n,
		LastName:   a,
		Email:      i,
		Tag:        t,
		Permission: p,
		ListPerms:  lp,
	}, nil
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
				UserId: user.ID,
				Type:   uint32(mgauth.AccessKey),
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
