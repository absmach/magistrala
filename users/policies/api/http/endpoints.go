// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package http

import (
	"context"

	"github.com/go-kit/kit/endpoint"
	"github.com/mainflux/mainflux/users/policies"
)

func authorizeEndpoint(svc policies.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(authorizeReq)
		if err := req.validate(); err != nil {
			return authorizeRes{Authorized: false}, err
		}
		aReq := policies.AccessRequest{
			Subject: req.Subject,
			Object:  req.Object,
			Action:  req.Action,
			Entity:  req.EntityType,
		}
		err := svc.Authorize(ctx, aReq)
		if err != nil {
			return authorizeRes{Authorized: false}, err
		}

		return authorizeRes{Authorized: true}, nil
	}
}

func createPolicyEndpoint(svc policies.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(createPolicyReq)
		if err := req.validate(); err != nil {
			return addPolicyRes{}, err
		}

		policy := policies.Policy{
			Subject: req.Subject,
			Object:  req.Object,
			Actions: req.Actions,
		}
		err := svc.AddPolicy(ctx, req.token, policy)
		if err != nil {
			return addPolicyRes{}, err
		}

		return addPolicyRes{created: true}, nil
	}
}

func updatePolicyEndpoint(svc policies.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(updatePolicyReq)
		if err := req.validate(); err != nil {
			return updatePolicyRes{}, err
		}

		policy := policies.Policy{
			Subject: req.Subject,
			Object:  req.Object,
			Actions: req.Actions,
		}

		err := svc.UpdatePolicy(ctx, req.token, policy)
		if err != nil {
			return updatePolicyRes{}, err
		}

		res := updatePolicyRes{updated: false}
		return res, nil
	}
}

func listPolicyEndpoint(svc policies.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(listPolicyReq)
		if err := req.validate(); err != nil {
			return listPolicyRes{}, err
		}
		pm := policies.Page{
			Total:   req.Total,
			Offset:  req.Offset,
			Limit:   req.Limit,
			OwnerID: req.OwnerID,
			Subject: req.Subject,
			Object:  req.Object,
			Action:  req.Actions,
		}
		page, err := svc.ListPolicies(ctx, req.token, pm)
		if err != nil {
			return listPolicyRes{}, err
		}
		return buildGroupsResponse(page), nil
	}
}

func deletePolicyEndpoint(svc policies.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(deletePolicyReq)
		if err := req.validate(); err != nil {
			return deletePolicyRes{}, err
		}
		policy := policies.Policy{
			Subject: req.Subject,
			Object:  req.Object,
		}
		if err := svc.DeletePolicy(ctx, req.token, policy); err != nil {
			return deletePolicyRes{}, err
		}

		return deletePolicyRes{}, nil
	}
}

func toViewPolicyRes(group policies.Policy) viewPolicyRes {
	return viewPolicyRes{
		OwnerID:   group.OwnerID,
		Subject:   group.Subject,
		Object:    group.Object,
		Actions:   group.Actions,
		CreatedAt: group.CreatedAt,
		UpdatedAt: group.UpdatedAt,
	}
}

func buildGroupsResponse(page policies.PolicyPage) listPolicyRes {
	res := listPolicyRes{
		pageRes: pageRes{
			Limit:  page.Limit,
			Offset: page.Offset,
			Total:  page.Total,
		},
		Policies: []viewPolicyRes{},
	}

	for _, group := range page.Policies {
		res.Policies = append(res.Policies, toViewPolicyRes(group))
	}

	return res
}
