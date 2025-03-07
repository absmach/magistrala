// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package auth

import (
	"context"
	"time"

	grpcAuthV1 "github.com/absmach/supermq/api/grpc/auth/v1"
	"github.com/absmach/supermq/auth"
	grpcapi "github.com/absmach/supermq/auth/api/grpc"
	"github.com/go-kit/kit/endpoint"
	kitgrpc "github.com/go-kit/kit/transport/grpc"
	"google.golang.org/grpc"
)

const authSvcName = "auth.v1.AuthService"

type authGrpcClient struct {
	authenticate    endpoint.Endpoint
	authenticatePAT endpoint.Endpoint
	authorize       endpoint.Endpoint
	authorizePAT    endpoint.Endpoint
	timeout         time.Duration
}

var _ grpcAuthV1.AuthServiceClient = (*authGrpcClient)(nil)

// NewAuthClient returns new auth gRPC client instance.
func NewAuthClient(conn *grpc.ClientConn, timeout time.Duration) grpcAuthV1.AuthServiceClient {
	return &authGrpcClient{
		authenticate: kitgrpc.NewClient(
			conn,
			authSvcName,
			"Authenticate",
			encodeIdentifyRequest,
			decodeIdentifyResponse,
			grpcAuthV1.AuthNRes{},
		).Endpoint(),
		authenticatePAT: kitgrpc.NewClient(
			conn,
			authSvcName,
			"AuthenticatePAT",
			encodeIdentifyRequest,
			decodeIdentifyPATResponse,
			grpcAuthV1.AuthNRes{},
		).Endpoint(),
		authorize: kitgrpc.NewClient(
			conn,
			authSvcName,
			"Authorize",
			encodeAuthorizeRequest,
			decodeAuthorizeResponse,
			grpcAuthV1.AuthZRes{},
		).Endpoint(),
		authorizePAT: kitgrpc.NewClient(
			conn,
			authSvcName,
			"AuthorizePAT",
			encodeAuthorizePATRequest,
			decodeAuthorizeResponse,
			grpcAuthV1.AuthZRes{},
		).Endpoint(),
		timeout: timeout,
	}
}

func (client authGrpcClient) Authenticate(ctx context.Context, token *grpcAuthV1.AuthNReq, _ ...grpc.CallOption) (*grpcAuthV1.AuthNRes, error) {
	ctx, cancel := context.WithTimeout(ctx, client.timeout)
	defer cancel()

	res, err := client.authenticate(ctx, authenticateReq{token: token.GetToken()})
	if err != nil {
		return &grpcAuthV1.AuthNRes{}, grpcapi.DecodeError(err)
	}
	ir := res.(authenticateRes)
	return &grpcAuthV1.AuthNRes{Id: ir.id, UserId: ir.userID, DomainId: ir.domainID}, nil
}

func encodeIdentifyRequest(_ context.Context, grpcReq interface{}) (interface{}, error) {
	req := grpcReq.(authenticateReq)
	return &grpcAuthV1.AuthNReq{Token: req.token}, nil
}

func decodeIdentifyResponse(_ context.Context, grpcRes interface{}) (interface{}, error) {
	res := grpcRes.(*grpcAuthV1.AuthNRes)
	return authenticateRes{id: res.GetId(), userID: res.GetUserId(), domainID: res.GetDomainId()}, nil
}

func (client authGrpcClient) AuthenticatePAT(ctx context.Context, token *grpcAuthV1.AuthNReq, _ ...grpc.CallOption) (*grpcAuthV1.AuthNRes, error) {
	ctx, cancel := context.WithTimeout(ctx, client.timeout)
	defer cancel()

	res, err := client.authenticatePAT(ctx, authenticateReq{token: token.GetToken()})
	if err != nil {
		return &grpcAuthV1.AuthNRes{}, grpcapi.DecodeError(err)
	}
	ir := res.(authenticateRes)
	return &grpcAuthV1.AuthNRes{Id: ir.id, UserId: ir.userID}, nil
}

func decodeIdentifyPATResponse(_ context.Context, grpcRes interface{}) (interface{}, error) {
	res := grpcRes.(*grpcAuthV1.AuthNRes)
	return authenticateRes{id: res.GetId(), userID: res.GetUserId()}, nil
}

func (client authGrpcClient) Authorize(ctx context.Context, req *grpcAuthV1.AuthZReq, _ ...grpc.CallOption) (r *grpcAuthV1.AuthZRes, err error) {
	ctx, cancel := context.WithTimeout(ctx, client.timeout)
	defer cancel()

	res, err := client.authorize(ctx, authReq{
		Domain:      req.GetDomain(),
		SubjectType: req.GetSubjectType(),
		Subject:     req.GetSubject(),
		SubjectKind: req.GetSubjectKind(),
		Relation:    req.GetRelation(),
		Permission:  req.GetPermission(),
		ObjectType:  req.GetObjectType(),
		Object:      req.GetObject(),
	})
	if err != nil {
		return &grpcAuthV1.AuthZRes{}, grpcapi.DecodeError(err)
	}

	ar := res.(authorizeRes)
	return &grpcAuthV1.AuthZRes{Authorized: ar.authorized, Id: ar.id}, nil
}

func decodeAuthorizeResponse(_ context.Context, grpcRes interface{}) (interface{}, error) {
	res := grpcRes.(*grpcAuthV1.AuthZRes)
	return authorizeRes{authorized: res.Authorized, id: res.Id}, nil
}

func encodeAuthorizeRequest(_ context.Context, grpcReq interface{}) (interface{}, error) {
	req := grpcReq.(authReq)
	return &grpcAuthV1.AuthZReq{
		Domain:      req.Domain,
		SubjectType: req.SubjectType,
		Subject:     req.Subject,
		SubjectKind: req.SubjectKind,
		Relation:    req.Relation,
		Permission:  req.Permission,
		ObjectType:  req.ObjectType,
		Object:      req.Object,
	}, nil
}

func (client authGrpcClient) AuthorizePAT(ctx context.Context, req *grpcAuthV1.AuthZPatReq, _ ...grpc.CallOption) (r *grpcAuthV1.AuthZRes, err error) {
	ctx, cancel := context.WithTimeout(ctx, client.timeout)
	defer cancel()

	res, err := client.authorizePAT(ctx, authPATReq{
		userID:           req.GetUserId(),
		patID:            req.GetPatId(),
		entityType:       auth.EntityType(req.GetEntityType()),
		optionalDomainID: req.GetOptionalDomainId(),
		operation:        auth.Operation(req.GetOperation()),
		entityID:         req.GetEntityId(),
	})
	if err != nil {
		return &grpcAuthV1.AuthZRes{}, grpcapi.DecodeError(err)
	}

	ar := res.(authorizeRes)
	return &grpcAuthV1.AuthZRes{Authorized: ar.authorized, Id: ar.id}, nil
}

func encodeAuthorizePATRequest(_ context.Context, grpcReq interface{}) (interface{}, error) {
	req := grpcReq.(authPATReq)
	return &grpcAuthV1.AuthZPatReq{
		UserId:           req.userID,
		PatId:            req.patID,
		EntityType:       uint32(req.entityType),
		OptionalDomainId: req.optionalDomainID,
		Operation:        uint32(req.operation),
		EntityId:         req.entityID,
	}, nil
}
