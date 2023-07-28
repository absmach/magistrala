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

// NewAuthService creates mock of users service.
func NewAuthService(users map[string]string, authzDB map[string][]SubjectSet) policies.AuthServiceClient {
	return &authServiceMock{users, authzDB}
}

func (svc authServiceMock) Identify(ctx context.Context, req *policies.IdentifyReq, opts ...grpc.CallOption) (*policies.IdentifyRes, error) {
	if id, ok := svc.users[req.GetToken()]; ok {
		return &policies.IdentifyRes{Id: id}, nil
	}
	return nil, errors.ErrAuthentication
}

func (svc authServiceMock) Authorize(ctx context.Context, req *policies.AuthorizeReq, _ ...grpc.CallOption) (r *policies.AuthorizeRes, err error) {
	for _, policy := range svc.authz[req.GetSubject()] {
		for _, r := range policy.Relation {
			if r == req.GetAction() && policy.Subject == req.GetObject() {
				return &policies.AuthorizeRes{Authorized: true}, nil
			}
		}
	}
	return &policies.AuthorizeRes{Authorized: false}, nil
}
