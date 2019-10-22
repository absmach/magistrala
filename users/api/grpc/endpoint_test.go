// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package grpc_test

import (
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/mainflux/mainflux"
	"github.com/mainflux/mainflux/users"
	grpcapi "github.com/mainflux/mainflux/users/api/grpc"
	"github.com/mainflux/mainflux/users/jwt"
	"github.com/mainflux/mainflux/users/mocks"
	"github.com/opentracing/opentracing-go/mocktracer"
	"github.com/stretchr/testify/assert"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const port = 8081

var (
	user = users.User{
		Email:    "john.doe@email.com",
		Password: "pass",
	}
	svc users.Service
)

func newService() users.Service {
	repo := mocks.NewUserRepository()
	hasher := mocks.NewHasher()
	idp := mocks.NewIdentityProvider()
	token := mocks.NewTokenizer()
	email := mocks.NewEmailer()

	return users.New(repo, hasher, idp, email, token)
}

func startGRPCServer(svc users.Service, port int) {
	listener, _ := net.Listen("tcp", fmt.Sprintf(":%d", port))
	server := grpc.NewServer()
	mainflux.RegisterUsersServiceServer(server, grpcapi.NewServer(mocktracer.New(), svc))
	go server.Serve(listener)
}

func TestIdentify(t *testing.T) {
	svc.Register(context.Background(), user)

	usersAddr := fmt.Sprintf("localhost:%d", port)
	conn, _ := grpc.Dial(usersAddr, grpc.WithInsecure())
	client := grpcapi.NewClient(mocktracer.New(), conn, time.Second)
	j := jwt.New("secret")
	token, _ := j.TemporaryKey(user.Email)

	cases := map[string]struct {
		token string
		id    string
		err   error
	}{
		"identify user with valid token":   {token, user.Email, nil},
		"identify user that doesn't exist": {"", "", status.Error(codes.InvalidArgument, "received invalid token request")},
	}

	for desc, tc := range cases {
		id, err := client.Identify(context.Background(), &mainflux.Token{Value: tc.token})
		assert.Equal(t, tc.id, id.GetValue(), fmt.Sprintf("%s: expected %s got %s", desc, tc.id, id.GetValue()))
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected %s got %s", desc, tc.err, err))
	}
}
