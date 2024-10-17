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
	"github.com/absmach/magistrala/pkg/errors"
	svcerr "github.com/absmach/magistrala/pkg/errors/service"
	"github.com/absmach/magistrala/pkg/policies"
	"github.com/absmach/magistrala/things"
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
)

func startGRPCServer(svc *mocks.Service, port int) {
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		panic(fmt.Sprintf("failed to obtain port: %s", err))
	}
	server := grpc.NewServer()
	magistrala.RegisterThingsServiceServer(server, grpcapi.NewServer(svc))
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
	conn, _ := grpc.NewClient(authAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	client := grpcapi.NewClient(conn, time.Second)

	cases := []struct {
		desc         string
		req          *magistrala.ThingsAuthzReq
		res          *magistrala.ThingsAuthzRes
		thingID      string
		identifyKey  string
		authorizeReq things.AuthzReq
		authorizeRes string
		authorizeErr error
		identifyErr  error
		err          error
		code         codes.Code
	}{
		{
			desc:    "authorize successfully",
			thingID: thingID,
			req: &magistrala.ThingsAuthzReq{
				ThingKey:   thingKey,
				ChannelID:  channelID,
				Permission: policies.PublishPermission,
			},
			authorizeReq: things.AuthzReq{
				ThingKey:   thingKey,
				ChannelID:  channelID,
				Permission: policies.PublishPermission,
			},
			authorizeRes: thingID,
			identifyKey:  thingKey,
			res:          &magistrala.ThingsAuthzRes{Authorized: true, Id: thingID},
			err:          nil,
		},
		{
			desc: "authorize with invalid key",
			req: &magistrala.ThingsAuthzReq{
				ThingKey:   invalid,
				ChannelID:  channelID,
				Permission: policies.PublishPermission,
			},
			authorizeReq: things.AuthzReq{
				ThingKey:   invalid,
				ChannelID:  channelID,
				Permission: policies.PublishPermission,
			},
			authorizeErr: svcerr.ErrAuthentication,
			identifyKey:  invalid,
			identifyErr:  svcerr.ErrAuthentication,
			res:          &magistrala.ThingsAuthzRes{},
			err:          svcerr.ErrAuthentication,
		},
		{
			desc:    "authorize with failed authorization",
			thingID: thingID,
			req: &magistrala.ThingsAuthzReq{
				ThingKey:   thingKey,
				ChannelID:  channelID,
				Permission: policies.PublishPermission,
			},
			authorizeReq: things.AuthzReq{
				ThingKey:   thingKey,
				ChannelID:  channelID,
				Permission: policies.PublishPermission,
			},
			authorizeErr: svcerr.ErrAuthorization,
			identifyKey:  thingKey,
			res:          &magistrala.ThingsAuthzRes{Authorized: false},
			err:          svcerr.ErrAuthorization,
		},

		{
			desc:    "authorize with invalid permission",
			thingID: thingID,
			req: &magistrala.ThingsAuthzReq{
				ThingKey:   thingKey,
				ChannelID:  channelID,
				Permission: invalid,
			},
			authorizeReq: things.AuthzReq{
				ChannelID:  channelID,
				ThingKey:   thingKey,
				Permission: invalid,
			},
			identifyKey:  thingKey,
			authorizeErr: svcerr.ErrAuthorization,
			res:          &magistrala.ThingsAuthzRes{Authorized: false},
			err:          svcerr.ErrAuthorization,
		},
		{
			desc:    "authorize with invalid channel ID",
			thingID: thingID,
			req: &magistrala.ThingsAuthzReq{
				ThingKey:   thingKey,
				ChannelID:  invalid,
				Permission: policies.PublishPermission,
			},
			authorizeReq: things.AuthzReq{
				ChannelID:  invalid,
				ThingKey:   thingKey,
				Permission: policies.PublishPermission,
			},
			identifyKey:  thingKey,
			authorizeErr: svcerr.ErrAuthorization,
			res:          &magistrala.ThingsAuthzRes{Authorized: false},
			err:          svcerr.ErrAuthorization,
		},
		{
			desc:    "authorize with empty channel ID",
			thingID: thingID,
			req: &magistrala.ThingsAuthzReq{
				ThingKey:   thingKey,
				ChannelID:  "",
				Permission: policies.PublishPermission,
			},
			authorizeReq: things.AuthzReq{
				ThingKey:   thingKey,
				ChannelID:  "",
				Permission: policies.PublishPermission,
			},
			authorizeErr: svcerr.ErrAuthorization,
			identifyKey:  thingKey,
			res:          &magistrala.ThingsAuthzRes{Authorized: false},
			err:          svcerr.ErrAuthorization,
		},
		{
			desc:    "authorize with empty permission",
			thingID: thingID,
			req: &magistrala.ThingsAuthzReq{
				ThingKey:   thingKey,
				ChannelID:  channelID,
				Permission: "",
			},
			authorizeReq: things.AuthzReq{
				ChannelID:  channelID,
				Permission: "",
				ThingKey:   thingKey,
			},
			identifyKey:  thingKey,
			authorizeErr: svcerr.ErrAuthorization,
			res:          &magistrala.ThingsAuthzRes{Authorized: false},
			err:          svcerr.ErrAuthorization,
		},
	}

	for _, tc := range cases {
		svcCall1 := svc.On("Identify", mock.Anything, tc.identifyKey).Return(tc.thingID, tc.identifyErr)
		svcCall2 := svc.On("Authorize", mock.Anything, tc.authorizeReq).Return(tc.thingID, tc.authorizeErr)
		res, err := client.Authorize(context.Background(), tc.req)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s", tc.desc, tc.err, err))
		assert.Equal(t, tc.res, res, fmt.Sprintf("%s: expected %s got %s", tc.desc, tc.res, res))
		svcCall1.Unset()
		svcCall2.Unset()
	}
}
