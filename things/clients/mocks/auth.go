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

type MockSubjectSet struct {
	Object   string
	Relation []string
}

type authServiceMock struct {
	users    map[string]string
	policies map[string][]MockSubjectSet
}

// NewAuthService creates mock of users service.
func NewAuthService(users map[string]string, policies map[string][]MockSubjectSet) policies.AuthServiceClient {
	return &authServiceMock{users, policies}
}

func (svc authServiceMock) Identify(ctx context.Context, in *policies.Token, opts ...grpc.CallOption) (*policies.UserIdentity, error) {
	if id, ok := svc.users[in.Value]; ok {
		return &policies.UserIdentity{Id: id}, nil
	}
	return nil, errors.ErrAuthentication
}

func (svc authServiceMock) Issue(ctx context.Context, in *policies.IssueReq, opts ...grpc.CallOption) (*policies.Token, error) {
	if id, ok := svc.users[in.GetEmail()]; ok {
		switch in.Type {
		default:
			return &policies.Token{Value: id}, nil
		}
	}
	return nil, errors.ErrAuthentication
}

func (svc authServiceMock) Authorize(ctx context.Context, req *policies.AuthorizeReq, _ ...grpc.CallOption) (r *policies.AuthorizeRes, err error) {
	for _, policy := range svc.policies[req.GetSub()] {
		for _, r := range policy.Relation {
			if r == req.GetAct() && policy.Object == req.GetObj() {
				return &policies.AuthorizeRes{Authorized: true}, nil
			}
		}

	}
	return nil, errors.ErrAuthorization
}

func (svc authServiceMock) AddPolicy(ctx context.Context, in *policies.AddPolicyReq, opts ...grpc.CallOption) (*policies.AddPolicyRes, error) {
	if len(in.GetAct()) == 0 || in.GetObj() == "" || in.GetSub() == "" {
		return &policies.AddPolicyRes{}, errors.ErrMalformedEntity
	}

	obj := in.GetObj()
	svc.policies[in.GetSub()] = append(svc.policies[in.GetSub()], MockSubjectSet{Object: obj, Relation: in.GetAct()})
	return &policies.AddPolicyRes{Authorized: true}, nil
}

func (svc authServiceMock) ListPolicies(ctx context.Context, in *policies.ListPoliciesReq, opts ...grpc.CallOption) (*policies.ListPoliciesRes, error) {
	res := policies.ListPoliciesRes{}
	for key := range svc.policies {
		res.Objects = append(res.Objects, key)
	}
	return &res, nil
}

func (svc authServiceMock) DeletePolicy(ctx context.Context, in *policies.DeletePolicyReq, opts ...grpc.CallOption) (*policies.DeletePolicyRes, error) {
	// Not implemented yet
	return &policies.DeletePolicyRes{Deleted: true}, nil
}
