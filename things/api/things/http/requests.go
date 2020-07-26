// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package http

import (
	"github.com/mainflux/mainflux/things"
)

const maxLimitSize = 100
const maxNameSize = 1024

type apiReq interface {
	validate() error
}

type createThingReq struct {
	token    string
	Name     string                 `json:"name,omitempty"`
	Key      string                 `json:"key,omitempty"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

func (req createThingReq) validate() error {
	if req.token == "" {
		return things.ErrUnauthorizedAccess
	}

	if len(req.Name) > maxNameSize {
		return things.ErrMalformedEntity
	}

	return nil
}

type createThingsReq struct {
	token  string
	Things []createThingReq
}

func (req createThingsReq) validate() error {
	if req.token == "" {
		return things.ErrUnauthorizedAccess
	}

	if len(req.Things) <= 0 {
		return things.ErrMalformedEntity
	}

	for _, thing := range req.Things {
		if len(thing.Name) > maxNameSize {
			return things.ErrMalformedEntity
		}
	}

	return nil
}

type updateThingReq struct {
	token    string
	id       string
	Name     string                 `json:"name,omitempty"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

func (req updateThingReq) validate() error {
	if req.token == "" {
		return things.ErrUnauthorizedAccess
	}

	if req.id == "" {
		return things.ErrMalformedEntity
	}

	if len(req.Name) > maxNameSize {
		return things.ErrMalformedEntity
	}

	return nil
}

type updateKeyReq struct {
	token string
	id    string
	Key   string `json:"key"`
}

func (req updateKeyReq) validate() error {
	if req.token == "" {
		return things.ErrUnauthorizedAccess
	}

	if req.id == "" || req.Key == "" {
		return things.ErrMalformedEntity
	}

	return nil
}

type createChannelReq struct {
	token    string
	Name     string                 `json:"name,omitempty"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

func (req createChannelReq) validate() error {
	if req.token == "" {
		return things.ErrUnauthorizedAccess
	}

	if len(req.Name) > maxNameSize {
		return things.ErrMalformedEntity
	}

	return nil
}

type createChannelsReq struct {
	token    string
	Channels []createChannelReq
}

func (req createChannelsReq) validate() error {
	if req.token == "" {
		return things.ErrUnauthorizedAccess
	}

	if len(req.Channels) <= 0 {
		return things.ErrMalformedEntity
	}

	for _, channel := range req.Channels {
		if len(channel.Name) > maxNameSize {
			return things.ErrMalformedEntity
		}
	}

	return nil
}

type updateChannelReq struct {
	token    string
	id       string
	Name     string                 `json:"name,omitempty"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

func (req updateChannelReq) validate() error {
	if req.token == "" {
		return things.ErrUnauthorizedAccess
	}

	if req.id == "" {
		return things.ErrMalformedEntity
	}

	if len(req.Name) > maxNameSize {
		return things.ErrMalformedEntity
	}

	return nil
}

type viewResourceReq struct {
	token string
	id    string
}

func (req viewResourceReq) validate() error {
	if req.token == "" {
		return things.ErrUnauthorizedAccess
	}

	if req.id == "" {
		return things.ErrMalformedEntity
	}

	return nil
}

type listResourcesReq struct {
	token    string
	offset   uint64
	limit    uint64
	name     string
	metadata map[string]interface{}
}

func (req *listResourcesReq) validate() error {
	if req.token == "" {
		return things.ErrUnauthorizedAccess
	}

	if req.limit == 0 || req.limit > maxLimitSize {
		return things.ErrMalformedEntity
	}

	if len(req.name) > maxNameSize {
		return things.ErrMalformedEntity
	}

	return nil
}

type listByConnectionReq struct {
	token     string
	id        string
	offset    uint64
	limit     uint64
	connected bool
}

func (req listByConnectionReq) validate() error {
	if req.token == "" {
		return things.ErrUnauthorizedAccess
	}

	if req.id == "" {
		return things.ErrMalformedEntity
	}

	if req.limit == 0 || req.limit > maxLimitSize {
		return things.ErrMalformedEntity
	}

	return nil
}

type connectionReq struct {
	token   string
	chanID  string
	thingID string
}

func (req connectionReq) validate() error {
	if req.token == "" {
		return things.ErrUnauthorizedAccess
	}

	if req.chanID == "" || req.thingID == "" {
		return things.ErrMalformedEntity
	}

	return nil
}

type createConnectionsReq struct {
	token      string
	ChannelIDs []string `json:"channel_ids,omitempty"`
	ThingIDs   []string `json:"thing_ids,omitempty"`
}

func (req createConnectionsReq) validate() error {
	if req.token == "" {
		return things.ErrUnauthorizedAccess
	}

	if len(req.ChannelIDs) == 0 || len(req.ThingIDs) == 0 {
		return things.ErrMalformedEntity
	}

	for _, chID := range req.ChannelIDs {
		if chID == "" {
			return things.ErrMalformedEntity
		}
	}
	for _, thingID := range req.ThingIDs {
		if thingID == "" {
			return things.ErrMalformedEntity
		}
	}

	return nil
}
