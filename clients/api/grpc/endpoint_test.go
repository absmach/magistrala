// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package grpc_test

import (
	"context"
	"fmt"
	"net"
	"testing"
	"time"

	grpcClientsV1 "github.com/absmach/supermq/api/grpc/clients/v1"
	grpcCommonV1 "github.com/absmach/supermq/api/grpc/common/v1"
	"github.com/absmach/supermq/clients"
	grpcapi "github.com/absmach/supermq/clients/api/grpc"
	"github.com/absmach/supermq/clients/private/mocks"
	"github.com/absmach/supermq/internal/testsutil"
	"github.com/absmach/supermq/pkg/connections"
	"github.com/absmach/supermq/pkg/errors"
	svcerr "github.com/absmach/supermq/pkg/errors/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

const port = 7006

var (
	validID       = testsutil.GenerateUUID(&testing.T{})
	validSecret   = "validSecret"
	invalidSecret = "invalidSecret"
	validClient   = clients.Client{
		ID:     validID,
		Domain: validID,
		Status: clients.EnabledStatus,
	}
)

func startGRPCServer(svc *mocks.Service, port int) *grpc.Server {
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		panic(fmt.Sprintf("failed to obtain port: %s", err))
	}
	server := grpc.NewServer()
	grpcClientsV1.RegisterClientsServiceServer(server, grpcapi.NewServer(svc))
	go func() {
		if err := server.Serve(listener); err != nil {
			panic(fmt.Sprintf("failed to serve: %s", err))
		}
	}()

	return server
}

