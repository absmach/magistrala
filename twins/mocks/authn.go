// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package mocks

import (
	"context"

	"github.com/mainflux/mainflux"
	"github.com/mainflux/mainflux/users"
	"google.golang.org/grpc"
)

var _ mainflux.AuthNServiceClient = (*authNServiceClient)(nil)

type authNServiceClient struct {
	users map[string]string
}

// NewAuthNServiceClient creates mock of auth service.
func NewAuthNServiceClient(users map[string]string) mainflux.AuthNServiceClient {
	return &authNServiceClient{users}
}

func (svc authNServiceClient) Identify(ctx context.Context, in *mainflux.Token, opts ...grpc.CallOption) (*mainflux.UserID, error) {
	if id, ok := svc.users[in.Value]; ok {
		return &mainflux.UserID{Value: id}, nil
	}
	return nil, users.ErrUnauthorizedAccess
}

func (c *authNServiceClient) Issue(ctx context.Context, in *mainflux.IssueReq, opts ...grpc.CallOption) (*mainflux.Token, error) {
	return new(mainflux.Token), nil
}
