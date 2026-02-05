// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package http

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	api "github.com/absmach/supermq/api/http"
	apiutil "github.com/absmach/supermq/api/http/util"
	"github.com/absmach/supermq/domains"
	"github.com/absmach/supermq/pkg/errors"
	"github.com/go-chi/chi/v5"
)

const (
	inviteeUserIDKey = "invitee_user_id"
	domainIDKey      = "domain_id"
	invitedByKey     = "invited_by"
	roleIDKey        = "role_id"
	stateKey         = "state"
)

func decodeCreateDomainRequest(_ context.Context, r *http.Request) (any, error) {
	if !strings.Contains(r.Header.Get("Content-Type"), api.ContentType) {
		return nil, errors.Wrap(apiutil.ErrValidation, apiutil.ErrUnsupportedContentType)
	}
	req := createDomainReq{}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, errors.Wrap(apiutil.ErrMalformedRequestBody, err)
	}

	return req, nil
}

func decodeRetrieveDomainRequest(_ context.Context, r *http.Request) (any, error) {
	roles, err := apiutil.ReadBoolQuery(r, api.RolesKey, false)
	if err != nil {
		return nil, errors.Wrap(apiutil.ErrValidation, err)
	}
	req := retrieveDomainRequest{
		domainID: chi.URLParam(r, "domainID"),
		roles:    roles,
	}
	return req, nil
}

func decodeUpdateDomainRequest(_ context.Context, r *http.Request) (any, error) {
	if !strings.Contains(r.Header.Get("Content-Type"), api.ContentType) {
		return nil, errors.Wrap(apiutil.ErrValidation, apiutil.ErrUnsupportedContentType)
	}

	req := updateDomainReq{
		domainID: chi.URLParam(r, "domainID"),
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, errors.Wrap(apiutil.ErrMalformedRequestBody, err)
	}

	return req, nil
}

func decodeListDomainRequest(ctx context.Context, r *http.Request) (any, error) {
	page, err := decodePageRequest(ctx, r)
	if err != nil {
		return nil, err
	}
	req := listDomainsReq{
		page,
	}

	return req, nil
}

func decodeEnableDomainRequest(_ context.Context, r *http.Request) (any, error) {
	req := enableDomainReq{
		domainID: chi.URLParam(r, "domainID"),
	}
	return req, nil
}

func decodeDisableDomainRequest(_ context.Context, r *http.Request) (any, error) {
	req := disableDomainReq{
		domainID: chi.URLParam(r, "domainID"),
	}
	return req, nil
}

func decodeFreezeDomainRequest(_ context.Context, r *http.Request) (any, error) {
	req := freezeDomainReq{
		domainID: chi.URLParam(r, "domainID"),
	}
	return req, nil
}

func decodePageRequest(_ context.Context, r *http.Request) (domains.Page, error) {
	s, err := apiutil.ReadStringQuery(r, api.StatusKey, api.DefClientStatus)
	if err != nil {
		return domains.Page{}, errors.Wrap(apiutil.ErrValidation, err)
	}
	st, err := domains.ToStatus(s)
	if err != nil {
		return domains.Page{}, errors.Wrap(apiutil.ErrValidation, err)
	}
	o, err := apiutil.ReadNumQuery[uint64](r, api.OffsetKey, api.DefOffset)
	if err != nil {
		return domains.Page{}, errors.Wrap(apiutil.ErrValidation, err)
	}
	or, err := apiutil.ReadStringQuery(r, api.OrderKey, api.DefOrder)
	if err != nil {
		return domains.Page{}, errors.Wrap(apiutil.ErrValidation, err)
	}
	dir, err := apiutil.ReadStringQuery(r, api.DirKey, api.DefDir)
	if err != nil {
		return domains.Page{}, errors.Wrap(apiutil.ErrValidation, err)
	}
	l, err := apiutil.ReadNumQuery[uint64](r, api.LimitKey, api.DefLimit)
	if err != nil {
		return domains.Page{}, errors.Wrap(apiutil.ErrValidation, err)
	}
	m, err := apiutil.ReadMetadataQuery(r, api.MetadataKey, nil)
	if err != nil {
		return domains.Page{}, errors.Wrap(apiutil.ErrValidation, err)
	}
	n, err := apiutil.ReadStringQuery(r, api.NameKey, "")
	if err != nil {
		return domains.Page{}, errors.Wrap(apiutil.ErrValidation, err)
	}
	t, err := apiutil.ReadStringQuery(r, api.TagsKey, "")
	if err != nil {
		return domains.Page{}, errors.Wrap(apiutil.ErrValidation, err)
	}
	var tq domains.TagsQuery
	if t != "" {
		tq = domains.ToTagsQuery(t)
	}

	allActions, err := apiutil.ReadStringQuery(r, api.ActionsKey, "")
	if err != nil {
		return domains.Page{}, errors.Wrap(apiutil.ErrValidation, err)
	}

	var actions []string

	allActions = strings.TrimSpace(allActions)
	if allActions != "" {
		actions = strings.Split(allActions, ",")
	}
	roleID, err := apiutil.ReadStringQuery(r, api.RoleIDKey, "")
	if err != nil {
		return domains.Page{}, errors.Wrap(apiutil.ErrValidation, err)
	}

	roleName, err := apiutil.ReadStringQuery(r, api.RoleNameKey, "")
	if err != nil {
		return domains.Page{}, errors.Wrap(apiutil.ErrValidation, err)
	}

	id, err := apiutil.ReadStringQuery(r, api.IDOrder, "")
	if err != nil {
		return domains.Page{}, errors.Wrap(apiutil.ErrValidation, err)
	}

	ot, err := apiutil.ReadBoolQuery(r, api.OnlyTotal, false)
	if err != nil {
		return domains.Page{}, errors.Wrap(apiutil.ErrValidation, err)
	}

	cfrom, err := apiutil.ReadStringQuery(r, "created_from", "")
	if err != nil {
		return domains.Page{}, errors.Wrap(apiutil.ErrValidation, err)
	}
	cto, err := apiutil.ReadStringQuery(r, "created_to", "")
	if err != nil {
		return domains.Page{}, errors.Wrap(apiutil.ErrValidation, err)
	}

	var createdFrom, createdTo time.Time
	if cfrom != "" {
		if createdFrom, err = time.Parse(time.RFC3339, cfrom); err != nil {
			return domains.Page{}, errors.Wrap(apiutil.ErrInvalidQueryParams, err)
		}
	}
	if cto != "" {
		if createdTo, err = time.Parse(time.RFC3339, cto); err != nil {
			return domains.Page{}, errors.Wrap(apiutil.ErrInvalidQueryParams, err)
		}
	}

	return domains.Page{
		Offset:      o,
		Order:       or,
		Dir:         dir,
		Limit:       l,
		Name:        n,
		Metadata:    m,
		Tags:        tq,
		RoleID:      roleID,
		RoleName:    roleName,
		Actions:     actions,
		Status:      st,
		ID:          id,
		OnlyTotal:   ot,
		CreatedFrom: createdFrom,
		CreatedTo:   createdTo,
	}, nil
}

