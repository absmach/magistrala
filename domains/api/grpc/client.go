// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package grpc

import (
	"context"
	"time"

	grpcCommonV1 "github.com/absmach/magistrala/api/grpc/common/v1"
	grpcDomainsV1 "github.com/absmach/magistrala/api/grpc/domains/v1"
	grpcapi "github.com/absmach/magistrala/auth/api/grpc"
	"github.com/go-kit/kit/endpoint"
	kitgrpc "github.com/go-kit/kit/transport/grpc"
	"google.golang.org/grpc"
)

const domainsSvcName = "domains.v1.DomainsService"

var _ grpcDomainsV1.DomainsServiceClient = (*domainsGrpcClient)(nil)

type domainsGrpcClient struct {
	deleteUserFromDomains endpoint.Endpoint
	retrieveStatus        endpoint.Endpoint
	retrieveIDByRoute     endpoint.Endpoint
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
		retrieveStatus: kitgrpc.NewClient(
			conn,
			domainsSvcName,
			"RetrieveStatus",
			encodeRetrieveStatusRequest,
			decodeRetrieveStatusResponse,
			grpcCommonV1.RetrieveEntityRes{},
		).Endpoint(),
		retrieveIDByRoute: kitgrpc.NewClient(
			conn,
			domainsSvcName,
			"RetrieveIDByRoute",
			encodeRetrieveIDByRouteRequest,
			decodeRetrieveIDByRouteResponse,
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

func decodeDeleteUserResponse(_ context.Context, grpcRes any) (any, error) {
	res := grpcRes.(*grpcDomainsV1.DeleteUserRes)
	return deleteUserRes{deleted: res.GetDeleted()}, nil
}

func encodeDeleteUserRequest(_ context.Context, grpcReq any) (any, error) {
	req := grpcReq.(deleteUserPoliciesReq)
	return &grpcDomainsV1.DeleteUserReq{
		Id: req.ID,
	}, nil
}

func (client domainsGrpcClient) RetrieveStatus(ctx context.Context, in *grpcCommonV1.RetrieveEntityReq, opts ...grpc.CallOption) (*grpcCommonV1.RetrieveEntityRes, error) {
	ctx, cancel := context.WithTimeout(ctx, client.timeout)
	defer cancel()

	res, err := client.retrieveStatus(ctx, retrieveStatusReq{
		ID: in.GetId(),
	})
	if err != nil {
		return &grpcCommonV1.RetrieveEntityRes{}, grpcapi.DecodeError(err)
	}

	rdsr := res.(retrieveStatusRes)
	return &grpcCommonV1.RetrieveEntityRes{
		Entity: &grpcCommonV1.EntityBasic{
			Status: uint32(rdsr.status),
		},
	}, nil
}

func decodeRetrieveStatusResponse(_ context.Context, grpcRes any) (any, error) {
	res := grpcRes.(*grpcCommonV1.RetrieveEntityRes)
	return retrieveStatusRes{status: uint8(res.Entity.GetStatus())}, nil
}

func encodeRetrieveStatusRequest(_ context.Context, grpcReq any) (any, error) {
	req := grpcReq.(retrieveStatusReq)
	return &grpcCommonV1.RetrieveEntityReq{
		Id: req.ID,
	}, nil
}

func (client domainsGrpcClient) RetrieveIDByRoute(ctx context.Context, in *grpcCommonV1.RetrieveIDByRouteReq, opts ...grpc.CallOption) (*grpcCommonV1.RetrieveEntityRes, error) {
	ctx, cancel := context.WithTimeout(ctx, client.timeout)
	defer cancel()

	res, err := client.retrieveIDByRoute(ctx, retrieveIDByRouteReq{
		Route: in.GetRoute(),
	})
	if err != nil {
		return &grpcCommonV1.RetrieveEntityRes{}, grpcapi.DecodeError(err)
	}

	rbr := res.(retrieveIDByRouteRes)
	return &grpcCommonV1.RetrieveEntityRes{
		Entity: &grpcCommonV1.EntityBasic{
			Id: rbr.id,
		},
	}, nil
}

func decodeRetrieveIDByRouteResponse(_ context.Context, grpcRes any) (any, error) {
	res := grpcRes.(*grpcCommonV1.RetrieveEntityRes)
	return retrieveIDByRouteRes{id: res.Entity.GetId()}, nil
}

func encodeRetrieveIDByRouteRequest(_ context.Context, grpcReq any) (any, error) {
	req := grpcReq.(retrieveIDByRouteReq)
	return &grpcCommonV1.RetrieveIDByRouteReq{
		Route: req.Route,
	}, nil
}
