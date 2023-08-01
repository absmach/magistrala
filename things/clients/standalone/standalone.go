// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package standalone

import (
	"context"

	"github.com/mainflux/mainflux/pkg/errors"
	"github.com/mainflux/mainflux/users/policies"
	"google.golang.org/grpc"
)

var _ policies.AuthServiceClient = (*singleUserRepo)(nil)

type singleUserRepo struct {
	id    string
	token string
}

// NewAuthService creates single user repository for constrained environments.
func NewAuthService(id, token string) policies.AuthServiceClient {
	return singleUserRepo{
		id:    id,
		token: token,
	}
}

func (repo singleUserRepo) Identify(ctx context.Context, req *policies.IdentifyReq, opts ...grpc.CallOption) (*policies.IdentifyRes, error) {
	if repo.token != req.GetToken() {
		return nil, errors.ErrAuthentication
	}

	return &policies.IdentifyRes{Id: repo.id}, nil
}

func (repo singleUserRepo) Authorize(ctx context.Context, req *policies.AuthorizeReq, _ ...grpc.CallOption) (r *policies.AuthorizeRes, err error) {
	if repo.id != req.GetSubject() {
		return &policies.AuthorizeRes{}, errors.ErrAuthorization
	}

	return &policies.AuthorizeRes{Authorized: true}, nil
}
