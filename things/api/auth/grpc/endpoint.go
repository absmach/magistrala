//
// Copyright (c) 2019
// Mainflux
//
// SPDX-License-Identifier: Apache-2.0
//

package grpc

import (
	"github.com/go-kit/kit/endpoint"
	"github.com/mainflux/mainflux/things"
	context "golang.org/x/net/context"
)

func canAccessEndpoint(svc things.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(accessReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		id, err := svc.CanAccess(ctx, req.chanID, req.thingKey)
		if err != nil {
			return identityRes{err: err}, err
		}
		return identityRes{id: id, err: nil}, nil
	}
}

func canAccessByIDEndpoint(svc things.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(accessByIDReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		err := svc.CanAccessByID(ctx, req.chanID, req.thingID)
		return emptyRes{err: err}, err
	}
}

func identifyEndpoint(svc things.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(identifyReq)
		id, err := svc.Identify(ctx, req.key)
		if err != nil {
			return identityRes{err: err}, err
		}
		return identityRes{id: id, err: nil}, nil
	}
}
