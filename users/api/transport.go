// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/mainflux/mainflux/pkg/errors"

	kitot "github.com/go-kit/kit/tracing/opentracing"
	kithttp "github.com/go-kit/kit/transport/http"
	"github.com/go-zoo/bone"
	"github.com/mainflux/mainflux"
	log "github.com/mainflux/mainflux/logger"
	"github.com/mainflux/mainflux/users"
	opentracing "github.com/opentracing/opentracing-go"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const (
	contentType = "application/json"

	offsetKey   = "offset"
	limitKey    = "limit"
	nameKey     = "name"
	metadataKey = "metadata"

	defOffset = 0
	defLimit  = 10
)

var (
	errInvalidQueryParams = errors.New("invalid query params")

	// ErrUnsupportedContentType indicates unacceptable or lack of Content-Type
	ErrUnsupportedContentType = errors.New("unsupported content type")

	// ErrFailedDecode indicates failed to decode request body
	ErrFailedDecode = errors.New("failed to decode request body")
	logger          log.Logger
)

// MakeHandler returns a HTTP handler for API endpoints.
func MakeHandler(svc users.Service, tracer opentracing.Tracer, l log.Logger) http.Handler {
	logger = l

	opts := []kithttp.ServerOption{
		kithttp.ServerErrorEncoder(encodeError),
	}

	mux := bone.New()

	mux.Post("/users", kithttp.NewServer(
		kitot.TraceServer(tracer, "register")(registrationEndpoint(svc)),
		decodeCredentials,
		encodeResponse,
		opts...,
	))

	mux.Get("/users", kithttp.NewServer(
		kitot.TraceServer(tracer, "view_user")(viewUserEndpoint(svc)),
		decodeViewUser,
		encodeResponse,
		opts...,
	))

	mux.Put("/users", kithttp.NewServer(
		kitot.TraceServer(tracer, "update_user")(updateUserEndpoint(svc)),
		decodeUpdateUser,
		encodeResponse,
		opts...,
	))

	mux.Get("/users/:userID/groups", kithttp.NewServer(
		kitot.TraceServer(tracer, "memberships")(listUserGroupsEndpoint(svc)),
		decodeListUserGroupRequest,
		encodeResponse,
		opts...,
	))

	mux.Post("/password/reset-request", kithttp.NewServer(
		kitot.TraceServer(tracer, "res-req")(passwordResetRequestEndpoint(svc)),
		decodePasswordResetRequest,
		encodeResponse,
		opts...,
	))

	mux.Put("/password/reset", kithttp.NewServer(
		kitot.TraceServer(tracer, "reset")(passwordResetEndpoint(svc)),
		decodePasswordReset,
		encodeResponse,
		opts...,
	))

	mux.Patch("/password", kithttp.NewServer(
		kitot.TraceServer(tracer, "reset")(passwordChangeEndpoint(svc)),
		decodePasswordChange,
		encodeResponse,
		opts...,
	))

	mux.Post("/groups", kithttp.NewServer(
		kitot.TraceServer(tracer, "add_group")(createGroupEndpoint(svc)),
		decodeGroupCreate,
		encodeResponse,
		opts...,
	))

	mux.Get("/groups", kithttp.NewServer(
		kitot.TraceServer(tracer, "groups")(listGroupsEndpoint(svc)),
		decodeListUserGroupRequest,
		encodeResponse,
		opts...,
	))

	mux.Delete("/groups/:groupID", kithttp.NewServer(
		kitot.TraceServer(tracer, "delete_group")(deleteGroupEndpoint(svc)),
		decodeGroupRequest,
		encodeResponse,
		opts...,
	))

	mux.Put("/groups/:groupID/users/:userID", kithttp.NewServer(
		kitot.TraceServer(tracer, "assign_user_to_group")(assignUserToGroup(svc)),
		decodeUserGroupRequest,
		encodeResponse,
		opts...,
	))

	mux.Delete("/groups/:groupID/users/:userID", kithttp.NewServer(
		kitot.TraceServer(tracer, "remove_user_from_group")(removeUserFromGroup(svc)),
		decodeUserGroupRequest,
		encodeResponse,
		opts...,
	))

	mux.Get("/groups/:groupID/users", kithttp.NewServer(
		kitot.TraceServer(tracer, "members")(listUsersForGroupEndpoint(svc)),
		decodeListUserGroupRequest,
		encodeResponse,
		opts...,
	))

	mux.Patch("/groups/:groupID", kithttp.NewServer(
		kitot.TraceServer(tracer, "update_group")(updateGroupEndpoint(svc)),
		decodeGroupCreate,
		encodeResponse,
		opts...,
	))

	mux.Get("/groups/:groupID/groups", kithttp.NewServer(
		kitot.TraceServer(tracer, "list_children_groups")(listGroupsEndpoint(svc)),
		decodeListUserGroupRequest,
		encodeResponse,
		opts...,
	))

	mux.Get("/groups/:groupID", kithttp.NewServer(
		kitot.TraceServer(tracer, "group")(viewGroupEndpoint(svc)),
		decodeGroupRequest,
		encodeResponse,
		opts...,
	))

	mux.Post("/tokens", kithttp.NewServer(
		kitot.TraceServer(tracer, "login")(loginEndpoint(svc)),
		decodeCredentials,
		encodeResponse,
		opts...,
	))

	mux.GetFunc("/version", mainflux.Version("users"))
	mux.Handle("/metrics", promhttp.Handler())

	return mux
}

