//
// Copyright (c) 2018
// Mainflux
//
// SPDX-License-Identifier: Apache-2.0
//

package http

import (
	"github.com/mainflux/mainflux/things"
)

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
	key   string
	thing things.Thing
}

func (req addThingReq) validate() error {
	if req.key == "" {
		return things.ErrUnauthorizedAccess
	}

	return req.thing.Validate()
}

type updateThingReq struct {
	key   string
	id    uint64
	thing things.Thing
}

func (req updateThingReq) validate() error {
	if req.key == "" {
		return things.ErrUnauthorizedAccess
	}

	if req.id < 1 {
		return things.ErrNotFound
	}

	return req.thing.Validate()
}

type createChannelReq struct {
	key     string
	channel things.Channel
}

func (req createChannelReq) validate() error {
	if req.key == "" {
		return things.ErrUnauthorizedAccess
	}

	return nil
}

type updateChannelReq struct {
	key     string
	id      uint64
	channel things.Channel
}

func (req updateChannelReq) validate() error {
	if req.key == "" {
		return things.ErrUnauthorizedAccess
	}

	if req.id < 1 {
		return things.ErrNotFound
	}

	return nil
}

type viewResourceReq struct {
	key string
	id  uint64
}

func (req viewResourceReq) validate() error {
	if req.key == "" {
		return things.ErrUnauthorizedAccess
	}

	if req.id < 1 {
		return things.ErrNotFound
	}

	return nil
}

type listResourcesReq struct {
	key    string
	offset int
	limit  int
}

func (req *listResourcesReq) validate() error {
	if req.key == "" {
		return things.ErrUnauthorizedAccess
	}

	if req.offset >= 0 && req.limit > 0 && req.limit <= maxLimitSize {
		return nil
	}

	return things.ErrMalformedEntity
}

type connectionReq struct {
	key     string
	chanID  uint64
	thingID uint64
}

func (req connectionReq) validate() error {
	if req.key == "" {
		return things.ErrUnauthorizedAccess
	}

	if req.chanID == 0 || req.thingID == 0 {
		return things.ErrNotFound
	}

	return nil
}
