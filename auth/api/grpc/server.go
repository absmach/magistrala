// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package grpc

import (
	"context"

	"github.com/absmach/magistrala"
	"github.com/absmach/magistrala/auth"
	"github.com/absmach/magistrala/pkg/apiutil"
	"github.com/absmach/magistrala/pkg/errors"
	svcerr "github.com/absmach/magistrala/pkg/errors/service"
	kitgrpc "github.com/go-kit/kit/transport/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var (
	_ magistrala.AuthzServiceServer  = (*authzGrpcServer)(nil)
	_ magistrala.AuthnServiceServer  = (*authnGrpcServer)(nil)
	_ magistrala.PolicyServiceServer = (*policyGrpcServer)(nil)
)

type authzGrpcServer struct {
	magistrala.UnimplementedAuthzServiceServer
	authorize kitgrpc.Handler
}

// NewAuthzServer returns new AuthzServiceServer instance.
func NewAuthzServer(svc auth.Service) magistrala.AuthzServiceServer {
	return &authzGrpcServer{
		authorize: kitgrpc.NewServer(
			(authorizeEndpoint(svc)),
			decodeAuthorizeRequest,
			encodeAuthorizeResponse,
		),
	}
}

func (s *authzGrpcServer) Authorize(ctx context.Context, req *magistrala.AuthorizeReq) (*magistrala.AuthorizeRes, error) {
	_, res, err := s.authorize.ServeGRPC(ctx, req)
	if err != nil {
		return nil, encodeError(err)
	}
	return res.(*magistrala.AuthorizeRes), nil
}

type authnGrpcServer struct {
	magistrala.UnimplementedAuthnServiceServer
	issue    kitgrpc.Handler
	refresh  kitgrpc.Handler
	identify kitgrpc.Handler
}

// NewAuthnServer returns new AuthnServiceServer instance.
func NewAuthnServer(svc auth.Service) magistrala.AuthnServiceServer {
	return &authnGrpcServer{
		issue: kitgrpc.NewServer(
			(issueEndpoint(svc)),
			decodeIssueRequest,
			encodeIssueResponse,
		),
		refresh: kitgrpc.NewServer(
			(refreshEndpoint(svc)),
			decodeRefreshRequest,
			encodeIssueResponse,
		),
		identify: kitgrpc.NewServer(
			(identifyEndpoint(svc)),
			decodeIdentifyRequest,
			encodeIdentifyResponse,
		),
	}
}

func (s *authnGrpcServer) Issue(ctx context.Context, req *magistrala.IssueReq) (*magistrala.Token, error) {
	_, res, err := s.issue.ServeGRPC(ctx, req)
	if err != nil {
		return nil, encodeError(err)
	}
	return res.(*magistrala.Token), nil
}

func (s *authnGrpcServer) Refresh(ctx context.Context, req *magistrala.RefreshReq) (*magistrala.Token, error) {
	_, res, err := s.refresh.ServeGRPC(ctx, req)
	if err != nil {
		return nil, encodeError(err)
	}
	return res.(*magistrala.Token), nil
}

func (s *authnGrpcServer) Identify(ctx context.Context, token *magistrala.IdentityReq) (*magistrala.IdentityRes, error) {
	_, res, err := s.identify.ServeGRPC(ctx, token)
	if err != nil {
		return nil, encodeError(err)
	}
	return res.(*magistrala.IdentityRes), nil
}

type policyGrpcServer struct {
	magistrala.UnimplementedPolicyServiceServer
	deleteUserPolicies kitgrpc.Handler
}

func NewPolicyServer(svc auth.Service) magistrala.PolicyServiceServer {
	return &policyGrpcServer{
		deleteUserPolicies: kitgrpc.NewServer(
			(deleteUserPoliciesEndpoint(svc)),
			decodeDeleteUserPoliciesRequest,
			encodeDeleteUserPoliciesResponse,
		),
	}
}

func (s *policyGrpcServer) DeleteUserPolicies(ctx context.Context, req *magistrala.DeleteUserPoliciesReq) (*magistrala.DeletePolicyRes, error) {
	_, res, err := s.deleteUserPolicies.ServeGRPC(ctx, req)
	if err != nil {
		return nil, encodeError(err)
	}
	return res.(*magistrala.DeletePolicyRes), nil
}

