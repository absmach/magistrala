// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package grpc

import (
	"context"

	grpcCommonV1 "github.com/absmach/magistrala/api/grpc/common/v1"
	grpcDomainsV1 "github.com/absmach/magistrala/api/grpc/domains/v1"
	grpcapi "github.com/absmach/magistrala/auth/api/grpc"
	domains "github.com/absmach/magistrala/domains/private"
	kitgrpc "github.com/go-kit/kit/transport/grpc"
)

var _ grpcDomainsV1.DomainsServiceServer = (*domainsGrpcServer)(nil)

type domainsGrpcServer struct {
	grpcDomainsV1.UnimplementedDomainsServiceServer
	deleteUserFromDomains kitgrpc.Handler
	retrieveStatus        kitgrpc.Handler
	retrieveIDByRoute     kitgrpc.Handler
}

func NewDomainsServer(svc domains.Service) grpcDomainsV1.DomainsServiceServer {
	return &domainsGrpcServer{
		deleteUserFromDomains: kitgrpc.NewServer(
			(deleteUserFromDomainsEndpoint(svc)),
			decodeDeleteUserRequest,
			encodeDeleteUserResponse,
		),
		retrieveStatus: kitgrpc.NewServer(
			retrieveStatusEndpoint(svc),
			decodeRetrieveStatusRequest,
			encodeRetrieveStatusResponse,
		),
		retrieveIDByRoute: kitgrpc.NewServer(
			retrieveIDByRouteEndpoint(svc),
			decodeRetrieveIDByRouteRequest,
			encodeRetrieveIDByRouteResponse,
		),
	}
}

func decodeDeleteUserRequest(_ context.Context, grpcReq any) (any, error) {
	req := grpcReq.(*grpcDomainsV1.DeleteUserReq)
	return deleteUserPoliciesReq{
		ID: req.GetId(),
	}, nil
}

func encodeDeleteUserResponse(_ context.Context, grpcRes any) (any, error) {
	res := grpcRes.(deleteUserRes)
	return &grpcDomainsV1.DeleteUserRes{Deleted: res.deleted}, nil
}

func (s *domainsGrpcServer) DeleteUserFromDomains(ctx context.Context, req *grpcDomainsV1.DeleteUserReq) (*grpcDomainsV1.DeleteUserRes, error) {
	_, res, err := s.deleteUserFromDomains.ServeGRPC(ctx, req)
	if err != nil {
		return nil, grpcapi.EncodeError(err)
	}
	return res.(*grpcDomainsV1.DeleteUserRes), nil
}

func decodeRetrieveStatusRequest(_ context.Context, grpcReq any) (any, error) {
	req := grpcReq.(*grpcCommonV1.RetrieveEntityReq)

	return retrieveStatusReq{
		ID: req.GetId(),
	}, nil
}

func encodeRetrieveStatusResponse(_ context.Context, grpcRes any) (any, error) {
	res := grpcRes.(retrieveStatusRes)

	return &grpcCommonV1.RetrieveEntityRes{
		Entity: &grpcCommonV1.EntityBasic{
			Status: uint32(res.status),
		},
	}, nil
}

func (s *domainsGrpcServer) RetrieveStatus(ctx context.Context, req *grpcCommonV1.RetrieveEntityReq) (*grpcCommonV1.RetrieveEntityRes, error) {
	_, res, err := s.retrieveStatus.ServeGRPC(ctx, req)
	if err != nil {
		return nil, grpcapi.EncodeError(err)
	}

	return res.(*grpcCommonV1.RetrieveEntityRes), nil
}

func decodeRetrieveIDByRouteRequest(_ context.Context, grpcReq any) (any, error) {
	req := grpcReq.(*grpcCommonV1.RetrieveIDByRouteReq)

	return retrieveIDByRouteReq{
		Route: req.GetRoute(),
	}, nil
}

func encodeRetrieveIDByRouteResponse(_ context.Context, grpcRes any) (any, error) {
	res := grpcRes.(retrieveIDByRouteRes)

	return &grpcCommonV1.RetrieveEntityRes{
		Entity: &grpcCommonV1.EntityBasic{
			Id: res.id,
		},
	}, nil
}

func (s *domainsGrpcServer) RetrieveIDByRoute(ctx context.Context, req *grpcCommonV1.RetrieveIDByRouteReq) (*grpcCommonV1.RetrieveEntityRes, error) {
	_, res, err := s.retrieveIDByRoute.ServeGRPC(ctx, req)
	if err != nil {
		return nil, grpcapi.EncodeError(err)
	}

	return res.(*grpcCommonV1.RetrieveEntityRes), nil
}
