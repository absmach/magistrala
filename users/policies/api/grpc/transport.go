// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package grpc

import (
	"context"

	kitgrpc "github.com/go-kit/kit/transport/grpc"
	"github.com/mainflux/mainflux/internal/apiutil"
	"github.com/mainflux/mainflux/pkg/errors"
	"github.com/mainflux/mainflux/users/clients"
	"github.com/mainflux/mainflux/users/policies"
	"go.opentelemetry.io/contrib/instrumentation/github.com/go-kit/kit/otelkit"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var _ policies.AuthServiceServer = (*grpcServer)(nil)

type grpcServer struct {
	authorize    kitgrpc.Handler
	issue        kitgrpc.Handler
	identify     kitgrpc.Handler
	addPolicy    kitgrpc.Handler
	deletePolicy kitgrpc.Handler
	listPolicies kitgrpc.Handler
	policies.UnimplementedAuthServiceServer
}

// NewServer returns new AuthServiceServer instance.
func NewServer(csvc clients.Service, psvc policies.Service) policies.AuthServiceServer {
	return &grpcServer{
		authorize: kitgrpc.NewServer(
			otelkit.EndpointMiddleware(otelkit.WithOperation("authorize"))(authorizeEndpoint(psvc)),
			decodeAuthorizeRequest,
			encodeAuthorizeResponse,
		),
		issue: kitgrpc.NewServer(
			otelkit.EndpointMiddleware(otelkit.WithOperation("issue"))(issueEndpoint(csvc)),
			decodeIssueRequest,
			encodeIssueResponse,
		),
		identify: kitgrpc.NewServer(
			otelkit.EndpointMiddleware(otelkit.WithOperation("identify"))(identifyEndpoint(csvc)),
			decodeIdentifyRequest,
			encodeIdentifyResponse,
		),
		addPolicy: kitgrpc.NewServer(
			otelkit.EndpointMiddleware(otelkit.WithOperation("add_policy"))(addPolicyEndpoint(psvc)),
			decodeAddPolicyRequest,
			encodeAddPolicyResponse,
		),
		deletePolicy: kitgrpc.NewServer(
			otelkit.EndpointMiddleware(otelkit.WithOperation("delete_policy"))(deletePolicyEndpoint(psvc)),
			decodeDeletePolicyRequest,
			encodeDeletePolicyResponse,
		),
		listPolicies: kitgrpc.NewServer(
			otelkit.EndpointMiddleware(otelkit.WithOperation("list_policies"))(listPoliciesEndpoint(psvc)),
			decodeListPoliciesRequest,
			encodeListPoliciesResponse,
		),
	}
}

func (s *grpcServer) Authorize(ctx context.Context, req *policies.AuthorizeReq) (*policies.AuthorizeRes, error) {
	_, res, err := s.authorize.ServeGRPC(ctx, req)
	if err != nil {
		return nil, encodeError(err)
	}
	return res.(*policies.AuthorizeRes), nil
}

func (s *grpcServer) Issue(ctx context.Context, req *policies.IssueReq) (*policies.Token, error) {
	_, res, err := s.issue.ServeGRPC(ctx, req)
	if err != nil {
		return nil, encodeError(err)
	}
	return res.(*policies.Token), nil
}

func (s *grpcServer) Identify(ctx context.Context, token *policies.Token) (*policies.UserIdentity, error) {
	_, res, err := s.identify.ServeGRPC(ctx, token)
	if err != nil {
		return nil, encodeError(err)
	}
	return res.(*policies.UserIdentity), nil
}

func (s *grpcServer) AddPolicy(ctx context.Context, req *policies.AddPolicyReq) (*policies.AddPolicyRes, error) {
	_, res, err := s.addPolicy.ServeGRPC(ctx, req)
	if err != nil {
		return nil, encodeError(err)
	}
	return res.(*policies.AddPolicyRes), nil
}

func (s *grpcServer) DeletePolicy(ctx context.Context, req *policies.DeletePolicyReq) (*policies.DeletePolicyRes, error) {
	_, res, err := s.deletePolicy.ServeGRPC(ctx, req)
	if err != nil {
		return nil, encodeError(err)
	}
	return res.(*policies.DeletePolicyRes), nil
}

func (s *grpcServer) ListPolicies(ctx context.Context, req *policies.ListPoliciesReq) (*policies.ListPoliciesRes, error) {
	_, res, err := s.listPolicies.ServeGRPC(ctx, req)
	if err != nil {
		return nil, encodeError(err)
	}
	return res.(*policies.ListPoliciesRes), nil
}

func decodeAuthorizeRequest(_ context.Context, grpcReq interface{}) (interface{}, error) {
	req := grpcReq.(*policies.AuthorizeReq)
	return authReq{Act: req.GetAct(), Obj: req.GetObj(), Sub: req.GetSub(), EntityType: req.GetEntityType()}, nil
}

func encodeAuthorizeResponse(_ context.Context, grpcRes interface{}) (interface{}, error) {
	res := grpcRes.(authorizeRes)
	return &policies.AuthorizeRes{Authorized: res.authorized}, nil
}

func decodeIssueRequest(_ context.Context, grpcReq interface{}) (interface{}, error) {
	req := grpcReq.(*policies.IssueReq)
	return issueReq{email: req.GetEmail(), password: req.GetPassword()}, nil
}

func encodeIssueResponse(_ context.Context, grpcRes interface{}) (interface{}, error) {
	res := grpcRes.(issueRes)
	return &policies.Token{Value: res.value}, nil
}

func decodeIdentifyRequest(_ context.Context, grpcReq interface{}) (interface{}, error) {
	req := grpcReq.(*policies.Token)
	return identityReq{token: req.GetValue()}, nil
}

func encodeIdentifyResponse(_ context.Context, grpcRes interface{}) (interface{}, error) {
	res := grpcRes.(identityRes)
	return &policies.UserIdentity{Id: res.id}, nil
}

func decodeAddPolicyRequest(_ context.Context, grpcReq interface{}) (interface{}, error) {
	req := grpcReq.(*policies.AddPolicyReq)
	return addPolicyReq{Token: req.GetToken(), Sub: req.GetSub(), Obj: req.GetObj(), Act: req.GetAct()}, nil
}

func encodeAddPolicyResponse(_ context.Context, grpcRes interface{}) (interface{}, error) {
	res := grpcRes.(addPolicyRes)
	return &policies.AddPolicyRes{Authorized: res.authorized}, nil
}

func decodeDeletePolicyRequest(_ context.Context, grpcReq interface{}) (interface{}, error) {
	req := grpcReq.(*policies.DeletePolicyReq)
	return policyReq{Token: req.GetToken(), Sub: req.GetSub(), Obj: req.GetObj(), Act: req.GetAct()}, nil
}

func encodeDeletePolicyResponse(_ context.Context, grpcRes interface{}) (interface{}, error) {
	res := grpcRes.(deletePolicyRes)
	return &policies.DeletePolicyRes{Deleted: res.deleted}, nil
}

func decodeListPoliciesRequest(_ context.Context, grpcReq interface{}) (interface{}, error) {
	req := grpcReq.(*policies.ListPoliciesReq)
	return listPoliciesReq{Token: req.GetToken(), Sub: req.GetSub(), Obj: req.GetObj(), Act: req.GetAct()}, nil
}

func encodeListPoliciesResponse(_ context.Context, grpcRes interface{}) (interface{}, error) {
	res := grpcRes.(listPoliciesRes)
	return &policies.ListPoliciesRes{Objects: res.objects}, nil
}

func encodeError(err error) error {
	switch {
	case errors.Contains(err, nil):
		return nil
	case errors.Contains(err, errors.ErrMalformedEntity),
		err == apiutil.ErrInvalidAuthKey,
		err == apiutil.ErrMissingID,
		err == apiutil.ErrMissingPolicySub,
		err == apiutil.ErrMissingPolicyObj,
		err == apiutil.ErrMalformedPolicyAct,
		err == apiutil.ErrMalformedPolicy,
		err == apiutil.ErrMissingPolicyOwner:
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Contains(err, errors.ErrAuthentication),
		err == apiutil.ErrBearerToken:
		return status.Error(codes.Unauthenticated, err.Error())
	case errors.Contains(err, errors.ErrAuthorization):
		return status.Error(codes.PermissionDenied, err.Error())
	default:
		return status.Error(codes.Internal, "internal server error")
	}
}