func decodeIssueRequest(_ context.Context, grpcReq interface{}) (interface{}, error) {
	req := grpcReq.(*magistrala.IssueReq)
	return issueReq{
		userID:   req.GetUserId(),
		domainID: req.GetDomainId(),
		keyType:  auth.KeyType(req.GetType()),
	}, nil
}

func decodeRefreshRequest(_ context.Context, grpcReq interface{}) (interface{}, error) {
	req := grpcReq.(*magistrala.RefreshReq)
	return refreshReq{refreshToken: req.GetRefreshToken(), domainID: req.GetDomainId()}, nil
}

func encodeIssueResponse(_ context.Context, grpcRes interface{}) (interface{}, error) {
	res := grpcRes.(issueRes)

	return &magistrala.Token{
		AccessToken:  res.accessToken,
		RefreshToken: &res.refreshToken,
		AccessType:   res.accessType,
	}, nil
}

func decodeIdentifyRequest(_ context.Context, grpcReq interface{}) (interface{}, error) {
	req := grpcReq.(*magistrala.IdentityReq)
	return identityReq{token: req.GetToken()}, nil
}

func encodeIdentifyResponse(_ context.Context, grpcRes interface{}) (interface{}, error) {
	res := grpcRes.(identityRes)
	return &magistrala.IdentityRes{Id: res.id, UserId: res.userID, DomainId: res.domainID}, nil
}

func decodeAuthorizeRequest(_ context.Context, grpcReq interface{}) (interface{}, error) {
	req := grpcReq.(*magistrala.AuthorizeReq)
	return authReq{
		Domain:      req.GetDomain(),
		SubjectType: req.GetSubjectType(),
		SubjectKind: req.GetSubjectKind(),
		Subject:     req.GetSubject(),
		Relation:    req.GetRelation(),
		Permission:  req.GetPermission(),
		ObjectType:  req.GetObjectType(),
		Object:      req.GetObject(),
	}, nil
}

func encodeAuthorizeResponse(_ context.Context, grpcRes interface{}) (interface{}, error) {
	res := grpcRes.(authorizeRes)
	return &magistrala.AuthorizeRes{Authorized: res.authorized, Id: res.id}, nil
}

func decodeDeleteUserPoliciesRequest(_ context.Context, grpcReq interface{}) (interface{}, error) {
	req := grpcReq.(*magistrala.DeleteUserPoliciesReq)
	return deleteUserPoliciesReq{
		ID: req.GetId(),
	}, nil
}

func encodeDeleteUserPoliciesResponse(_ context.Context, grpcRes interface{}) (interface{}, error) {
	res := grpcRes.(deletePolicyRes)
	return &magistrala.DeletePolicyRes{Deleted: res.deleted}, nil
}

func encodeError(err error) error {
	switch {
	case errors.Contains(err, nil):
		return nil
	case errors.Contains(err, errors.ErrMalformedEntity),
		errors.Contains(err, svcerr.ErrInvalidPolicy),
		err == apiutil.ErrInvalidAuthKey,
		err == apiutil.ErrMissingID,
		err == apiutil.ErrMissingMemberType,
		err == apiutil.ErrMissingPolicySub,
		err == apiutil.ErrMissingPolicyObj,
		err == apiutil.ErrMalformedPolicyAct:
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Contains(err, svcerr.ErrAuthentication),
		errors.Contains(err, auth.ErrKeyExpired),
		err == apiutil.ErrMissingEmail,
		err == apiutil.ErrBearerToken:
		return status.Error(codes.Unauthenticated, err.Error())
	case errors.Contains(err, svcerr.ErrAuthorization),
		errors.Contains(err, svcerr.ErrDomainAuthorization):
		return status.Error(codes.PermissionDenied, err.Error())
	case errors.Contains(err, svcerr.ErrNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Contains(err, svcerr.ErrConflict):
		return status.Error(codes.AlreadyExists, err.Error())
	default:
		return status.Error(codes.Internal, err.Error())
	}
}
