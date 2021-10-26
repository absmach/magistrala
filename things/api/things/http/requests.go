// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package http

import (
	"github.com/mainflux/mainflux/auth"
	"github.com/mainflux/mainflux/things"
)

const (
	maxLimitSize = 100
	maxNameSize  = 1024
	nameOrder    = "name"
	idOrder      = "id"
	ascDir       = "asc"
	descDir      = "desc"
	readPolicy   = "read"
	writePolicy  = "write"
	deletePolicy = "delete"
)

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

type shareThingReq struct {
	token    string
	thingID  string
	UserIDs  []string `json:"user_ids"`
	Policies []string `json:"policies"`
}

func (req shareThingReq) validate() error {
	if req.token == "" {
		return things.ErrUnauthorizedAccess
	}

	if req.thingID == "" || len(req.UserIDs) == 0 || len(req.Policies) == 0 {
		return things.ErrMalformedEntity
	}
	for _, p := range req.Policies {
		if p != readPolicy && p != writePolicy && p != deletePolicy {
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
	token        string
	pageMetadata things.PageMetadata
}

func (req *listResourcesReq) validate() error {
	if req.token == "" {
		return things.ErrUnauthorizedAccess
	}

	if req.pageMetadata.Limit == 0 {
		req.pageMetadata.Limit = defLimit
	}

	if req.pageMetadata.Limit > maxLimitSize {
		return things.ErrMalformedEntity
	}

	if len(req.pageMetadata.Name) > maxNameSize {
		return things.ErrMalformedEntity
	}

	if req.pageMetadata.Order != "" &&
		req.pageMetadata.Order != nameOrder && req.pageMetadata.Order != idOrder {
		return things.ErrMalformedEntity
	}

	if req.pageMetadata.Dir != "" &&
		req.pageMetadata.Dir != ascDir && req.pageMetadata.Dir != descDir {
		return things.ErrMalformedEntity
	}

	return nil
}

type listByConnectionReq struct {
	token        string
	id           string
	pageMetadata things.PageMetadata
}

func (req listByConnectionReq) validate() error {
	if req.token == "" {
		return things.ErrUnauthorizedAccess
	}

	if req.id == "" {
		return things.ErrMalformedEntity
	}

	if req.pageMetadata.Limit == 0 || req.pageMetadata.Limit > maxLimitSize {
		return things.ErrMalformedEntity
	}

	if req.pageMetadata.Order != "" &&
		req.pageMetadata.Order != nameOrder && req.pageMetadata.Order != idOrder {
		return things.ErrMalformedEntity
	}

	if req.pageMetadata.Dir != "" &&
		req.pageMetadata.Dir != ascDir && req.pageMetadata.Dir != descDir {
		return things.ErrMalformedEntity
	}

	return nil
}

type connectThingReq struct {
	token   string
	chanID  string
	thingID string
}

func (req connectThingReq) validate() error {
	if req.token == "" {
		return things.ErrUnauthorizedAccess
	}

	if req.chanID == "" || req.thingID == "" {
		return things.ErrMalformedEntity
	}

	return nil
}

type connectReq struct {
	token      string
	ChannelIDs []string `json:"channel_ids,omitempty"`
	ThingIDs   []string `json:"thing_ids,omitempty"`
}

func (req connectReq) validate() error {
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

type listThingsGroupReq struct {
	token        string
	groupID      string
	pageMetadata things.PageMetadata
}

func (req listThingsGroupReq) validate() error {
	if req.token == "" {
		return auth.ErrUnauthorizedAccess
	}

	if req.groupID == "" {
		return auth.ErrMalformedEntity
	}

	if req.pageMetadata.Limit == 0 || req.pageMetadata.Limit > maxLimitSize {
		return things.ErrMalformedEntity
	}

	if len(req.pageMetadata.Name) > maxNameSize {
		return things.ErrMalformedEntity
	}

	if req.pageMetadata.Order != "" &&
		req.pageMetadata.Order != "name" && req.pageMetadata.Order != "id" {
		return things.ErrMalformedEntity
	}

	if req.pageMetadata.Dir != "" &&
		req.pageMetadata.Dir != "asc" && req.pageMetadata.Dir != "desc" {
		return things.ErrMalformedEntity
	}

	return nil

}
