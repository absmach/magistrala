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

func (svc authServiceMock) Identify(ctx context.Context, req *policies.IdentifyReq, opts ...grpc.CallOption) (*policies.IdentifyRes, error) {
	if id, ok := svc.users[req.GetToken()]; ok {
		return &policies.IdentifyRes{Id: id}, nil
	}
	return nil, errors.ErrAuthentication
}

func (svc authServiceMock) Authorize(ctx context.Context, req *policies.AuthorizeReq, _ ...grpc.CallOption) (r *policies.AuthorizeRes, err error) {
	for _, policy := range svc.policies[req.GetSubject()] {
		for _, r := range policy.Relation {
			if r == req.GetAction() && policy.Object == req.GetObject() {
				return &policies.AuthorizeRes{Authorized: true}, nil
			}
		}
	}
	return nil, errors.ErrAuthorization
}
