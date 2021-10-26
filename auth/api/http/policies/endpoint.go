package policies

import (
	"context"
	"github.com/go-kit/kit/endpoint"
	"github.com/mainflux/mainflux/auth"
)

func createPolicyEndpoint(svc auth.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(createPolicyReq)
		if err := req.validate(); err != nil {
			return createPolicyRes{}, err
		}

		if err := svc.AddPolicies(ctx, req.token, req.Object, req.SubjectIDs, req.Policies); err != nil {
			return createPolicyRes{}, err
		}

		return createPolicyRes{created: true}, nil
	}
}
