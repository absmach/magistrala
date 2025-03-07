// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package http

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	api "github.com/absmach/supermq/api/http"
	apiutil "github.com/absmach/supermq/api/http/util"
	"github.com/absmach/supermq/clients"
	"github.com/absmach/supermq/pkg/errors"
	"github.com/go-chi/chi/v5"
)

func decodeViewChannel(_ context.Context, r *http.Request) (interface{}, error) {
	req := viewChannelReq{
		id: chi.URLParam(r, "channelID"),
	}

	return req, nil
}

func decodeCreateChannelReq(_ context.Context, r *http.Request) (interface{}, error) {
	if !strings.Contains(r.Header.Get("Content-Type"), api.ContentType) {
		return nil, errors.Wrap(apiutil.ErrValidation, apiutil.ErrUnsupportedContentType)
	}

	req := createChannelReq{}
	if err := json.NewDecoder(r.Body).Decode(&req.Channel); err != nil {
		return nil, errors.Wrap(apiutil.ErrValidation, errors.Wrap(errors.ErrMalformedEntity, err))
	}

	return req, nil
}

func decodeCreateChannelsReq(_ context.Context, r *http.Request) (interface{}, error) {
	if !strings.Contains(r.Header.Get("Content-Type"), api.ContentType) {
		return nil, errors.Wrap(apiutil.ErrValidation, apiutil.ErrUnsupportedContentType)
	}

	req := createChannelsReq{}
	if err := json.NewDecoder(r.Body).Decode(&req.Channels); err != nil {
		return nil, errors.Wrap(apiutil.ErrValidation, errors.Wrap(errors.ErrMalformedEntity, err))
	}

	return req, nil
}

func decodeListChannels(_ context.Context, r *http.Request) (interface{}, error) {
	name, err := apiutil.ReadStringQuery(r, api.NameKey, "")
	if err != nil {
		return listChannelsReq{}, errors.Wrap(apiutil.ErrValidation, err)
	}

	tag, err := apiutil.ReadStringQuery(r, api.TagKey, "")
	if err != nil {
		return nil, errors.Wrap(apiutil.ErrValidation, err)
	}

	s, err := apiutil.ReadStringQuery(r, api.StatusKey, api.DefGroupStatus)
	if err != nil {
		return listChannelsReq{}, errors.Wrap(apiutil.ErrValidation, err)
	}
	status, err := clients.ToStatus(s)
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

	groupID, err := apiutil.ReadStringQuery(r, api.GroupKey, "")
	if err != nil {
		return listChannelsReq{}, errors.Wrap(apiutil.ErrValidation, err)
	}

	clientID, err := apiutil.ReadStringQuery(r, api.ClientKey, "")
	if err != nil {
		return listChannelsReq{}, errors.Wrap(apiutil.ErrValidation, err)
	}

	req := listChannelsReq{
		name:       name,
		tag:        tag,
		status:     status,
		metadata:   meta,
		roleName:   roleName,
		roleID:     roleID,
		actions:    actions,
		accessType: accessType,
		order:      order,
		dir:        dir,
		offset:     offset,
		limit:      limit,
		groupID:    groupID,
		clientID:   clientID,
		userID:     userID,
	}
	return req, nil
}

func decodeUpdateChannel(_ context.Context, r *http.Request) (interface{}, error) {
	if !strings.Contains(r.Header.Get("Content-Type"), api.ContentType) {
		return nil, errors.Wrap(apiutil.ErrValidation, apiutil.ErrUnsupportedContentType)
	}

	req := updateChannelReq{
		id: chi.URLParam(r, "channelID"),
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, errors.Wrap(apiutil.ErrValidation, errors.Wrap(errors.ErrMalformedEntity, err))
	}

	return req, nil
}

func decodeUpdateChannelTags(_ context.Context, r *http.Request) (interface{}, error) {
	if !strings.Contains(r.Header.Get("Content-Type"), api.ContentType) {
		return nil, errors.Wrap(apiutil.ErrValidation, apiutil.ErrUnsupportedContentType)
	}

	req := updateChannelTagsReq{
		id: chi.URLParam(r, "channelID"),
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, errors.Wrap(apiutil.ErrValidation, errors.Wrap(errors.ErrMalformedEntity, err))
	}

	return req, nil
}

func decodeSetChannelParentGroupStatus(_ context.Context, r *http.Request) (interface{}, error) {
	if !strings.Contains(r.Header.Get("Content-Type"), api.ContentType) {
		return nil, errors.Wrap(apiutil.ErrValidation, apiutil.ErrUnsupportedContentType)
	}

	req := setChannelParentGroupReq{
		id: chi.URLParam(r, "channelID"),
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, errors.Wrap(apiutil.ErrValidation, errors.Wrap(errors.ErrMalformedEntity, err))
	}
	return req, nil
}

func decodeRemoveChannelParentGroupStatus(_ context.Context, r *http.Request) (interface{}, error) {
	req := removeChannelParentGroupReq{
		id: chi.URLParam(r, "channelID"),
	}

	return req, nil
}

func decodeChangeChannelStatus(_ context.Context, r *http.Request) (interface{}, error) {
	req := changeChannelStatusReq{
		id: chi.URLParam(r, "channelID"),
	}

	return req, nil
}

func decodeDeleteChannelReq(_ context.Context, r *http.Request) (interface{}, error) {
	req := deleteChannelReq{
		id: chi.URLParam(r, "channelID"),
	}
	return req, nil
}

func decodeConnectChannelClientRequest(_ context.Context, r *http.Request) (interface{}, error) {
	if !strings.Contains(r.Header.Get("Content-Type"), api.ContentType) {
		return nil, errors.Wrap(apiutil.ErrValidation, apiutil.ErrUnsupportedContentType)
	}
	req := connectChannelClientsRequest{
		channelID: chi.URLParam(r, "channelID"),
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, errors.Wrap(apiutil.ErrValidation, errors.Wrap(errors.ErrMalformedEntity, err))
	}

	return req, nil
}

func decodeDisconnectChannelClientsRequest(_ context.Context, r *http.Request) (interface{}, error) {
	if !strings.Contains(r.Header.Get("Content-Type"), api.ContentType) {
		return nil, errors.Wrap(apiutil.ErrValidation, apiutil.ErrUnsupportedContentType)
	}
	req := disconnectChannelClientsRequest{
		channelID: chi.URLParam(r, "channelID"),
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, errors.Wrap(apiutil.ErrValidation, errors.Wrap(errors.ErrMalformedEntity, err))
	}

	return req, nil
}

func decodeConnectRequest(_ context.Context, r *http.Request) (interface{}, error) {
	if !strings.Contains(r.Header.Get("Content-Type"), api.ContentType) {
		return nil, errors.Wrap(apiutil.ErrValidation, apiutil.ErrUnsupportedContentType)
	}

	req := connectRequest{}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, errors.Wrap(apiutil.ErrValidation, errors.Wrap(errors.ErrMalformedEntity, err))
	}

	return req, nil
}

func decodeDisconnectRequest(_ context.Context, r *http.Request) (interface{}, error) {
	if !strings.Contains(r.Header.Get("Content-Type"), api.ContentType) {
		return nil, errors.Wrap(apiutil.ErrValidation, apiutil.ErrUnsupportedContentType)
	}

	req := disconnectRequest{}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, errors.Wrap(apiutil.ErrValidation, errors.Wrap(errors.ErrMalformedEntity, err))
	}

	return req, nil
}
