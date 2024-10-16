// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package domains

import (
	"context"
	"time"

	"github.com/absmach/magistrala"
	grpcapi "github.com/absmach/magistrala/auth/api/grpc"
	"github.com/go-kit/kit/endpoint"
	kitgrpc "github.com/go-kit/kit/transport/grpc"
	"google.golang.org/grpc"
)

const domainsSvcName = "magistrala.DomainsService"

var _ magistrala.DomainsServiceClient = (*domainsGrpcClient)(nil)

type domainsGrpcClient struct {
	deleteUserFromDomains endpoint.Endpoint
	timeout               time.Duration
}

// NewDomainsClient returns new domains gRPC client instance.
func NewDomainsClient(conn *grpc.ClientConn, timeout time.Duration) magistrala.DomainsServiceClient {
	return &domainsGrpcClient{
		deleteUserFromDomains: kitgrpc.NewClient(
			conn,
			domainsSvcName,
			"DeleteUserFromDomains",
			encodeDeleteUserRequest,
			decodeDeleteUserResponse,
			magistrala.DeleteUserRes{},
		).Endpoint(),

		timeout: timeout,
	}
}

func (client domainsGrpcClient) DeleteUserFromDomains(ctx context.Context, in *magistrala.DeleteUserReq, opts ...grpc.CallOption) (*magistrala.DeleteUserRes, error) {
	ctx, cancel := context.WithTimeout(ctx, client.timeout)
	defer cancel()

	res, err := client.deleteUserFromDomains(ctx, deleteUserPoliciesReq{
		ID: in.GetId(),
	})
	if err != nil {
		return &magistrala.DeleteUserRes{}, grpcapi.DecodeError(err)
	}

	dpr := res.(deleteUserRes)
	return &magistrala.DeleteUserRes{Deleted: dpr.deleted}, nil
}

func decodeDeleteUserResponse(_ context.Context, grpcRes interface{}) (interface{}, error) {
	res := grpcRes.(*magistrala.DeleteUserRes)
	return deleteUserRes{deleted: res.GetDeleted()}, nil
}

func encodeDeleteUserRequest(_ context.Context, grpcReq interface{}) (interface{}, error) {
	req := grpcReq.(deleteUserPoliciesReq)
	return &magistrala.DeleteUserReq{
		Id: req.ID,
	}, nil
}
