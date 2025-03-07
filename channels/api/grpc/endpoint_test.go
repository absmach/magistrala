// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package grpc_test

import (
	"context"
	"fmt"
	"net"
	"testing"
	"time"

	grpcChannelsV1 "github.com/absmach/supermq/api/grpc/channels/v1"
	grpcCommonV1 "github.com/absmach/supermq/api/grpc/common/v1"
	ch "github.com/absmach/supermq/channels"
	grpcapi "github.com/absmach/supermq/channels/api/grpc"
	"github.com/absmach/supermq/channels/private/mocks"
	"github.com/absmach/supermq/clients"
	"github.com/absmach/supermq/internal/testsutil"
	"github.com/absmach/supermq/pkg/connections"
	"github.com/absmach/supermq/pkg/errors"
	svcerr "github.com/absmach/supermq/pkg/errors/service"
	"github.com/absmach/supermq/pkg/policies"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
)

const port = 7005

var (
	validID      = testsutil.GenerateUUID(&testing.T{})
	validChannel = ch.Channel{
		ID:     validID,
		Domain: testsutil.GenerateUUID(&testing.T{}),
		Status: clients.EnabledStatus,
	}
)

func startGRPCServer(svc *mocks.Service, port int) *grpc.Server {
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		panic(fmt.Sprintf("failed to obtain port: %s", err))
	}
	server := grpc.NewServer()
	grpcChannelsV1.RegisterChannelsServiceServer(server, grpcapi.NewServer(svc))
	go func() {
		if err := server.Serve(listener); err != nil {
			panic(fmt.Sprintf("failed to serve: %s", err))
		}
	}()
	return server
}

func TestAuthorize(t *testing.T) {
	svc := new(mocks.Service)
	server := startGRPCServer(svc, port)
	defer server.GracefulStop()
	authAddr := fmt.Sprintf("localhost:%d", port)
	conn, _ := grpc.NewClient(authAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	client := grpcapi.NewClient(conn, time.Second)

	cases := []struct {
		desc       string
		domainID   string
		clientID   string
		clientType string
		channelID  string
		connType   connections.ConnType
		err        error
		authzErr   error
		res        *grpcChannelsV1.AuthzRes
		code       codes.Code
	}{
		{
			desc:       "authorize successfully",
			domainID:   validID,
			clientID:   validID,
			clientType: policies.UserType,
			channelID:  validID,
			connType:   connections.Publish,
			res:        &grpcChannelsV1.AuthzRes{Authorized: true},
			err:        nil,
		},
		{
			desc:       "authorize with authorization  error",
			domainID:   validID,
			clientID:   validID,
			clientType: policies.UserType,
			channelID:  validID,
			connType:   connections.Publish,
			res:        &grpcChannelsV1.AuthzRes{Authorized: false},
			authzErr:   svcerr.ErrAuthorization,
			err:        svcerr.ErrAuthorization,
		},
		{
			desc:       "authorize withnot found error",
			domainID:   validID,
			clientID:   validID,
			clientType: policies.UserType,
			channelID:  validID,
			connType:   connections.Publish,
			res:        &grpcChannelsV1.AuthzRes{Authorized: false},
			authzErr:   svcerr.ErrNotFound,
			err:        svcerr.ErrNotFound,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			authReq := ch.AuthzReq{
				DomainID:   tc.domainID,
				ClientID:   tc.clientID,
				ClientType: tc.clientType,
				ChannelID:  tc.channelID,
				Type:       tc.connType,
			}
			svcCall := svc.On("Authorize", mock.Anything, authReq).Return(tc.authzErr)
			res, err := client.Authorize(context.Background(), &grpcChannelsV1.AuthzReq{
				DomainId:   tc.domainID,
				ClientId:   tc.clientID,
				ClientType: tc.clientType,
				ChannelId:  tc.channelID,
				Type:       uint32(tc.connType),
			})
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s", tc.desc, tc.err, err))
			assert.Equal(t, tc.res, res, fmt.Sprintf("%s: expected %s got %s", tc.desc, tc.res, res))
			svcCall.Unset()
		})
	}
}

