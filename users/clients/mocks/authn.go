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

type SubjectSet struct {
	Subject  string
	Relation []string
}

type authServiceMock struct {
	users map[string]string
	authz map[string][]SubjectSet
}

func (svc authServiceMock) ListPolicies(ctx context.Context, in *policies.ListPoliciesReq, opts ...grpc.CallOption) (*policies.ListPoliciesRes, error) {
	res := policies.ListPoliciesRes{}
	for key := range svc.authz {
		res.Objects = append(res.Objects, key)
	}
	return &res, nil
}

// NewAuthService creates mock of users service.
func NewAuthService(users map[string]string, authzDB map[string][]SubjectSet) policies.AuthServiceClient {
	return &authServiceMock{users, authzDB}
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
	for _, policy := range svc.authz[req.GetSub()] {
		for _, r := range policy.Relation {
			if r == req.GetAct() && policy.Subject == req.GetObj() {
				return &policies.AuthorizeRes{Authorized: true}, nil
			}
		}
	}
	return &policies.AuthorizeRes{Authorized: false}, nil
}

func (svc authServiceMock) AddPolicy(ctx context.Context, in *policies.AddPolicyReq, opts ...grpc.CallOption) (*policies.AddPolicyRes, error) {
	if len(in.GetAct()) == 0 || in.GetObj() == "" || in.GetSub() == "" {
		return &policies.AddPolicyRes{}, errors.ErrMalformedEntity
	}

	svc.authz[in.GetSub()] = append(svc.authz[in.GetSub()], SubjectSet{Subject: in.GetSub(), Relation: in.GetAct()})
	return &policies.AddPolicyRes{Authorized: true}, nil
}

func (svc authServiceMock) DeletePolicy(ctx context.Context, in *policies.DeletePolicyReq, opts ...grpc.CallOption) (*policies.DeletePolicyRes, error) {
	if in.GetObj() == "" || in.GetSub() == "" {
		return &policies.DeletePolicyRes{}, errors.ErrMalformedEntity
	}
	delete(svc.authz, in.GetSub())
	return &policies.DeletePolicyRes{Deleted: true}, nil
}
