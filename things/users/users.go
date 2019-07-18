//
// Copyright (c) 2019
// Mainflux
//
// SPDX-License-Identifier: Apache-2.0
//

// Package users contains implementation for users service in
// single user scenario.
package users

import (
	"context"
	"time"

	"github.com/mainflux/mainflux/things"

	"github.com/mainflux/mainflux"
	"google.golang.org/grpc"
)

var _ mainflux.UsersServiceClient = (*singleUserRepo)(nil)

type singleUserRepo struct {
	email string
	token string
}

// NewSingleUserService creates single user repository for constraind environments.
func NewSingleUserService(email, token string) mainflux.UsersServiceClient {
	return singleUserRepo{
		email: email,
		token: token,
	}
}

func (repo singleUserRepo) Identify(ctx context.Context, token *mainflux.Token, opts ...grpc.CallOption) (*mainflux.UserID, error) {
	ctx, cancel := context.WithTimeout(ctx, time.Second)
	defer cancel()

	if repo.token != token.GetValue() {
		return nil, things.ErrUnauthorizedAccess
	}

	return &mainflux.UserID{Value: repo.email}, nil
}
