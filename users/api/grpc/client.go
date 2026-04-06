// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package grpc

import (
	"context"
	"time"

	grpcUsersV1 "github.com/absmach/magistrala/api/grpc/users/v1"
	grpcapi "github.com/absmach/magistrala/auth/api/grpc"
	"github.com/absmach/magistrala/users"
	"github.com/go-kit/kit/endpoint"
	kitgrpc "github.com/go-kit/kit/transport/grpc"
	"google.golang.org/grpc"
)

const usersSvcName = "users.v1.UsersService"

var _ grpcUsersV1.UsersServiceClient = (*usersGrpcClient)(nil)

type usersGrpcClient struct {
	retrieveUsers endpoint.Endpoint
	timeout       time.Duration
}

// NewClient returns new users gRPC client instance.
func NewClient(conn *grpc.ClientConn, timeout time.Duration) grpcUsersV1.UsersServiceClient {
	return &usersGrpcClient{
		retrieveUsers: kitgrpc.NewClient(
			conn,
			usersSvcName,
			"RetrieveUsers",
			encodeRetrieveUsersRequest,
			decodeRetrieveUsersResponse,
			grpcUsersV1.RetrieveUsersRes{},
		).Endpoint(),
		timeout: timeout,
	}
}

func (client usersGrpcClient) RetrieveUsers(ctx context.Context, in *grpcUsersV1.RetrieveUsersReq, opts ...grpc.CallOption) (*grpcUsersV1.RetrieveUsersRes, error) {
	ctx, cancel := context.WithTimeout(ctx, client.timeout)
	defer cancel()

	res, err := client.retrieveUsers(ctx, retrieveUsersReq{
		ids:    in.GetIds(),
		offset: in.GetOffset(),
		limit:  in.GetLimit(),
	})
	if err != nil {
		return &grpcUsersV1.RetrieveUsersRes{}, grpcapi.DecodeError(err)
	}

	rur := res.(retrieveUsersRes)

	usersPB, err := toProtoUsers(rur.users)
	if err != nil {
		return &grpcUsersV1.RetrieveUsersRes{}, err
	}

	return &grpcUsersV1.RetrieveUsersRes{
		Total:  rur.total,
		Limit:  rur.limit,
		Offset: rur.offset,
		Users:  usersPB,
	}, nil
}

func decodeRetrieveUsersResponse(_ context.Context, grpcRes any) (any, error) {
	res := grpcRes.(*grpcUsersV1.RetrieveUsersRes)

	usersDomain, err := usersFromProto(res.GetUsers())
	if err != nil {
		return nil, err
	}

	return retrieveUsersRes{
		users:  usersDomain,
		total:  res.GetTotal(),
		limit:  res.GetLimit(),
		offset: res.GetOffset(),
	}, nil
}

func encodeRetrieveUsersRequest(_ context.Context, grpcReq any) (any, error) {
	req := grpcReq.(retrieveUsersReq)
	return &grpcUsersV1.RetrieveUsersReq{
		Ids:    req.ids,
		Offset: req.offset,
		Limit:  req.limit,
	}, nil
}

func usersFromProto(us []*grpcUsersV1.User) ([]users.User, error) {
	var res []users.User
	for _, u := range us {
		du, err := userFromProto(u)
		if err != nil {
			return nil, err
		}
		res = append(res, du)
	}

	return res, nil
}

func userFromProto(u *grpcUsersV1.User) (users.User, error) {
	metadata := users.Metadata(nil)
	if u.GetMetadata() != nil {
		metadata = users.Metadata(u.GetMetadata().AsMap())
	}
	privateMetadata := users.Metadata(nil)
	if u.GetPrivateMetadata() != nil {
		privateMetadata = users.Metadata(u.GetPrivateMetadata().AsMap())
	}

	user := users.User{
		ID:              u.GetId(),
		FirstName:       u.GetFirstName(),
		LastName:        u.GetLastName(),
		Tags:            u.GetTags(),
		Metadata:        metadata,
		PrivateMetadata: privateMetadata,
		Status:          users.Status(u.GetStatus()),
		Role:            users.Role(u.GetRole()),
		ProfilePicture:  u.GetProfilePicture(),
		Credentials: users.Credentials{
			Username: u.GetUsername(),
		},
		Email:        u.GetEmail(),
		UpdatedBy:    u.GetUpdatedBy(),
		Permissions:  u.GetPermissions(),
		AuthProvider: u.GetAuthProvider(),
	}

	if u.GetCreatedAt() != nil {
		user.CreatedAt = u.GetCreatedAt().AsTime()
	}
	if u.GetUpdatedAt() != nil {
		user.UpdatedAt = u.GetUpdatedAt().AsTime()
	}
	if u.GetVerifiedAt() != nil {
		user.VerifiedAt = u.GetVerifiedAt().AsTime()
	}

	return user, nil
}