func TestAuthenticate(t *testing.T) {
	svc := new(mocks.Service)
	server := startGRPCServer(svc, port)
	defer server.GracefulStop()
	authAddr := fmt.Sprintf("localhost:%d", port)
	conn, _ := grpc.NewClient(authAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	client := grpcapi.NewClient(conn, time.Second)

	cases := []struct {
		desc         string
		clientSecret string
		clientID     string
		resp         *grpcClientsV1.AuthnRes
		svcErr       error
		err          error
	}{
		{
			desc:         "authenticate successfully",
			clientSecret: validSecret,
			resp: &grpcClientsV1.AuthnRes{
				Authenticated: true,
				Id:            validID,
			},
			clientID: validID,
			svcErr:   nil,
			err:      nil,
		},
		{
			desc:         "failed to authenticate",
			clientSecret: invalidSecret,
			resp: &grpcClientsV1.AuthnRes{
				Authenticated: false,
				Id:            "",
			},
			clientID: "",
			svcErr:   svcerr.ErrAuthentication,
			err:      svcerr.ErrAuthentication,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			svcCall := svc.On("Authenticate", mock.Anything, tc.clientSecret).Return(tc.clientID, tc.svcErr)
			res, err := client.Authenticate(context.Background(), &grpcClientsV1.AuthnReq{ClientSecret: tc.clientSecret})
			assert.True(t, errors.Contains(err, tc.err))
			assert.Equal(t, tc.resp, res)
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
		svcRes clients.Client
		resp   *grpcCommonV1.RetrieveEntityRes
		svcErr error
		err    error
	}{
		{
			desc:   "retrieve entity successfully",
			id:     validID,
			svcRes: validClient,
			resp: &grpcCommonV1.RetrieveEntityRes{
				Entity: &grpcCommonV1.EntityBasic{
					Id:       validID,
					DomainId: validID,
					Status:   uint32(clients.EnabledStatus),
				},
			},
			err: nil,
		},
		{
			desc:   "retrieve entity with empty ID",
			id:     "",
			resp:   &grpcCommonV1.RetrieveEntityRes{},
			svcErr: svcerr.ErrNotFound,
			err:    svcerr.ErrNotFound,
		},
		{
			desc:   "retrieve entity with invalid ID",
			id:     "invalidID",
			resp:   &grpcCommonV1.RetrieveEntityRes{},
			svcErr: svcerr.ErrNotFound,
			err:    svcerr.ErrNotFound,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			svcCall := svc.On("RetrieveById", mock.Anything, tc.id).Return(tc.svcRes, tc.svcErr)
			res, err := client.RetrieveEntity(context.Background(), &grpcCommonV1.RetrieveEntityReq{Id: tc.id})
			assert.True(t, errors.Contains(err, tc.err))
			assert.Equal(t, tc.resp, res)
			svcCall.Unset()
		})
	}
}

func TestRetrieveEntities(t *testing.T) {
	svc := new(mocks.Service)
	server := startGRPCServer(svc, port)
	defer server.GracefulStop()
	authAddr := fmt.Sprintf("localhost:%d", port)
	conn, _ := grpc.NewClient(authAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	client := grpcapi.NewClient(conn, time.Second)

	cases := []struct {
		desc   string
		ids    []string
		svcRes clients.ClientsPage
		resp   *grpcCommonV1.RetrieveEntitiesRes
		svcErr error
		err    error
	}{
		{
			desc: "retrieve entities successfully",
			ids:  []string{validID},
			svcRes: clients.ClientsPage{
				Page: clients.Page{
					Total: 1,
					Limit: 1,
				},
				Clients: []clients.Client{validClient},
			},
			resp: &grpcCommonV1.RetrieveEntitiesRes{
				Total:  1,
				Limit:  1,
				Offset: 0,
				Entities: []*grpcCommonV1.EntityBasic{
					{
						Id:       validID,
						DomainId: validID,
						Status:   uint32(clients.EnabledStatus),
					},
				},
			},
			err: nil,
		},
		{
			desc:   "retrieve entities with empty IDs",
			ids:    []string(nil),
			resp:   &grpcCommonV1.RetrieveEntitiesRes{},
			svcErr: svcerr.ErrNotFound,
			err:    svcerr.ErrNotFound,
		},
		{
			desc:   "retrieve entities with invalid IDs",
			ids:    []string{"invalidID"},
			resp:   &grpcCommonV1.RetrieveEntitiesRes{},
			svcErr: svcerr.ErrNotFound,
			err:    svcerr.ErrNotFound,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			svcCall := svc.On("RetrieveByIds", mock.Anything, tc.ids).Return(tc.svcRes, tc.svcErr)
			res, err := client.RetrieveEntities(context.Background(), &grpcCommonV1.RetrieveEntitiesReq{Ids: tc.ids})
			assert.True(t, errors.Contains(err, tc.err))
			assert.Equal(t, tc.resp, res)
			svcCall.Unset()
		})
	}
}

func TestAddConnections(t *testing.T) {
	svc := new(mocks.Service)
	server := startGRPCServer(svc, port)
	defer server.GracefulStop()
	authAddr := fmt.Sprintf("localhost:%d", port)
	conn, _ := grpc.NewClient(authAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	client := grpcapi.NewClient(conn, time.Second)

	cases := []struct {
		desc   string
		req    *grpcCommonV1.AddConnectionsReq
		svcErr error
		err    error
	}{
		{
			desc: "add connections successfully",
			req: &grpcCommonV1.AddConnectionsReq{
				Connections: []*grpcCommonV1.Connection{
					{
						ClientId:  validID,
						ChannelId: validID,
						DomainId:  validID,
						Type:      uint32(connections.Publish),
					},
				},
			},
			err: nil,
		},
		{
			desc: "add connections with invalid request",
			req: &grpcCommonV1.AddConnectionsReq{
				Connections: []*grpcCommonV1.Connection{
					{
						ClientId:  "",
						ChannelId: "",
						DomainId:  "",
						Type:      uint32(connections.Publish),
					},
				},
			},
			svcErr: svcerr.ErrCreateEntity,
			err:    svcerr.ErrCreateEntity,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			svcCall := svc.On("AddConnections", mock.Anything, mock.Anything).Return(tc.svcErr)
			_, err := client.AddConnections(context.Background(), tc.req)
			assert.True(t, errors.Contains(err, tc.err))
			svcCall.Unset()
		})
	}
}

func TestRemoveConnections(t *testing.T) {
	svc := new(mocks.Service)
	server := startGRPCServer(svc, port)
	defer server.GracefulStop()
	authAddr := fmt.Sprintf("localhost:%d", port)
	conn, _ := grpc.NewClient(authAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	client := grpcapi.NewClient(conn, time.Second)

	cases := []struct {
		desc   string
		req    *grpcCommonV1.RemoveConnectionsReq
		svcErr error
		err    error
	}{
		{
			desc: "remove connections successfully",
			req: &grpcCommonV1.RemoveConnectionsReq{
				Connections: []*grpcCommonV1.Connection{
					{
						ClientId:  validID,
						ChannelId: validID,
						DomainId:  validID,
						Type:      uint32(connections.Publish),
					},
				},
			},
			err: nil,
		},
		{
			desc: "remove connections with invalid request",
			req: &grpcCommonV1.RemoveConnectionsReq{
				Connections: []*grpcCommonV1.Connection{
					{
						ClientId:  "",
						ChannelId: "",
						DomainId:  "",
						Type:      uint32(connections.Publish),
					},
				},
			},
			svcErr: svcerr.ErrRemoveEntity,
			err:    svcerr.ErrRemoveEntity,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			svcCall := svc.On("RemoveConnections", mock.Anything, mock.Anything).Return(tc.svcErr)
			_, err := client.RemoveConnections(context.Background(), tc.req)
			assert.True(t, errors.Contains(err, tc.err))
			svcCall.Unset()
		})
	}
}

func TestRemoveChannelConnections(t *testing.T) {
	svc := new(mocks.Service)
	server := startGRPCServer(svc, port)
	defer server.GracefulStop()
	authAddr := fmt.Sprintf("localhost:%d", port)
	conn, _ := grpc.NewClient(authAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	client := grpcapi.NewClient(conn, time.Second)

	cases := []struct {
		desc   string
		req    *grpcClientsV1.RemoveChannelConnectionsReq
		svcErr error
		err    error
	}{
		{
			desc: "remove channel connections successfully",
			req: &grpcClientsV1.RemoveChannelConnectionsReq{
				ChannelId: validID,
			},
			err: nil,
		},
		{
			desc: "remove channel connections with invalid request",
			req: &grpcClientsV1.RemoveChannelConnectionsReq{
				ChannelId: "",
			},
			svcErr: svcerr.ErrRemoveEntity,
			err:    svcerr.ErrRemoveEntity,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			svcCall := svc.On("RemoveChannelConnections", mock.Anything, tc.req.ChannelId).Return(tc.svcErr)
			_, err := client.RemoveChannelConnections(context.Background(), tc.req)
			assert.True(t, errors.Contains(err, tc.err))
			svcCall.Unset()
		})
	}
}

func TestUnsetParentGroupFromClient(t *testing.T) {
	svc := new(mocks.Service)
	server := startGRPCServer(svc, port)
	defer server.GracefulStop()
	authAddr := fmt.Sprintf("localhost:%d", port)
	conn, _ := grpc.NewClient(authAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	client := grpcapi.NewClient(conn, time.Second)

	cases := []struct {
		desc   string
		req    *grpcClientsV1.UnsetParentGroupFromClientReq
		svcErr error
		err    error
	}{
		{
			desc: "unset parent group successfully",
			req: &grpcClientsV1.UnsetParentGroupFromClientReq{
				ParentGroupId: validID,
			},
			err: nil,
		},
		{
			desc: "unset parent group with invalid request",
			req: &grpcClientsV1.UnsetParentGroupFromClientReq{
				ParentGroupId: "",
			},
			svcErr: svcerr.ErrRemoveEntity,
			err:    svcerr.ErrRemoveEntity,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			svcCall := svc.On("UnsetParentGroupFromClient", mock.Anything, tc.req.ParentGroupId).Return(tc.svcErr)
			_, err := client.UnsetParentGroupFromClient(context.Background(), tc.req)
			assert.True(t, errors.Contains(err, tc.err))
			svcCall.Unset()
		})
	}
}
