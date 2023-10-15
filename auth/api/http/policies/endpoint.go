package policies

import (
	"context"

	"github.com/go-kit/kit/endpoint"
	"github.com/mainflux/mainflux/auth"
)

func createPolicyEndpoint(svc auth.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(policiesReq)
		if err := req.validate(); err != nil {
			return createPolicyRes{}, err
		}

		if err := svc.AddPolicies(ctx, req.token, req.Object, req.SubjectIDs, req.Policies); err != nil {
			return createPolicyRes{}, err
		}

		return createPolicyRes{created: true}, nil
	}
}

func deletePoliciesEndpoint(svc auth.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(policiesReq)
		if err := req.validate(); err != nil {
			return deletePoliciesRes{}, err
		}

		if err := svc.DeletePolicies(ctx, req.token, req.Object, req.SubjectIDs, req.Policies); err != nil {
			return deletePoliciesRes{}, err
		}

		return deletePoliciesRes{deleted: true}, nil
	}
}
