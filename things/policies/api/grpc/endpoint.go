// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package grpc

import (
	"context"

	"github.com/go-kit/kit/endpoint"
	"github.com/mainflux/mainflux/internal/apiutil"
	"github.com/mainflux/mainflux/pkg/errors"
	"github.com/mainflux/mainflux/things/clients"
	"github.com/mainflux/mainflux/things/policies"
)

func authorizeEndpoint(svc policies.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(authorizeReq)
		if err := req.validate(); err != nil {
			return authorizeRes{}, err
		}
		ar := policies.AccessRequest{
			Subject: req.subject,
			Object:  req.object,
			Action:  req.action,
			Entity:  req.entityType,
		}
		policy, err := svc.Authorize(ctx, ar)
		if err != nil {
			return authorizeRes{}, err
		}

		return authorizeRes{authorized: true, thingID: policy.Subject}, nil
	}
}

func identifyEndpoint(svc clients.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(identifyReq)
		if err := req.validate(); err != nil {
			return identityRes{}, err
		}
		id, err := svc.Identify(ctx, req.secret)
		if err != nil {
			return identityRes{}, errors.Wrap(apiutil.ErrValidation, err)
		}
		return identityRes{id: id}, nil
	}
}
