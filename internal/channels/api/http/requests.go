// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package http

import (
	"github.com/absmach/magistrala/internal/api"
	"github.com/absmach/magistrala/pkg/apiutil"
	"github.com/absmach/magistrala/pkg/channels"
	mgclients "github.com/absmach/magistrala/pkg/clients"
)

type createChannelReq struct {
	Channel channels.Channel
	token   string
}

func (req createChannelReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}
	if len(req.Channel.Name) > api.MaxNameSize {
		return apiutil.ErrNameSize
	}
	if req.Channel.ID != "" {
		return api.ValidateUUID(req.Channel.ID)
	}

	return nil
}

type createChannelsReq struct {
	token    string
	Channels []channels.Channel
}

func (req createChannelsReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}
	if len(req.Channels) == 0 {
		return apiutil.ErrEmptyList
	}
	for _, channel := range req.Channels {
		if channel.ID != "" {
			if err := api.ValidateUUID(channel.ID); err != nil {
				return err
			}
		}
		if len(channel.Name) > api.MaxNameSize {
			return apiutil.ErrNameSize
		}
	}

	return nil
}

type viewChannelReq struct {
	token string
	id    string
}

func (req viewChannelReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}
	if req.id == "" {
		return apiutil.ErrMissingID
	}
	return nil
}

type listChannelsReq struct {
	token      string
	status     mgclients.Status
	offset     uint64
	limit      uint64
	name       string
	tag        string
	permission string
	visibility string
	userID     string
	listPerms  bool
	metadata   mgclients.Metadata
	id         string
}

func (req listChannelsReq) validate() error {
	if req.limit > api.MaxLimitSize || req.limit < 1 {
		return apiutil.ErrLimitSize
	}
	if req.visibility != "" &&
		req.visibility != api.AllVisibility &&
		req.visibility != api.MyVisibility &&
		req.visibility != api.SharedVisibility {
		return apiutil.ErrInvalidVisibilityType
	}
	if len(req.name) > api.MaxNameSize {
		return apiutil.ErrNameSize
	}

	return nil
}

type updateChannelReq struct {
	token    string
	id       string
	Name     string                 `json:"name,omitempty"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
	Tags     []string               `json:"tags,omitempty"`
}

func (req updateChannelReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}
	if req.id == "" {
		return apiutil.ErrMissingID
	}
	if len(req.Name) > api.MaxNameSize {
		return apiutil.ErrNameSize
	}

	return nil
}

type updateChannelTagsReq struct {
	id    string
	token string
	Tags  []string `json:"tags,omitempty"`
}

func (req updateChannelTagsReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}
	if req.id == "" {
		return apiutil.ErrMissingID
	}

	return nil
}

type changeChannelStatusReq struct {
	token string
	id    string
}

func (req changeChannelStatusReq) validate() error {
	if req.id == "" {
		return apiutil.ErrMissingID
	}

	return nil
}

type connectChannelThingRequest struct {
	token     string
	ThingID   string `json:"thing_id,omitempty"`
	ChannelID string `json:"channel_id,omitempty"`
}

func (req *connectChannelThingRequest) validate() error {
	if req.ThingID == "" || req.ChannelID == "" {
		return apiutil.ErrMissingID
	}
	return nil
}

type disconnectChannelThingRequest struct {
	token     string
	ThingID   string `json:"thing_id,omitempty"`
	ChannelID string `json:"channel_id,omitempty"`
}

func (req *disconnectChannelThingRequest) validate() error {
	if req.ThingID == "" || req.ChannelID == "" {
		return apiutil.ErrMissingID
	}
	return nil
}

type deleteChannelReq struct {
	token string
	id    string
}

func (req deleteChannelReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}
	if req.id == "" {
		return apiutil.ErrMissingID
	}
	return nil
}
