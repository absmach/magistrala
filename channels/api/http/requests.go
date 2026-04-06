// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package http

import (
	"strings"

	api "github.com/absmach/magistrala/api/http"
	apiutil "github.com/absmach/magistrala/api/http/util"
	"github.com/absmach/magistrala/channels"
	"github.com/absmach/magistrala/pkg/connections"
)

type createChannelReq struct {
	Channel channels.Channel
}

func (req createChannelReq) validate() error {
	if len(req.Channel.Name) > api.MaxNameSize {
		return apiutil.ErrNameSize
	}
	if req.Channel.ID != "" {
		if strings.TrimSpace(req.Channel.ID) == "" {
			return apiutil.ErrMissingChannelID
		}
	}
	if req.Channel.Route != "" {
		if err := api.ValidateRoute(req.Channel.Route); err != nil {
			return err
		}
		if err := api.ValidateUUID(req.Channel.Route); err == nil {
			return apiutil.ErrInvalidRouteFormat
		}
	}

	return nil
}

type createChannelsReq struct {
	Channels []channels.Channel
}

func (req createChannelsReq) validate() error {
	if len(req.Channels) == 0 {
		return apiutil.ErrEmptyList
	}
	for _, channel := range req.Channels {
		if channel.ID != "" {
			if strings.TrimSpace(channel.ID) == "" {
				return apiutil.ErrMissingChannelID
			}
		}
		if len(channel.Name) > api.MaxNameSize {
			return apiutil.ErrNameSize
		}
		if channel.Route != "" {
			if err := api.ValidateRoute(channel.Route); err != nil {
				return err
			}
			if err := api.ValidateUUID(channel.Route); err == nil {
				return apiutil.ErrInvalidRouteFormat
			}
		}
	}

	return nil
}

type viewChannelReq struct {
	id    string
	roles bool
}

func (req viewChannelReq) validate() error {
	if req.id == "" {
		return apiutil.ErrMissingID
	}
	return nil
}

type listChannelsReq struct {
	channels.Page
	userID string
}

func (req listChannelsReq) validate() error {
	if req.Limit > api.MaxLimitSize || req.Limit < 1 {
		return apiutil.ErrLimitSize
	}

	if len(req.Name) > api.MaxNameSize {
		return apiutil.ErrNameSize
	}

	switch req.Order {
	case "", api.NameOrder, api.CreatedAtOrder, api.UpdatedAtOrder:
	default:
		return apiutil.ErrInvalidOrder
	}

	if req.Dir != "" && (req.Dir != api.DescDir && req.Dir != api.AscDir) {
		return apiutil.ErrInvalidDirection
	}

	if req.ConnectionType != "" {
		if _, err := connections.ParseConnType(req.ConnectionType); err != nil {
			return apiutil.ErrValidation
		}
	}

	return nil
}

type updateChannelReq struct {
	id       string
	Name     string         `json:"name,omitempty"`
	Metadata map[string]any `json:"metadata,omitempty"`
	Tags     []string       `json:"tags,omitempty"`
}

func (req updateChannelReq) validate() error {
	if req.id == "" {
		return apiutil.ErrMissingID
	}
	if len(req.Name) > api.MaxNameSize {
		return apiutil.ErrNameSize
	}

	return nil
}

type updateChannelTagsReq struct {
	id   string
	Tags []string `json:"tags,omitempty"`
}

func (req updateChannelTagsReq) validate() error {
	if req.id == "" {
		return apiutil.ErrMissingID
	}

	return nil
}

type setChannelParentGroupReq struct {
	id            string
	ParentGroupID string `json:"parent_group_id"`
}

func (req setChannelParentGroupReq) validate() error {
	if req.id == "" {
		return apiutil.ErrMissingID
	}
	if req.ParentGroupID == "" {
		return apiutil.ErrMissingParentGroupID
	}

	return nil
}

type removeChannelParentGroupReq struct {
	id string
}

func (req removeChannelParentGroupReq) validate() error {
	if req.id == "" {
		return apiutil.ErrMissingID
	}

	return nil
}

type changeChannelStatusReq struct {
	id string
}

func (req changeChannelStatusReq) validate() error {
	if req.id == "" {
		return apiutil.ErrMissingID
	}

	return nil
}

type connectChannelClientsRequest struct {
	channelID string
	ClientIDs []string               `json:"client_ids,omitempty"`
	Types     []connections.ConnType `json:"types,omitempty"`
}

func (req *connectChannelClientsRequest) validate() error {
	if req.channelID == "" || strings.TrimSpace(req.channelID) == "" {
		return apiutil.ErrMissingID
	}

	if len(req.ClientIDs) == 0 {
		return apiutil.ErrMissingID
	}

	for _, tid := range req.ClientIDs {
		if err := api.ValidateUUID(tid); err != nil {
			return err
		}
	}

	if len(req.Types) == 0 {
		return apiutil.ErrMissingConnectionType
	}

	return nil
}

type disconnectChannelClientsRequest struct {
	channelID string
	ClientIds []string               `json:"client_ids,omitempty"`
	Types     []connections.ConnType `json:"types,omitempty"`
}

func (req *disconnectChannelClientsRequest) validate() error {
	if req.channelID == "" {
		return apiutil.ErrMissingID
	}

	if err := api.ValidateUUID(req.channelID); err != nil {
		return err
	}

	if len(req.ClientIds) == 0 {
		return apiutil.ErrMissingID
	}

	for _, tid := range req.ClientIds {
		if err := api.ValidateUUID(tid); err != nil {
			return err
		}
	}

	if len(req.Types) == 0 {
		return apiutil.ErrMissingConnectionType
	}

	return nil
}

type connectRequest struct {
	ChannelIds []string               `json:"channel_ids,omitempty"`
	ClientIds  []string               `json:"client_ids,omitempty"`
	Types      []connections.ConnType `json:"types,omitempty"`
}

func (req *connectRequest) validate() error {
	if len(req.ChannelIds) == 0 {
		return apiutil.ErrMissingID
	}
	for _, cid := range req.ChannelIds {
		if strings.TrimSpace(cid) == "" {
			return apiutil.ErrMissingChannelID
		}
	}

	if len(req.ClientIds) == 0 {
		return apiutil.ErrMissingID
	}

	for _, tid := range req.ClientIds {
		if strings.TrimSpace(tid) == "" {
			return apiutil.ErrMissingChannelID
		}
	}

	if len(req.Types) == 0 {
		return apiutil.ErrMissingConnectionType
	}

	return nil
}

type disconnectRequest struct {
	ChannelIds []string               `json:"channel_ids,omitempty"`
	ClientIds  []string               `json:"client_ids,omitempty"`
	Types      []connections.ConnType `json:"types,omitempty"`
}

func (req *disconnectRequest) validate() error {
	if len(req.ChannelIds) == 0 {
		return apiutil.ErrMissingID
	}
	for _, cid := range req.ChannelIds {
		if err := api.ValidateUUID(cid); err != nil {
			return err
		}
	}

	if len(req.ClientIds) == 0 {
		return apiutil.ErrMissingID
	}

	for _, tid := range req.ClientIds {
		if err := api.ValidateUUID(tid); err != nil {
			return err
		}
	}

	if len(req.Types) == 0 {
		return apiutil.ErrMissingConnectionType
	}

	return nil
}

type deleteChannelReq struct {
	id string
}

func (req deleteChannelReq) validate() error {
	if req.id == "" {
		return apiutil.ErrMissingID
	}
	return nil
}
