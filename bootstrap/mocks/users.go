// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package mocks

import (
	"context"

	"github.com/mainflux/mainflux/pkg/errors"
	"github.com/mainflux/mainflux/users/policies"
	"google.golang.org/grpc"
)

var _ policies.AuthServiceClient = (*serviceMock)(nil)

type serviceMock struct {
	users map[string]string
}

// NewAuthClient creates mock of users service.
func NewAuthClient(users map[string]string) policies.AuthServiceClient {
	return &serviceMock{users}
}

func (svc serviceMock) Identify(ctx context.Context, in *policies.Token, opts ...grpc.CallOption) (*policies.UserIdentity, error) {
	if id, ok := svc.users[in.GetValue()]; ok {
		return &policies.UserIdentity{Id: id}, nil
	}
	return nil, errors.ErrAuthentication
}

func (svc serviceMock) Issue(ctx context.Context, in *policies.IssueReq, opts ...grpc.CallOption) (*policies.Token, error) {
	if id, ok := svc.users[in.GetEmail()]; ok {
		return &policies.Token{Value: id}, nil
	}
	return nil, errors.ErrAuthentication
}

func (svc serviceMock) Authorize(ctx context.Context, req *policies.AuthorizeReq, _ ...grpc.CallOption) (r *policies.AuthorizeRes, err error) {
	panic("not implemented")
}

func (svc serviceMock) AddPolicy(ctx context.Context, req *policies.AddPolicyReq, _ ...grpc.CallOption) (r *policies.AddPolicyRes, err error) {
	panic("not implemented")
}
func (svc serviceMock) DeletePolicy(ctx context.Context, req *policies.DeletePolicyReq, _ ...grpc.CallOption) (r *policies.DeletePolicyRes, err error) {
	panic("not implemented")
}
func (svc serviceMock) ListPolicies(ctx context.Context, req *policies.ListPoliciesReq, _ ...grpc.CallOption) (r *policies.ListPoliciesRes, err error) {
	panic("not implemented")
}
