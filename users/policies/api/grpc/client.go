// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package grpc

import (
	"context"
	"time"

	"github.com/go-kit/kit/endpoint"
	kitgrpc "github.com/go-kit/kit/transport/grpc"
	"github.com/mainflux/mainflux/users/policies"
	"go.opentelemetry.io/contrib/instrumentation/github.com/go-kit/kit/otelkit"
	"google.golang.org/grpc"
)

const svcName = "mainflux.users.policies.AuthService"

var _ policies.AuthServiceClient = (*grpcClient)(nil)

type grpcClient struct {
	authorize    endpoint.Endpoint
	issue        endpoint.Endpoint
	identify     endpoint.Endpoint
	addPolicy    endpoint.Endpoint
	deletePolicy endpoint.Endpoint
	listPolicies endpoint.Endpoint
	timeout      time.Duration
}

// NewClient returns new gRPC client instance.
func NewClient(conn *grpc.ClientConn, timeout time.Duration) policies.AuthServiceClient {
	return &grpcClient{
		authorize: otelkit.EndpointMiddleware(otelkit.WithOperation("authorize"))(kitgrpc.NewClient(
			conn,
			svcName,
			"Authorize",
			encodeAuthorizeRequest,
			decodeAuthorizeResponse,
			policies.AuthorizeRes{},
		).Endpoint()),
		issue: otelkit.EndpointMiddleware(otelkit.WithOperation("issue"))(kitgrpc.NewClient(
			conn,
			svcName,
			"Issue",
			encodeIssueRequest,
			decodeIssueResponse,
			policies.UserIdentity{},
		).Endpoint()),
		identify: otelkit.EndpointMiddleware(otelkit.WithOperation("identify"))(kitgrpc.NewClient(
			conn,
			svcName,
			"Identify",
			encodeIdentifyRequest,
			decodeIdentifyResponse,
			policies.UserIdentity{},
		).Endpoint()),
		addPolicy: otelkit.EndpointMiddleware(otelkit.WithOperation("add_policy"))(kitgrpc.NewClient(
			conn,
			svcName,
			"AddPolicy",
			encodeAddPolicyRequest,
			decodeAddPolicyResponse,
			policies.AddPolicyRes{},
		).Endpoint()),
		deletePolicy: otelkit.EndpointMiddleware(otelkit.WithOperation("delete_policy"))(kitgrpc.NewClient(
			conn,
			svcName,
			"DeletePolicy",
			encodeDeletePolicyRequest,
			decodeDeletePolicyResponse,
			policies.DeletePolicyRes{},
		).Endpoint()),
		listPolicies: otelkit.EndpointMiddleware(otelkit.WithOperation("list_policies"))(kitgrpc.NewClient(
			conn,
			svcName,
			"ListPolicies",
			encodeListPoliciesRequest,
			decodeListPoliciesResponse,
			policies.ListPoliciesRes{},
		).Endpoint()),

		timeout: timeout,
	}
}

func (client grpcClient) Authorize(ctx context.Context, req *policies.AuthorizeReq, _ ...grpc.CallOption) (r *policies.AuthorizeRes, err error) {
	ctx, close := context.WithTimeout(ctx, client.timeout)
	defer close()
	areq := authReq{Act: req.GetAct(), Obj: req.GetObj(), Sub: req.GetSub(), EntityType: req.GetEntityType()}
	res, err := client.authorize(ctx, areq)
	if err != nil {
		return &policies.AuthorizeRes{}, err
	}

	ar := res.(authorizeRes)
	return &policies.AuthorizeRes{Authorized: ar.authorized}, err
}

func decodeAuthorizeResponse(_ context.Context, grpcRes interface{}) (interface{}, error) {
	res := grpcRes.(*policies.AuthorizeRes)
	return authorizeRes{authorized: res.Authorized}, nil
}

func encodeAuthorizeRequest(_ context.Context, grpcReq interface{}) (interface{}, error) {
	req := grpcReq.(authReq)
	return &policies.AuthorizeReq{
		Sub:        req.Sub,
		Obj:        req.Obj,
		Act:        req.Act,
		EntityType: req.EntityType,
	}, nil
}

func (client grpcClient) Issue(ctx context.Context, req *policies.IssueReq, _ ...grpc.CallOption) (*policies.Token, error) {
	ctx, close := context.WithTimeout(ctx, client.timeout)
	defer close()
	ireq := issueReq{email: req.GetEmail(), password: req.GetPassword()}
	res, err := client.issue(ctx, ireq)
	if err != nil {
		return nil, err
	}

	ir := res.(identityRes)
	return &policies.Token{Value: ir.id}, nil
}

func encodeIssueRequest(_ context.Context, grpcReq interface{}) (interface{}, error) {
	req := grpcReq.(issueReq)
	return &policies.IssueReq{Email: req.email, Password: req.password}, nil
}