func decodeViewUser(_ context.Context, r *http.Request) (interface{}, error) {
	req := viewUserReq{
		token: r.Header.Get("Authorization"),
	}
	return req, nil
}

func decodeUpdateUser(_ context.Context, r *http.Request) (interface{}, error) {
	var req updateUserReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, errors.Wrap(users.ErrMalformedEntity, err)
	}

	req.token = r.Header.Get("Authorization")
	return req, nil
}

func decodeCredentials(_ context.Context, r *http.Request) (interface{}, error) {
	if !strings.Contains(r.Header.Get("Content-Type"), contentType) {
		return nil, ErrUnsupportedContentType
	}

	var user users.User
	if err := json.NewDecoder(r.Body).Decode(&user); err != nil {
		return nil, errors.Wrap(users.ErrMalformedEntity, err)
	}

	return userReq{user}, nil
}

func decodePasswordResetRequest(_ context.Context, r *http.Request) (interface{}, error) {
	if !strings.Contains(r.Header.Get("Content-Type"), contentType) {
		return nil, ErrUnsupportedContentType
	}

	var req passwResetReq

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, errors.Wrap(ErrFailedDecode, err)
	}

	req.Host = r.Header.Get("Referer")
	return req, nil
}

func decodePasswordReset(_ context.Context, r *http.Request) (interface{}, error) {
	if !strings.Contains(r.Header.Get("Content-Type"), contentType) {
		return nil, ErrUnsupportedContentType
	}

	var req resetTokenReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, errors.Wrap(ErrFailedDecode, err)
	}

	return req, nil
}

func decodePasswordChange(_ context.Context, r *http.Request) (interface{}, error) {
	if !strings.Contains(r.Header.Get("Content-Type"), contentType) {
		return nil, ErrUnsupportedContentType
	}

	var req passwChangeReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, errors.Wrap(ErrFailedDecode, err)
	}

	req.Token = r.Header.Get("Authorization")

	return req, nil
}

// Group related methods
func decodeGroupCreate(_ context.Context, r *http.Request) (interface{}, error) {
	if !strings.Contains(r.Header.Get("Content-Type"), contentType) {
		return nil, ErrUnsupportedContentType
	}

	var req createGroupReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, errors.Wrap(ErrFailedDecode, err)
	}

	req.token = r.Header.Get("Authorization")

	return req, nil
}

func decodeGroupRequest(_ context.Context, r *http.Request) (interface{}, error) {
	if !strings.Contains(r.Header.Get("Content-Type"), contentType) {
		return nil, ErrUnsupportedContentType
	}

	req := groupReq{
		token:   r.Header.Get("Authorization"),
		groupID: bone.GetValue(r, "groupID"),
		name:    bone.GetValue(r, "name"),
	}

	return req, nil
}

