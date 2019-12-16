// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package grpc_test

import (
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/mainflux/mainflux"
	"github.com/mainflux/mainflux/authn"
	grpcapi "github.com/mainflux/mainflux/authn/api/grpc"
	"github.com/mainflux/mainflux/authn/jwt"
	"github.com/mainflux/mainflux/authn/mocks"
	"github.com/opentracing/opentracing-go/mocktracer"
	"github.com/stretchr/testify/assert"
	"golang.org/x/net/context"
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
	idp := mocks.NewIdentityProvider()
	t := jwt.New(secret)

	return authn.New(repo, idp, t)
}

func startGRPCServer(svc authn.Service, port int) {
	listener, _ := net.Listen("tcp", fmt.Sprintf(":%d", port))
	server := grpc.NewServer()
	mainflux.RegisterAuthNServiceServer(server, grpcapi.NewServer(mocktracer.New(), svc))
	go server.Serve(listener)
}

func TestIssue(t *testing.T) {
	loginKey, err := svc.Issue(context.Background(), email, authn.Key{Type: authn.UserKey, IssuedAt: time.Now()})
	assert.Nil(t, err, fmt.Sprintf("Issuing login key expected to succeed: %s", err))

	authAddr := fmt.Sprintf("localhost:%d", port)
	conn, _ := grpc.Dial(authAddr, grpc.WithInsecure())
	client := grpcapi.NewClient(mocktracer.New(), conn, time.Second)

	cases := map[string]struct {
		token string
		id    string
		kind  uint32
		err   error
	}{
		"issue for user with valid token":   {"", email, authn.UserKey, nil},
		"issue for user that doesn't exist": {"", loginKey.Secret, 32, status.Error(codes.InvalidArgument, "received invalid token request")},
	}

	for desc, tc := range cases {
		_, err := client.Issue(context.Background(), &mainflux.IssueReq{Issuer: tc.id, Type: tc.kind})
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected %s got %s", desc, tc.err, err))
	}
}

func TestIdentify(t *testing.T) {
	loginKey, err := svc.Issue(context.Background(), email, authn.Key{Type: authn.UserKey, IssuedAt: time.Now()})
	assert.Nil(t, err, fmt.Sprintf("Issuing login key expected to succeed: %s", err))

	resetKey, err := svc.Issue(context.Background(), loginKey.Secret, authn.Key{Type: authn.RecoveryKey, IssuedAt: time.Now()})
	assert.Nil(t, err, fmt.Sprintf("Issuing reset key expected to succeed: %s", err))

	userKey, err := svc.Issue(context.Background(), loginKey.Secret, authn.Key{Type: authn.APIKey, IssuedAt: time.Now(), ExpiresAt: time.Now().Add(time.Minute)})
	assert.Nil(t, err, fmt.Sprintf("Issuing user key expected to succeed: %s", err))

	authAddr := fmt.Sprintf("localhost:%d", port)
	conn, _ := grpc.Dial(authAddr, grpc.WithInsecure())
	client := grpcapi.NewClient(mocktracer.New(), conn, time.Second)

	cases := []struct {
		desc  string
		token string
		id    string
		err   error
	}{
		{
			desc:  "identify user with reset token",
			token: resetKey.Secret,
			id:    email,
			err:   nil,
		},
		{
			desc:  "identify user with user token",
			token: userKey.Secret,
			id:    email,
			err:   nil,
		},
		{
			desc:  "identify user with invalid login token",
			token: "invalid",
			id:    "",
			err:   status.Error(codes.Unauthenticated, "unauthorized access"),
		},
		{
			desc:  "identify user that doesn't exist",
			token: "",
			id:    "",
			err:   status.Error(codes.InvalidArgument, "received invalid token request"),
		},
	}

	for _, tc := range cases {
		id, err := client.Identify(context.Background(), &mainflux.Token{Value: tc.token})
		assert.Equal(t, tc.id, id.GetValue(), fmt.Sprintf("%s: expected %s got %s", tc.desc, tc.id, id.GetValue()))
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected %s got %s", tc.desc, tc.err, err))
	}
}
