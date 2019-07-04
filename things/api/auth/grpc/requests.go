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

type identifyReq struct {
	key string
}