func decodeListUserGroupRequest(_ context.Context, r *http.Request) (interface{}, error) {
	if !strings.Contains(r.Header.Get("Content-Type"), contentType) {
		return nil, ErrUnsupportedContentType
	}
	o, err := readUintQuery(r, offsetKey, defOffset)
	if err != nil {
		return nil, err
	}

	l, err := readUintQuery(r, limitKey, defLimit)
	if err != nil {
		return nil, err
	}

	n, err := readStringQuery(r, nameKey)
	if err != nil {
		return nil, err
	}

	m, err := readMetadataQuery(r, metadataKey)
	if err != nil {
		return nil, err
	}

	groupID := bone.GetValue(r, "groupID")
	userID := bone.GetValue(r, "userID")

	req := listUserGroupReq{
		token:    r.Header.Get("Authorization"),
		groupID:  groupID,
		userID:   userID,
		offset:   o,
		limit:    l,
		name:     n,
		metadata: m,
	}
	return req, nil
}

func decodeUserGroupRequest(_ context.Context, r *http.Request) (interface{}, error) {
	if !strings.Contains(r.Header.Get("Content-Type"), contentType) {
		return nil, ErrUnsupportedContentType
	}

	req := userGroupReq{
		token:   r.Header.Get("Authorization"),
		groupID: bone.GetValue(r, "groupID"),
		userID:  bone.GetValue(r, "userID"),
	}
	return req, nil
}

func encodeResponse(_ context.Context, w http.ResponseWriter, response interface{}) error {
	if ar, ok := response.(mainflux.Response); ok {
		for k, v := range ar.Headers() {
			w.Header().Set(k, v)
		}
		w.Header().Set("Content-Type", contentType)
		w.WriteHeader(ar.Code())

		if ar.Empty() {
			return nil
		}
	}

	return json.NewEncoder(w).Encode(response)
}

func encodeError(_ context.Context, err error, w http.ResponseWriter) {
	switch errorVal := err.(type) {
	case errors.Error:
		w.Header().Set("Content-Type", contentType)
		switch {
		case errors.Contains(errorVal, users.ErrMalformedEntity):
			w.WriteHeader(http.StatusBadRequest)
		case errors.Contains(errorVal, users.ErrUnauthorizedAccess):
			w.WriteHeader(http.StatusForbidden)
		case errors.Contains(errorVal, users.ErrConflict):
			w.WriteHeader(http.StatusConflict)
		case errors.Contains(errorVal, users.ErrGroupConflict):
			w.WriteHeader(http.StatusConflict)
		case errors.Contains(errorVal, ErrUnsupportedContentType):
			w.WriteHeader(http.StatusUnsupportedMediaType)
		case errors.Contains(errorVal, ErrFailedDecode):
			w.WriteHeader(http.StatusBadRequest)
		case errors.Contains(errorVal, io.ErrUnexpectedEOF):
			w.WriteHeader(http.StatusBadRequest)
		case errors.Contains(errorVal, io.EOF):
			w.WriteHeader(http.StatusBadRequest)
		case errors.Contains(errorVal, users.ErrUserNotFound):
			w.WriteHeader(http.StatusBadRequest)
		case errors.Contains(errorVal, users.ErrRecoveryToken):
			w.WriteHeader(http.StatusNotFound)
		default:
			w.WriteHeader(http.StatusInternalServerError)
		}
		if errorVal.Msg() != "" {
			if err := json.NewEncoder(w).Encode(errorRes{Err: errorVal.Msg()}); err != nil {
				w.WriteHeader(http.StatusInternalServerError)
			}
		}
	default:
		w.WriteHeader(http.StatusInternalServerError)
	}
}

func readUintQuery(r *http.Request, key string, def uint64) (uint64, error) {
	vals := bone.GetQuery(r, key)
	if len(vals) > 1 {
		return 0, errInvalidQueryParams
	}

	if len(vals) == 0 {
		return def, nil
	}

	strval := vals[0]
	val, err := strconv.ParseUint(strval, 10, 64)
	if err != nil {
		return 0, errInvalidQueryParams
	}

	return val, nil
}

func readStringQuery(r *http.Request, key string) (string, error) {
	vals := bone.GetQuery(r, key)
	if len(vals) > 1 {
		return "", errInvalidQueryParams
	}

	if len(vals) == 0 {
		return "", nil
	}

	return vals[0], nil
}

func readMetadataQuery(r *http.Request, key string) (map[string]interface{}, error) {
	vals := bone.GetQuery(r, key)
	if len(vals) > 1 {
		return nil, errInvalidQueryParams
	}

	if len(vals) == 0 {
		return nil, nil
	}

	m := make(map[string]interface{})
	err := json.Unmarshal([]byte(vals[0]), &m)
	if err != nil {
		return nil, errors.Wrap(errInvalidQueryParams, err)
	}

	return m, nil
}
