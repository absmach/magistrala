// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package http

import (
	"context"

	"github.com/go-kit/kit/endpoint"
	"github.com/mainflux/mainflux/authz"
)

func addPolicy(svc authz.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(addPolicyReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		policy := authz.Policy{
			Action:  req.Action,
			Subject: req.Subject,
			Object:  req.Object,
		}

		added, err := svc.AddPolicy(ctx, req.token, policy)
		if err != nil {
			return addPolicyRes{created: false}, err
		}

		return addPolicyRes{created: added}, nil
	}
}

func removePolicy(svc authz.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(removePolicyReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		policy := authz.Policy{
			Action:  req.Action,
			Subject: req.Subject,
			Object:  req.Object,
		}

		if _, err := svc.RemovePolicy(ctx, req.token, policy); err != nil {
			return removePolicyRes{removed: false}, err
		}

		return removePolicyRes{removed: true}, nil
	}
}
