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
	"github.com/absmach/magistrala/auth"
	"github.com/absmach/magistrala/internal/apiutil"
	"github.com/absmach/magistrala/pkg/errors"
	grpcapi "github.com/absmach/magistrala/things/api/grpc"
	"github.com/absmach/magistrala/things/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
)

const port = 7000

var (
	thingID   = "testID"
	channelID = "testID"
	invalid   = "invalid"
	valid     = "valid"
)

func startGRPCServer(svc *mocks.Service, port int) {
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		panic(fmt.Sprintf("failed to obtain port: %s", err))
	}
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
		res          *magistrala.AuthorizeRes
		thingID      string
		authorizeErr error
		err          error
		code         codes.Code
	}{
		{
			desc: "authorize successfully",
			authorizeReq: &magistrala.AuthorizeReq{
				Subject:     thingID,
				SubjectKind: auth.ThingsKind,
				Permission:  valid,
				SubjectType: auth.ThingType,
				Object:      channelID,
				ObjectType:  auth.GroupType,
			},
			thingID: thingID,
			res:     &magistrala.AuthorizeRes{Authorized: true, Id: thingID},
			err:     nil,
		},
		{
			desc: "authorize with invalid id",
			authorizeReq: &magistrala.AuthorizeReq{
				Subject:     invalid,
				SubjectKind: auth.ThingsKind,
				Permission:  "publish",
				SubjectType: auth.ThingType,
				Object:      channelID,
				ObjectType:  auth.GroupType,
			},
			res: &magistrala.AuthorizeRes{},
			err: errors.ErrAuthentication,
		},
		{
			desc: "authorize with missing ID",
			authorizeReq: &magistrala.AuthorizeReq{
				Subject:     "",
				SubjectKind: auth.ThingsKind,
				Permission:  valid,
				SubjectType: auth.ThingType,
				Object:      channelID,
				ObjectType:  auth.GroupType,
			},
			res: &magistrala.AuthorizeRes{},
			err: apiutil.ErrMissingID,
		},
		{
			desc: "authorize with unauthorized id",
			authorizeReq: &magistrala.AuthorizeReq{
				Subject:     invalid,
				SubjectKind: auth.ThingsKind,
				Permission:  valid,
				SubjectType: auth.ThingType,
				Object:      channelID,
				ObjectType:  auth.GroupType,
			},
			res: &magistrala.AuthorizeRes{},
			err: errors.ErrAuthorization,
		},
		{
			desc: "authorize with invalid permission",
			authorizeReq: &magistrala.AuthorizeReq{
				Subject:     thingID,
				SubjectKind: auth.ThingsKind,
				Permission:  invalid,
				SubjectType: auth.ThingType,
				Object:      channelID,
				ObjectType:  auth.GroupType,
			},
			res: &magistrala.AuthorizeRes{},
			err: errors.ErrAuthorization,
		},
		{
			desc: "authorize with invalid channel ID",
			authorizeReq: &magistrala.AuthorizeReq{
				Subject:     thingID,
				SubjectKind: auth.ThingsKind,
				Permission:  valid,
				SubjectType: auth.ThingType,
				Object:      invalid,
				ObjectType:  auth.GroupType,
			},
			res: &magistrala.AuthorizeRes{},
			err: errors.ErrAuthorization,
		},
		{
			desc: "authorize with empty channel ID",
			authorizeReq: &magistrala.AuthorizeReq{
				Subject:     thingID,
				SubjectKind: auth.ThingsKind,
				Permission:  valid,
				SubjectType: auth.ThingType,
				Object:      "",
				ObjectType:  auth.GroupType,
			},
			res: &magistrala.AuthorizeRes{},
			err: errors.ErrAuthorization,
		},
		{
			desc: "authorize with empty permission",
			authorizeReq: &magistrala.AuthorizeReq{
				Subject:     thingID,
				SubjectKind: auth.ThingsKind,
				Permission:  "",
				SubjectType: auth.ThingType,
				Object:      channelID,
				ObjectType:  auth.GroupType,
			},
			res: &magistrala.AuthorizeRes{},
			err: errors.ErrAuthorization,
		},
	}

	for _, tc := range cases {
		svcCall := svc.On("Authorize", mock.Anything, tc.authorizeReq).Return(tc.thingID, tc.err)
		res, err := client.Authorize(context.Background(), tc.authorizeReq)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s", tc.desc, tc.err, err))
		assert.Equal(t, tc.res, res, fmt.Sprintf("%s: expected %s got %s", tc.desc, tc.res, res))
		svcCall.Unset()
	}
}
