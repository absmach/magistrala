// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package grpc

import (
	"context"
	"time"

	"github.com/go-kit/kit/endpoint"
	"github.com/mainflux/mainflux/authn"
)

func issueEndpoint(svc authn.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(issueReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		now := time.Now().UTC()
		key := authn.Key{
			Type:     req.keyType,
			Subject:  req.email,
			IssuerID: req.id,
			IssuedAt: now,
		}

		_, secret, err := svc.Issue(ctx, "", key)
		if err != nil {
			return nil, err
		}

		return issueRes{secret, nil}, nil
	}
}

func identifyEndpoint(svc authn.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(identityReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		id, err := svc.Identify(ctx, req.token)
		if err != nil {
			return nil, err
		}

		ret := identityRes{
			id:    id.ID,
			email: id.Email,
			err:   nil,
		}
		return ret, nil
	}
}
