// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package standalone

import (
	"context"

	"github.com/absmach/magistrala"
	authclient "github.com/absmach/magistrala/pkg/auth"
	svcerr "github.com/absmach/magistrala/pkg/errors/service"
	"github.com/absmach/magistrala/pkg/policies"
	"google.golang.org/grpc"
)

var (
	_ authclient.AuthClient = (*singleUserAuth)(nil)
	_ policies.PolicyClient   = (*singleUserPolicyClient)(nil)
)

type singleUserAuth struct {
	id    string
	token string
}

// NewAuthClient creates single user auth client for constrained environments.
func NewAuthClient(id, token string) authclient.AuthClient {
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

// NewPolicyClient creates single user policies client for constrained environments.
func NewPolicyClient(id, token string) policies.PolicyClient {
	return singleUserPolicyClient{
		id:    id,
		token: token,
	}
}

func (repo singleUserPolicyClient) AddPolicy(ctx context.Context, pr policies.PolicyReq) error {
	return nil
}

func (repo singleUserPolicyClient) AddPolicies(ctx context.Context, prs []policies.PolicyReq) error {
	return nil
}

func (repo singleUserPolicyClient) DeletePolicyFilter(ctx context.Context, pr policies.PolicyReq) error {
	return nil
}

func (repo singleUserPolicyClient) DeletePolicies(ctx context.Context, prs []policies.PolicyReq) error {
	return nil
}

func (repo singleUserPolicyClient) ListObjects(ctx context.Context, pr policies.PolicyReq, nextPageToken string, limit uint64) (policies.PolicyPage, error) {
	return policies.PolicyPage{}, nil
}

func (repo singleUserPolicyClient) ListAllObjects(ctx context.Context, pr policies.PolicyReq) (policies.PolicyPage, error) {
	return policies.PolicyPage{}, nil
}

func (repo singleUserPolicyClient) CountObjects(ctx context.Context, pr policies.PolicyReq) (uint64, error) {
	return 0, nil
}

func (repo singleUserPolicyClient) ListSubjects(ctx context.Context, pr policies.PolicyReq, nextPageToken string, limit uint64) (policies.PolicyPage, error) {
	return policies.PolicyPage{}, nil
}

func (repo singleUserPolicyClient) ListAllSubjects(ctx context.Context, pr policies.PolicyReq) (policies.PolicyPage, error) {
	return policies.PolicyPage{}, nil
}

func (repo singleUserPolicyClient) CountSubjects(ctx context.Context, pr policies.PolicyReq) (uint64, error) {
	return 0, nil
}

func (repo singleUserPolicyClient) ListPermissions(ctx context.Context, pr policies.PolicyReq, permissionsFilter []string) (policies.Permissions, error) {
	return nil, nil
}

func (repo singleUserPolicyClient) DeleteEntityPolicies(ctx context.Context, entityType, id string) error {
	return nil
}
