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
	"github.com/absmach/supermq/channels"
	"github.com/absmach/supermq/internal/nullable"
	"github.com/absmach/supermq/pkg/errors"
	"github.com/go-chi/chi/v5"
)

func decodeViewChannel(_ context.Context, r *http.Request) (any, error) {
	roles, err := apiutil.ReadBoolQuery(r, api.RolesKey, false)
	if err != nil {
		return viewChannelReq{}, errors.Wrap(apiutil.ErrValidation, err)
	}

	req := viewChannelReq{
		id:    chi.URLParam(r, "channelID"),
		roles: roles,
	}

	return req, nil
}

func decodeCreateChannelReq(_ context.Context, r *http.Request) (any, error) {
	if !strings.Contains(r.Header.Get("Content-Type"), api.ContentType) {
		return nil, errors.Wrap(apiutil.ErrValidation, apiutil.ErrUnsupportedContentType)
	}

	req := createChannelReq{}
	if err := json.NewDecoder(r.Body).Decode(&req.Channel); err != nil {
		return nil, errors.Wrap(apiutil.ErrMalformedRequestBody, err)
	}

	return req, nil
}

func decodeCreateChannelsReq(_ context.Context, r *http.Request) (any, error) {
	if !strings.Contains(r.Header.Get("Content-Type"), api.ContentType) {
		return nil, errors.Wrap(apiutil.ErrValidation, apiutil.ErrUnsupportedContentType)
	}

	req := createChannelsReq{}
	if err := json.NewDecoder(r.Body).Decode(&req.Channels); err != nil {
		return nil, errors.Wrap(apiutil.ErrMalformedRequestBody, err)
	}

	return req, nil
}

func decodeListChannels(_ context.Context, r *http.Request) (any, error) {
	name, err := apiutil.ReadStringQuery(r, api.NameKey, "")
	if err != nil {
		return listChannelsReq{}, errors.Wrap(apiutil.ErrValidation, err)
	}

	tags, err := apiutil.ReadStringQuery(r, api.TagsKey, "")
	if err != nil {
		return nil, errors.Wrap(apiutil.ErrValidation, err)
	}
	var tq channels.TagsQuery
	if tags != "" {
		tq = channels.ToTagsQuery(tags)
	}

	s, err := apiutil.ReadStringQuery(r, api.StatusKey, api.DefGroupStatus)
	if err != nil {
		return listChannelsReq{}, errors.Wrap(apiutil.ErrValidation, err)
	}
	status, err := channels.ToStatus(s)
	if err != nil {
		return listChannelsReq{}, errors.Wrap(apiutil.ErrValidation, err)
	}

	meta, err := apiutil.ReadMetadataQuery(r, api.MetadataKey, nil)
	if err != nil {
		return listChannelsReq{}, errors.Wrap(apiutil.ErrValidation, err)
	}

	offset, err := apiutil.ReadNumQuery[uint64](r, api.OffsetKey, api.DefOffset)
	if err != nil {
		return listChannelsReq{}, errors.Wrap(apiutil.ErrValidation, err)
	}
	limit, err := apiutil.ReadNumQuery[uint64](r, api.LimitKey, api.DefLimit)
	if err != nil {
		return listChannelsReq{}, errors.Wrap(apiutil.ErrValidation, err)
	}

	dir, err := apiutil.ReadStringQuery(r, api.DirKey, api.DefDir)
	if err != nil {
		return listChannelsReq{}, errors.Wrap(apiutil.ErrValidation, err)
	}

	order, err := apiutil.ReadStringQuery(r, api.OrderKey, api.DefOrder)
	if err != nil {
		return listChannelsReq{}, errors.Wrap(apiutil.ErrValidation, err)
	}

	allActions, err := apiutil.ReadStringQuery(r, api.ActionsKey, "")
	if err != nil {
		return listChannelsReq{}, errors.Wrap(apiutil.ErrValidation, err)
	}

	actions := []string{}

	allActions = strings.TrimSpace(allActions)
	if allActions != "" {
		actions = strings.Split(allActions, ",")
	}
	roleID, err := apiutil.ReadStringQuery(r, api.RoleIDKey, "")
	if err != nil {
		return listChannelsReq{}, errors.Wrap(apiutil.ErrValidation, err)
	}

	roleName, err := apiutil.ReadStringQuery(r, api.RoleNameKey, "")
	if err != nil {
		return listChannelsReq{}, errors.Wrap(apiutil.ErrValidation, err)
	}

	accessType, err := apiutil.ReadStringQuery(r, api.AccessTypeKey, "")
	if err != nil {
		return listChannelsReq{}, errors.Wrap(apiutil.ErrValidation, err)
	}

	userID, err := apiutil.ReadStringQuery(r, api.UserKey, "")
	if err != nil {
		return listChannelsReq{}, errors.Wrap(apiutil.ErrValidation, err)
	}
	groupID, err := nullable.Parse(r.URL.Query(), api.GroupKey, nullable.ParseString)
	if err != nil {
		return listChannelsReq{}, errors.Wrap(apiutil.ErrValidation, err)
	}

	clientID, err := apiutil.ReadStringQuery(r, api.ClientKey, "")
	if err != nil {
		return listChannelsReq{}, errors.Wrap(apiutil.ErrValidation, err)
	}

	id, err := apiutil.ReadStringQuery(r, api.IDOrder, "")
	if err != nil {
		return listChannelsReq{}, errors.Wrap(apiutil.ErrValidation, err)
	}
	ot, err := apiutil.ReadBoolQuery(r, api.OnlyTotal, false)
	if err != nil {
		return listChannelsReq{}, errors.Wrap(apiutil.ErrValidation, err)
	}
	connectionType, err := apiutil.ReadStringQuery(r, api.ConnTypeKey, "")
	if err != nil {
		return listChannelsReq{}, errors.Wrap(apiutil.ErrValidation, err)
	}

	cfrom, err := apiutil.ReadStringQuery(r, "created_from", "")
	if err != nil {
		return listChannelsReq{}, errors.Wrap(apiutil.ErrValidation, err)
	}
	cto, err := apiutil.ReadStringQuery(r, "created_to", "")
	if err != nil {
		return listChannelsReq{}, errors.Wrap(apiutil.ErrValidation, err)
	}

	var createdFrom, createdTo time.Time
	if cfrom != "" {
		if createdFrom, err = time.Parse(time.RFC3339, cfrom); err != nil {
			return listChannelsReq{}, errors.Wrap(apiutil.ErrInvalidQueryParams, err)
		}
	}
	if cto != "" {
		if createdTo, err = time.Parse(time.RFC3339, cto); err != nil {
			return listChannelsReq{}, errors.Wrap(apiutil.ErrInvalidQueryParams, err)
		}
	}

	req := listChannelsReq{
		Page: channels.Page{
			Name:           name,
			Tags:           tq,
			Status:         status,
			Metadata:       meta,
			RoleName:       roleName,
			RoleID:         roleID,
			Actions:        actions,
			AccessType:     accessType,
			Order:          order,
			Dir:            dir,
			Offset:         offset,
			Limit:          limit,
			Group:          groupID,
			Client:         clientID,
			ConnectionType: connectionType,
			ID:             id,
			OnlyTotal:      ot,
			CreatedFrom:    createdFrom,
			CreatedTo:      createdTo,
		},
		userID: userID,
	}
	return req, nil
}

