// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package auth_test

import (
	"context"
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/absmach/magistrala"
	"github.com/absmach/magistrala/auth"
	grpcapi "github.com/absmach/magistrala/auth/api/grpc/auth"
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
	domainID = testsutil.GenerateUUID(&testing.T{})
	authAddr = fmt.Sprintf("localhost:%d", port)
)

func startGRPCServer(svc auth.Service, port int) *grpc.Server {
	listener, _ := net.Listen("tcp", fmt.Sprintf(":%d", port))
	server := grpc.NewServer()
	magistrala.RegisterAuthServiceServer(server, grpcapi.NewAuthServer(svc))
	go func() {
		err := server.Serve(listener)
		assert.Nil(&testing.T{}, err, fmt.Sprintf(`"Unexpected error creating auth server %s"`, err))
	}()

	return server
}

func TestIdentify(t *testing.T) {
	conn, err := grpc.NewClient(authAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	assert.Nil(t, err, fmt.Sprintf("Unexpected error creating client connection %s", err))
	grpcClient := grpcapi.NewAuthClient(conn, time.Second)

	cases := []struct {
		desc   string
		token  string
		idt    *magistrala.AuthNRes
		svcErr error
		err    error
	}{
		{
			desc:  "authenticate user with valid user token",
			token: validToken,
			idt:   &magistrala.AuthNRes{Id: id, UserId: email, DomainId: domainID},
			err:   nil,
		},
		{
			desc:   "authenticate user with invalid user token",
			token:  "invalid",
			idt:    &magistrala.AuthNRes{},
			svcErr: svcerr.ErrAuthentication,
			err:    svcerr.ErrAuthentication,
		},
		{
			desc:  "authenticate user with empty token",
			token: "",
			idt:   &magistrala.AuthNRes{},
			err:   apiutil.ErrBearerToken,
		},
	}

	for _, tc := range cases {
		svcCall := svc.On("Identify", mock.Anything, mock.Anything, mock.Anything).Return(auth.Key{Subject: id, User: email, Domain: domainID}, tc.svcErr)
		idt, err := grpcClient.Authenticate(context.Background(), &magistrala.AuthNReq{Token: tc.token})
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
	grpcClient := grpcapi.NewAuthClient(conn, time.Second)

	cases := []struct {
		desc         string
		token        string
		authRequest  *magistrala.AuthZReq
		authResponse *magistrala.AuthZRes
		err          error
	}{
		{
			desc:  "authorize user with authorized token",
			token: validToken,
			authRequest: &magistrala.AuthZReq{
				Subject:     id,
				SubjectType: usersType,
				Object:      authoritiesObj,
				ObjectType:  usersType,
				Relation:    memberRelation,
				Permission:  adminpermission,
			},
			authResponse: &magistrala.AuthZRes{Authorized: true},
			err:          nil,
		},
		{
			desc:  "authorize user with unauthorized token",
			token: inValidToken,
			authRequest: &magistrala.AuthZReq{
				Subject:     id,
				SubjectType: usersType,
				Object:      authoritiesObj,
				ObjectType:  usersType,
				Relation:    memberRelation,
				Permission:  adminpermission,
			},
			authResponse: &magistrala.AuthZRes{Authorized: false},
			err:          svcerr.ErrAuthorization,
		},
		{
			desc:  "authorize user with empty subject",
			token: validToken,
			authRequest: &magistrala.AuthZReq{
				Subject:     "",
				SubjectType: usersType,
				Object:      authoritiesObj,
				ObjectType:  usersType,
				Relation:    memberRelation,
				Permission:  adminpermission,
			},
			authResponse: &magistrala.AuthZRes{Authorized: false},
			err:          apiutil.ErrMissingPolicySub,
		},
		{
			desc:  "authorize user with empty subject type",
			token: validToken,
			authRequest: &magistrala.AuthZReq{
				Subject:     id,
				SubjectType: "",
				Object:      authoritiesObj,
				ObjectType:  usersType,
				Relation:    memberRelation,
				Permission:  adminpermission,
			},
			authResponse: &magistrala.AuthZRes{Authorized: false},
			err:          apiutil.ErrMissingPolicySub,
		},
		{
			desc:  "authorize user with empty object",
			token: validToken,
			authRequest: &magistrala.AuthZReq{
				Subject:     id,
				SubjectType: usersType,
				Object:      "",
				ObjectType:  usersType,
				Relation:    memberRelation,
				Permission:  adminpermission,
			},
			authResponse: &magistrala.AuthZRes{Authorized: false},
			err:          apiutil.ErrMissingPolicyObj,
		},
		{
			desc:  "authorize user with empty object type",
			token: validToken,
			authRequest: &magistrala.AuthZReq{
				Subject:     id,
				SubjectType: usersType,
				Object:      authoritiesObj,
				ObjectType:  "",
				Relation:    memberRelation,
				Permission:  adminpermission,
			},
			authResponse: &magistrala.AuthZRes{Authorized: false},
			err:          apiutil.ErrMissingPolicyObj,
		},
		{
			desc:  "authorize user with empty permission",
			token: validToken,
			authRequest: &magistrala.AuthZReq{
				Subject:     id,
				SubjectType: usersType,
				Object:      authoritiesObj,
				ObjectType:  usersType,
				Relation:    memberRelation,
				Permission:  "",
			},
			authResponse: &magistrala.AuthZRes{Authorized: false},
			err:          apiutil.ErrMalformedPolicyPer,
		},
	}
	for _, tc := range cases {
		svccall := svc.On("Authorize", mock.Anything, mock.Anything).Return(tc.err)
		ar, err := grpcClient.Authorize(context.Background(), tc.authRequest)
		if ar != nil {
			assert.Equal(t, tc.authResponse, ar, fmt.Sprintf("%s: expected %v got %v", tc.desc, tc.authResponse, ar))
		}
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		svccall.Unset()
	}
}
