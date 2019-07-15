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
	"github.com/mainflux/mainflux/things"
)

func identifyEndpoint(svc things.Service) endpoint.Endpoint {
	return func(_ context.Context, request interface{}) (interface{}, error) {
		req := request.(identifyReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		id, err := svc.Identify(req.Token)
		if err != nil {
			return nil, err
		}

		res := identityRes{
			ID: id,
		}

		return res, nil
	}
}

func canAccessEndpoint(svc things.Service) endpoint.Endpoint {
	return func(_ context.Context, request interface{}) (interface{}, error) {
		req := request.(canAccessReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		id, err := svc.CanAccess(req.chanID, req.Token)
		if err != nil {
			return nil, err
		}

		res := identityRes{
			ID: id,
		}

		return res, nil
	}
}

func canAccessByIDEndpoint(svc things.Service) endpoint.Endpoint {
	return func(_ context.Context, request interface{}) (interface{}, error) {
		req := request.(canAccessByIDReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		if err := svc.CanAccessByID(req.chanID, req.ThingID); err != nil {
			return nil, err
		}

		res := canAccessByIDRes{}
		return res, nil
	}
}
