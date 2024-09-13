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
	authmocks "github.com/absmach/magistrala/pkg/auth/mocks"
	"github.com/absmach/magistrala/pkg/errors"
	svcerr "github.com/absmach/magistrala/pkg/errors/service"
	"github.com/absmach/magistrala/pkg/policy"
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
	thingKey  = "testKey"
	channelID = "testID"
	invalid   = "invalid"
	valid     = "valid"
)

func startGRPCServer(svc *mocks.Service, auth *authmocks.AuthClient, port int) {
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		panic(fmt.Sprintf("failed to obtain port: %s", err))
	}
	server := grpc.NewServer()
	magistrala.RegisterAuthzServiceServer(server, grpcapi.NewServer(svc, auth))
	go func() {
		if err := server.Serve(listener); err != nil {
			panic(fmt.Sprintf("failed to serve: %s", err))
		}
	}()
}

func TestAuthorize(t *testing.T) {
	svc := new(mocks.Service)
	auth := new(authmocks.AuthClient)
	startGRPCServer(svc, auth, port)
	authAddr := fmt.Sprintf("localhost:%d", port)
	conn, _ := grpc.NewClient(authAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	client := grpcapi.NewClient(conn, time.Second)

	cases := []struct {
		desc         string
		req          *magistrala.AuthorizeReq
		res          *magistrala.AuthorizeRes
		thingID      string
		identifyKey  string
		authorizeReq *magistrala.AuthorizeReq
		authorizeErr error
		identifyErr  error
		err          error
		code         codes.Code
	}{
		{
			desc:    "authorize successfully",
			thingID: thingID,
			req: &magistrala.AuthorizeReq{
				SubjectType: policy.ThingType,
				Permission:  policy.PublishPermission,
				Subject:     thingKey,
				Object:      channelID,
				ObjectType:  policy.GroupType,
			},
			authorizeReq: &magistrala.AuthorizeReq{
				SubjectType: policy.GroupType,
				Subject:     channelID,
				ObjectType:  policy.ThingType,
				Object:      thingID,
				Permission:  policy.PublishPermission,
			},
			identifyKey: thingKey,
			res:         &magistrala.AuthorizeRes{Authorized: true, Id: thingID},
			err:         nil,
		},
		{
			desc: "authorize with invalid key",
			req: &magistrala.AuthorizeReq{
				Subject:     invalid,
				SubjectKind: policy.ThingsKind,
				Permission:  policy.PublishPermission,
				SubjectType: policy.ThingType,
				Object:      channelID,
				ObjectType:  policy.GroupType,
			},
			identifyKey: invalid,
			identifyErr: svcerr.ErrAuthentication,
			res:         &magistrala.AuthorizeRes{},
			err:         svcerr.ErrAuthentication,
		},
		{
			desc:    "authorize with failed authorization",
			thingID: thingID,
			req: &magistrala.AuthorizeReq{
				SubjectType: policy.ThingType,
				Permission:  policy.PublishPermission,
				Subject:     thingKey,
				Object:      channelID,
				ObjectType:  policy.GroupType,
			},
			authorizeReq: &magistrala.AuthorizeReq{
				SubjectType: policy.GroupType,
				Subject:     channelID,
				ObjectType:  policy.ThingType,
				Object:      thingID,
				Permission:  policy.PublishPermission,
			},
			identifyKey: thingKey,
			res:         &magistrala.AuthorizeRes{Authorized: false},
			err:         svcerr.ErrAuthorization,
		},

		{
			desc:    "authorize with invalid permission",
			thingID: thingID,
			req: &magistrala.AuthorizeReq{
				SubjectType: policy.ThingType,
				Permission:  invalid,
				Subject:     thingKey,
				Object:      channelID,
				ObjectType:  policy.GroupType,
			},
			authorizeReq: &magistrala.AuthorizeReq{
				SubjectType: policy.GroupType,
				Subject:     channelID,
				ObjectType:  policy.ThingType,
				Object:      thingID,
				Permission:  invalid,
			},
			identifyKey:  thingKey,
			authorizeErr: svcerr.ErrAuthorization,
			res:          &magistrala.AuthorizeRes{Authorized: false},
			err:          svcerr.ErrAuthorization,
		},
		{
			desc:    "authorize with invalid channel ID",
			thingID: thingID,
			req: &magistrala.AuthorizeReq{
				SubjectType: policy.ThingType,
				Permission:  policy.PublishPermission,
				Subject:     thingKey,
				Object:      invalid,
				ObjectType:  policy.GroupType,
			},
			authorizeReq: &magistrala.AuthorizeReq{
				SubjectType: policy.GroupType,
				Subject:     invalid,
				ObjectType:  policy.ThingType,
				Object:      thingID,
				Permission:  policy.PublishPermission,
			},
			identifyKey:  thingKey,
			authorizeErr: svcerr.ErrAuthorization,
			res:          &magistrala.AuthorizeRes{Authorized: false},
			err:          svcerr.ErrAuthorization,
		},
		{
			desc:    "authorize with empty channel ID",
			thingID: thingID,
			req: &magistrala.AuthorizeReq{
				SubjectType: policy.ThingType,
				Permission:  policy.PublishPermission,
				Subject:     thingKey,
				Object:      "",
				ObjectType:  policy.GroupType,
			},
			authorizeReq: &magistrala.AuthorizeReq{
				SubjectType: policy.GroupType,
				Subject:     "",
				ObjectType:  policy.ThingType,
				Object:      thingID,
				Permission:  policy.PublishPermission,
			},
			identifyKey: thingKey,
			res:         &magistrala.AuthorizeRes{Authorized: false},
			err:         svcerr.ErrAuthorization,
		},
		{
			desc:    "authorize with empty permission",
			thingID: thingID,
			req: &magistrala.AuthorizeReq{
				SubjectType: policy.ThingType,
				Permission:  "",
				Subject:     thingKey,
				Object:      channelID,
				ObjectType:  policy.GroupType,
			},
			authorizeReq: &magistrala.AuthorizeReq{
				SubjectType: policy.GroupType,
				Subject:     channelID,
				ObjectType:  policy.ThingType,
				Object:      thingID,
				Permission:  "",
			},
			identifyKey:  thingKey,
			authorizeErr: svcerr.ErrAuthorization,
			res:          &magistrala.AuthorizeRes{Authorized: false},
			err:          svcerr.ErrAuthorization,
		},
	}

	for _, tc := range cases {
		svcCall := svc.On("Identify", mock.Anything, tc.identifyKey).Return(tc.thingID, tc.identifyErr)
		authCall := auth.On("Authorize", mock.Anything, tc.authorizeReq).Return(tc.res, tc.authorizeErr)
		res, err := client.Authorize(context.Background(), tc.req)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s", tc.desc, tc.err, err))
		assert.Equal(t, tc.res, res, fmt.Sprintf("%s: expected %s got %s", tc.desc, tc.res, res))
		svcCall.Unset()
		authCall.Unset()
	}
}
