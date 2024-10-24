// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package token_test

import (
	"context"
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/absmach/magistrala/auth"
	grpcapi "github.com/absmach/magistrala/auth/api/grpc/token"
	grpcTokenV1 "github.com/absmach/magistrala/internal/grpc/token/v1"
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
	clientsType     = "clients"
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
	authAddr = fmt.Sprintf("localhost:%d", port)
)

func startGRPCServer(svc auth.Service, port int) *grpc.Server {
	listener, _ := net.Listen("tcp", fmt.Sprintf(":%d", port))
	server := grpc.NewServer()
	grpcTokenV1.RegisterTokenServiceServer(server, grpcapi.NewTokenServer(svc))
	go func() {
		err := server.Serve(listener)
		assert.Nil(&testing.T{}, err, fmt.Sprintf(`"Unexpected error creating auth server %s"`, err))
	}()

	return server
}

func TestIssue(t *testing.T) {
	conn, err := grpc.NewClient(authAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	assert.Nil(t, err, fmt.Sprintf("Unexpected error creating client connection %s", err))
	grpcClient := grpcapi.NewTokenClient(conn, time.Second)

	cases := []struct {
		desc          string
		userId        string
		kind          auth.KeyType
		issueResponse auth.Token
		err           error
	}{
		{
			desc:   "issue for user with valid token",
			userId: validID,
			kind:   auth.AccessKey,
			issueResponse: auth.Token{
				AccessToken:  validToken,
				RefreshToken: validToken,
			},
			err: nil,
		},
		{
			desc:   "issue recovery key",
			userId: validID,
			kind:   auth.RecoveryKey,
			issueResponse: auth.Token{
				AccessToken:  validToken,
				RefreshToken: validToken,
			},
			err: nil,
		},
		{
			desc:          "issue API key unauthenticated",
			userId:        validID,
			kind:          auth.APIKey,
			issueResponse: auth.Token{},
			err:           svcerr.ErrAuthentication,
		},
		{
			desc:          "issue for invalid key type",
			userId:        validID,
			kind:          32,
			issueResponse: auth.Token{},
			err:           errors.ErrMalformedEntity,
		},
		{
			desc:          "issue for user that does notexist",
			userId:        "",
			kind:          auth.APIKey,
			issueResponse: auth.Token{},
			err:           svcerr.ErrAuthentication,
		},
	}

	for _, tc := range cases {
		svcCall := svc.On("Issue", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(tc.issueResponse, tc.err)
		_, err := grpcClient.Issue(context.Background(), &grpcTokenV1.IssueReq{UserId: tc.userId, Type: uint32(tc.kind)})
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		svcCall.Unset()
	}
}

func TestRefresh(t *testing.T) {
	conn, err := grpc.NewClient(authAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	assert.Nil(t, err, fmt.Sprintf("Unexpected error creating client connection %s", err))
	grpcClient := grpcapi.NewTokenClient(conn, time.Second)

	cases := []struct {
		desc          string
		token         string
		issueResponse auth.Token
		err           error
	}{
		{
			desc:  "refresh token with valid token",
			token: validToken,
			issueResponse: auth.Token{
				AccessToken:  validToken,
				RefreshToken: validToken,
			},
			err: nil,
		},
		{
			desc:          "refresh token with invalid token",
			token:         inValidToken,
			issueResponse: auth.Token{},
			err:           svcerr.ErrAuthentication,
		},
		{
			desc:          "refresh token with empty token",
			token:         "",
			issueResponse: auth.Token{},
			err:           apiutil.ErrMissingSecret,
		},
	}

	for _, tc := range cases {
		svcCall := svc.On("Issue", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(tc.issueResponse, tc.err)
		_, err := grpcClient.Refresh(context.Background(), &grpcTokenV1.RefreshReq{RefreshToken: tc.token})
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		svcCall.Unset()
	}
}
