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
	clientsType     = "clients"
	usersType       = "users"
	description     = "Description"
	groupName       = "smqx"
	adminpermission = "admin"

	authoritiesObj  = "authorities"
	memberRelation  = "member"
	loginDuration   = 30 * time.Minute
	refreshDuration = 24 * time.Hour
	invalidDuration = 7 * 24 * time.Hour
	validToken      = "valid"
	inValidToken    = "invalid"
	validPATToken   = "valid"
	inValidPATToken = "invalid"
	validPolicy     = "valid"
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
		idt    *grpcAuthV1.AuthNRes
		svcErr error
		err    error
	}{
		{
			desc:  "authenticate user with valid user token",
			token: validToken,
			idt:   &grpcAuthV1.AuthNRes{Id: id, UserId: email, DomainId: domainID},
			err:   nil,
		},
		{
			desc:   "authenticate user with invalid user token",
			token:  "invalid",
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
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			svcCall := svc.On("Identify", mock.Anything, mock.Anything).Return(auth.Key{Subject: id, User: email, Domain: domainID}, tc.svcErr)
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
				Subject:     id,
				SubjectType: usersType,
				Object:      authoritiesObj,
				ObjectType:  usersType,
				Relation:    memberRelation,
				Permission:  adminpermission,
			},
			authResponse: &grpcAuthV1.AuthZRes{Authorized: true},
			err:          nil,
		},
		{
			desc:  "authorize user with unauthorized token",
			token: inValidToken,
			authRequest: &grpcAuthV1.AuthZReq{
				Subject:     id,
				SubjectType: usersType,
				Object:      authoritiesObj,
				ObjectType:  usersType,
				Relation:    memberRelation,
				Permission:  adminpermission,
			},
			authResponse: &grpcAuthV1.AuthZRes{Authorized: false},
			err:          svcerr.ErrAuthorization,
		},
		{
			desc:  "authorize user with empty subject",
			token: validToken,
			authRequest: &grpcAuthV1.AuthZReq{
				Subject:     "",
				SubjectType: usersType,
				Object:      authoritiesObj,
				ObjectType:  usersType,
				Relation:    memberRelation,
				Permission:  adminpermission,
			},
			authResponse: &grpcAuthV1.AuthZRes{Authorized: false},
			err:          apiutil.ErrMissingPolicySub,
		},
		{
			desc:  "authorize user with empty subject type",
			token: validToken,
			authRequest: &grpcAuthV1.AuthZReq{
				Subject:     id,
				SubjectType: "",
				Object:      authoritiesObj,
				ObjectType:  usersType,
				Relation:    memberRelation,
				Permission:  adminpermission,
			},
			authResponse: &grpcAuthV1.AuthZRes{Authorized: false},
			err:          apiutil.ErrMissingPolicySub,
		},
		{
			desc:  "authorize user with empty object",
			token: validToken,
			authRequest: &grpcAuthV1.AuthZReq{
				Subject:     id,
				SubjectType: usersType,
				Object:      "",
				ObjectType:  usersType,
				Relation:    memberRelation,
				Permission:  adminpermission,
			},
			authResponse: &grpcAuthV1.AuthZRes{Authorized: false},
			err:          apiutil.ErrMissingPolicyObj,
		},
		{
			desc:  "authorize user with empty object type",
			token: validToken,
			authRequest: &grpcAuthV1.AuthZReq{
				Subject:     id,
				SubjectType: usersType,
				Object:      authoritiesObj,
				ObjectType:  "",
				Relation:    memberRelation,
				Permission:  adminpermission,
			},
			authResponse: &grpcAuthV1.AuthZRes{Authorized: false},
			err:          apiutil.ErrMissingPolicyObj,
		},
		{
			desc:  "authorize user with empty permission",
			token: validToken,
			authRequest: &grpcAuthV1.AuthZReq{
				Subject:     id,
				SubjectType: usersType,
				Object:      authoritiesObj,
				ObjectType:  usersType,
				Relation:    memberRelation,
				Permission:  "",
			},
			authResponse: &grpcAuthV1.AuthZRes{Authorized: false},
			err:          apiutil.ErrMalformedPolicyPer,
		},
	}
	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			svccall := svc.On("Authorize", mock.Anything, mock.Anything).Return(tc.err)
			ar, err := grpcClient.Authorize(context.Background(), tc.authRequest)
			if ar != nil {
				assert.Equal(t, tc.authResponse, ar, fmt.Sprintf("%s: expected %v got %v", tc.desc, tc.authResponse, ar))
			}
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
			svccall.Unset()
		})
	}
}

