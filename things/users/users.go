// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

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

var _ mainflux.AuthNServiceClient = (*singleUserRepo)(nil)

type singleUserRepo struct {
	email string
	token string
}

// NewSingleUserService creates single user repository for constrained environments.
func NewSingleUserService(email, token string) mainflux.AuthNServiceClient {
	return singleUserRepo{
		email: email,
		token: token,
	}
}

func (repo singleUserRepo) Issue(ctx context.Context, req *mainflux.IssueReq, opts ...grpc.CallOption) (*mainflux.Token, error) {
	ctx, cancel := context.WithTimeout(ctx, time.Second)
	defer cancel()

	if repo.token != req.GetIssuer() {
		return nil, things.ErrUnauthorizedAccess
	}

	return &mainflux.Token{Value: repo.token}, nil
}

func (repo singleUserRepo) Identify(ctx context.Context, token *mainflux.Token, opts ...grpc.CallOption) (*mainflux.UserID, error) {
	ctx, cancel := context.WithTimeout(ctx, time.Second)
	defer cancel()

	if repo.token != token.GetValue() {
		return nil, things.ErrUnauthorizedAccess
	}

	return &mainflux.UserID{Value: repo.email}, nil
}
