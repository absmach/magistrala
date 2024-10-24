// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package grpc

import (
	"context"

	grpcapi "github.com/absmach/magistrala/auth/api/grpc"
	"github.com/absmach/magistrala/domains"
	grpcDomainsV1 "github.com/absmach/magistrala/internal/grpc/domains/v1"
	kitgrpc "github.com/go-kit/kit/transport/grpc"
)

var _ grpcDomainsV1.DomainsServiceServer = (*domainsGrpcServer)(nil)

type domainsGrpcServer struct {
	grpcDomainsV1.UnimplementedDomainsServiceServer
	deleteUserFromDomains kitgrpc.Handler
}

func NewDomainsServer(svc domains.Service) grpcDomainsV1.DomainsServiceServer {
	return &domainsGrpcServer{
		deleteUserFromDomains: kitgrpc.NewServer(
			(deleteUserFromDomainsEndpoint(svc)),
			decodeDeleteUserRequest,
			encodeDeleteUserResponse,
		),
	}
}

func decodeDeleteUserRequest(_ context.Context, grpcReq interface{}) (interface{}, error) {
	req := grpcReq.(*grpcDomainsV1.DeleteUserReq)
	return deleteUserPoliciesReq{
		ID: req.GetId(),
	}, nil
}

func encodeDeleteUserResponse(_ context.Context, grpcRes interface{}) (interface{}, error) {
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
