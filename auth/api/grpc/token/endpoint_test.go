// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package token_test

import (
	"context"
	"fmt"
	"net"
	"testing"
	"time"

	grpcTokenV1 "github.com/absmach/supermq/api/grpc/token/v1"
	apiutil "github.com/absmach/supermq/api/http/util"
	"github.com/absmach/supermq/auth"
	grpcapi "github.com/absmach/supermq/auth/api/grpc/token"
	"github.com/absmach/supermq/internal/testsutil"
	"github.com/absmach/supermq/pkg/errors"
	svcerr "github.com/absmach/supermq/pkg/errors/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

const (
	port         = 8082
	validToken   = "valid"
	inValidToken = "invalid"
	invalidID    = "invalid"
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
	defer conn.Close()

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
	defer conn.Close()

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
		svcCall := svc.On("Issue", mock.Anything, mock.Anything, mock.Anything).Return(tc.issueResponse, tc.err)
		_, err := grpcClient.Refresh(context.Background(), &grpcTokenV1.RefreshReq{RefreshToken: tc.token})
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		svcCall.Unset()
	}
}

func TestRevoke(t *testing.T) {
	conn, err := grpc.NewClient(authAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	assert.Nil(t, err, fmt.Sprintf("Unexpected error creating client connection %s", err))
	grpcClient := grpcapi.NewTokenClient(conn, time.Second)
	defer conn.Close()

	cases := []struct {
		desc string
		id   string
		err  error
	}{
		{
			desc: "revoke token with valid id",
			id:   validID,
			err:  nil,
		},
		{
			desc: "revoke token with invalid id",
			id:   invalidID,
			err:  svcerr.ErrAuthentication,
		},
		{
			desc: "revoke token with empty id",
			id:   "",
			err:  apiutil.ErrMissingID,
		},
		{
			desc: "revoke already revoked token",
			id:   validID,
			err:  svcerr.ErrConflict,
		},
	}

	for _, tc := range cases {
		svcCall := svc.On("RevokeToken", mock.Anything, tc.id).Return(tc.err)
		_, err := grpcClient.Revoke(context.Background(), &grpcTokenV1.RevokeReq{TokenId: tc.id})
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		svcCall.Unset()
	}
}

func TestListUserRefreshTokens(t *testing.T) {
	conn, err := grpc.NewClient(authAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	assert.Nil(t, err, fmt.Sprintf("Unexpected error creating client connection %s", err))
	grpcClient := grpcapi.NewTokenClient(conn, time.Second)
	defer conn.Close()

	cases := []struct {
		desc         string
		userID       string
		listResponse []auth.TokenInfo
		err          error
	}{
		{
			desc:   "list tokens for user with valid id",
			userID: validID,
			listResponse: []auth.TokenInfo{
				{ID: testsutil.GenerateUUID(&testing.T{}), Description: "Token 1"},
				{ID: testsutil.GenerateUUID(&testing.T{}), Description: "Token 2"},
			},
			err: nil,
		},
		{
			desc:         "list tokens for user with empty list",
			userID:       validID,
			listResponse: []auth.TokenInfo{},
			err:          nil,
		},
		{
			desc:         "list tokens with invalid user id",
			userID:       invalidID,
			listResponse: nil,
			err:          svcerr.ErrAuthentication,
		},
		{
			desc:         "list tokens with empty user id",
			userID:       "",
			listResponse: nil,
			err:          apiutil.ErrMissingID,
		},
	}

	for _, tc := range cases {
		svcCall := svc.On("ListUserRefreshTokens", mock.Anything, tc.userID).Return(tc.listResponse, tc.err)
		_, err := grpcClient.ListUserRefreshTokens(context.Background(), &grpcTokenV1.ListUserRefreshTokensReq{UserId: tc.userID})
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		svcCall.Unset()
	}
}
