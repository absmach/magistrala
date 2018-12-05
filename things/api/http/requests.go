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

type identityReq struct {
	key string
}

func (req identityReq) validate() error {
	if req.key == "" {
		return things.ErrUnauthorizedAccess
	}

	return nil
}

type addThingReq struct {
	key      string
	Type     string `json:"type"`
	Name     string `json:"name,omitempty"`
	Metadata string `json:"metadata,omitempty"`
}

func (req addThingReq) validate() error {
	if req.key == "" {
		return things.ErrUnauthorizedAccess
	}

	if req.Type == "" {
		return things.ErrMalformedEntity
	}

	return nil
}

type updateThingReq struct {
	key      string
	id       string
	Type     string `json:"type"`
	Name     string `json:"name,omitempty"`
	Metadata string `json:"metadata,omitempty"`
}

func (req updateThingReq) validate() error {
	if req.key == "" {
		return things.ErrUnauthorizedAccess
	}

	if req.id == "" || req.Type == "" {
		return things.ErrMalformedEntity
	}

	return nil
}

type createChannelReq struct {
	key      string
	Name     string `json:"name,omitempty"`
	Metadata string `json:"metadata,omitempty"`
}

func (req createChannelReq) validate() error {
	if req.key == "" {
		return things.ErrUnauthorizedAccess
	}

	return nil
}

type updateChannelReq struct {
	key      string
	id       string
	Name     string `json:"name,omitempty"`
	Metadata string `json:"metadata,omitempty"`
}

func (req updateChannelReq) validate() error {
	if req.key == "" {
		return things.ErrUnauthorizedAccess
	}

	if req.id == "" {
		return things.ErrMalformedEntity
	}

	return nil
}

type viewResourceReq struct {
	key string
	id  string
}

func (req viewResourceReq) validate() error {
	if req.key == "" {
		return things.ErrUnauthorizedAccess
	}

	if req.id == "" {
		return things.ErrMalformedEntity
	}

	return nil
}

type listResourcesReq struct {
	key    string
	offset uint64
	limit  uint64
}

func (req *listResourcesReq) validate() error {
	if req.key == "" {
		return things.ErrUnauthorizedAccess
	}

	if req.limit == 0 || req.limit > maxLimitSize {
		return things.ErrMalformedEntity
	}

	return nil
}

type connectionReq struct {
	key     string
	chanID  string
	thingID string
}

func (req connectionReq) validate() error {
	if req.key == "" {
		return things.ErrUnauthorizedAccess
	}

	if req.chanID == "" || req.thingID == "" {
		return things.ErrMalformedEntity
	}

	return nil
}
