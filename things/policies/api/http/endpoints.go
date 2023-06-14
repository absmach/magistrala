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

		id, err := svc.Identify(ctx, req.Secret)
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
			return nil, err
		}
		ar := policies.AccessRequest{
			Subject: req.Subject,
			Object:  req.Object,
			Action:  req.Action,
			Entity:  req.EntityType,
		}
		policy, err := svc.Authorize(ctx, ar)
		if err != nil {
			return authorizeRes{Authorized: false}, err
		}

		return authorizeRes{ThingID: policy.Subject, Authorized: true}, nil
	}
}

func connectEndpoint(svc policies.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		cr := request.(createPolicyReq)

		if err := cr.validate(); err != nil {
			return nil, err
		}
		if len(cr.Actions) == 0 {
			cr.Actions = policies.PolicyTypes
		}
		policy := policies.Policy{
			Subject: cr.ClientID,
			Object:  cr.GroupID,
			Actions: cr.Actions,
		}
		policy, err := svc.AddPolicy(ctx, cr.token, policy)
		if err != nil {
			return nil, err
		}

		return policyRes{[]policies.Policy{policy}, true}, nil
	}
}

func connectThingsEndpoint(svc policies.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		cr := request.(createPoliciesReq)

		if err := cr.validate(); err != nil {
			return nil, err
		}
		if len(cr.Actions) == 0 {
			cr.Actions = policies.PolicyTypes
		}
		ps := []policies.Policy{}
		for _, tid := range cr.ClientIDs {
			for _, cid := range cr.GroupIDs {
				policy := policies.Policy{
					Subject: tid,
					Object:  cid,
					Actions: cr.Actions,
				}
				if _, err := svc.AddPolicy(ctx, cr.token, policy); err != nil {
					return nil, err
				}
				ps = append(ps, policy)
			}
		}

		return policyRes{created: true, Policies: ps}, nil
	}
}

func updatePolicyEndpoint(svc policies.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		cr := request.(policyReq)

		if err := cr.validate(); err != nil {
			return nil, err
		}
		policy := policies.Policy{
			Subject: cr.Subject,
			Object:  cr.Object,
			Actions: cr.Actions,
		}
		policy, err := svc.UpdatePolicy(ctx, cr.token, policy)
		if err != nil {
			return nil, err
		}

		return policyRes{[]policies.Policy{policy}, false}, nil
	}
}

func listPoliciesEndpoint(svc policies.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		lpr := request.(listPoliciesReq)

		if err := lpr.validate(); err != nil {
			return nil, err
		}
		policy := policies.Page{
			Limit:   lpr.limit,
			Offset:  lpr.offset,
			Subject: lpr.client,
			Object:  lpr.group,
			Action:  lpr.action,
			OwnerID: lpr.owner,
		}
		policyPage, err := svc.ListPolicies(ctx, lpr.token, policy)
		if err != nil {
			return nil, err
		}

		return listPolicyRes{policyPage}, nil
	}
}

func disconnectEndpoint(svc policies.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		cr := request.(createPolicyReq)
		if err := cr.validate(); err != nil {
			return nil, err
		}

		if len(cr.Actions) == 0 {
			cr.Actions = policies.PolicyTypes
		}
		policy := policies.Policy{
			Subject: cr.ClientID,
			Object:  cr.GroupID,
			Actions: cr.Actions,
		}
		if err := svc.DeletePolicy(ctx, cr.token, policy); err != nil {
			return nil, err
		}

		return deletePolicyRes{}, nil
	}
}

func disconnectThingsEndpoint(svc policies.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(createPoliciesReq)
		if err := req.validate(); err != nil {
			return nil, err
		}
		for _, tid := range req.ClientIDs {
			for _, cid := range req.GroupIDs {
				policy := policies.Policy{
					Subject: tid,
					Object:  cid,
				}
				if err := svc.DeletePolicy(ctx, req.token, policy); err != nil {
					return nil, err
				}
			}
		}

		return deletePolicyRes{}, nil
	}
}
