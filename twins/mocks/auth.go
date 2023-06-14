// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package mocks

import (
	"context"

	"github.com/mainflux/mainflux/pkg/errors"
	"github.com/mainflux/mainflux/users/policies"
	"google.golang.org/grpc"
)

var _ policies.AuthServiceClient = (*authServiceClient)(nil)

type authServiceClient struct {
	users map[string]string
}

func (svc authServiceClient) ListPolicies(ctx context.Context, in *policies.ListPoliciesReq, opts ...grpc.CallOption) (*policies.ListPoliciesRes, error) {
	panic("not implemented")
}

// NewAuthServiceClient creates mock of auth service.
func NewAuthServiceClient(users map[string]string) policies.AuthServiceClient {
	return &authServiceClient{users}
}

func (svc authServiceClient) Identify(ctx context.Context, in *policies.Token, opts ...grpc.CallOption) (*policies.UserIdentity, error) {
	if id, ok := svc.users[in.Value]; ok {
		return &policies.UserIdentity{Id: id}, nil
	}
	return nil, errors.ErrAuthentication
}

func (svc *authServiceClient) Issue(ctx context.Context, in *policies.IssueReq, opts ...grpc.CallOption) (*policies.Token, error) {
	return new(policies.Token), nil
}

func (svc *authServiceClient) Authorize(ctx context.Context, req *policies.AuthorizeReq, _ ...grpc.CallOption) (r *policies.AuthorizeRes, err error) {
	panic("not implemented")
}

func (svc authServiceClient) AddPolicy(ctx context.Context, in *policies.AddPolicyReq, opts ...grpc.CallOption) (*policies.AddPolicyRes, error) {
	panic("not implemented")
}

func (svc authServiceClient) DeletePolicy(ctx context.Context, in *policies.DeletePolicyReq, opts ...grpc.CallOption) (*policies.DeletePolicyRes, error) {
	panic("not implemented")
}
