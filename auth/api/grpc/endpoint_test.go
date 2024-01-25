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
	"github.com/absmach/magistrala/internal/apiutil"
	"github.com/absmach/magistrala/internal/testsutil"
	"github.com/absmach/magistrala/pkg/errors"
	svcerr "github.com/absmach/magistrala/pkg/errors/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
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
	magistrala.RegisterAuthServiceServer(server, grpcapi.NewServer(svc))
	go func() {
		err := server.Serve(listener)
		assert.Nil(&testing.T{}, err, fmt.Sprintf(`"Unexpected error creating server %s"`, err))
	}()
}

func TestIssue(t *testing.T) {
	conn, err := grpc.Dial(authAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	assert.Nil(t, err, fmt.Sprintf("Unexpected error creating client connection %s", err))
	client := grpcapi.NewClient(conn, time.Second)

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
			err:           errors.ErrAuthentication,
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
			err:           errors.ErrAuthentication,
		},
	}

	for _, tc := range cases {
		repoCall := svc.On("Issue", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(tc.issueResponse, tc.err)
		_, err := client.Issue(context.Background(), &magistrala.IssueReq{UserId: tc.userId, DomainId: &tc.domainID, Type: uint32(tc.kind)})
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		repoCall.Unset()
	}
}

func TestRefresh(t *testing.T) {
	conn, err := grpc.Dial(authAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	assert.Nil(t, err, fmt.Sprintf("Unexpected error creating client connection %s", err))
	client := grpcapi.NewClient(conn, time.Second)

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
			err:           errors.ErrAuthentication,
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
		repoCall := svc.On("Issue", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(tc.issueResponse, tc.err)
		_, err := client.Refresh(context.Background(), &magistrala.RefreshReq{DomainId: &tc.domainID, RefreshToken: tc.token})
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		repoCall.Unset()
	}
}

func TestIdentify(t *testing.T) {
	conn, err := grpc.Dial(authAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	assert.Nil(t, err, fmt.Sprintf("Unexpected error creating client connection %s", err))
	client := grpcapi.NewClient(conn, time.Second)

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
		repoCall := svc.On("Identify", mock.Anything, mock.Anything, mock.Anything).Return(auth.Key{Subject: id, User: email, Domain: domainID}, tc.svcErr)
		idt, err := client.Identify(context.Background(), &magistrala.IdentityReq{Token: tc.token})
		if idt != nil {
			assert.Equal(t, tc.idt, idt, fmt.Sprintf("%s: expected %v got %v", tc.desc, tc.idt, idt))
		}
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		repoCall.Unset()
	}
}

func TestAuthorize(t *testing.T) {
	conn, err := grpc.Dial(authAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	assert.Nil(t, err, fmt.Sprintf("Unexpected error creating client connection %s", err))
	client := grpcapi.NewClient(conn, time.Second)

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
		repocall := svc.On("Authorize", mock.Anything, mock.Anything).Return(tc.err)
		ar, err := client.Authorize(context.Background(), tc.authRequest)
		if ar != nil {
			assert.Equal(t, tc.authResponse, ar, fmt.Sprintf("%s: expected %v got %v", tc.desc, tc.authResponse, ar))
		}
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		repocall.Unset()
	}
}

func TestAddPolicy(t *testing.T) {
	conn, err := grpc.Dial(authAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	assert.Nil(t, err, fmt.Sprintf("Unexpected error creating client connection %s", err))
	client := grpcapi.NewClient(conn, time.Second)

	groupAdminObj := "groupadmin"

	cases := []struct {
		desc         string
		token        string
		addPolicyReq *magistrala.AddPolicyReq
		addPolicyRes *magistrala.AddPolicyRes
		err          error
	}{
		{
			desc:  "add groupadmin policy to user",
			token: validToken,
			addPolicyReq: &magistrala.AddPolicyReq{
				Subject:     id,
				SubjectType: usersType,
				Object:      groupAdminObj,
				ObjectType:  usersType,
				Relation:    memberRelation,
				Permission:  adminpermission,
			},
			addPolicyRes: &magistrala.AddPolicyRes{Added: true},
			err:          nil,
		},
		{
			desc:  "add groupadmin policy to user with invalid token",
			token: inValidToken,
			addPolicyReq: &magistrala.AddPolicyReq{
				Subject:     id,
				SubjectType: usersType,
				Object:      groupAdminObj,
				ObjectType:  usersType,
				Relation:    memberRelation,
				Permission:  adminpermission,
			},
			addPolicyRes: &magistrala.AddPolicyRes{Added: false},
			err:          svcerr.ErrAuthorization,
		},
	}
	for _, tc := range cases {
		repoCall := svc.On("AddPolicy", mock.Anything, mock.Anything).Return(tc.err)
		apr, err := client.AddPolicy(context.Background(), tc.addPolicyReq)
		if apr != nil {
			assert.Equal(t, tc.addPolicyRes, apr, fmt.Sprintf("%s: expected %v got %v", tc.desc, tc.addPolicyRes, apr))
		}
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		repoCall.Unset()
	}
}

func TestAddPolicies(t *testing.T) {
	conn, err := grpc.Dial(authAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	assert.Nil(t, err, fmt.Sprintf("Unexpected error creating client connection %s", err))
	client := grpcapi.NewClient(conn, time.Second)

	groupAdminObj := "groupadmin"

	cases := []struct {
		desc  string
		token string
		pr    *magistrala.AddPoliciesReq
		ar    *magistrala.AddPoliciesRes
		err   error
	}{
		{
			desc:  "add groupadmin policy to user",
			token: validToken,
			pr: &magistrala.AddPoliciesReq{
				AddPoliciesReq: []*magistrala.AddPolicyReq{
					{
						Subject:     id,
						SubjectType: usersType,
						Object:      groupAdminObj,
						ObjectType:  usersType,
						Relation:    memberRelation,
						Permission:  adminpermission,
					},
				},
			},
			ar:  &magistrala.AddPoliciesRes{Added: true},
			err: nil,
		},
		{
			desc:  "add groupadmin policy to user with invalid token",
			token: inValidToken,
			pr: &magistrala.AddPoliciesReq{
				AddPoliciesReq: []*magistrala.AddPolicyReq{
					{
						Subject:     id,
						SubjectType: usersType,
						Object:      groupAdminObj,
						ObjectType:  usersType,
						Relation:    memberRelation,
						Permission:  adminpermission,
					},
				},
			},
			ar:  &magistrala.AddPoliciesRes{Added: false},
			err: svcerr.ErrAuthorization,
		},
	}
	for _, tc := range cases {
		repoCall := svc.On("AddPolicies", mock.Anything, mock.Anything).Return(tc.err)
		apr, err := client.AddPolicies(context.Background(), tc.pr)
		if apr != nil {
			assert.Equal(t, tc.ar, apr, fmt.Sprintf("%s: expected %v got %v", tc.desc, tc.ar, apr))
		}
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		repoCall.Unset()
	}
}

func TestDeletePolicy(t *testing.T) {
	conn, err := grpc.Dial(authAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	assert.Nil(t, err, fmt.Sprintf("Unexpected error creating client connection %s", err))
	client := grpcapi.NewClient(conn, time.Second)

	readRelation := "read"
	thingID := "thing"

	cases := []struct {
		desc            string
		token           string
		deletePolicyReq *magistrala.DeletePolicyReq
		deletePolicyRes *magistrala.DeletePolicyRes
		err             error
	}{
		{
			desc:  "delete valid policy",
			token: validToken,
			deletePolicyReq: &magistrala.DeletePolicyReq{
				Subject:     id,
				SubjectType: usersType,
				Object:      thingID,
				ObjectType:  thingsType,
				Relation:    readRelation,
				Permission:  readRelation,
			},
			deletePolicyRes: &magistrala.DeletePolicyRes{Deleted: true},
			err:             nil,
		},
		{
			desc:  "delete invalid policy with invalid token",
			token: inValidToken,
			deletePolicyReq: &magistrala.DeletePolicyReq{
				Subject:     id,
				SubjectType: usersType,
				Object:      thingID,
				ObjectType:  thingsType,
				Relation:    readRelation,
				Permission:  readRelation,
			},
			deletePolicyRes: &magistrala.DeletePolicyRes{Deleted: false},
			err:             svcerr.ErrAuthorization,
		},
	}
	for _, tc := range cases {
		repoCall := svc.On("DeletePolicy", mock.Anything, mock.Anything).Return(tc.err)
		dpr, err := client.DeletePolicy(context.Background(), tc.deletePolicyReq)
		assert.Equal(t, tc.deletePolicyRes.GetDeleted(), dpr.GetDeleted(), fmt.Sprintf("%s: expected %v got %v", tc.desc, tc.deletePolicyRes.GetDeleted(), dpr.GetDeleted()))
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		repoCall.Unset()
	}
}

func TestDeletePolicies(t *testing.T) {
	conn, err := grpc.Dial(authAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	assert.Nil(t, err, fmt.Sprintf("Unexpected error creating client connection %s", err))
	client := grpcapi.NewClient(conn, time.Second)

	readRelation := "read"
	thingID := "thing"

	cases := []struct {
		desc              string
		token             string
		deletePoliciesReq *magistrala.DeletePoliciesReq
		deletePoliciesRes *magistrala.DeletePoliciesRes
		err               error
	}{
		{
			desc:  "delete policies with valid token",
			token: validToken,
			deletePoliciesReq: &magistrala.DeletePoliciesReq{
				DeletePoliciesReq: []*magistrala.DeletePolicyReq{
					{
						Subject:     id,
						SubjectType: usersType,
						Object:      thingID,
						ObjectType:  thingsType,
						Relation:    readRelation,
						Permission:  readRelation,
					},
				},
			},
			deletePoliciesRes: &magistrala.DeletePoliciesRes{Deleted: true},
			err:               nil,
		},
		{
			desc:  "delete policies with invalid token",
			token: inValidToken,
			deletePoliciesReq: &magistrala.DeletePoliciesReq{
				DeletePoliciesReq: []*magistrala.DeletePolicyReq{
					{
						Subject:     id,
						SubjectType: usersType,
						Object:      thingID,
						ObjectType:  thingsType,
						Relation:    readRelation,
						Permission:  readRelation,
					},
				},
			},
			deletePoliciesRes: &magistrala.DeletePoliciesRes{Deleted: false},
			err:               svcerr.ErrAuthorization,
		},
	}
	for _, tc := range cases {
		repoCall := svc.On("DeletePolicies", mock.Anything, mock.Anything).Return(tc.err)
		apr, err := client.DeletePolicies(context.Background(), tc.deletePoliciesReq)
		if apr != nil {
			assert.Equal(t, tc.deletePoliciesRes, apr, fmt.Sprintf("%s: expected %v got %v", tc.desc, tc.deletePoliciesRes, apr))
		}
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		repoCall.Unset()
	}
}

func TestListObjects(t *testing.T) {
	conn, err := grpc.Dial(authAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	assert.Nil(t, err, fmt.Sprintf("Unexpected error creating client connection %s", err))
	client := grpcapi.NewClient(conn, time.Second)

	cases := []struct {
		desc           string
		token          string
		listObjectsReq *magistrala.ListObjectsReq
		listObjectsRes *magistrala.ListObjectsRes
		err            error
	}{
		{
			desc:  "list objects with valid token",
			token: validToken,
			listObjectsReq: &magistrala.ListObjectsReq{
				Domain:     domainID,
				ObjectType: thingsType,
				Relation:   memberRelation,
				Permission: adminpermission,
			},
			listObjectsRes: &magistrala.ListObjectsRes{
				Policies: []string{validPolicy},
			},
			err: nil,
		},
		{
			desc:  "list objects with invalid token",
			token: inValidToken,
			listObjectsReq: &magistrala.ListObjectsReq{
				Domain:     domainID,
				ObjectType: thingsType,
				Relation:   memberRelation,
				Permission: adminpermission,
			},
			listObjectsRes: &magistrala.ListObjectsRes{},
			err:            svcerr.ErrAuthorization,
		},
	}
	for _, tc := range cases {
		repoCall := svc.On("ListObjects", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(auth.PolicyPage{Policies: tc.listObjectsRes.Policies}, tc.err)
		apr, err := client.ListObjects(context.Background(), tc.listObjectsReq)
		if apr != nil {
			assert.Equal(t, tc.listObjectsRes, apr, fmt.Sprintf("%s: expected %v got %v", tc.desc, tc.listObjectsRes, apr))
		}
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		repoCall.Unset()
	}
}

func TestListAllObjects(t *testing.T) {
	conn, err := grpc.Dial(authAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	assert.Nil(t, err, fmt.Sprintf("Unexpected error creating client connection %s", err))
	client := grpcapi.NewClient(conn, time.Second)

	cases := []struct {
		desc              string
		token             string
		listAllObjectsReq *magistrala.ListObjectsReq
		listAllObjectsRes *magistrala.ListObjectsRes
		err               error
	}{
		{
			desc:  "list all objects with valid token",
			token: validToken,
			listAllObjectsReq: &magistrala.ListObjectsReq{
				Domain:     domainID,
				ObjectType: thingsType,
				Relation:   memberRelation,
				Permission: adminpermission,
			},
			listAllObjectsRes: &magistrala.ListObjectsRes{
				Policies: []string{validPolicy},
			},
			err: nil,
		},
		{
			desc:  "list all objects with invalid token",
			token: inValidToken,
			listAllObjectsReq: &magistrala.ListObjectsReq{
				Domain:     domainID,
				ObjectType: thingsType,
				Relation:   memberRelation,
				Permission: adminpermission,
			},
			listAllObjectsRes: &magistrala.ListObjectsRes{},
			err:               svcerr.ErrAuthorization,
		},
	}
	for _, tc := range cases {
		repoCall := svc.On("ListAllObjects", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(auth.PolicyPage{Policies: tc.listAllObjectsRes.Policies}, tc.err)
		apr, err := client.ListAllObjects(context.Background(), tc.listAllObjectsReq)
		if apr != nil {
			assert.Equal(t, tc.listAllObjectsRes, apr, fmt.Sprintf("%s: expected %v got %v", tc.desc, tc.listAllObjectsRes, apr))
		}
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		repoCall.Unset()
	}
}

func TestCountObects(t *testing.T) {
	conn, err := grpc.Dial(authAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	assert.Nil(t, err, fmt.Sprintf("Unexpected error creating client connection %s", err))
	client := grpcapi.NewClient(conn, time.Second)

	cases := []struct {
		desc            string
		token           string
		countObjectsReq *magistrala.CountObjectsReq
		countObjectsRes *magistrala.CountObjectsRes
		err             error
	}{
		{
			desc:  "count objects with valid token",
			token: validToken,
			countObjectsReq: &magistrala.CountObjectsReq{
				Domain:     domainID,
				ObjectType: thingsType,
				Relation:   memberRelation,
				Permission: adminpermission,
			},
			countObjectsRes: &magistrala.CountObjectsRes{
				Count: 1,
			},
			err: nil,
		},
		{
			desc:  "count objects with invalid token",
			token: inValidToken,
			countObjectsReq: &magistrala.CountObjectsReq{
				Domain:     domainID,
				ObjectType: thingsType,
				Relation:   memberRelation,
				Permission: adminpermission,
			},
			countObjectsRes: &magistrala.CountObjectsRes{},
			err:             svcerr.ErrAuthorization,
		},
	}
	for _, tc := range cases {
		repoCall := svc.On("CountObjects", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(int(tc.countObjectsRes.Count), tc.err)
		apr, err := client.CountObjects(context.Background(), tc.countObjectsReq)
		if apr != nil {
			assert.Equal(t, tc.countObjectsRes, apr, fmt.Sprintf("%s: expected %v got %v", tc.desc, tc.countObjectsRes, apr))
		}
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		repoCall.Unset()
	}
}

func TestListSubjects(t *testing.T) {
	conn, err := grpc.Dial(authAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	assert.Nil(t, err, fmt.Sprintf("Unexpected error creating client connection %s", err))
	client := grpcapi.NewClient(conn, time.Second)

	cases := []struct {
		desc            string
		token           string
		listSubjectsReq *magistrala.ListSubjectsReq
		listSubjectsRes *magistrala.ListSubjectsRes
		err             error
	}{
		{
			desc:  "list subjects with valid token",
			token: validToken,
			listSubjectsReq: &magistrala.ListSubjectsReq{
				Domain:      domainID,
				SubjectType: usersType,
				Relation:    memberRelation,
				Permission:  adminpermission,
			},
			listSubjectsRes: &magistrala.ListSubjectsRes{
				Policies: []string{validPolicy},
			},
			err: nil,
		},
		{
			desc:  "list subjects with invalid token",
			token: inValidToken,
			listSubjectsReq: &magistrala.ListSubjectsReq{
				Domain:      domainID,
				SubjectType: usersType,
				Relation:    memberRelation,
				Permission:  adminpermission,
			},
			listSubjectsRes: &magistrala.ListSubjectsRes{},
			err:             svcerr.ErrAuthorization,
		},
	}
	for _, tc := range cases {
		repoCall := svc.On("ListSubjects", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(auth.PolicyPage{Policies: tc.listSubjectsRes.Policies}, tc.err)
		apr, err := client.ListSubjects(context.Background(), tc.listSubjectsReq)
		if apr != nil {
			assert.Equal(t, tc.listSubjectsRes, apr, fmt.Sprintf("%s: expected %v got %v", tc.desc, tc.listSubjectsRes, apr))
		}
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		repoCall.Unset()
	}
}

func TestListAllSubjects(t *testing.T) {
	conn, err := grpc.Dial(authAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	assert.Nil(t, err, fmt.Sprintf(`"Unexpected error creating client connection %s"`, err))
	client := grpcapi.NewClient(conn, time.Second)

	cases := []struct {
		desc            string
		token           string
		listSubjectsReq *magistrala.ListSubjectsReq
		listSubjectsRes *magistrala.ListSubjectsRes
		err             error
	}{
		{
			desc:  "list all subjects with valid token",
			token: validToken,
			listSubjectsReq: &magistrala.ListSubjectsReq{
				Domain:      domainID,
				SubjectType: auth.UserType,
				Relation:    memberRelation,
				Permission:  adminpermission,
			},
			listSubjectsRes: &magistrala.ListSubjectsRes{
				Policies: []string{validPolicy},
			},
			err: nil,
		},
		{
			desc:  "list all subjects with invalid token",
			token: inValidToken,
			listSubjectsReq: &magistrala.ListSubjectsReq{
				Domain:      domainID,
				SubjectType: usersType,
				Relation:    memberRelation,
				Permission:  adminpermission,
			},
			listSubjectsRes: &magistrala.ListSubjectsRes{},
			err:             svcerr.ErrAuthorization,
		},
	}
	for _, tc := range cases {
		repoCall := svc.On("ListAllSubjects", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(auth.PolicyPage{Policies: tc.listSubjectsRes.Policies}, tc.err)
		apr, err := client.ListAllSubjects(context.Background(), tc.listSubjectsReq)
		if apr != nil {
			assert.Equal(t, tc.listSubjectsRes, apr, fmt.Sprintf("%s: expected %v got %v", tc.desc, tc.listSubjectsRes, apr))
		}
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		repoCall.Unset()
	}
}

func TestCountSubjects(t *testing.T) {
	conn, err := grpc.Dial(authAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	assert.Nil(t, err, fmt.Sprintf("Unexpected error creating client connection %s", err))
	client := grpcapi.NewClient(conn, time.Second)

	cases := []struct {
		desc             string
		token            string
		countSubjectsReq *magistrala.CountSubjectsReq
		countSubjectsRes *magistrala.CountSubjectsRes
		err              error
		code             codes.Code
	}{
		{
			desc:  "count subjects with valid token",
			token: validToken,
			countSubjectsReq: &magistrala.CountSubjectsReq{
				Domain:      domainID,
				SubjectType: usersType,
				Relation:    memberRelation,
				Permission:  adminpermission,
			},
			countSubjectsRes: &magistrala.CountSubjectsRes{
				Count: 1,
			},
			code: codes.OK,
			err:  nil,
		},
		{
			desc:  "count subjects with invalid token",
			token: inValidToken,
			countSubjectsReq: &magistrala.CountSubjectsReq{
				Domain:      domainID,
				SubjectType: usersType,
				Relation:    memberRelation,
				Permission:  adminpermission,
			},
			countSubjectsRes: &magistrala.CountSubjectsRes{},
			err:              svcerr.ErrAuthentication,
			code:             codes.Unauthenticated,
		},
	}
	for _, tc := range cases {
		repoCall := svc.On("CountSubjects", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(int(tc.countSubjectsRes.Count), tc.err)
		apr, err := client.CountSubjects(context.Background(), tc.countSubjectsReq)
		if apr != nil {
			assert.Equal(t, tc.countSubjectsRes, apr, fmt.Sprintf("%s: expected %v got %v", tc.desc, tc.countSubjectsRes, apr))
		}
		e, ok := status.FromError(err)
		assert.True(t, ok, "gRPC status can't be extracted from the error")
		assert.Equal(t, tc.code, e.Code(), fmt.Sprintf("%s: expected %s got %s", tc.desc, tc.code, e.Code()))
		repoCall.Unset()
	}
}

func TestListPermissions(t *testing.T) {
	conn, err := grpc.Dial(authAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	assert.Nil(t, err, fmt.Sprintf("Unexpected error creating client connection %s", err))
	client := grpcapi.NewClient(conn, time.Second)

	cases := []struct {
		desc               string
		token              string
		listPermissionsReq *magistrala.ListPermissionsReq
		listPermissionsRes *magistrala.ListPermissionsRes
		err                error
	}{
		{
			desc:  "list permissions of thing type with valid token",
			token: validToken,
			listPermissionsReq: &magistrala.ListPermissionsReq{
				Domain:            domainID,
				SubjectType:       auth.UserType,
				Subject:           id,
				ObjectType:        auth.ThingType,
				Object:            validID,
				FilterPermissions: []string{"view"},
			},
			listPermissionsRes: &magistrala.ListPermissionsRes{
				SubjectType: auth.UserType,
				Subject:     id,
				ObjectType:  auth.ThingType,
				Object:      validID,
				Permissions: []string{"view"},
			},
			err: nil,
		},
		{
			desc:  "list permissions of group type with valid token",
			token: validToken,
			listPermissionsReq: &magistrala.ListPermissionsReq{
				Domain:            domainID,
				SubjectType:       auth.UserType,
				Subject:           id,
				ObjectType:        auth.GroupType,
				Object:            validID,
				FilterPermissions: []string{"view"},
			},
			listPermissionsRes: &magistrala.ListPermissionsRes{
				SubjectType: auth.UserType,
				Subject:     id,
				ObjectType:  auth.GroupType,
				Object:      validID,
				Permissions: []string{"view"},
			},
			err: nil,
		},
		{
			desc:  "list permissions of platform type with valid token",
			token: validToken,
			listPermissionsReq: &magistrala.ListPermissionsReq{
				Domain:            domainID,
				SubjectType:       auth.UserType,
				Subject:           id,
				ObjectType:        auth.PlatformType,
				Object:            validID,
				FilterPermissions: []string{"view"},
			},
			listPermissionsRes: &magistrala.ListPermissionsRes{
				SubjectType: auth.UserType,
				Subject:     id,
				ObjectType:  auth.PlatformType,
				Object:      validID,
				Permissions: []string{"view"},
			},
			err: nil,
		},
		{
			desc:  "list permissions of domain type with valid token",
			token: validToken,
			listPermissionsReq: &magistrala.ListPermissionsReq{
				Domain:            domainID,
				SubjectType:       auth.UserType,
				Subject:           id,
				ObjectType:        auth.DomainType,
				Object:            validID,
				FilterPermissions: []string{"view"},
			},
			listPermissionsRes: &magistrala.ListPermissionsRes{
				SubjectType: auth.UserType,
				Subject:     id,
				ObjectType:  auth.DomainType,
				Object:      validID,
				Permissions: []string{"view"},
			},
			err: nil,
		},
		{
			desc:  "list permissions of thing type with invalid token",
			token: inValidToken,
			listPermissionsReq: &magistrala.ListPermissionsReq{
				Domain:            domainID,
				SubjectType:       auth.UserType,
				Subject:           id,
				ObjectType:        auth.ThingType,
				Object:            validID,
				FilterPermissions: []string{"view"},
			},
			listPermissionsRes: &magistrala.ListPermissionsRes{},
			err:                svcerr.ErrAuthentication,
		},
		{
			desc:  "list permissions with invalid object type",
			token: validToken,
			listPermissionsReq: &magistrala.ListPermissionsReq{
				Domain:      domainID,
				SubjectType: auth.UserType,
				Subject:     id,
				ObjectType:  "invalid",
				Object:      validID,
			},
			listPermissionsRes: &magistrala.ListPermissionsRes{},
			err:                apiutil.ErrMalformedPolicy,
		},
	}
	for _, tc := range cases {
		repoCall := svc.On("ListPermissions", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(auth.Permissions{"view"}, tc.err)
		apr, err := client.ListPermissions(context.Background(), tc.listPermissionsReq)
		if apr != nil {
			assert.Equal(t, tc.listPermissionsRes, apr, fmt.Sprintf("%s: expected %v got %v", tc.desc, tc.listPermissionsRes, apr))
		}
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		repoCall.Unset()
	}
}
