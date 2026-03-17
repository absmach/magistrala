// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package grpc

import (
	"context"

	"github.com/absmach/supermq/certs"
	"github.com/absmach/supermq/certs/api/http"
	"github.com/absmach/supermq/pkg/errors"
	kitgrpc "github.com/go-kit/kit/transport/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

var _ certs.CertsServiceServer = (*grpcServer)(nil)

type grpcServer struct {
	getEntity   kitgrpc.Handler
	revokeCerts kitgrpc.Handler
	certs.UnimplementedCertsServiceServer
}

func NewServer(svc certs.Service) certs.CertsServiceServer {
	return &grpcServer{
		getEntity: kitgrpc.NewServer(
			(getEntityEndpoint(svc)),
			decodeGetEntityReq,
			encodeGetEntityRes,
		),
		revokeCerts: kitgrpc.NewServer(
			(revokeCertsEndpoint(svc)),
			decodeRevokeCertsReq,
			encodeRevokeCertsRes,
		),
	}
}

func decodeGetEntityReq(_ context.Context, req any) (any, error) {
	return req.(*certs.EntityReq), nil
}

func encodeGetEntityRes(_ context.Context, res any) (any, error) {
	return res.(*certs.EntityRes), nil
}

func decodeRevokeCertsReq(_ context.Context, req any) (any, error) {
	return req.(*certs.RevokeReq), nil
}

func encodeRevokeCertsRes(_ context.Context, res any) (any, error) {
	return res.(*emptypb.Empty), nil
}

// GetEntityID returns the entity ID for the given entity request.
func (g *grpcServer) GetEntityID(ctx context.Context, req *certs.EntityReq) (*certs.EntityRes, error) {
	_, res, err := g.getEntity.ServeGRPC(ctx, req)
	if err != nil {
		return &certs.EntityRes{}, encodeError(err)
	}
	return res.(*certs.EntityRes), nil
}

func (g *grpcServer) RevokeCerts(ctx context.Context, req *certs.RevokeReq) (*emptypb.Empty, error) {
	_, res, err := g.revokeCerts.ServeGRPC(ctx, req)
	if err != nil {
		return &emptypb.Empty{}, encodeError(err)
	}
	return res.(*emptypb.Empty), nil
}

func encodeError(err error) error {
	switch {
	case errors.Contains(err, nil):
		return nil
	case errors.Contains(err, certs.ErrMalformedEntity),
		errors.Contains(err, http.ErrMissingEntityID):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Contains(err, certs.ErrNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Contains(err, certs.ErrConflict):
		return status.Error(codes.AlreadyExists, err.Error())
	case errors.Contains(err, certs.ErrCreateEntity),
		errors.Contains(err, certs.ErrUpdateEntity),
		errors.Contains(err, certs.ErrViewEntity):
		return status.Error(codes.Internal, err.Error())
	default:
		return status.Error(codes.Internal, "internal server error")
	}
}
