// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package grpc

import (
	"context"

	"github.com/absmach/magistrala/things"
	"github.com/go-kit/kit/endpoint"
)

func authorizeEndpoint(svc things.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(authorizeReq)

		thingID, err := svc.Authorize(ctx, things.AuthzReq{
			ChannelID:  req.ChannelID,
			ThingID:    req.ThingID,
			ThingKey:   req.ThingKey,
			Permission: req.Permission,
		})
		if err != nil {
			return authorizeRes{}, err
		}
		return authorizeRes{
			authorized: true,
			id:         thingID,
		}, err
	}
}
