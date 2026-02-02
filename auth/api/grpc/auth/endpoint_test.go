// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package auth_test

import (
	"context"
	"fmt"
	"net"
	"testing"
	"time"

	grpcAuthV1 "github.com/absmach/supermq/api/grpc/auth/v1"
	apiutil "github.com/absmach/supermq/api/http/util"
	"github.com/absmach/supermq/auth"
	grpcapi "github.com/absmach/supermq/auth/api/grpc/auth"
	"github.com/absmach/supermq/internal/testsutil"
	"github.com/absmach/supermq/pkg/errors"
	svcerr "github.com/absmach/supermq/pkg/errors/service"
	"github.com/absmach/supermq/pkg/policies"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

const (
	port            = 8081
	id              = "testID"
	usersType       = "users"
	adminPermission = "admin"
	authoritiesObj  = "authorities"
	memberRelation  = "member"
	validToken      = "valid"
	inValidToken    = "invalid"
	validPATToken   = "valid"
)

var (
	domainID = testsutil.GenerateUUID(&testing.T{})
	authAddr = fmt.Sprintf("localhost:%d", port)
	clientID = testsutil.GenerateUUID(&testing.T{})
)

func startGRPCServer(svc auth.Service, port int) *grpc.Server {
	listener, _ := net.Listen("tcp", fmt.Sprintf(":%d", port))
	server := grpc.NewServer()
	grpcAuthV1.RegisterAuthServiceServer(server, grpcapi.NewAuthServer(svc))
	go func() {
		err := server.Serve(listener)
		assert.Nil(&testing.T{}, err, fmt.Sprintf(`"Unexpected error creating auth server %s"`, err))
	}()

	return server
}

func TestIdentify(t *testing.T) {
	conn, err := grpc.NewClient(authAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	assert.Nil(t, err, fmt.Sprintf("Unexpected error creating client connection %s", err))
	defer conn.Close()
	grpcClient := grpcapi.NewAuthClient(conn, time.Second)

	cases := []struct {
		desc   string
		token  string
		key    auth.Key
		idt    *grpcAuthV1.AuthNRes
		svcErr error
		err    error
	}{
		{
			desc:  "authenticate user with valid user token",
			token: validToken,
			key:   auth.Key{ID: "", Subject: id, Role: auth.UserRole},
			idt:   &grpcAuthV1.AuthNRes{UserId: id, UserRole: uint32(auth.UserRole)},
			err:   nil,
		},
		{
			desc:   "authenticate user with invalid user token",
			token:  "invalid",
			key:    auth.Key{},
			idt:    &grpcAuthV1.AuthNRes{},
			svcErr: svcerr.ErrAuthentication,
			err:    svcerr.ErrAuthentication,
		},
		{
			desc:  "authenticate user with empty token",
			token: "",
			idt:   &grpcAuthV1.AuthNRes{},
			err:   apiutil.ErrBearerToken,
		},
		{
			desc:  "authenticate user with valid PAT token",
			token: "pat_" + validPATToken,
			key:   auth.Key{ID: id, Type: auth.PersonalAccessToken, Subject: clientID, Role: auth.UserRole},
			idt:   &grpcAuthV1.AuthNRes{Id: id, UserId: clientID, UserRole: uint32(auth.UserRole)},
			err:   nil,
		},
		{
			desc:   "authenticate user with invalid PAT token",
			token:  "pat_invalid",
			key:    auth.Key{},
			idt:    &grpcAuthV1.AuthNRes{},
			svcErr: svcerr.ErrAuthentication,
			err:    svcerr.ErrAuthentication,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			svcCall := svc.On("Identify", mock.Anything, tc.token).Return(tc.key, tc.svcErr)
			idt, err := grpcClient.Authenticate(context.Background(), &grpcAuthV1.AuthNReq{Token: tc.token})
			if idt != nil {
				assert.Equal(t, tc.idt, idt, fmt.Sprintf("%s: expected %v got %v", tc.desc, tc.idt, idt))
			}
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
			svcCall.Unset()
		})
	}
}

func TestAuthorize(t *testing.T) {
	conn, err := grpc.NewClient(authAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	assert.Nil(t, err, fmt.Sprintf("Unexpected error creating client connection %s", err))
	defer conn.Close()

	grpcClient := grpcapi.NewAuthClient(conn, time.Second)

	cases := []struct {
		desc         string
		token        string
		authRequest  *grpcAuthV1.AuthZReq
		authResponse *grpcAuthV1.AuthZRes
		err          error
	}{
		{
			desc:  "authorize user with authorized token",
			token: validToken,
			authRequest: &grpcAuthV1.AuthZReq{
				PolicyReq: &grpcAuthV1.PolicyReq{
					Subject:     id,
					SubjectType: usersType,
					Object:      authoritiesObj,
					ObjectType:  usersType,
					Relation:    memberRelation,
					Permission:  adminPermission,
				},
			},
			authResponse: &grpcAuthV1.AuthZRes{Authorized: true},
			err:          nil,
		},
		{
			desc:  "authorize user with unauthorized token",
			token: inValidToken,
			authRequest: &grpcAuthV1.AuthZReq{
				PolicyReq: &grpcAuthV1.PolicyReq{
					Subject:     id,
					SubjectType: usersType,
					Object:      authoritiesObj,
					ObjectType:  usersType,
					Relation:    memberRelation,
					Permission:  adminPermission,
				},
			},
			authResponse: &grpcAuthV1.AuthZRes{Authorized: false},
			err:          svcerr.ErrAuthorization,
		},
		{
			desc:  "authorize user with empty subject",
			token: validToken,
			authRequest: &grpcAuthV1.AuthZReq{
				PolicyReq: &grpcAuthV1.PolicyReq{
					Subject:     "",
					SubjectType: usersType,
					Object:      authoritiesObj,
					ObjectType:  usersType,
					Relation:    memberRelation,
					Permission:  adminPermission,
				},
			},
			authResponse: &grpcAuthV1.AuthZRes{Authorized: false},
			err:          apiutil.ErrMissingPolicySub,
		},
		{
			desc:  "authorize user with empty subject type",
			token: validToken,
			authRequest: &grpcAuthV1.AuthZReq{
				PolicyReq: &grpcAuthV1.PolicyReq{
					Subject:     id,
					SubjectType: "",
					Object:      authoritiesObj,
					ObjectType:  usersType,
					Relation:    memberRelation,
					Permission:  adminPermission,
				},
			},
			authResponse: &grpcAuthV1.AuthZRes{Authorized: false},
			err:          apiutil.ErrMissingPolicySub,
		},
		{
			desc:  "authorize user with empty object",
			token: validToken,
			authRequest: &grpcAuthV1.AuthZReq{
				PolicyReq: &grpcAuthV1.PolicyReq{
					Subject:     id,
					SubjectType: usersType,
					Object:      "",
					ObjectType:  usersType,
					Relation:    memberRelation,
					Permission:  adminPermission,
				},
			},
			authResponse: &grpcAuthV1.AuthZRes{Authorized: false},
			err:          apiutil.ErrMissingPolicyObj,
		},
		{
			desc:  "authorize user with empty object type",
			token: validToken,
			authRequest: &grpcAuthV1.AuthZReq{
				PolicyReq: &grpcAuthV1.PolicyReq{
					Subject:     id,
					SubjectType: usersType,
					Object:      authoritiesObj,
					ObjectType:  "",
					Relation:    memberRelation,
					Permission:  adminPermission,
				},
			},
			authResponse: &grpcAuthV1.AuthZRes{Authorized: false},
			err:          apiutil.ErrMissingPolicyObj,
		},
		{
			desc:  "authorize user with empty permission",
			token: validToken,
			authRequest: &grpcAuthV1.AuthZReq{
				PolicyReq: &grpcAuthV1.PolicyReq{
					Subject:     id,
					SubjectType: usersType,
					Object:      authoritiesObj,
					ObjectType:  usersType,
					Relation:    memberRelation,
					Permission:  "",
				},
			},
			authResponse: &grpcAuthV1.AuthZRes{Authorized: false},
			err:          apiutil.ErrMalformedPolicyPer,
		},
		{
			desc:  "authorize user with valid PAT token",
			token: validPATToken,
			authRequest: &grpcAuthV1.AuthZReq{
				PolicyReq: &grpcAuthV1.PolicyReq{
					Subject:     id,
					SubjectType: policies.UserType,
					SubjectKind: policies.UsersKind,
					Permission:  policies.ViewPermission,
					ObjectType:  policies.ClientType,
					Domain:      domainID,
					Object:      clientID,
				},
				PatReq: &grpcAuthV1.PATReq{
					PatId:      id,
					Domain:     domainID,
					Operation:  "view",
					UserId:     id,
					EntityId:   clientID,
					EntityType: auth.ClientsScopeStr,
				},
			},
			authResponse: &grpcAuthV1.AuthZRes{Authorized: true},
			err:          nil,
		},
		{
			desc:  "authorize user with unauthorized PAT token",
			token: inValidToken,
			authRequest: &grpcAuthV1.AuthZReq{
				PolicyReq: &grpcAuthV1.PolicyReq{
					Subject:     id,
					SubjectType: policies.UserType,
					SubjectKind: policies.UsersKind,
					Permission:  policies.ViewPermission,
					ObjectType:  policies.ClientType,
					Domain:      domainID,
					Object:      clientID,
				},
				PatReq: &grpcAuthV1.PATReq{
					PatId:      id,
					Domain:     domainID,
					Operation:  "view",
					UserId:     id,
					EntityId:   clientID,
					EntityType: auth.ClientsScopeStr,
				},
			},
			authResponse: &grpcAuthV1.AuthZRes{Authorized: false},
			err:          svcerr.ErrAuthorization,
		},
		{
			desc:  "authorize PAT with missing user id",
			token: validPATToken,
			authRequest: &grpcAuthV1.AuthZReq{
				PolicyReq: &grpcAuthV1.PolicyReq{
					Subject:     id,
					SubjectType: policies.UserType,
					SubjectKind: policies.UsersKind,
					Permission:  policies.ViewPermission,
					ObjectType:  policies.ClientType,
					Domain:      domainID,
					Object:      clientID,
				},
				PatReq: &grpcAuthV1.PATReq{
					PatId:      id,
					Domain:     domainID,
					Operation:  "view",
					EntityId:   clientID,
					EntityType: auth.ClientsScopeStr,
				},
			},
			authResponse: &grpcAuthV1.AuthZRes{Authorized: false},
			err:          apiutil.ErrMissingUserID,
		},
		{
			desc:  "authorize PAT with missing entity id",
			token: validPATToken,
			authRequest: &grpcAuthV1.AuthZReq{
				PolicyReq: &grpcAuthV1.PolicyReq{
					Subject:     id,
					SubjectType: policies.UserType,
					SubjectKind: policies.UsersKind,
					Permission:  policies.ViewPermission,
					ObjectType:  policies.ClientType,
					Domain:      domainID,
					Object:      clientID,
				},
				PatReq: &grpcAuthV1.PATReq{
					PatId:      id,
					Domain:     domainID,
					Operation:  "view",
					UserId:     id,
					EntityType: auth.ClientsScopeStr,
				},
			},
			authResponse: &grpcAuthV1.AuthZRes{Authorized: false},
			err:          apiutil.ErrMissingID,
		},
	}
	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			svcCall := svc.On("Authorize", mock.Anything, mock.Anything, mock.Anything).Return(tc.err)
			ar, err := grpcClient.Authorize(context.Background(), tc.authRequest)
			if ar != nil {
				assert.Equal(t, tc.authResponse, ar, fmt.Sprintf("%s: expected %v got %v", tc.desc, tc.authResponse, ar))
			}
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
			svcCall.Unset()
		})
	}
}
