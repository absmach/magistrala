// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package auth

import (
	"context"

	"github.com/absmach/magistrala/auth"
	"github.com/absmach/magistrala/pkg/policies"
	"github.com/go-kit/kit/endpoint"
)

func authenticateEndpoint(svc auth.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(authenticateReq)
		if err := req.validate(); err != nil {
			return authenticateRes{}, err
		}

		key, err := svc.Identify(ctx, req.token)
		if err != nil {
			return authenticateRes{}, err
		}

		return authenticateRes{id: key.Subject, userID: key.User, domainID: key.Domain}, nil
	}
}

func authorizeEndpoint(svc auth.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(authReq)

		if err := req.validate(); err != nil {
			return authorizeRes{}, err
		}
		err := svc.Authorize(ctx, policies.Policy{
			Domain:      req.Domain,
			SubjectType: req.SubjectType,
			SubjectKind: req.SubjectKind,
			Subject:     req.Subject,
			Relation:    req.Relation,
			Permission:  req.Permission,
			ObjectType:  req.ObjectType,
			Object:      req.Object,
		})
		if err != nil {
			return authorizeRes{authorized: false}, err
		}
		return authorizeRes{authorized: true}, nil
	}
}