func decodeSendInvitationReq(_ context.Context, r *http.Request) (any, error) {
	if !strings.Contains(r.Header.Get("Content-Type"), api.ContentType) {
		return nil, errors.Wrap(apiutil.ErrValidation, apiutil.ErrUnsupportedContentType)
	}

	var req sendInvitationReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, errors.Wrap(apiutil.ErrMalformedRequestBody, err)
	}

	return req, nil
}

func decodeListInvitationsReq(_ context.Context, r *http.Request) (any, error) {
	offset, err := apiutil.ReadNumQuery[uint64](r, api.OffsetKey, api.DefOffset)
	if err != nil {
		return nil, errors.Wrap(apiutil.ErrValidation, err)
	}
	limit, err := apiutil.ReadNumQuery[uint64](r, api.LimitKey, api.DefLimit)
	if err != nil {
		return nil, errors.Wrap(apiutil.ErrValidation, err)
	}
	inviteeUserID, err := apiutil.ReadStringQuery(r, inviteeUserIDKey, "")
	if err != nil {
		return nil, errors.Wrap(apiutil.ErrValidation, err)
	}
	invitedBy, err := apiutil.ReadStringQuery(r, invitedByKey, "")
	if err != nil {
		return nil, errors.Wrap(apiutil.ErrValidation, err)
	}
	roleID, err := apiutil.ReadStringQuery(r, roleIDKey, "")
	if err != nil {
		return nil, errors.Wrap(apiutil.ErrValidation, err)
	}
	domainID, err := apiutil.ReadStringQuery(r, domainIDKey, "")
	if err != nil {
		return nil, errors.Wrap(apiutil.ErrValidation, err)
	}
	st, err := apiutil.ReadStringQuery(r, stateKey, domains.AllState.String())
	if err != nil {
		return nil, errors.Wrap(apiutil.ErrValidation, err)
	}
	state, err := domains.ToState(st)
	if err != nil {
		return nil, errors.Wrap(apiutil.ErrValidation, err)
	}
	ot, err := apiutil.ReadBoolQuery(r, api.OnlyTotal, false)
	if err != nil {
		return nil, errors.Wrap(apiutil.ErrValidation, err)
	}
	req := listInvitationsReq{
		InvitationPageMeta: domains.InvitationPageMeta{
			Offset:        offset,
			Limit:         limit,
			InvitedBy:     invitedBy,
			InviteeUserID: inviteeUserID,
			RoleID:        roleID,
			DomainID:      domainID,
			State:         state,
			OnlyTotal:     ot,
		},
	}

	return req, nil
}

func decodeAcceptInvitationReq(_ context.Context, r *http.Request) (any, error) {
	if !strings.Contains(r.Header.Get("Content-Type"), api.ContentType) {
		return nil, errors.Wrap(apiutil.ErrValidation, apiutil.ErrUnsupportedContentType)
	}

	var req acceptInvitationReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, errors.Wrap(apiutil.ErrMalformedRequestBody, err)
	}

	return req, nil
}

func decodeDeleteInvitationReq(_ context.Context, r *http.Request) (any, error) {
	req := deleteInvitationReq{
		domainID: chi.URLParam(r, "domainID"),
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, errors.Wrap(apiutil.ErrMalformedRequestBody, err)
	}

	return req, nil
}
