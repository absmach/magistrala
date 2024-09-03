// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package standalone

import (
	"context"

	"github.com/absmach/magistrala"
	grpcclient "github.com/absmach/magistrala/auth/api/grpc"
	svcerr "github.com/absmach/magistrala/pkg/errors/service"
	"google.golang.org/grpc"
)

var (
	_ grpcclient.AuthServiceClient   = (*singleUserAuth)(nil)
	_ magistrala.PolicyServiceClient = (*singleUserPolicyClient)(nil)
)

type singleUserAuth struct {
	id    string
	token string
}

// NewAuthService creates single user repository for constrained environments.
func NewAuthService(id, token string) grpcclient.AuthServiceClient {
	return singleUserAuth{
		id:    id,
		token: token,
	}
}

func (repo singleUserAuth) Login(ctx context.Context, in *magistrala.IssueReq, opts ...grpc.CallOption) (*magistrala.Token, error) {
	return nil, nil
}

func (repo singleUserAuth) Refresh(ctx context.Context, in *magistrala.RefreshReq, opts ...grpc.CallOption) (*magistrala.Token, error) {
	return nil, nil
}

func (repo singleUserAuth) Issue(ctx context.Context, in *magistrala.IssueReq, opts ...grpc.CallOption) (*magistrala.Token, error) {
	return nil, nil
}

func (repo singleUserAuth) Identify(ctx context.Context, in *magistrala.IdentityReq, opts ...grpc.CallOption) (*magistrala.IdentityRes, error) {
	if repo.token != in.GetToken() {
		return nil, svcerr.ErrAuthentication
	}

	return &magistrala.IdentityRes{Id: repo.id}, nil
}

func (repo singleUserAuth) Authorize(ctx context.Context, in *magistrala.AuthorizeReq, opts ...grpc.CallOption) (*magistrala.AuthorizeRes, error) {
	if repo.id != in.Subject {
		return &magistrala.AuthorizeRes{Authorized: false}, svcerr.ErrAuthorization
	}

	return &magistrala.AuthorizeRes{Authorized: true}, nil
}

type singleUserPolicyClient struct {
	id    string
	token string
}

// NewPolicyService creates single user policy service for constrained environments.
func NewPolicyService(id, token string) magistrala.PolicyServiceClient {
	return singleUserPolicyClient{
		id:    id,
		token: token,
	}
}

func (repo singleUserPolicyClient) AddPolicy(ctx context.Context, in *magistrala.AddPolicyReq, opts ...grpc.CallOption) (*magistrala.AddPolicyRes, error) {
	return nil, nil
}

func (repo singleUserPolicyClient) AddPolicies(ctx context.Context, in *magistrala.AddPoliciesReq, opts ...grpc.CallOption) (*magistrala.AddPoliciesRes, error) {
	return nil, nil
}

func (repo singleUserPolicyClient) DeletePolicyFilter(ctx context.Context, in *magistrala.DeletePolicyFilterReq, opts ...grpc.CallOption) (*magistrala.DeletePolicyRes, error) {
	return nil, nil
}

func (repo singleUserPolicyClient) DeletePolicies(ctx context.Context, in *magistrala.DeletePoliciesReq, opts ...grpc.CallOption) (*magistrala.DeletePolicyRes, error) {
	return nil, nil
}

func (repo singleUserPolicyClient) ListObjects(ctx context.Context, in *magistrala.ListObjectsReq, opts ...grpc.CallOption) (*magistrala.ListObjectsRes, error) {
	return nil, nil
}

func (repo singleUserPolicyClient) ListAllObjects(ctx context.Context, in *magistrala.ListObjectsReq, opts ...grpc.CallOption) (*magistrala.ListObjectsRes, error) {
	return nil, nil
}

func (repo singleUserPolicyClient) CountObjects(ctx context.Context, in *magistrala.CountObjectsReq, opts ...grpc.CallOption) (*magistrala.CountObjectsRes, error) {
	return nil, nil
}

func (repo singleUserPolicyClient) ListSubjects(ctx context.Context, in *magistrala.ListSubjectsReq, opts ...grpc.CallOption) (*magistrala.ListSubjectsRes, error) {
	return nil, nil
}

func (repo singleUserPolicyClient) ListAllSubjects(ctx context.Context, in *magistrala.ListSubjectsReq, opts ...grpc.CallOption) (*magistrala.ListSubjectsRes, error) {
	return nil, nil
}

func (repo singleUserPolicyClient) CountSubjects(ctx context.Context, in *magistrala.CountSubjectsReq, opts ...grpc.CallOption) (*magistrala.CountSubjectsRes, error) {
	return nil, nil
}

func (repo singleUserPolicyClient) ListPermissions(ctx context.Context, in *magistrala.ListPermissionsReq, opts ...grpc.CallOption) (*magistrala.ListPermissionsRes, error) {
	return nil, nil
}

func (repo singleUserPolicyClient) DeleteEntityPolicies(ctx context.Context, in *magistrala.DeleteEntityPoliciesReq, opts ...grpc.CallOption) (*magistrala.DeletePolicyRes, error) {
	return nil, nil
}
