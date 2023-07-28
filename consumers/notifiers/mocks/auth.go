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

func (svc authServiceMock) Identify(ctx context.Context, req *policies.IdentifyReq, opts ...grpc.CallOption) (*policies.IdentifyRes, error) {
	if id, ok := svc.users[req.GetToken()]; ok {
		return &policies.IdentifyRes{Id: id}, nil
	}
	return nil, errors.ErrAuthentication
}

func (svc authServiceMock) Authorize(ctx context.Context, req *policies.AuthorizeReq, _ ...grpc.CallOption) (r *policies.AuthorizeRes, err error) {
	panic("not implemented")
}
