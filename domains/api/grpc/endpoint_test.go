// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package grpc_test

import (
	"context"
	"fmt"
	"net"
	"testing"
	"time"

	grpcCommonV1 "github.com/absmach/magistrala/api/grpc/common/v1"
	grpcDomainsV1 "github.com/absmach/magistrala/api/grpc/domains/v1"
	apiutil "github.com/absmach/magistrala/api/http/util"
	"github.com/absmach/magistrala/domains"
	grpcapi "github.com/absmach/magistrala/domains/api/grpc"
	pDomains "github.com/absmach/magistrala/domains/private"
	"github.com/absmach/magistrala/pkg/errors"
	svcerr "github.com/absmach/magistrala/pkg/errors/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

const (
	port            = 8081
	secret          = "secret"
	email           = "test@example.com"
	id              = "testID"
	clientsType     = "clients"
	usersType       = "users"
	description     = "Description"
	groupName       = "smqx"
	adminPermission = "admin"
	authoritiesObj  = "authorities"
	memberRelation  = "member"
	loginDuration   = 30 * time.Minute
	refreshDuration = 24 * time.Hour
	invalidDuration = 7 * 24 * time.Hour
	validToken      = "valid"
	inValidToken    = "invalid"
	validPolicy     = "valid"
)

var authAddr = fmt.Sprintf("localhost:%d", port)

func startGRPCServer(svc pDomains.Service, port int) *grpc.Server {
	listener, _ := net.Listen("tcp", fmt.Sprintf(":%d", port))
	server := grpc.NewServer()
	grpcDomainsV1.RegisterDomainsServiceServer(server, grpcapi.NewDomainsServer(svc))
	go func() {
		err := server.Serve(listener)
		assert.Nil(&testing.T{}, err, fmt.Sprintf(`"Unexpected error creating auth server %s"`, err))
	}()

	return server
}

func TestDeleteUserFromDomains(t *testing.T) {
	conn, err := grpc.NewClient(authAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	assert.Nil(t, err, fmt.Sprintf("Unexpected error creating client connection %s", err))
	grpcClient := grpcapi.NewDomainsClient(conn, time.Second)

	cases := []struct {
		desc          string
		token         string
		deleteUserReq *grpcDomainsV1.DeleteUserReq
		deleteUserRes *grpcDomainsV1.DeleteUserRes
		err           error
	}{
		{
			desc:  "delete valid req",
			token: validToken,
			deleteUserReq: &grpcDomainsV1.DeleteUserReq{
				Id: id,
			},
			deleteUserRes: &grpcDomainsV1.DeleteUserRes{Deleted: true},
			err:           nil,
		},
		{
			desc:          "delete invalid req with invalid token",
			token:         inValidToken,
			deleteUserReq: &grpcDomainsV1.DeleteUserReq{},
			deleteUserRes: &grpcDomainsV1.DeleteUserRes{Deleted: false},
			err:           apiutil.ErrMissingID,
		},
		{
			desc:  "delete invalid req with invalid token",
			token: inValidToken,
			deleteUserReq: &grpcDomainsV1.DeleteUserReq{
				Id: id,
			},
			deleteUserRes: &grpcDomainsV1.DeleteUserRes{Deleted: false},
			err:           apiutil.ErrMissingPolicyEntityType,
		},
	}
	for _, tc := range cases {
		repoCall := svc.On("DeleteUserFromDomains", mock.Anything, tc.deleteUserReq.Id).Return(tc.err)
		dpr, err := grpcClient.DeleteUserFromDomains(context.Background(), tc.deleteUserReq)
		assert.Equal(t, tc.deleteUserRes.GetDeleted(), dpr.GetDeleted(), fmt.Sprintf("%s: expected %v got %v", tc.desc, tc.deleteUserRes.GetDeleted(), dpr.GetDeleted()))
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		repoCall.Unset()
	}
}

func TestRetrieveStatus(t *testing.T) {
	conn, err := grpc.NewClient(authAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	assert.Nil(t, err, fmt.Sprintf("Unexpected error creating client connection %s", err))
	grpcClient := grpcapi.NewDomainsClient(conn, time.Second)

	cases := []struct {
		desc        string
		token       string
		retrieveReq *grpcCommonV1.RetrieveEntityReq
		svcRes      domains.Status
		svcErr      error
		retrieveRes *grpcCommonV1.RetrieveEntityRes
		err         error
	}{
		{
			desc:  "retrieve status with valid req",
			token: validToken,
			retrieveReq: &grpcCommonV1.RetrieveEntityReq{
				Id: id,
			},
			svcRes: domains.EnabledStatus,
			retrieveRes: &grpcCommonV1.RetrieveEntityRes{
				Entity: &grpcCommonV1.EntityBasic{
					Status: uint32(domains.EnabledStatus),
				},
			},
			err: nil,
		},
		{
			desc: "retrieve status with empty id",
			retrieveReq: &grpcCommonV1.RetrieveEntityReq{
				Id: "",
			},
			svcRes:      domains.AllStatus,
			retrieveRes: &grpcCommonV1.RetrieveEntityRes{},
			err:         apiutil.ErrMissingID,
		},
		{
			desc: "retrieve status with invalid id",
			retrieveReq: &grpcCommonV1.RetrieveEntityReq{
				Id: "invalid",
			},
			svcRes:      domains.AllStatus,
			svcErr:      svcerr.ErrNotFound,
			retrieveRes: &grpcCommonV1.RetrieveEntityRes{},
			err:         svcerr.ErrNotFound,
		},
	}
	for _, tc := range cases {
		svcCall := svc.On("RetrieveStatus", mock.Anything, tc.retrieveReq.Id).Return(tc.svcRes, tc.svcErr)
		dpr, err := grpcClient.RetrieveStatus(context.Background(), tc.retrieveReq)
		assert.Equal(t, tc.retrieveRes.Entity, dpr.Entity, fmt.Sprintf("%s: expected %v got %v", tc.desc, tc.retrieveRes.Entity, dpr.Entity))
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		svcCall.Unset()
	}
}

func TestRetrieveIDByRoute(t *testing.T) {
	conn, err := grpc.NewClient(authAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	assert.Nil(t, err, fmt.Sprintf("Unexpected error creating client connection %s", err))
	grpcClient := grpcapi.NewDomainsClient(conn, time.Second)

	validRoute := "validRoute"

	cases := []struct {
		desc        string
		retrieveReq *grpcCommonV1.RetrieveIDByRouteReq
		svcRes      string
		svcErr      error
		retrieveRes *grpcCommonV1.RetrieveEntityRes
		err         error
	}{
		{
			desc: "retrieve id with valid route",
			retrieveReq: &grpcCommonV1.RetrieveIDByRouteReq{
				Route: validRoute,
			},
			svcRes: id,
			retrieveRes: &grpcCommonV1.RetrieveEntityRes{
				Entity: &grpcCommonV1.EntityBasic{
					Id: id,
				},
			},
			err: nil,
		},
		{
			desc: "retrieve id with empty route",
			retrieveReq: &grpcCommonV1.RetrieveIDByRouteReq{
				Route: "",
			},
			svcRes:      "",
			retrieveRes: &grpcCommonV1.RetrieveEntityRes{},
			err:         apiutil.ErrMissingRoute,
		},
		{
			desc: "retrieve id with invalid route",
			retrieveReq: &grpcCommonV1.RetrieveIDByRouteReq{
				Route: "invalid",
			},
			svcRes:      "",
			svcErr:      svcerr.ErrNotFound,
			retrieveRes: &grpcCommonV1.RetrieveEntityRes{},
			err:         svcerr.ErrNotFound,
		},
	}
	for _, tc := range cases {
		svcCall := svc.On("RetrieveIDByRoute", mock.Anything, tc.retrieveReq.Route).Return(tc.svcRes, tc.svcErr)
		dpr, err := grpcClient.RetrieveIDByRoute(context.Background(), tc.retrieveReq)
		assert.Equal(t, tc.retrieveRes.Entity, dpr.Entity, fmt.Sprintf("%s: expected %v got %v", tc.desc, tc.retrieveRes.Entity, dpr.Entity))
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		svcCall.Unset()
	}
}
