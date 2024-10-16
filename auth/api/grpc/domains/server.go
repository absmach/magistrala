// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package domains

import (
	"context"

	"github.com/absmach/magistrala"
	"github.com/absmach/magistrala/auth"
	grpcapi "github.com/absmach/magistrala/auth/api/grpc"
	kitgrpc "github.com/go-kit/kit/transport/grpc"
)

var _ magistrala.DomainsServiceServer = (*domainsGrpcServer)(nil)

type domainsGrpcServer struct {
	magistrala.UnimplementedDomainsServiceServer
	deleteUserFromDomains kitgrpc.Handler
}

func NewDomainsServer(svc auth.Service) magistrala.DomainsServiceServer {
	return &domainsGrpcServer{
		deleteUserFromDomains: kitgrpc.NewServer(
			(deleteUserFromDomainsEndpoint(svc)),
			decodeDeleteUserRequest,
			encodeDeleteUserResponse,
		),
	}
}

func decodeDeleteUserRequest(_ context.Context, grpcReq interface{}) (interface{}, error) {
	req := grpcReq.(*magistrala.DeleteUserReq)
	return deleteUserPoliciesReq{
		ID: req.GetId(),
	}, nil
}

func encodeDeleteUserResponse(_ context.Context, grpcRes interface{}) (interface{}, error) {
	res := grpcRes.(deleteUserRes)
	return &magistrala.DeleteUserRes{Deleted: res.deleted}, nil
}

func (s *domainsGrpcServer) DeleteUserFromDomains(ctx context.Context, req *magistrala.DeleteUserReq) (*magistrala.DeleteUserRes, error) {
	_, res, err := s.deleteUserFromDomains.ServeGRPC(ctx, req)
	if err != nil {
		return nil, grpcapi.EncodeError(err)
	}
	return res.(*magistrala.DeleteUserRes), nil
}
