// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package grpc

import (
	"context"

	"github.com/go-kit/kit/endpoint"
	"github.com/mainflux/mainflux/users/clients"
	"github.com/mainflux/mainflux/users/policies"
)

func authorizeEndpoint(svc policies.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(authReq)

		if err := req.validate(); err != nil {
			return authorizeRes{}, err
		}
		aReq := policies.AccessRequest{Subject: req.Sub, Object: req.Obj, Action: req.Act, Entity: req.EntityType}
		err := svc.Authorize(ctx, aReq)
		if err != nil {
			return authorizeRes{}, err
		}
		return authorizeRes{authorized: true}, err
	}
}

func issueEndpoint(svc clients.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(issueReq)
		if err := req.validate(); err != nil {
			return issueRes{}, err
		}

		tkn, err := svc.IssueToken(ctx, req.email, req.password)
		if err != nil {
			return issueRes{}, err
		}

		return issueRes{value: tkn.AccessToken}, nil
	}
}

func identifyEndpoint(svc clients.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(identityReq)
		if err := req.validate(); err != nil {
			return identityRes{}, err
		}

		id, err := svc.Identify(ctx, req.token)
		if err != nil {
			return identityRes{}, err
		}

		ret := identityRes{
			id: id,
		}
		return ret, nil
	}
}

func addPolicyEndpoint(svc policies.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(addPolicyReq)
		if err := req.validate(); err != nil {
			return addPolicyRes{}, err
		}
		policy := policies.Policy{Subject: req.Sub, Object: req.Obj, Actions: req.Act}
		err := svc.AddPolicy(ctx, req.Token, policy)
		if err != nil {
			return addPolicyRes{}, err
		}
		return addPolicyRes{authorized: true}, err
	}
}

func deletePolicyEndpoint(svc policies.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(policyReq)
		if err := req.validate(); err != nil {
			return deletePolicyRes{}, err
		}

		policy := policies.Policy{Subject: req.Sub, Object: req.Obj, Actions: []string{req.Act}}
		err := svc.DeletePolicy(ctx, req.Token, policy)
		if err != nil {
			return deletePolicyRes{}, err
		}
		return deletePolicyRes{deleted: true}, nil
	}
}

func listPoliciesEndpoint(svc policies.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(listPoliciesReq)
		pp := policies.Page{Subject: req.Sub, Object: req.Obj, Action: req.Act, Limit: 10}
		page, err := svc.ListPolicies(ctx, req.Token, pp)
		if err != nil {
			return listPoliciesRes{}, err
		}
		var objects []string
		for _, p := range page.Policies {
			objects = append(objects, p.Object)
		}
		return listPoliciesRes{objects: objects}, nil
	}
}
