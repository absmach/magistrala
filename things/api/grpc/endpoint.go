// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package grpc

import (
	"context"

	"github.com/absmach/magistrala"
	"github.com/absmach/magistrala/things"
	"github.com/go-kit/kit/endpoint"
)

func authorizeEndpoint(svc things.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(*magistrala.AuthorizeReq)

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
