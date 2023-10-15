// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package standalone

import (
	"context"

	"github.com/mainflux/mainflux"
	"github.com/mainflux/mainflux/pkg/errors"
	"google.golang.org/grpc"
)

var _ mainflux.AuthServiceClient = (*singleUserRepo)(nil)

type singleUserRepo struct {
	id    string
	token string
}

// NewAuthService creates single user repository for constrained environments.
func NewAuthService(id, token string) mainflux.AuthServiceClient {
	return singleUserRepo{
		id:    id,
		token: token,
	}
}

func (repo singleUserRepo) Login(ctx context.Context, in *mainflux.LoginReq, opts ...grpc.CallOption) (*mainflux.Token, error) {
	return nil, nil
}
func (repo singleUserRepo) Refresh(ctx context.Context, in *mainflux.RefreshReq, opts ...grpc.CallOption) (*mainflux.Token, error) {
	return nil, nil
}

func (repo singleUserRepo) Issue(ctx context.Context, in *mainflux.IssueReq, opts ...grpc.CallOption) (*mainflux.Token, error) {
	return nil, nil
}

func (repo singleUserRepo) Identify(ctx context.Context, in *mainflux.IdentityReq, opts ...grpc.CallOption) (*mainflux.IdentityRes, error) {
	if repo.token != in.GetToken() {
		return nil, errors.ErrAuthentication
	}

	return &mainflux.IdentityRes{Id: repo.id}, nil
}

func (repo singleUserRepo) Authorize(ctx context.Context, in *mainflux.AuthorizeReq, opts ...grpc.CallOption) (*mainflux.AuthorizeRes, error) {
	if repo.id != in.Subject {
		return &mainflux.AuthorizeRes{Authorized: false}, errors.ErrAuthorization
	}

	return &mainflux.AuthorizeRes{Authorized: true}, nil
}

func (repo singleUserRepo) AddPolicy(ctx context.Context, in *mainflux.AddPolicyReq, opts ...grpc.CallOption) (*mainflux.AddPolicyRes, error) {
	return nil, nil
}
func (repo singleUserRepo) DeletePolicy(ctx context.Context, in *mainflux.DeletePolicyReq, opts ...grpc.CallOption) (*mainflux.DeletePolicyRes, error) {
	return nil, nil
}
func (repo singleUserRepo) ListObjects(ctx context.Context, in *mainflux.ListObjectsReq, opts ...grpc.CallOption) (*mainflux.ListObjectsRes, error) {
	return nil, nil
}
func (repo singleUserRepo) ListAllObjects(ctx context.Context, in *mainflux.ListObjectsReq, opts ...grpc.CallOption) (*mainflux.ListObjectsRes, error) {
	return nil, nil
}
func (repo singleUserRepo) CountObjects(ctx context.Context, in *mainflux.CountObjectsReq, opts ...grpc.CallOption) (*mainflux.CountObjectsRes, error) {
	return nil, nil
}
func (repo singleUserRepo) ListSubjects(ctx context.Context, in *mainflux.ListSubjectsReq, opts ...grpc.CallOption) (*mainflux.ListSubjectsRes, error) {
	return nil, nil
}
func (repo singleUserRepo) ListAllSubjects(ctx context.Context, in *mainflux.ListSubjectsReq, opts ...grpc.CallOption) (*mainflux.ListSubjectsRes, error) {
	return nil, nil
}
func (repo singleUserRepo) CountSubjects(ctx context.Context, in *mainflux.CountSubjectsReq, opts ...grpc.CallOption) (*mainflux.CountSubjectsRes, error) {
	return nil, nil
}