func TestIdentifyPAT(t *testing.T) {
	conn, err := grpc.NewClient(authAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	assert.Nil(t, err, fmt.Sprintf("Unexpected error creating client connection %s", err))
	defer conn.Close()
	grpcClient := grpcapi.NewAuthClient(conn, time.Second)

	cases := []struct {
		desc   string
		token  string
		idt    *grpcAuthV1.AuthNRes
		svcErr error
		err    error
	}{
		{
			desc:  "authenticate user with valid user token",
			token: validToken,
			idt:   &grpcAuthV1.AuthNRes{Id: id, UserId: clientID},
			err:   nil,
		},
		{
			desc:   "authenticate user with invalid user token",
			token:  "invalid",
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
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			svcCall := svc.On("IdentifyPAT", mock.Anything, tc.token).Return(auth.PAT{ID: id, User: clientID, IssuedAt: time.Now()}, tc.svcErr)
			idt, err := grpcClient.AuthenticatePAT(context.Background(), &grpcAuthV1.AuthNReq{Token: tc.token})
			if idt != nil {
				assert.Equal(t, tc.idt, idt, fmt.Sprintf("%s: expected %v got %v", tc.desc, tc.idt, idt))
			}
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
			svcCall.Unset()
		})
	}
}

func TestAuthorizePAT(t *testing.T) {
	conn, err := grpc.NewClient(authAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	assert.Nil(t, err, fmt.Sprintf("Unexpected error creating client connection %s", err))
	defer conn.Close()

	grpcClient := grpcapi.NewAuthClient(conn, time.Second)
	cases := []struct {
		desc         string
		token        string
		authRequest  *grpcAuthV1.AuthZPatReq
		authResponse *grpcAuthV1.AuthZRes
		err          error
	}{
		{
			desc:  "authorize user with authorized token",
			token: validPATToken,
			authRequest: &grpcAuthV1.AuthZPatReq{
				UserId:           id,
				PatId:            id,
				EntityType:       uint32(auth.ClientsType),
				OptionalDomainId: domainID,
				Operation:        uint32(auth.CreateOp),
				EntityId:         clientID,
			},
			authResponse: &grpcAuthV1.AuthZRes{Authorized: true},
			err:          nil,
		},
		{
			desc:  "authorize user with unauthorized token",
			token: inValidPATToken,
			authRequest: &grpcAuthV1.AuthZPatReq{
				UserId:           id,
				PatId:            id,
				EntityType:       uint32(auth.ClientsType),
				OptionalDomainId: domainID,
				Operation:        uint32(auth.CreateOp),
				EntityId:         clientID,
			},
			authResponse: &grpcAuthV1.AuthZRes{Authorized: false},
			err:          svcerr.ErrAuthorization,
		},
		{
			desc:  "authorize user with missing user id",
			token: validPATToken,
			authRequest: &grpcAuthV1.AuthZPatReq{
				PatId:            id,
				EntityType:       uint32(auth.ClientsType),
				OptionalDomainId: domainID,
				Operation:        uint32(auth.CreateOp),
				EntityId:         clientID,
			},
			authResponse: &grpcAuthV1.AuthZRes{Authorized: false},
			err:          apiutil.ErrMissingUserID,
		},
		{
			desc:  "authorize user with missing pat id",
			token: validPATToken,
			authRequest: &grpcAuthV1.AuthZPatReq{
				UserId:           id,
				EntityType:       uint32(auth.ClientsType),
				OptionalDomainId: domainID,
				Operation:        uint32(auth.CreateOp),
				EntityId:         clientID,
			},
			authResponse: &grpcAuthV1.AuthZRes{Authorized: false},
			err:          apiutil.ErrMissingPATID,
		},
	}
	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			svccall := svc.On("AuthorizePAT",
				mock.Anything,
				tc.authRequest.UserId,
				tc.authRequest.PatId,
				mock.Anything,
				tc.authRequest.OptionalDomainId,
				mock.Anything,
				mock.Anything,
				mock.Anything).Return(tc.err)
			ar, err := grpcClient.AuthorizePAT(context.Background(), tc.authRequest)
			if ar != nil {
				assert.Equal(t, tc.authResponse, ar, fmt.Sprintf("%s: expected %v got %v", tc.desc, tc.authResponse, ar))
			}
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
			svccall.Unset()
		})
	}
}
