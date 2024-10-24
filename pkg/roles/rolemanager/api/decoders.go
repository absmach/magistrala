// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package http

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/absmach/magistrala/internal/api"
	"github.com/absmach/magistrala/pkg/apiutil"
	"github.com/absmach/magistrala/pkg/errors"
	"github.com/go-chi/chi/v5"
)

type Decoder struct {
	entityIDTemplate string
}

func NewDecoder(entityIDTemplate string) Decoder {
	return Decoder{entityIDTemplate}
}
func (d Decoder) DecodeCreateRole(_ context.Context, r *http.Request) (interface{}, error) {
	if !strings.Contains(r.Header.Get("Content-Type"), api.ContentType) {
		return nil, errors.Wrap(apiutil.ErrValidation, apiutil.ErrUnsupportedContentType)
	}
	req := createRoleReq{
		token:    apiutil.ExtractBearerToken(r),
		entityID: chi.URLParam(r, d.entityIDTemplate),
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, errors.Wrap(apiutil.ErrValidation, errors.Wrap(errors.ErrMalformedEntity, err))
	}
	return req, nil
}

func (d Decoder) DecodeListRoles(_ context.Context, r *http.Request) (interface{}, error) {
	o, err := apiutil.ReadNumQuery[uint64](r, api.OffsetKey, api.DefOffset)
	if err != nil {
		return nil, errors.Wrap(apiutil.ErrValidation, err)
	}
	l, err := apiutil.ReadNumQuery[uint64](r, api.LimitKey, api.DefLimit)
	if err != nil {
		return nil, errors.Wrap(apiutil.ErrValidation, err)
	}
	req := listRolesReq{
		token:    apiutil.ExtractBearerToken(r),
		entityID: chi.URLParam(r, d.entityIDTemplate),
		limit:    l,
		offset:   o,
	}
	return req, nil
}

func (d Decoder) DecodeViewRole(_ context.Context, r *http.Request) (interface{}, error) {
	req := viewRoleReq{
		token:    apiutil.ExtractBearerToken(r),
		entityID: chi.URLParam(r, d.entityIDTemplate),
		roleName: chi.URLParam(r, "roleName"),
	}
	return req, nil
}

func (d Decoder) DecodeUpdateRole(_ context.Context, r *http.Request) (interface{}, error) {
	if !strings.Contains(r.Header.Get("Content-Type"), api.ContentType) {
		return nil, errors.Wrap(apiutil.ErrValidation, apiutil.ErrUnsupportedContentType)
	}
	req := updateRoleReq{
		token:    apiutil.ExtractBearerToken(r),
		entityID: chi.URLParam(r, d.entityIDTemplate),
		roleName: chi.URLParam(r, "roleName"),
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, errors.Wrap(apiutil.ErrValidation, errors.Wrap(errors.ErrMalformedEntity, err))
	}
	return req, nil
}

func (d Decoder) DecodeDeleteRole(_ context.Context, r *http.Request) (interface{}, error) {
	req := deleteRoleReq{
		token:    apiutil.ExtractBearerToken(r),
		entityID: chi.URLParam(r, d.entityIDTemplate),
		roleName: chi.URLParam(r, "roleName"),
	}
	return req, nil
}
func (d Decoder) DecodeListAvailableActions(_ context.Context, r *http.Request) (interface{}, error) {
	req := listAvailableActionsReq{
		token: apiutil.ExtractBearerToken(r),
	}
	return req, nil
}

func (d Decoder) DecodeAddRoleActions(_ context.Context, r *http.Request) (interface{}, error) {
	if !strings.Contains(r.Header.Get("Content-Type"), api.ContentType) {
		return nil, errors.Wrap(apiutil.ErrValidation, apiutil.ErrUnsupportedContentType)
	}
	req := addRoleActionsReq{
		token:    apiutil.ExtractBearerToken(r),
		entityID: chi.URLParam(r, d.entityIDTemplate),
		roleName: chi.URLParam(r, "roleName"),
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, errors.Wrap(apiutil.ErrValidation, errors.Wrap(errors.ErrMalformedEntity, err))
	}
	return req, nil
}

