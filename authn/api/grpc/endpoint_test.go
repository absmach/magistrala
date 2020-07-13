// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package grpc_test

import (
	"context"
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/mainflux/mainflux"
	"github.com/mainflux/mainflux/authn"
	grpcapi "github.com/mainflux/mainflux/authn/api/grpc"
	"github.com/mainflux/mainflux/authn/jwt"
	"github.com/mainflux/mainflux/authn/mocks"
	"github.com/mainflux/mainflux/pkg/uuid"
	"github.com/opentracing/opentracing-go/mocktracer"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	port   = 8081
	secret = "secret"
	email  = "test@example.com"
)

var svc authn.Service

func newService() authn.Service {
	repo := mocks.NewKeyRepository()
	uuidProvider := uuid.NewMock()
	t := jwt.New(secret)

	return authn.New(repo, uuidProvider, t)
}

func startGRPCServer(svc authn.Service, port int) {
	listener, _ := net.Listen("tcp", fmt.Sprintf(":%d", port))
	server := grpc.NewServer()
	mainflux.RegisterAuthNServiceServer(server, grpcapi.NewServer(mocktracer.New(), svc))
	go server.Serve(listener)
}

func TestIssue(t *testing.T) {
	userKey, err := svc.Issue(context.Background(), email, authn.Key{Type: authn.UserKey, IssuedAt: time.Now()})
	assert.Nil(t, err, fmt.Sprintf("Issuing user key expected to succeed: %s", err))

	authAddr := fmt.Sprintf("localhost:%d", port)
	conn, _ := grpc.Dial(authAddr, grpc.WithInsecure())
	client := grpcapi.NewClient(mocktracer.New(), conn, time.Second)

	cases := []struct {
		desc string
		id   string
		kind uint32
		err  error
		code codes.Code
	}{
		{
			desc: "issue for user with valid token",
			id:   email,
			kind: authn.UserKey,
			err:  nil,
			code: codes.OK,
		},
		{
			desc: "issue recovery key",
			id:   email,
			kind: authn.RecoveryKey,
			err:  nil,
			code: codes.OK,
		},
		{
			desc: "issue API key",
			id:   userKey.Secret,
			kind: authn.APIKey,
			err:  nil,
			code: codes.OK,
		},
		{
			desc: "issue for invalid key type",
			id:   email,
			kind: 32,
			err:  status.Error(codes.InvalidArgument, "received invalid token request"),
			code: codes.InvalidArgument,
		},
		{
			desc: "issue for user that  exist",
			id:   "",
			kind: authn.APIKey,
			err:  status.Error(codes.Unauthenticated, "unauthorized access"),
			code: codes.Unauthenticated,
		},
	}

	for _, tc := range cases {
		_, err := client.Issue(context.Background(), &mainflux.IssueReq{Issuer: tc.id, Type: tc.kind})
		e, ok := status.FromError(err)
		assert.True(t, ok, "gRPC status can't be extracted from the error")
		assert.Equal(t, tc.code, e.Code(), fmt.Sprintf("%s: expected %s got %s", tc.desc, tc.code, e.Code()))
	}
}

func TestIdentify(t *testing.T) {
	userKey, err := svc.Issue(context.Background(), email, authn.Key{Type: authn.UserKey, IssuedAt: time.Now()})
	assert.Nil(t, err, fmt.Sprintf("Issuing user key expected to succeed: %s", err))

	recoveryKey, err := svc.Issue(context.Background(), email, authn.Key{Type: authn.RecoveryKey, IssuedAt: time.Now()})
	assert.Nil(t, err, fmt.Sprintf("Issuing recovery key expected to succeed: %s", err))

	apiKey, err := svc.Issue(context.Background(), userKey.Secret, authn.Key{Type: authn.APIKey, IssuedAt: time.Now(), ExpiresAt: time.Now().Add(time.Minute)})
	assert.Nil(t, err, fmt.Sprintf("Issuing API key expected to succeed: %s", err))

	authAddr := fmt.Sprintf("localhost:%d", port)
	conn, _ := grpc.Dial(authAddr, grpc.WithInsecure())
	client := grpcapi.NewClient(mocktracer.New(), conn, time.Second)

	cases := []struct {
		desc  string
		token string
		id    string
		err   error
		code  codes.Code
	}{
		{
			desc:  "identify user with recovery token",
			token: recoveryKey.Secret,
			id:    email,
			err:   nil,
			code:  codes.OK,
		},
		{
			desc:  "identify user with API token",
			token: apiKey.Secret,
			id:    email,
			err:   nil,
			code:  codes.OK,
		},
		{
			desc:  "identify user with invalid user token",
			token: "invalid",
			id:    "",
			err:   status.Error(codes.Unauthenticated, "unauthorized access"),
			code:  codes.Unauthenticated,
		},
		{
			desc:  "identify user that doesn't exist",
			token: "",
			id:    "",
			err:   status.Error(codes.InvalidArgument, "received invalid token request"),
			code:  codes.InvalidArgument,
		},
	}

	for _, tc := range cases {
		id, err := client.Identify(context.Background(), &mainflux.Token{Value: tc.token})
		assert.Equal(t, tc.id, id.GetValue(), fmt.Sprintf("%s: expected %s got %s", tc.desc, tc.id, id.GetValue()))
		e, ok := status.FromError(err)
		assert.True(t, ok, "gRPC status can't be extracted from the error")
		assert.Equal(t, tc.code, e.Code(), fmt.Sprintf("%s: expected %s got %s", tc.desc, tc.code, e.Code()))
	}
}
