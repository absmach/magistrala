// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package mocks

import (
	"context"

	"github.com/mainflux/mainflux/pkg/errors"
	"github.com/mainflux/mainflux/users/policies"
	"google.golang.org/grpc"
)

var _ policies.AuthServiceClient = (*authServiceMock)(nil)

type authServiceMock struct {
	users map[string]string
}

// NewAuth creates mock of auth service.
func NewAuth(users map[string]string) policies.AuthServiceClient {
	return &authServiceMock{users}
}

func (svc authServiceMock) Identify(ctx context.Context, in *policies.Token, opts ...grpc.CallOption) (*policies.UserIdentity, error) {
	if id, ok := svc.users[in.Value]; ok {
		return &policies.UserIdentity{Id: id}, nil
	}
	return nil, errors.ErrAuthentication
}

func (svc authServiceMock) Issue(ctx context.Context, in *policies.IssueReq, opts ...grpc.CallOption) (*policies.Token, error) {
	if id, ok := svc.users[in.GetEmail()]; ok {
		return &policies.Token{Value: id}, nil
	}
	return nil, errors.ErrAuthentication
}

func (svc authServiceMock) Authorize(ctx context.Context, req *policies.AuthorizeReq, _ ...grpc.CallOption) (r *policies.AuthorizeRes, err error) {
	panic("not implemented")
}

func (svc authServiceMock) AddPolicy(ctx context.Context, in *policies.AddPolicyReq, opts ...grpc.CallOption) (*policies.AddPolicyRes, error) {
	panic("not implemented")
}

func (svc authServiceMock) DeletePolicy(ctx context.Context, in *policies.DeletePolicyReq, opts ...grpc.CallOption) (*policies.DeletePolicyRes, error) {
	panic("not implemented")
}

func (svc authServiceMock) ListPolicies(ctx context.Context, in *policies.ListPoliciesReq, opts ...grpc.CallOption) (*policies.ListPoliciesRes, error) {
	panic("not implemented")
}
