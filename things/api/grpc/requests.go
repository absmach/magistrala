//
// Copyright (c) 2018
// Mainflux
//
// SPDX-License-Identifier: Apache-2.0
//

package grpc

import (
	"github.com/mainflux/mainflux/things"
)

type accessReq struct {
	thingKey string
	chanID   uint64
}

func (req accessReq) validate() error {
	if req.chanID == 0 || req.thingKey == "" {
		return things.ErrMalformedEntity
	}
	return nil
}

type identifyReq struct {
	key string
}
