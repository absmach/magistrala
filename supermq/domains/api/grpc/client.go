// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package grpc

import (
	"context"
	"time"

	grpcCommonV1 "github.com/absmach/supermq/api/grpc/common/v1"
	grpcDomainsV1 "github.com/absmach/supermq/api/grpc/domains/v1"
	grpcapi "github.com/absmach/supermq/auth/api/grpc"
	"github.com/go-kit/kit/endpoint"
	kitgrpc "github.com/go-kit/kit/transport/grpc"
	"google.golang.org/grpc"
)

const domainsSvcName = "domains.v1.DomainsService"

var _ grpcDomainsV1.DomainsServiceClient = (*domainsGrpcClient)(nil)

type domainsGrpcClient struct {
	deleteUserFromDomains endpoint.Endpoint
	retrieveEntity        endpoint.Endpoint
	timeout               time.Duration
}

// NewDomainsClient returns new domains gRPC client instance.
func NewDomainsClient(conn *grpc.ClientConn, timeout time.Duration) grpcDomainsV1.DomainsServiceClient {
	return &domainsGrpcClient{
		deleteUserFromDomains: kitgrpc.NewClient(
			conn,
			domainsSvcName,
			"DeleteUserFromDomains",
			encodeDeleteUserRequest,
			decodeDeleteUserResponse,
			grpcDomainsV1.DeleteUserRes{},
		).Endpoint(),
		retrieveEntity: kitgrpc.NewClient(
			conn,
			domainsSvcName,
			"RetrieveEntity",
			encodeRetrieveEntityRequest,
			decodeRetrieveEntityResponse,
			grpcCommonV1.RetrieveEntityRes{},
		).Endpoint(),
		timeout: timeout,
	}
}

func (client domainsGrpcClient) DeleteUserFromDomains(ctx context.Context, in *grpcDomainsV1.DeleteUserReq, opts ...grpc.CallOption) (*grpcDomainsV1.DeleteUserRes, error) {
	ctx, cancel := context.WithTimeout(ctx, client.timeout)
	defer cancel()

	res, err := client.deleteUserFromDomains(ctx, deleteUserPoliciesReq{
		ID: in.GetId(),
	})
	if err != nil {
		return &grpcDomainsV1.DeleteUserRes{}, grpcapi.DecodeError(err)
	}

	dpr := res.(deleteUserRes)
	return &grpcDomainsV1.DeleteUserRes{Deleted: dpr.deleted}, nil
}

func decodeDeleteUserResponse(_ context.Context, grpcRes interface{}) (interface{}, error) {
	res := grpcRes.(*grpcDomainsV1.DeleteUserRes)
	return deleteUserRes{deleted: res.GetDeleted()}, nil
}

func encodeDeleteUserRequest(_ context.Context, grpcReq interface{}) (interface{}, error) {
	req := grpcReq.(deleteUserPoliciesReq)
	return &grpcDomainsV1.DeleteUserReq{
		Id: req.ID,
	}, nil
}

func (client domainsGrpcClient) RetrieveEntity(ctx context.Context, in *grpcCommonV1.RetrieveEntityReq, opts ...grpc.CallOption) (*grpcCommonV1.RetrieveEntityRes, error) {
	ctx, cancel := context.WithTimeout(ctx, client.timeout)
	defer cancel()

	res, err := client.retrieveEntity(ctx, retrieveEntityReq{
		ID: in.GetId(),
	})
	if err != nil {
		return &grpcCommonV1.RetrieveEntityRes{}, grpcapi.DecodeError(err)
	}

	rdsr := res.(retrieveEntityRes)
	return &grpcCommonV1.RetrieveEntityRes{
		Entity: &grpcCommonV1.EntityBasic{
			Id:     rdsr.id,
			Status: uint32(rdsr.status),
		},
	}, nil
}

func decodeRetrieveEntityResponse(_ context.Context, grpcRes interface{}) (interface{}, error) {
	res := grpcRes.(*grpcCommonV1.RetrieveEntityRes)
	return retrieveEntityRes{id: res.Entity.GetId(), status: uint8(res.Entity.GetStatus())}, nil
}

func encodeRetrieveEntityRequest(_ context.Context, grpcReq interface{}) (interface{}, error) {
	req := grpcReq.(retrieveEntityReq)
	return &grpcCommonV1.RetrieveEntityReq{
		Id: req.ID,
	}, nil
}
