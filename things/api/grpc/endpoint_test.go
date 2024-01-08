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
	"github.com/absmach/magistrala/internal/apiutil"
	"github.com/absmach/magistrala/pkg/errors"
	grpcapi "github.com/absmach/magistrala/things/api/grpc"
	"github.com/absmach/magistrala/things/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
)

const port = 7000

func startGRPCServer(svc *mocks.Service, port int) {
	listener, _ := net.Listen("tcp", fmt.Sprintf(":%d", port))
	server := grpc.NewServer()
	magistrala.RegisterAuthzServiceServer(server, grpcapi.NewServer(svc))
	go func() {
		if err := server.Serve(listener); err != nil {
			panic(fmt.Sprintf("failed to serve: %s", err))
		}
	}()
}

func TestAuthorize(t *testing.T) {
	svc := new(mocks.Service)
	startGRPCServer(svc, port)
	authAddr := fmt.Sprintf("localhost:%d", port)
	conn, _ := grpc.Dial(authAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	client := grpcapi.NewClient(conn, time.Second)

	cases := []struct {
		desc         string
		authorizeReq *magistrala.AuthorizeReq
		thingID      string
		authorizeErr error
		err          error
		code         codes.Code
	}{
		{
			desc: "authorize successfully",
			authorizeReq: &magistrala.AuthorizeReq{
				Subject:     "testID",
				SubjectKind: "Thing",
				SubjectType: "ID",
				Domain:      "testDomain",
				Object:      "testID",
			},
			thingID: "testID",
			code:    codes.OK,
			err:     nil,
		},
		{
			desc: "authorize with invalid id",
			authorizeReq: &magistrala.AuthorizeReq{
				Subject:     "testID",
				SubjectKind: "Thing",
				SubjectType: "ID",
				Domain:      "testDomain",
				Object:      "testID",
			},
			err:  errors.ErrAuthentication,
			code: codes.Unauthenticated,
		},
		{
			desc: "authorize with missing ID",
			authorizeReq: &magistrala.AuthorizeReq{
				Subject:     "",
				SubjectKind: "Thing",
				SubjectType: "ID",
				Domain:      "testDomain",
				Object:      "testID",
			},
			err:  apiutil.ErrMissingID,
			code: codes.InvalidArgument,
		},
		{
			desc: "authorize with unauthorized id",
			authorizeReq: &magistrala.AuthorizeReq{
				Subject:     "invalidID",
				SubjectKind: "Thing",
				SubjectType: "ID",
				Domain:      "testDomain",
				Object:      "testID",
			},
			err:  errors.ErrAuthorization,
			code: codes.PermissionDenied,
		},
		{
			desc: "authorize with unfound entity",
			authorizeReq: &magistrala.AuthorizeReq{
				Subject:     "invalidID",
				SubjectKind: "Thing",
				SubjectType: "ID",
				Domain:      "testDomain",
				Object:      "testID",
			},
			err:  errors.ErrNotFound,
			code: codes.Internal,
		},
	}

	for _, tc := range cases {
		repocall := svc.On("Authorize", mock.Anything, &magistrala.AuthorizeReq{}).Return(tc.thingID, tc.err)
		_, err := client.Authorize(context.Background(), &magistrala.AuthorizeReq{})
		e, ok := status.FromError(err)
		assert.True(t, ok, "gRPC status can't be extracted from the error")
		assert.Equal(t, tc.code, e.Code(), fmt.Sprintf("%s: expected %s got %s", tc.desc, tc.code, e.Code()))
		repocall.Unset()
	}
}
