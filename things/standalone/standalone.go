// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package standalone

import (
	"context"

	"github.com/absmach/magistrala"
	svcerr "github.com/absmach/magistrala/pkg/errors"
	"google.golang.org/grpc"
)

var _ magistrala.AuthServiceClient = (*singleUserRepo)(nil)

type singleUserRepo struct {
	id    string
	token string
}

// NewAuthService creates single user repository for constrained environments.
func NewAuthService(id, token string) magistrala.AuthServiceClient {
	return singleUserRepo{
		id:    id,
		token: token,
	}
}

func (repo singleUserRepo) Login(ctx context.Context, in *magistrala.IssueReq, opts ...grpc.CallOption) (*magistrala.Token, error) {
	return nil, nil
}

func (repo singleUserRepo) Refresh(ctx context.Context, in *magistrala.RefreshReq, opts ...grpc.CallOption) (*magistrala.Token, error) {
	return nil, nil
}

func (repo singleUserRepo) Issue(ctx context.Context, in *magistrala.IssueReq, opts ...grpc.CallOption) (*magistrala.Token, error) {
	return nil, nil
}

func (repo singleUserRepo) Identify(ctx context.Context, in *magistrala.IdentityReq, opts ...grpc.CallOption) (*magistrala.IdentityRes, error) {
	if repo.token != in.GetToken() {
		return nil, svcerr.ErrAuthentication
	}

	return &magistrala.IdentityRes{Id: repo.id}, nil
}

func (repo singleUserRepo) Authorize(ctx context.Context, in *magistrala.AuthorizeReq, opts ...grpc.CallOption) (*magistrala.AuthorizeRes, error) {
	if repo.id != in.Subject {
		return &magistrala.AuthorizeRes{Authorized: false}, svcerr.ErrAuthorization
	}

	return &magistrala.AuthorizeRes{Authorized: true}, nil
}

func (repo singleUserRepo) AddPolicy(ctx context.Context, in *magistrala.AddPolicyReq, opts ...grpc.CallOption) (*magistrala.AddPolicyRes, error) {
	return nil, nil
}

func (repo singleUserRepo) AddPolicies(ctx context.Context, in *magistrala.AddPoliciesReq, opts ...grpc.CallOption) (*magistrala.AddPoliciesRes, error) {
	return nil, nil
}

func (repo singleUserRepo) DeletePolicy(ctx context.Context, in *magistrala.DeletePolicyReq, opts ...grpc.CallOption) (*magistrala.DeletePolicyRes, error) {
	return nil, nil
}

func (repo singleUserRepo) DeletePolicies(ctx context.Context, in *magistrala.DeletePoliciesReq, opts ...grpc.CallOption) (*magistrala.DeletePoliciesRes, error) {
	return nil, nil
}

func (repo singleUserRepo) ListObjects(ctx context.Context, in *magistrala.ListObjectsReq, opts ...grpc.CallOption) (*magistrala.ListObjectsRes, error) {
	return nil, nil
}

func (repo singleUserRepo) ListAllObjects(ctx context.Context, in *magistrala.ListObjectsReq, opts ...grpc.CallOption) (*magistrala.ListObjectsRes, error) {
	return nil, nil
}

func (repo singleUserRepo) CountObjects(ctx context.Context, in *magistrala.CountObjectsReq, opts ...grpc.CallOption) (*magistrala.CountObjectsRes, error) {
	return nil, nil
}

func (repo singleUserRepo) ListSubjects(ctx context.Context, in *magistrala.ListSubjectsReq, opts ...grpc.CallOption) (*magistrala.ListSubjectsRes, error) {
	return nil, nil
}

func (repo singleUserRepo) ListAllSubjects(ctx context.Context, in *magistrala.ListSubjectsReq, opts ...grpc.CallOption) (*magistrala.ListSubjectsRes, error) {
	return nil, nil
}

func (repo singleUserRepo) CountSubjects(ctx context.Context, in *magistrala.CountSubjectsReq, opts ...grpc.CallOption) (*magistrala.CountSubjectsRes, error) {
	return nil, nil
}

func (repo singleUserRepo) ListPermissions(ctx context.Context, in *magistrala.ListPermissionsReq, opts ...grpc.CallOption) (*magistrala.ListPermissionsRes, error) {
	return nil, nil
}
