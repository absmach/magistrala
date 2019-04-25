//
// Copyright (c) 2018
// Mainflux
//
// SPDX-License-Identifier: Apache-2.0
//

package http

import "github.com/mainflux/mainflux/things"

const maxLimitSize = 100

type apiReq interface {
	validate() error
}

type addThingReq struct {
	token    string
	Name     string                 `json:"name,omitempty"`
	Key      string                 `json:"key,omitempty"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

func (req addThingReq) validate() error {
	if req.token == "" {
		return things.ErrUnauthorizedAccess
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
	token  string
	offset uint64
	limit  uint64
}

func (req *listResourcesReq) validate() error {
	if req.token == "" {
		return things.ErrUnauthorizedAccess
	}

	if req.limit == 0 || req.limit > maxLimitSize {
		return things.ErrMalformedEntity
	}

	return nil
}

type listByConnectionReq struct {
	token  string
	id     string
	offset uint64
	limit  uint64
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