func decodeIssueResponse(_ context.Context, grpcRes interface{}) (interface{}, error) {
	res := grpcRes.(*policies.UserIdentity)
	return identityRes{id: res.GetId()}, nil
}

func (client grpcClient) Identify(ctx context.Context, token *policies.Token, _ ...grpc.CallOption) (*policies.UserIdentity, error) {
	ctx, close := context.WithTimeout(ctx, client.timeout)
	defer close()

	res, err := client.identify(ctx, identityReq{token: token.GetValue()})
	if err != nil {
		return nil, err
	}

	ir := res.(identityRes)
	return &policies.UserIdentity{Id: ir.id}, nil
}

func encodeIdentifyRequest(_ context.Context, grpcReq interface{}) (interface{}, error) {
	req := grpcReq.(identityReq)
	return &policies.Token{Value: req.token}, nil
}

func decodeIdentifyResponse(_ context.Context, grpcRes interface{}) (interface{}, error) {
	res := grpcRes.(*policies.UserIdentity)
	return identityRes{id: res.GetId()}, nil
}

func (client grpcClient) AddPolicy(ctx context.Context, in *policies.AddPolicyReq, opts ...grpc.CallOption) (*policies.AddPolicyRes, error) {
	ctx, close := context.WithTimeout(ctx, client.timeout)
	defer close()
	areq := addPolicyReq{Token: in.GetToken(), Act: in.GetAct(), Obj: in.GetObj(), Sub: in.GetSub()}
	res, err := client.addPolicy(ctx, areq)
	if err != nil {
		return &policies.AddPolicyRes{}, err
	}

	apr := res.(addPolicyRes)
	return &policies.AddPolicyRes{Authorized: apr.authorized}, err
}

func decodeAddPolicyResponse(_ context.Context, grpcRes interface{}) (interface{}, error) {
	res := grpcRes.(*policies.AddPolicyRes)
	return addPolicyRes{authorized: res.Authorized}, nil
}

func encodeAddPolicyRequest(_ context.Context, grpcReq interface{}) (interface{}, error) {
	req := grpcReq.(addPolicyReq)
	return &policies.AddPolicyReq{
		Token: req.Token,
		Sub:   req.Sub,
		Obj:   req.Obj,
		Act:   req.Act,
	}, nil
}

func (client grpcClient) DeletePolicy(ctx context.Context, in *policies.DeletePolicyReq, opts ...grpc.CallOption) (*policies.DeletePolicyRes, error) {
	ctx, close := context.WithTimeout(ctx, client.timeout)
	defer close()
	preq := policyReq{Token: in.GetToken(), Act: in.GetAct(), Obj: in.GetObj(), Sub: in.GetSub()}
	res, err := client.deletePolicy(ctx, preq)
	if err != nil {
		return &policies.DeletePolicyRes{}, err
	}

	dpr := res.(deletePolicyRes)
	return &policies.DeletePolicyRes{Deleted: dpr.deleted}, err
}

func decodeDeletePolicyResponse(_ context.Context, grpcRes interface{}) (interface{}, error) {
	res := grpcRes.(*policies.DeletePolicyRes)
	return deletePolicyRes{deleted: res.GetDeleted()}, nil
}

func encodeDeletePolicyRequest(_ context.Context, grpcReq interface{}) (interface{}, error) {
	req := grpcReq.(policyReq)
	return &policies.DeletePolicyReq{
		Token: req.Token,
		Sub:   req.Sub,
		Obj:   req.Obj,
		Act:   req.Act,
	}, nil
}

func (client grpcClient) ListPolicies(ctx context.Context, in *policies.ListPoliciesReq, opts ...grpc.CallOption) (*policies.ListPoliciesRes, error) {
	ctx, close := context.WithTimeout(ctx, client.timeout)
	defer close()
	lreq := listPoliciesReq{Token: in.GetToken(), Obj: in.GetObj(), Act: in.GetAct(), Sub: in.GetSub()}
	res, err := client.listPolicies(ctx, lreq)
	if err != nil {
		return &policies.ListPoliciesRes{}, err
	}

	lpr := res.(listPoliciesRes)
	return &policies.ListPoliciesRes{Objects: lpr.objects}, err
}

func decodeListPoliciesResponse(_ context.Context, grpcRes interface{}) (interface{}, error) {
	res := grpcRes.(*policies.ListPoliciesRes)
	return listPoliciesRes{objects: res.GetObjects()}, nil
}

func encodeListPoliciesRequest(_ context.Context, grpcReq interface{}) (interface{}, error) {
	req := grpcReq.(listPoliciesReq)
	return &policies.ListPoliciesReq{
		Token: req.Token,
		Sub:   req.Sub,
		Obj:   req.Obj,
		Act:   req.Act,
	}, nil
}
