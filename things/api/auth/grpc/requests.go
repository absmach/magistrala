//
// Copyright (c) 2018
// Mainflux
//
// SPDX-License-Identifier: Apache-2.0
//

package grpc

import "github.com/mainflux/mainflux/things"

type accessReq struct {
	thingKey string
	chanID   string
}

func (req accessReq) validate() error {
	if req.chanID == "" || req.thingKey == "" {
		return things.ErrMalformedEntity
	}

	return nil
}

type accessByIDReq struct {
	thingID string
	chanID  string
}

func (req accessByIDReq) validate() error {
	if req.thingID == "" || req.chanID == "" {
		return things.ErrMalformedEntity
	}

	return nil
}

type identifyReq struct {
	key string
}

func (req identifyReq) validate() error {
	if req.key == "" {
		return things.ErrMalformedEntity
	}

	return nil
}
