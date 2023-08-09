// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package grpc

import (
	"context"

	"github.com/go-kit/kit/endpoint"
	"github.com/mainflux/mainflux/internal/apiutil"
	"github.com/mainflux/mainflux/pkg/errors"
	"github.com/mainflux/mainflux/users/clients"
	"github.com/mainflux/mainflux/users/policies"
)

func authorizeEndpoint(svc policies.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(authReq)

		if err := req.validate(); err != nil {
			return authorizeRes{}, errors.Wrap(apiutil.ErrValidation, err)
		}
		aReq := policies.AccessRequest{Subject: req.subject, Object: req.object, Action: req.action, Entity: req.entityType}
		err := svc.Authorize(ctx, aReq)
		if err != nil {
			return authorizeRes{}, errors.Wrap(apiutil.ErrValidation, err)
		}
		return authorizeRes{authorized: true}, err
	}
}

func identifyEndpoint(svc clients.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(identifyReq)
		if err := req.validate(); err != nil {
			return identifyRes{}, err
		}

		id, err := svc.Identify(ctx, req.token)
		if err != nil {
			return identifyRes{}, err
		}

		ret := identifyRes{
			id: id,
		}
		return ret, nil
	}
}
