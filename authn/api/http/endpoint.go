// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package http

import (
	"context"
	"time"

	"github.com/go-kit/kit/endpoint"
	"github.com/mainflux/mainflux/authn"
)

func issueEndpoint(svc authn.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(issueKeyReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		now := time.Now().UTC()
		newKey := authn.Key{
			Issuer:   req.issuer,
			IssuedAt: now,
			Type:     req.Type,
		}

		duration := time.Duration(req.Duration * time.Second)
		if duration != 0 {
			exp := now.Add(duration)
			newKey.ExpiresAt = exp
		}

		key, err := svc.Issue(ctx, req.issuer, newKey)
		if err != nil {
			return nil, err
		}

		res := issueKeyRes{
			ID:       key.ID,
			Value:    key.Secret,
			IssuedAt: key.IssuedAt,
		}
		if !key.ExpiresAt.IsZero() {
			res.ExpiresAt = &key.ExpiresAt
		}
		return res, nil
	}
}

func revokeEndpoint(svc authn.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(keyReq)

		if err := req.validate(); err != nil {
			return nil, err
		}

		if err := svc.Revoke(ctx, req.issuer, req.id); err != nil {
			return nil, err
		}

		return revokeKeyRes{}, nil
	}
}

func retrieveEndpoint(svc authn.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(keyReq)

		if err := req.validate(); err != nil {
			return nil, err
		}

		key, err := svc.Retrieve(ctx, req.issuer, req.id)

		if err != nil {
			return nil, err
		}

		return key, nil
	}
}
