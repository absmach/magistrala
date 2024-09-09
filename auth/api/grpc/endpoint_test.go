// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package grpc_test

import (
	"context"
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/absmach/magistrala"
	"github.com/absmach/magistrala/auth"
	grpcapi "github.com/absmach/magistrala/auth/api/grpc"
	client "github.com/absmach/magistrala/internal/auth"
	"github.com/absmach/magistrala/internal/testsutil"
	"github.com/absmach/magistrala/pkg/apiutil"
	"github.com/absmach/magistrala/pkg/errors"
	svcerr "github.com/absmach/magistrala/pkg/errors/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

const (
	port            = 8081
	secret          = "secret"
	email           = "test@example.com"
	id              = "testID"
	thingsType      = "things"
	usersType       = "users"
	description     = "Description"
	groupName       = "mgx"
	adminpermission = "admin"

	authoritiesObj  = "authorities"
	memberRelation  = "member"
	loginDuration   = 30 * time.Minute
	refreshDuration = 24 * time.Hour
	invalidDuration = 7 * 24 * time.Hour
	validToken      = "valid"
	inValidToken    = "invalid"
	validPolicy     = "valid"
)

var (
	validID  = testsutil.GenerateUUID(&testing.T{})
	domainID = testsutil.GenerateUUID(&testing.T{})
	authAddr = fmt.Sprintf("localhost:%d", port)
)

func startGRPCServer(svc auth.Service, port int) {
	listener, _ := net.Listen("tcp", fmt.Sprintf(":%d", port))
	server := grpc.NewServer()
	magistrala.RegisterAuthzServiceServer(server, grpcapi.NewAuthzServer(svc))
	magistrala.RegisterAuthnServiceServer(server, grpcapi.NewAuthnServer(svc))
	magistrala.RegisterPolicyServiceServer(server, grpcapi.NewPolicyServer(svc))
	go func() {
		err := server.Serve(listener)
		assert.Nil(&testing.T{}, err, fmt.Sprintf(`"Unexpected error creating auth server %s"`, err))
	}()
}

func TestIssue(t *testing.T) {
	conn, err := grpc.NewClient(authAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	assert.Nil(t, err, fmt.Sprintf("Unexpected error creating client connection %s", err))
	client := client.NewAuthClient(conn, time.Second)

	cases := []struct {
		desc          string
		userId        string
		domainID      string
		kind          auth.KeyType
		issueResponse auth.Token
		err           error
	}{
		{
			desc:     "issue for user with valid token",
			userId:   validID,
			domainID: domainID,
			kind:     auth.AccessKey,
			issueResponse: auth.Token{
				AccessToken:  validToken,
				RefreshToken: validToken,
			},
			err: nil,
		},
		{
			desc:     "issue recovery key",
			userId:   validID,
			domainID: domainID,
			kind:     auth.RecoveryKey,
			issueResponse: auth.Token{
				AccessToken:  validToken,
				RefreshToken: validToken,
			},
			err: nil,
		},
		{
			desc:          "issue API key unauthenticated",
			userId:        validID,
			domainID:      domainID,
			kind:          auth.APIKey,
			issueResponse: auth.Token{},
			err:           svcerr.ErrAuthentication,
		},
		{
			desc:          "issue for invalid key type",
			userId:        validID,
			domainID:      domainID,
			kind:          32,
			issueResponse: auth.Token{},
			err:           errors.ErrMalformedEntity,
		},
		{
			desc:          "issue for user that does notexist",
			userId:        "",
			domainID:      "",
			kind:          auth.APIKey,
			issueResponse: auth.Token{},
			err:           svcerr.ErrAuthentication,
		},
	}

	for _, tc := range cases {
		svcCall := svc.On("Issue", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(tc.issueResponse, tc.err)
		_, err := client.Issue(context.Background(), &magistrala.IssueReq{UserId: tc.userId, DomainId: &tc.domainID, Type: uint32(tc.kind)})
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		svcCall.Unset()
	}
}

func TestRefresh(t *testing.T) {
	conn, err := grpc.NewClient(authAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	assert.Nil(t, err, fmt.Sprintf("Unexpected error creating client connection %s", err))
	client := client.NewAuthClient(conn, time.Second)

	cases := []struct {
		desc          string
		token         string
		domainID      string
		issueResponse auth.Token
		err           error
	}{
		{
			desc:     "refresh token with valid token",
			token:    validToken,
			domainID: domainID,
			issueResponse: auth.Token{
				AccessToken:  validToken,
				RefreshToken: validToken,
			},
			err: nil,
		},
		{
			desc:          "refresh token with invalid token",
			token:         inValidToken,
			domainID:      domainID,
			issueResponse: auth.Token{},
			err:           svcerr.ErrAuthentication,
		},
		{
			desc:          "refresh token with empty token",
			token:         "",
			domainID:      domainID,
			issueResponse: auth.Token{},
			err:           apiutil.ErrMissingSecret,
		},
	}

	for _, tc := range cases {
		svcCall := svc.On("Issue", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(tc.issueResponse, tc.err)
		_, err := client.Refresh(context.Background(), &magistrala.RefreshReq{DomainId: &tc.domainID, RefreshToken: tc.token})
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		svcCall.Unset()
	}
}

func TestIdentify(t *testing.T) {
	conn, err := grpc.NewClient(authAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	assert.Nil(t, err, fmt.Sprintf("Unexpected error creating client connection %s", err))
	client := client.NewAuthClient(conn, time.Second)

	cases := []struct {
		desc   string
		token  string
		idt    *magistrala.IdentityRes
		svcErr error
		err    error
	}{
		{
			desc:  "identify user with valid user token",
			token: validToken,
			idt:   &magistrala.IdentityRes{Id: id, UserId: email, DomainId: domainID},
			err:   nil,
		},
		{
			desc:   "identify user with invalid user token",
			token:  "invalid",
			idt:    &magistrala.IdentityRes{},
			svcErr: svcerr.ErrAuthentication,
			err:    svcerr.ErrAuthentication,
		},
		{
			desc:  "identify user with empty token",
			token: "",
			idt:   &magistrala.IdentityRes{},
			err:   apiutil.ErrBearerToken,
		},
	}

	for _, tc := range cases {
		svcCall := svc.On("Identify", mock.Anything, mock.Anything, mock.Anything).Return(auth.Key{Subject: id, User: email, Domain: domainID}, tc.svcErr)
		idt, err := client.Identify(context.Background(), &magistrala.IdentityReq{Token: tc.token})
		if idt != nil {
			assert.Equal(t, tc.idt, idt, fmt.Sprintf("%s: expected %v got %v", tc.desc, tc.idt, idt))
		}
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		svcCall.Unset()
	}
}

func TestAuthorize(t *testing.T) {
	conn, err := grpc.NewClient(authAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	assert.Nil(t, err, fmt.Sprintf("Unexpected error creating client connection %s", err))
	client := client.NewAuthClient(conn, time.Second)

	cases := []struct {
		desc         string
		token        string
		authRequest  *magistrala.AuthorizeReq
		authResponse *magistrala.AuthorizeRes
		err          error
	}{
		{
			desc:  "authorize user with authorized token",
			token: validToken,
			authRequest: &magistrala.AuthorizeReq{
				Subject:     id,
				SubjectType: usersType,
				Object:      authoritiesObj,
				ObjectType:  usersType,
				Relation:    memberRelation,
				Permission:  adminpermission,
			},
			authResponse: &magistrala.AuthorizeRes{Authorized: true},
			err:          nil,
		},
		{
			desc:  "authorize user with unauthorized token",
			token: inValidToken,
			authRequest: &magistrala.AuthorizeReq{
				Subject:     id,
				SubjectType: usersType,
				Object:      authoritiesObj,
				ObjectType:  usersType,
				Relation:    memberRelation,
				Permission:  adminpermission,
			},
			authResponse: &magistrala.AuthorizeRes{Authorized: false},
			err:          svcerr.ErrAuthorization,
		},
		{
			desc:  "authorize user with empty subject",
			token: validToken,
			authRequest: &magistrala.AuthorizeReq{
				Subject:     "",
				SubjectType: usersType,
				Object:      authoritiesObj,
				ObjectType:  usersType,
				Relation:    memberRelation,
				Permission:  adminpermission,
			},
			authResponse: &magistrala.AuthorizeRes{Authorized: false},
			err:          apiutil.ErrMissingPolicySub,
		},
		{
			desc:  "authorize user with empty subject type",
			token: validToken,
			authRequest: &magistrala.AuthorizeReq{
				Subject:     id,
				SubjectType: "",
				Object:      authoritiesObj,
				ObjectType:  usersType,
				Relation:    memberRelation,
				Permission:  adminpermission,
			},
			authResponse: &magistrala.AuthorizeRes{Authorized: false},
			err:          apiutil.ErrMissingPolicySub,
		},
		{
			desc:  "authorize user with empty object",
			token: validToken,
			authRequest: &magistrala.AuthorizeReq{
				Subject:     id,
				SubjectType: usersType,
				Object:      "",
				ObjectType:  usersType,
				Relation:    memberRelation,
				Permission:  adminpermission,
			},
			authResponse: &magistrala.AuthorizeRes{Authorized: false},
			err:          apiutil.ErrMissingPolicyObj,
		},
		{
			desc:  "authorize user with empty object type",
			token: validToken,
			authRequest: &magistrala.AuthorizeReq{
				Subject:     id,
				SubjectType: usersType,
				Object:      authoritiesObj,
				ObjectType:  "",
				Relation:    memberRelation,
				Permission:  adminpermission,
			},
			authResponse: &magistrala.AuthorizeRes{Authorized: false},
			err:          apiutil.ErrMissingPolicyObj,
		},
		{
			desc:  "authorize user with empty permission",
			token: validToken,
			authRequest: &magistrala.AuthorizeReq{
				Subject:     id,
				SubjectType: usersType,
				Object:      authoritiesObj,
				ObjectType:  usersType,
				Relation:    memberRelation,
				Permission:  "",
			},
			authResponse: &magistrala.AuthorizeRes{Authorized: false},
			err:          apiutil.ErrMalformedPolicyPer,
		},
	}
	for _, tc := range cases {
		svccall := svc.On("Authorize", mock.Anything, mock.Anything).Return(tc.err)
		ar, err := client.Authorize(context.Background(), tc.authRequest)
		if ar != nil {
			assert.Equal(t, tc.authResponse, ar, fmt.Sprintf("%s: expected %v got %v", tc.desc, tc.authResponse, ar))
		}
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		svccall.Unset()
	}
}

func TestDeleteUserPolicies(t *testing.T) {
	conn, err := grpc.NewClient(authAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	assert.Nil(t, err, fmt.Sprintf("Unexpected error creating client connection %s", err))
	client := grpcapi.NewPolicyClient(conn, time.Second)

	cases := []struct {
		desc                    string
		token                   string
		deleteUserPoliciesReq *magistrala.DeleteUserPoliciesReq
		deletePolicyRes         *magistrala.DeletePolicyRes
		err                     error
	}{
		{
			desc:  "delete valid req",
			token: validToken,
			deleteUserPoliciesReq: &magistrala.DeleteUserPoliciesReq{
				Id: id,
			},
			deletePolicyRes: &magistrala.DeletePolicyRes{Deleted: true},
			err:             nil,
		},
		{
			desc:                    "delete invalid req with invalid token",
			token:                   inValidToken,
			deleteUserPoliciesReq: &magistrala.DeleteUserPoliciesReq{},
			deletePolicyRes:         &magistrala.DeletePolicyRes{Deleted: false},
			err:                     apiutil.ErrMissingID,
		},
		{
			desc:  "delete invalid req with invalid token",
			token: inValidToken,
			deleteUserPoliciesReq: &magistrala.DeleteUserPoliciesReq{
				Id: id,
			},
			deletePolicyRes: &magistrala.DeletePolicyRes{Deleted: false},
			err:             apiutil.ErrMissingPolicyEntityType,
		},
	}
	for _, tc := range cases {
		repoCall := svc.On("DeleteUserPolicies", mock.Anything, tc.deleteUserPoliciesReq.Id).Return(tc.err)
		dpr, err := client.DeleteUserPolicies(context.Background(), tc.deleteUserPoliciesReq)
		assert.Equal(t, tc.deletePolicyRes.GetDeleted(), dpr.GetDeleted(), fmt.Sprintf("%s: expected %v got %v", tc.desc, tc.deletePolicyRes.GetDeleted(), dpr.GetDeleted()))
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		repoCall.Unset()
	}
}
