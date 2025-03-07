// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package auth

import (
	"context"

	"github.com/absmach/supermq/auth"
	"github.com/absmach/supermq/pkg/policies"
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

func authenticatePATEndpoint(svc auth.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(authenticateReq)
		if err := req.validate(); err != nil {
			return authenticateRes{}, err
		}

		pat, err := svc.IdentifyPAT(ctx, req.token)
		if err != nil {
			return authenticateRes{}, err
		}

		return authenticateRes{id: pat.ID, userID: pat.User}, nil
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

func authorizePATEndpoint(svc auth.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(authPATReq)

		if err := req.validate(); err != nil {
			return authorizeRes{}, err
		}
		err := svc.AuthorizePAT(ctx, req.userID, req.patID, req.entityType, req.optionalDomainID, req.operation, req.entityID)
		if err != nil {
			return authorizeRes{authorized: false}, err
		}
		return authorizeRes{authorized: true}, nil
	}
}
