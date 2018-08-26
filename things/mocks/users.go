//
// Copyright (c) 2018
// Mainflux
//
// SPDX-License-Identifier: Apache-2.0
//

package mocks

import (
	"context"

	"github.com/mainflux/mainflux"
	"github.com/mainflux/mainflux/users"
	"google.golang.org/grpc"
)

var _ mainflux.UsersServiceClient = (*usersServiceMock)(nil)

type usersServiceMock struct {
	users map[string]string
}

// NewUsersService creates mock of users service.
func NewUsersService(users map[string]string) mainflux.UsersServiceClient {
	return &usersServiceMock{users}
}

func (svc usersServiceMock) Identify(ctx context.Context, in *mainflux.Token, opts ...grpc.CallOption) (*mainflux.UserID, error) {
	if id, ok := svc.users[in.Value]; ok {
		return &mainflux.UserID{Value: id}, nil
	}
	return nil, users.ErrUnauthorizedAccess
}