func decodeUpdateChannel(_ context.Context, r *http.Request) (any, error) {
	if !strings.Contains(r.Header.Get("Content-Type"), api.ContentType) {
		return nil, errors.Wrap(apiutil.ErrValidation, apiutil.ErrUnsupportedContentType)
	}

	req := updateChannelReq{
		id: chi.URLParam(r, "channelID"),
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, errors.Wrap(apiutil.ErrMalformedRequestBody, err)
	}

	return req, nil
}

func decodeUpdateChannelTags(_ context.Context, r *http.Request) (any, error) {
	if !strings.Contains(r.Header.Get("Content-Type"), api.ContentType) {
		return nil, errors.Wrap(apiutil.ErrValidation, apiutil.ErrUnsupportedContentType)
	}

	req := updateChannelTagsReq{
		id: chi.URLParam(r, "channelID"),
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, errors.Wrap(apiutil.ErrMalformedRequestBody, err)
	}

	return req, nil
}

func decodeSetChannelParentGroupStatus(_ context.Context, r *http.Request) (any, error) {
	if !strings.Contains(r.Header.Get("Content-Type"), api.ContentType) {
		return nil, errors.Wrap(apiutil.ErrValidation, apiutil.ErrUnsupportedContentType)
	}

	req := setChannelParentGroupReq{
		id: chi.URLParam(r, "channelID"),
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, errors.Wrap(apiutil.ErrMalformedRequestBody, err)
	}
	return req, nil
}

func decodeRemoveChannelParentGroupStatus(_ context.Context, r *http.Request) (any, error) {
	req := removeChannelParentGroupReq{
		id: chi.URLParam(r, "channelID"),
	}

	return req, nil
}

func decodeChangeChannelStatus(_ context.Context, r *http.Request) (any, error) {
	req := changeChannelStatusReq{
		id: chi.URLParam(r, "channelID"),
	}

	return req, nil
}

func decodeDeleteChannelReq(_ context.Context, r *http.Request) (any, error) {
	req := deleteChannelReq{
		id: chi.URLParam(r, "channelID"),
	}
	return req, nil
}

func decodeConnectChannelClientRequest(_ context.Context, r *http.Request) (any, error) {
	if !strings.Contains(r.Header.Get("Content-Type"), api.ContentType) {
		return nil, errors.Wrap(apiutil.ErrValidation, apiutil.ErrUnsupportedContentType)
	}
	req := connectChannelClientsRequest{
		channelID: chi.URLParam(r, "channelID"),
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, errors.Wrap(apiutil.ErrMalformedRequestBody, err)
	}

	return req, nil
}

func decodeDisconnectChannelClientsRequest(_ context.Context, r *http.Request) (any, error) {
	if !strings.Contains(r.Header.Get("Content-Type"), api.ContentType) {
		return nil, errors.Wrap(apiutil.ErrValidation, apiutil.ErrUnsupportedContentType)
	}
	req := disconnectChannelClientsRequest{
		channelID: chi.URLParam(r, "channelID"),
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, errors.Wrap(apiutil.ErrMalformedRequestBody, err)
	}

	return req, nil
}

func decodeConnectRequest(_ context.Context, r *http.Request) (any, error) {
	if !strings.Contains(r.Header.Get("Content-Type"), api.ContentType) {
		return nil, errors.Wrap(apiutil.ErrValidation, apiutil.ErrUnsupportedContentType)
	}

	req := connectRequest{}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, errors.Wrap(apiutil.ErrMalformedRequestBody, err)
	}

	return req, nil
}

func decodeDisconnectRequest(_ context.Context, r *http.Request) (any, error) {
	if !strings.Contains(r.Header.Get("Content-Type"), api.ContentType) {
		return nil, errors.Wrap(apiutil.ErrValidation, apiutil.ErrUnsupportedContentType)
	}

	req := disconnectRequest{}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, errors.Wrap(apiutil.ErrMalformedRequestBody, err)
	}

	return req, nil
}
