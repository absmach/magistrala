// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package grpc

import (
	"context"

	"github.com/go-kit/kit/endpoint"
	"github.com/mainflux/mainflux"
	"github.com/mainflux/mainflux/things"
)

func authorizeEndpoint(svc things.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(*mainflux.AuthorizeReq)

		id, err := svc.Authorize(ctx, req)
		if err != nil {
			return authorizeRes{}, err
		}
		return authorizeRes{
			authorized: true,
			id:         id,
		}, err
	}
}
