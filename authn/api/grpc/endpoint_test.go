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
	id     = "testID"
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
	authAddr := fmt.Sprintf("localhost:%d", port)
	conn, _ := grpc.Dial(authAddr, grpc.WithInsecure())
	client := grpcapi.NewClient(mocktracer.New(), conn, time.Second)

	cases := []struct {
		desc  string
		id    string
		email string
		kind  uint32
		err   error
		code  codes.Code
	}{
		{
			desc:  "issue for user with valid token",
			id:    email,
			email: email,
			kind:  authn.UserKey,
			err:   nil,
			code:  codes.OK,
		},
		{
			desc:  "issue recovery key",
			id:    email,
			email: email,
			kind:  authn.RecoveryKey,
			err:   nil,
			code:  codes.OK,
		},
		{
			desc:  "issue API key unauthenticated",
			id:    email,
			email: email,
			kind:  authn.APIKey,
			err:   nil,
			code:  codes.Unauthenticated,
		},
		{
			desc:  "issue for invalid key type",
			id:    email,
			email: email,
			kind:  32,
			err:   status.Error(codes.InvalidArgument, "received invalid token request"),
			code:  codes.InvalidArgument,
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
		_, err := client.Issue(context.Background(), &mainflux.IssueReq{Id: tc.id, Email: tc.email, Type: tc.kind})
		e, ok := status.FromError(err)
		assert.True(t, ok, "gRPC status can't be extracted from the error")
		assert.Equal(t, tc.code, e.Code(), fmt.Sprintf("%s: expected %s got %s", tc.desc, tc.code, e.Code()))
	}
}

func TestIdentify(t *testing.T) {
	_, loginSecret, err := svc.Issue(context.Background(), "", authn.Key{Type: authn.UserKey, IssuedAt: time.Now(), IssuerID: id, Subject: email})
	assert.Nil(t, err, fmt.Sprintf("Issuing user key expected to succeed: %s", err))

	_, recoverySecret, err := svc.Issue(context.Background(), "", authn.Key{Type: authn.RecoveryKey, IssuedAt: time.Now(), IssuerID: id, Subject: email})
	assert.Nil(t, err, fmt.Sprintf("Issuing recovery key expected to succeed: %s", err))

	_, apiSecret, err := svc.Issue(context.Background(), loginSecret, authn.Key{Type: authn.APIKey, IssuedAt: time.Now(), ExpiresAt: time.Now().Add(time.Minute), IssuerID: id, Subject: email})
	assert.Nil(t, err, fmt.Sprintf("Issuing API key expected to succeed: %s", err))

	authAddr := fmt.Sprintf("localhost:%d", port)
	conn, _ := grpc.Dial(authAddr, grpc.WithInsecure())
	client := grpcapi.NewClient(mocktracer.New(), conn, time.Second)

	cases := []struct {
		desc  string
		token string
		idt   mainflux.UserIdentity
		err   error
		code  codes.Code
	}{
		{
			desc:  "identify user with user token",
			token: loginSecret,
			idt:   mainflux.UserIdentity{Email: email, Id: id},
			err:   nil,
			code:  codes.OK,
		},
		{
			desc:  "identify user with recovery token",
			token: recoverySecret,
			idt:   mainflux.UserIdentity{Email: email, Id: id},
			err:   nil,
			code:  codes.OK,
		},
		{
			desc:  "identify user with API token",
			token: apiSecret,
			idt:   mainflux.UserIdentity{Email: email, Id: id},
			err:   nil,
			code:  codes.OK,
		},
		{
			desc:  "identify user with invalid user token",
			token: "invalid",
			idt:   mainflux.UserIdentity{},
			err:   status.Error(codes.Unauthenticated, "unauthorized access"),
			code:  codes.Unauthenticated,
		},
		{
			desc:  "identify user that doesn't exist",
			token: "",
			idt:   mainflux.UserIdentity{},
			err:   status.Error(codes.InvalidArgument, "received invalid token request"),
			code:  codes.InvalidArgument,
		},
	}

	for _, tc := range cases {
		idt, err := client.Identify(context.Background(), &mainflux.Token{Value: tc.token})
		if idt != nil {
			assert.Equal(t, tc.idt, *idt, fmt.Sprintf("%s: expected %v got %v", tc.desc, tc.idt, *idt))
		}
		e, ok := status.FromError(err)
		assert.True(t, ok, "gRPC status can't be extracted from the error")
		assert.Equal(t, tc.code, e.Code(), fmt.Sprintf("%s: expected %s got %s", tc.desc, tc.code, e.Code()))
	}
}