func (d Decoder) DecodeListRoleActions(_ context.Context, r *http.Request) (interface{}, error) {
	req := listRoleActionsReq{
		token:    apiutil.ExtractBearerToken(r),
		entityID: chi.URLParam(r, d.entityIDTemplate),
		roleName: chi.URLParam(r, "roleName"),
	}
	return req, nil
}

func (d Decoder) DecodeDeleteRoleActions(_ context.Context, r *http.Request) (interface{}, error) {
	if !strings.Contains(r.Header.Get("Content-Type"), api.ContentType) {
		return nil, errors.Wrap(apiutil.ErrValidation, apiutil.ErrUnsupportedContentType)
	}
	req := deleteRoleActionsReq{
		token:    apiutil.ExtractBearerToken(r),
		entityID: chi.URLParam(r, d.entityIDTemplate),
		roleName: chi.URLParam(r, "roleName"),
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, errors.Wrap(apiutil.ErrValidation, errors.Wrap(errors.ErrMalformedEntity, err))
	}
	return req, nil
}

func (d Decoder) DecodeDeleteAllRoleActions(_ context.Context, r *http.Request) (interface{}, error) {
	req := deleteAllRoleActionsReq{
		token:    apiutil.ExtractBearerToken(r),
		entityID: chi.URLParam(r, d.entityIDTemplate),
		roleName: chi.URLParam(r, "roleName"),
	}
	return req, nil
}

func (d Decoder) DecodeAddRoleMembers(_ context.Context, r *http.Request) (interface{}, error) {
	if !strings.Contains(r.Header.Get("Content-Type"), api.ContentType) {
		return nil, errors.Wrap(apiutil.ErrValidation, apiutil.ErrUnsupportedContentType)
	}
	req := addRoleMembersReq{
		token:    apiutil.ExtractBearerToken(r),
		entityID: chi.URLParam(r, d.entityIDTemplate),
		roleName: chi.URLParam(r, "roleName"),
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, errors.Wrap(apiutil.ErrValidation, errors.Wrap(errors.ErrMalformedEntity, err))
	}
	return req, nil
}

func (d Decoder) DecodeListRoleMembers(_ context.Context, r *http.Request) (interface{}, error) {
	o, err := apiutil.ReadNumQuery[uint64](r, api.OffsetKey, api.DefOffset)
	if err != nil {
		return nil, errors.Wrap(apiutil.ErrValidation, err)
	}
	l, err := apiutil.ReadNumQuery[uint64](r, api.LimitKey, api.DefLimit)
	if err != nil {
		return nil, errors.Wrap(apiutil.ErrValidation, err)
	}
	req := listRoleMembersReq{
		token:    apiutil.ExtractBearerToken(r),
		entityID: chi.URLParam(r, d.entityIDTemplate),
		roleName: chi.URLParam(r, "roleName"),
		limit:    l,
		offset:   o,
	}
	return req, nil
}

func (d Decoder) DecodeDeleteRoleMembers(_ context.Context, r *http.Request) (interface{}, error) {
	if !strings.Contains(r.Header.Get("Content-Type"), api.ContentType) {
		return nil, errors.Wrap(apiutil.ErrValidation, apiutil.ErrUnsupportedContentType)
	}
	req := deleteRoleMembersReq{
		token:    apiutil.ExtractBearerToken(r),
		entityID: chi.URLParam(r, d.entityIDTemplate),
		roleName: chi.URLParam(r, "roleName"),
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, errors.Wrap(apiutil.ErrValidation, errors.Wrap(errors.ErrMalformedEntity, err))
	}
	return req, nil
}

func (d Decoder) DecodeDeleteAllRoleMembers(_ context.Context, r *http.Request) (interface{}, error) {
	req := deleteAllRoleMembersReq{
		token:    apiutil.ExtractBearerToken(r),
		entityID: chi.URLParam(r, d.entityIDTemplate),
		roleName: chi.URLParam(r, "roleName"),
	}
	return req, nil
}
