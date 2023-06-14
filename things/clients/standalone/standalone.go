// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package standalone

import (
	"context"

	"github.com/mainflux/mainflux/pkg/errors"
	"github.com/mainflux/mainflux/users/policies"
	"google.golang.org/grpc"
)

var errUnsupported = errors.New("not supported in standalone mode")

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

func (repo singleUserRepo) Issue(ctx context.Context, req *policies.IssueReq, opts ...grpc.CallOption) (*policies.Token, error) {
	return &policies.Token{}, errUnsupported
}

func (repo singleUserRepo) Identify(ctx context.Context, token *policies.Token, opts ...grpc.CallOption) (*policies.UserIdentity, error) {
	if repo.token != token.GetValue() {
		return nil, errors.ErrAuthentication
	}

	return &policies.UserIdentity{Id: repo.id}, nil
}

func (repo singleUserRepo) Authorize(ctx context.Context, req *policies.AuthorizeReq, _ ...grpc.CallOption) (r *policies.AuthorizeRes, err error) {
	if repo.id != req.GetSub() {
		return &policies.AuthorizeRes{}, errors.ErrAuthorization
	}
	return &policies.AuthorizeRes{Authorized: true}, nil
}

func (repo singleUserRepo) AddPolicy(ctx context.Context, req *policies.AddPolicyReq, opts ...grpc.CallOption) (*policies.AddPolicyRes, error) {
	if repo.token != req.GetToken() {
		return &policies.AddPolicyRes{}, errors.ErrAuthorization
	}
	return &policies.AddPolicyRes{Authorized: true}, nil
}

func (repo singleUserRepo) DeletePolicy(ctx context.Context, req *policies.DeletePolicyReq, opts ...grpc.CallOption) (*policies.DeletePolicyRes, error) {
	if repo.token != req.GetToken() {
		return &policies.DeletePolicyRes{}, errors.ErrAuthorization
	}
	return &policies.DeletePolicyRes{Deleted: true}, nil
}

func (repo singleUserRepo) ListPolicies(ctx context.Context, in *policies.ListPoliciesReq, opts ...grpc.CallOption) (*policies.ListPoliciesRes, error) {
	return &policies.ListPoliciesRes{}, errUnsupported
}
