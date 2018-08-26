//
// Copyright (c) 2018
// Mainflux
//
// SPDX-License-Identifier: Apache-2.0
//

package grpc

import (
	"github.com/go-kit/kit/endpoint"
	"github.com/mainflux/mainflux/users"
	context "golang.org/x/net/context"
)

func identifyEndpoint(svc users.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(identityReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		id, err := svc.Identify(req.token)
		if err != nil {
			return identityRes{}, err
		}
		return identityRes{id, nil}, nil
	}
}
