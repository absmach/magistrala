//
// Copyright (c) 2019
// Mainflux
//
// SPDX-License-Identifier: Apache-2.0
//

package http

import (
	"context"

	"github.com/go-kit/kit/endpoint"
	"github.com/mainflux/mainflux/users"
)

func registrationEndpoint(svc users.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(userReq)

		if err := req.validate(); err != nil {
			return nil, err
		}

		err := svc.Register(ctx, req.user)
		return tokenRes{}, err
	}
}

func loginEndpoint(svc users.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(userReq)

		if err := req.validate(); err != nil {
			return nil, err
		}

		token, err := svc.Login(ctx, req.user)
		if err != nil {
			return nil, err
		}

		return tokenRes{token}, nil
	}
}
