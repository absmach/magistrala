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
	authenticate endpoint.Endpoint
	authorize    endpoint.Endpoint
	timeout      time.Duration
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
		authorize: kitgrpc.NewClient(
			conn,
			authSvcName,
			"Authorize",
			encodeAuthorizeRequest,
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
	return &grpcAuthV1.AuthNRes{Id: ir.id, UserId: ir.userID, UserRole: uint32(ir.userRole), Verified: ir.verified}, nil
}

func encodeIdentifyRequest(_ context.Context, grpcReq any) (any, error) {
	req := grpcReq.(authenticateReq)
	return &grpcAuthV1.AuthNReq{Token: req.token}, nil
}

func decodeIdentifyResponse(_ context.Context, grpcRes any) (any, error) {
	res := grpcRes.(*grpcAuthV1.AuthNRes)
	return authenticateRes{id: res.GetId(), userID: res.GetUserId(), userRole: auth.Role(res.UserRole), verified: res.GetVerified()}, nil
}

func (client authGrpcClient) Authorize(ctx context.Context, req *grpcAuthV1.AuthZReq, _ ...grpc.CallOption) (r *grpcAuthV1.AuthZRes, err error) {
	ctx, cancel := context.WithTimeout(ctx, client.timeout)
	defer cancel()

	var authReqData authReq

	if req != nil {
		policyReq := req.GetPolicyReq()
		patReq := req.GetPatReq()

		if policyReq != nil {
			authReqData = authReq{
				Domain:      policyReq.GetDomain(),
				SubjectType: policyReq.GetSubjectType(),
				Subject:     policyReq.GetSubject(),
				SubjectKind: policyReq.GetSubjectKind(),
				Relation:    policyReq.GetRelation(),
				Permission:  policyReq.GetPermission(),
				ObjectType:  policyReq.GetObjectType(),
				Object:      policyReq.GetObject(),
			}
		}

		if patReq != nil {
			authReqData.UserID = patReq.GetUserId()
			authReqData.PatID = patReq.GetPatId()
			authReqData.EntityType = patReq.GetEntityType()
			authReqData.Operation = patReq.GetOperation()
			authReqData.EntityID = patReq.GetEntityId()
		}
	}

	res, err := client.authorize(ctx, authReqData)
	if err != nil {
		return &grpcAuthV1.AuthZRes{}, grpcapi.DecodeError(err)
	}

	ar := res.(authorizeRes)
	return &grpcAuthV1.AuthZRes{Authorized: ar.authorized, Id: ar.id}, nil
}

func decodeAuthorizeResponse(_ context.Context, grpcRes any) (any, error) {
	res := grpcRes.(*grpcAuthV1.AuthZRes)
	return authorizeRes{authorized: res.Authorized, id: res.Id}, nil
}

func encodeAuthorizeRequest(_ context.Context, grpcReq any) (any, error) {
	req := grpcReq.(authReq)

	authZReq := &grpcAuthV1.AuthZReq{
		PolicyReq: &grpcAuthV1.PolicyReq{
			Domain:      req.Domain,
			SubjectType: req.SubjectType,
			Subject:     req.Subject,
			SubjectKind: req.SubjectKind,
			Relation:    req.Relation,
			Permission:  req.Permission,
			ObjectType:  req.ObjectType,
			Object:      req.Object,
		},
	}

	if req.PatID != "" {
		authZReq.PatReq = &grpcAuthV1.PATReq{
			PatId:      req.PatID,
			Domain:     req.Domain,
			Operation:  req.Operation,
			UserId:     req.UserID,
			EntityId:   req.EntityID,
			EntityType: req.EntityType,
		}
	}

	return authZReq, nil
}
