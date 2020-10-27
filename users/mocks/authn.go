// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package mocks

import (
	"context"

	"github.com/mainflux/mainflux"
	"github.com/mainflux/mainflux/users"
	"google.golang.org/grpc"
)

var _ mainflux.AuthNServiceClient = (*authNServiceMock)(nil)

type authNServiceMock struct {
	users map[string]string
}

// NewAuthService creates mock of users service.
func NewAuthService(users map[string]string) mainflux.AuthNServiceClient {
	return &authNServiceMock{users}
}

func (svc authNServiceMock) Identify(ctx context.Context, in *mainflux.Token, opts ...grpc.CallOption) (*mainflux.UserIdentity, error) {
	if id, ok := svc.users[in.Value]; ok {
		return &mainflux.UserIdentity{Id: id, Email: id}, nil
	}
	return nil, users.ErrUnauthorizedAccess
}

func (svc authNServiceMock) Issue(ctx context.Context, in *mainflux.IssueReq, opts ...grpc.CallOption) (*mainflux.Token, error) {
	if id, ok := svc.users[in.GetEmail()]; ok {
		switch in.Type {
		default:
			return &mainflux.Token{Value: id}, nil
		}
	}
	return nil, users.ErrUnauthorizedAccess
}
