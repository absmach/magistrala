// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package http

import (
	"context"

	"github.com/go-kit/kit/endpoint"
	"github.com/mainflux/mainflux/things/clients"
	"github.com/mainflux/mainflux/things/policies"
)

func identifyEndpoint(svc clients.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(identifyReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		id, err := svc.Identify(ctx, req.secret)
		if err != nil {
			return nil, err
		}

		return identityRes{ID: id}, nil
	}
}

func authorizeEndpoint(svc policies.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(authorizeReq)
		if err := req.validate(); err != nil {
			return authorizeRes{}, err
		}
		ar := policies.AccessRequest{
			Subject: req.Subject,
			Object:  req.Object,
			Action:  req.Action,
			Entity:  req.EntityType,
		}
		policy, err := svc.Authorize(ctx, ar)
		if err != nil {
			return authorizeRes{}, err
		}

		return authorizeRes{ThingID: policy.Subject, Authorized: true}, nil
	}
}

func connectEndpoint(svc policies.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		cr := request.(createPolicyReq)

		if err := cr.validate(); err != nil {
			return addPolicyRes{}, err
		}
		if len(cr.Actions) == 0 {
			cr.Actions = policies.PolicyTypes
		}
		policy := policies.Policy{
			Subject: cr.Subject,
			Object:  cr.Object,
			Actions: cr.Actions,
		}
		policy, err := svc.AddPolicy(ctx, cr.token, policy)
		if err != nil {
			return nil, err
		}

		return addPolicyRes{created: true, Policy: policy}, nil
	}
}

func connectThingsEndpoint(svc policies.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		cr := request.(createPoliciesReq)

		if err := cr.validate(); err != nil {
			return listPolicyRes{}, err
		}
		if len(cr.Actions) == 0 {
			cr.Actions = policies.PolicyTypes
		}
		var pols policies.PolicyPage
		for _, tid := range cr.Subjects {
			for _, cid := range cr.Objects {
				policy := policies.Policy{
					Subject: tid,
					Object:  cid,
					Actions: cr.Actions,
				}
				p, err := svc.AddPolicy(ctx, cr.token, policy)
				if err != nil {
					return listPolicyRes{}, err
				}
				pols.Policies = append(pols.Policies, p)
			}
		}

		return buildPoliciesResponse(pols), nil
	}
}

func updatePolicyEndpoint(svc policies.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		cr := request.(policyReq)

		if err := cr.validate(); err != nil {
			return updatePolicyRes{}, err
		}
		policy := policies.Policy{
			Subject: cr.Subject,
			Object:  cr.Object,
			Actions: cr.Actions,
		}
		policy, err := svc.UpdatePolicy(ctx, cr.token, policy)
		if err != nil {
			return updatePolicyRes{}, err
		}

		return updatePolicyRes{updated: true, Policy: policy}, nil
	}
}

func listPoliciesEndpoint(svc policies.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		lpr := request.(listPoliciesReq)

		if err := lpr.validate(); err != nil {
			return nil, err
		}
		pm := policies.Page{
			Limit:   lpr.limit,
			Offset:  lpr.offset,
			Subject: lpr.client,
			Object:  lpr.group,
			Action:  lpr.action,
			OwnerID: lpr.owner,
		}
		policyPage, err := svc.ListPolicies(ctx, lpr.token, pm)
		if err != nil {
			return nil, err
		}

		return buildPoliciesResponse(policyPage), nil
	}
}

func disconnectEndpoint(svc policies.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		cr := request.(createPolicyReq)
		if err := cr.validate(); err != nil {
			return deletePolicyRes{}, err
		}

		if len(cr.Actions) == 0 {
			cr.Actions = policies.PolicyTypes
		}
		policy := policies.Policy{
			Subject: cr.Subject,
			Object:  cr.Object,
			Actions: cr.Actions,
		}
		if err := svc.DeletePolicy(ctx, cr.token, policy); err != nil {
			return deletePolicyRes{}, err
		}

		return deletePolicyRes{deleted: true}, nil
	}
}

func disconnectThingsEndpoint(svc policies.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(createPoliciesReq)
		if err := req.validate(); err != nil {
			return deletePolicyRes{}, err
		}
		for _, tid := range req.Subjects {
			for _, cid := range req.Objects {
				policy := policies.Policy{
					Subject: tid,
					Object:  cid,
				}
				if err := svc.DeletePolicy(ctx, req.token, policy); err != nil {
					return deletePolicyRes{}, err
				}
			}
		}

		return deletePolicyRes{deleted: true}, nil
	}
}

func buildPoliciesResponse(page policies.PolicyPage) listPolicyRes {
	res := listPolicyRes{
		pageRes: pageRes{
			Limit:  page.Limit,
			Offset: page.Offset,
			Total:  page.Total,
		},
		Policies: []viewPolicyRes{},
	}

	for _, policy := range page.Policies {
		res.Policies = append(res.Policies, viewPolicyRes{policy})
	}

	return res
}