func TestRemoveClientConnections(t *testing.T) {
	svc := new(mocks.Service)
	server := startGRPCServer(svc, port)
	defer server.GracefulStop()
	authAddr := fmt.Sprintf("localhost:%d", port)
	conn, _ := grpc.NewClient(authAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	client := grpcapi.NewClient(conn, time.Second)

	cases := []struct {
		desc     string
		clientID string
		err      error
		code     codes.Code
	}{
		{
			desc:     "remove client connections successfully",
			clientID: validID,
			err:      nil,
		},
		{
			desc:     "remove client connections with error",
			clientID: validID,
			err:      svcerr.ErrNotFound,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			svcCall := svc.On("RemoveClientConnections", mock.Anything, tc.clientID).Return(tc.err)
			res, err := client.RemoveClientConnections(context.Background(), &grpcChannelsV1.RemoveClientConnectionsReq{
				ClientId: tc.clientID,
			})
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s", tc.desc, tc.err, err))
			assert.Equal(t, &grpcChannelsV1.RemoveClientConnectionsRes{}, res)
			svcCall.Unset()
		})
	}
}

func TestUnsetParentGroupFromChannelsEndpoint(t *testing.T) {
	svc := new(mocks.Service)
	server := startGRPCServer(svc, port)
	defer server.GracefulStop()
	authAddr := fmt.Sprintf("localhost:%d", port)
	conn, _ := grpc.NewClient(authAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	client := grpcapi.NewClient(conn, time.Second)

	cases := []struct {
		desc          string
		parentGroupID string
		err           error
		code          codes.Code
	}{
		{
			desc:          "unset parent group from channels successfully",
			parentGroupID: validID,
			err:           nil,
		},
		{
			desc:          "unset parent group from channels authorization error",
			parentGroupID: validID,
			err:           svcerr.ErrAuthorization,
		},
		{
			desc:          "unset parent group from channels with not found error",
			parentGroupID: validID,
			err:           svcerr.ErrNotFound,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			svcCall := svc.On("UnsetParentGroupFromChannels", mock.Anything, tc.parentGroupID).Return(tc.err)
			res, err := client.UnsetParentGroupFromChannels(context.Background(), &grpcChannelsV1.UnsetParentGroupFromChannelsReq{
				ParentGroupId: tc.parentGroupID,
			})
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s", tc.desc, tc.err, err))
			assert.Equal(t, &grpcChannelsV1.UnsetParentGroupFromChannelsRes{}, res)
			svcCall.Unset()
		})
	}
}

func TestRetrieveEntity(t *testing.T) {
	svc := new(mocks.Service)
	server := startGRPCServer(svc, port)
	defer server.GracefulStop()
	authAddr := fmt.Sprintf("localhost:%d", port)
	conn, _ := grpc.NewClient(authAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	client := grpcapi.NewClient(conn, time.Second)

	cases := []struct {
		desc   string
		id     string
		svcRes ch.Channel
		resp   *grpcCommonV1.RetrieveEntityRes
		code   codes.Code
		err    error
	}{
		{
			desc:   "retrieve entity successfully",
			id:     validID,
			svcRes: validChannel,
			resp: &grpcCommonV1.RetrieveEntityRes{
				Entity: &grpcCommonV1.EntityBasic{
					Id:            validChannel.ID,
					DomainId:      validChannel.Domain,
					ParentGroupId: validChannel.ParentGroup,
					Status:        uint32(validChannel.Status),
				},
			},
			err: nil,
		},
		{
			desc: "retrieve entity with error",
			id:   validID,
			resp: &grpcCommonV1.RetrieveEntityRes{},
			err:  svcerr.ErrNotFound,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			svcCall := svc.On("RetrieveByID", mock.Anything, tc.id).Return(tc.svcRes, tc.err)
			res, err := client.RetrieveEntity(context.Background(), &grpcCommonV1.RetrieveEntityReq{
				Id: tc.id,
			})
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s", tc.desc, tc.err, err))
			assert.Equal(t, tc.resp.Entity, res.Entity)
			svcCall.Unset()
		})
	}
}
