// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package grpc

import (
	"context"

	grpcUsersV1 "github.com/absmach/magistrala/api/grpc/users/v1"
	grpcapi "github.com/absmach/magistrala/auth/api/grpc"
	"github.com/absmach/magistrala/pkg/errors"
	svcerr "github.com/absmach/magistrala/pkg/errors/service"
	"github.com/absmach/magistrala/users"
	pusers "github.com/absmach/magistrala/users/private"
	kitgrpc "github.com/go-kit/kit/transport/grpc"
	structpb "google.golang.org/protobuf/types/known/structpb"
	timestamppb "google.golang.org/protobuf/types/known/timestamppb"
)

var _ grpcUsersV1.UsersServiceServer = (*usersGrpcServer)(nil)

type usersGrpcServer struct {
	grpcUsersV1.UnimplementedUsersServiceServer
	retrieveUsers kitgrpc.Handler
}

func NewServer(svc pusers.Service) grpcUsersV1.UsersServiceServer {
	return &usersGrpcServer{
		retrieveUsers: kitgrpc.NewServer(
			retrieveUsersEndpoint(svc),
			decodeRetrieveUsersRequest,
			encodeRetrieveUsersResponse,
		),
	}
}

func decodeRetrieveUsersRequest(_ context.Context, grpcReq any) (any, error) {
	req := grpcReq.(*grpcUsersV1.RetrieveUsersReq)
	return retrieveUsersReq{
		ids:    req.GetIds(),
		offset: req.GetOffset(),
		limit:  req.GetLimit(),
	}, nil
}

func encodeRetrieveUsersResponse(_ context.Context, grpcRes any) (any, error) {
	res := grpcRes.(retrieveUsersRes)

	usersPB, err := toProtoUsers(res.users)
	if err != nil {
		return nil, err
	}

	return &grpcUsersV1.RetrieveUsersRes{
		Total:  res.total,
		Limit:  res.limit,
		Offset: res.offset,
		Users:  usersPB,
	}, nil
}

func (s *usersGrpcServer) RetrieveUsers(ctx context.Context, req *grpcUsersV1.RetrieveUsersReq) (*grpcUsersV1.RetrieveUsersRes, error) {
	_, res, err := s.retrieveUsers.ServeGRPC(ctx, req)
	if err != nil {
		return nil, grpcapi.EncodeError(err)
	}

	return res.(*grpcUsersV1.RetrieveUsersRes), nil
}

func toProtoUsers(us []users.User) ([]*grpcUsersV1.User, error) {
	var res []*grpcUsersV1.User
	for _, u := range us {
		pu, err := toProtoUser(u)
		if err != nil {
			return nil, err
		}
		res = append(res, pu)
	}

	return res, nil
}

func toProtoUser(u users.User) (*grpcUsersV1.User, error) {
	var metadata, privateMetadata *structpb.Struct
	var err error
	if u.Metadata != nil {
		metadata, err = structpb.NewStruct(u.Metadata)
		if err != nil {
			return nil, errors.Wrap(svcerr.ErrViewEntity, err)
		}
	}
	if u.PrivateMetadata != nil {
		privateMetadata, err = structpb.NewStruct(u.PrivateMetadata)
		if err != nil {
			return nil, errors.Wrap(svcerr.ErrViewEntity, err)
		}
	}

	pu := &grpcUsersV1.User{
		Id:              u.ID,
		FirstName:       u.FirstName,
		LastName:        u.LastName,
		Tags:            u.Tags,
		Metadata:        metadata,
		PrivateMetadata: privateMetadata,
		Status:          uint32(u.Status),
		Role:            uint32(u.Role),
		ProfilePicture:  u.ProfilePicture,
		Username:        u.Credentials.Username,
		Email:           u.Email,
		UpdatedBy:       u.UpdatedBy,
		AuthProvider:    u.AuthProvider,
		Permissions:     u.Permissions,
	}

	if !u.CreatedAt.IsZero() {
		pu.CreatedAt = timestamppb.New(u.CreatedAt)
	}
	if !u.UpdatedAt.IsZero() {
		pu.UpdatedAt = timestamppb.New(u.UpdatedAt)
	}
	if !u.VerifiedAt.IsZero() {
		pu.VerifiedAt = timestamppb.New(u.VerifiedAt)
	}

	return pu, nil
}
