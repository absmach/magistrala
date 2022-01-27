// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package grpc

import "github.com/mainflux/mainflux/pkg/errors"

type AccessByKeyReq struct {
	thingKey string
	chanID   string
}

func (req AccessByKeyReq) validate() error {
	if req.chanID == "" || req.thingKey == "" {
		return errors.ErrMalformedEntity
	}

	return nil
}

type accessByIDReq struct {
	thingID string
	chanID  string
}

func (req accessByIDReq) validate() error {
	if req.thingID == "" || req.chanID == "" {
		return errors.ErrMalformedEntity
	}

	return nil
}

type channelOwnerReq struct {
	owner  string
	chanID string
}

func (req channelOwnerReq) validate() error {
	if req.owner == "" || req.chanID == "" {
		return errors.ErrMalformedEntity
	}

	return nil
}

type identifyReq struct {
	key string
}

func (req identifyReq) validate() error {
	if req.key == "" {
		return errors.ErrMalformedEntity
	}

	return nil
}
