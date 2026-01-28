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
	return func(ctx context.Context, request any) (any, error) {
		req := request.(authenticateReq)
		if err := req.validate(); err != nil {
			return authenticateRes{}, err
		}

		key, err := svc.Identify(ctx, req.token)
		if err != nil {
			return authenticateRes{}, err
		}

		return authenticateRes{id: key.ID, userID: key.Subject, userRole: key.Role, verified: key.Verified}, nil
	}
}

func authorizeEndpoint(svc auth.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req := request.(authReq)

		if err := req.validate(); err != nil {
			return authorizeRes{}, err
		}

		var pat *auth.PATAuthz
		if req.PatID != "" {
			entityType, err := auth.ParseEntityType(req.EntityType)
			if err != nil {
				return authorizeRes{authorized: false}, err
			}
			pat = &auth.PATAuthz{
				PatID:      req.PatID,
				UserID:     req.UserID,
				EntityType: entityType,
				EntityID:   req.EntityID,
				Operation:  req.Operation,
				Domain:     req.Domain,
			}
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
		}, pat)
		if err != nil {
			return authorizeRes{authorized: false}, err
		}
		return authorizeRes{authorized: true}, nil
	}
}
