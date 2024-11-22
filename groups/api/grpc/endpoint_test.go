// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package grpc_test

import (
	"context"
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/absmach/magistrala/groups"
	grpcapi "github.com/absmach/magistrala/groups/api/grpc"
	prmocks "github.com/absmach/magistrala/groups/private/mocks"
	grpcCommonV1 "github.com/absmach/magistrala/internal/grpc/common/v1"
	grpcGroupsV1 "github.com/absmach/magistrala/internal/grpc/groups/v1"
	"github.com/absmach/magistrala/internal/testsutil"
	"github.com/absmach/magistrala/pkg/errors"
	svcerr "github.com/absmach/magistrala/pkg/errors/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

const port = 7004

var (
	validID        = testsutil.GenerateUUID(&testing.T{})
	valid          = "valid"
	validGroupResp = groups.Group{
		ID:          testsutil.GenerateUUID(&testing.T{}),
		Name:        valid,
		Description: valid,
		Domain:      testsutil.GenerateUUID(&testing.T{}),
		Parent:      testsutil.GenerateUUID(&testing.T{}),
		Metadata: groups.Metadata{
			"name": "test",
		},
		Children:  []*groups.Group{},
		CreatedAt: time.Now().Add(-1 * time.Second),
		UpdatedAt: time.Now(),
		UpdatedBy: testsutil.GenerateUUID(&testing.T{}),
		Status:    groups.EnabledStatus,
	}
)

func startGRPCServer(svc *prmocks.Service, port int) {
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		panic(fmt.Sprintf("failed to obtain port: %s", err))
	}
	server := grpc.NewServer()
	grpcGroupsV1.RegisterGroupsServiceServer(server, grpcapi.NewServer(svc))
	go func() {
		if err := server.Serve(listener); err != nil {
			panic(fmt.Sprintf("failed to serve: %s", err))
		}
	}()
}

func TestRetrieveEntityEndpoint(t *testing.T) {
	svc := new(prmocks.Service)
	startGRPCServer(svc, port)
	grpAddr := fmt.Sprintf("localhost:%d", port)
	conn, _ := grpc.NewClient(grpAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	client := grpcapi.NewClient(conn, time.Second)

	cases := []struct {
		desc   string
		req    *grpcCommonV1.RetrieveEntityReq
		svcRes groups.Group
		svcErr error
		res    *grpcCommonV1.RetrieveEntityRes
		err    error
	}{
		{
			desc: "retrieve group successfully",
			req: &grpcCommonV1.RetrieveEntityReq{
				Id: validID,
			},
			svcRes: validGroupResp,
			svcErr: nil,
			res: &grpcCommonV1.RetrieveEntityRes{
				Entity: &grpcCommonV1.EntityBasic{
					Id:            validGroupResp.ID,
					DomainId:      validGroupResp.Domain,
					ParentGroupId: validGroupResp.Parent,
					Status:        uint32(validGroupResp.Status),
				},
			},
			err: nil,
		},
		{
			desc: "retrieve group with authentication error",
			req: &grpcCommonV1.RetrieveEntityReq{
				Id: validID,
			},
			svcErr: svcerr.ErrAuthentication,
			res:    &grpcCommonV1.RetrieveEntityRes{},
			err:    svcerr.ErrAuthentication,
		},
		{
			desc: "retrieve group with authorization error",
			req: &grpcCommonV1.RetrieveEntityReq{
				Id: validID,
			},
			svcErr: svcerr.ErrAuthorization,
			res:    &grpcCommonV1.RetrieveEntityRes{},
			err:    svcerr.ErrAuthorization,
		},
		{
			desc: "retrieve group with not found error",
			req: &grpcCommonV1.RetrieveEntityReq{
				Id: validID,
			},
			svcErr: svcerr.ErrNotFound,
			res:    &grpcCommonV1.RetrieveEntityRes{},
			err:    svcerr.ErrNotFound,
		},
		{
			desc: "retrieve group with malformed entity error",
			req: &grpcCommonV1.RetrieveEntityReq{
				Id: validID,
			},
			svcErr: errors.ErrMalformedEntity,
			res:    &grpcCommonV1.RetrieveEntityRes{},
			err:    errors.ErrMalformedEntity,
		},
		{
			desc: "retrieve group with conflict error",
			req: &grpcCommonV1.RetrieveEntityReq{
				Id: validID,
			},
			svcErr: svcerr.ErrConflict,
			res:    &grpcCommonV1.RetrieveEntityRes{},
			err:    svcerr.ErrConflict,
		},
		{
			desc: "retrieve group with unknown error",
			req: &grpcCommonV1.RetrieveEntityReq{
				Id: validID,
			},
			svcErr: errors.ErrUnidentified,
			res:    &grpcCommonV1.RetrieveEntityRes{},
			err:    errors.ErrUnidentified,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			svcCall := svc.On("RetrieveById", mock.Anything, tc.req.Id).Return(tc.svcRes, tc.svcErr)
			res, err := client.RetrieveEntity(context.Background(), tc.req)
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s", tc.desc, tc.err, err))
			assert.Equal(t, tc.res, res, fmt.Sprintf("%s: expected %s got %s", tc.desc, tc.res, res))
			svcCall.Unset()
		})
	}
}
