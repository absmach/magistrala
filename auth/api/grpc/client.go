// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package grpc

import (
	"context"
	"time"

	"github.com/go-kit/kit/endpoint"
	kitgrpc "github.com/go-kit/kit/transport/grpc"
	"github.com/mainflux/mainflux"
	"github.com/mainflux/mainflux/auth"
	"google.golang.org/grpc"
)

const svcName = "mainflux.AuthService"

var _ mainflux.AuthServiceClient = (*grpcClient)(nil)

type grpcClient struct {
	issue           endpoint.Endpoint
	login           endpoint.Endpoint
	refresh         endpoint.Endpoint
	identify        endpoint.Endpoint
	authorize       endpoint.Endpoint
	addPolicy       endpoint.Endpoint
	deletePolicy    endpoint.Endpoint
	listObjects     endpoint.Endpoint
	listAllObjects  endpoint.Endpoint
	countObjects    endpoint.Endpoint
	listSubjects    endpoint.Endpoint
	listAllSubjects endpoint.Endpoint
	countSubjects   endpoint.Endpoint
	timeout         time.Duration
}

// NewClient returns new gRPC client instance.
func NewClient(conn *grpc.ClientConn, timeout time.Duration) mainflux.AuthServiceClient {
	return &grpcClient{
		issue: kitgrpc.NewClient(
			conn,
			svcName,
			"Issue",
			encodeIssueRequest,
			decodeIssueResponse,
			mainflux.Token{},
		).Endpoint(),
		login: kitgrpc.NewClient(
			conn,
			svcName,
			"Login",
			encodeLoginRequest,
			decodeLoginResponse,
			mainflux.Token{},
		).Endpoint(),
		refresh: kitgrpc.NewClient(
			conn,
			svcName,
			"Refresh",
			encodeRefreshRequest,
			decodeRefreshResponse,
			mainflux.Token{},
		).Endpoint(),
		identify: kitgrpc.NewClient(
			conn,
			svcName,
			"Identify",
			encodeIdentifyRequest,
			decodeIdentifyResponse,
			mainflux.IdentityRes{},
		).Endpoint(),
		authorize: kitgrpc.NewClient(
			conn,
			svcName,
			"Authorize",
			encodeAuthorizeRequest,
			decodeAuthorizeResponse,
			mainflux.AuthorizeRes{},
		).Endpoint(),
		addPolicy: kitgrpc.NewClient(
			conn,
			svcName,
			"AddPolicy",
			encodeAddPolicyRequest,
			decodeAddPolicyResponse,
			mainflux.AddPolicyRes{},
		).Endpoint(),
		deletePolicy: kitgrpc.NewClient(
			conn,
			svcName,
			"DeletePolicy",
			encodeDeletePolicyRequest,
			decodeDeletePolicyResponse,
			mainflux.DeletePolicyRes{},
		).Endpoint(),
		listObjects: kitgrpc.NewClient(
			conn,
			svcName,
			"ListObjects",
			encodeListObjectsRequest,
			decodeListObjectsResponse,
			mainflux.ListObjectsRes{},
		).Endpoint(),
		listAllObjects: kitgrpc.NewClient(
			conn,
			svcName,
			"ListAllObjects",
			encodeListObjectsRequest,
			decodeListObjectsResponse,
			mainflux.ListObjectsRes{},
		).Endpoint(),
		countObjects: kitgrpc.NewClient(
			conn,
			svcName,
			"CountObjects",
			encodeCountObjectsRequest,
			decodeCountObjectsResponse,
			mainflux.CountObjectsRes{},
		).Endpoint(),
		listSubjects: kitgrpc.NewClient(
			conn,
			svcName,
			"ListSubjects",
			encodeListSubjectsRequest,
			decodeListSubjectsResponse,
			mainflux.ListSubjectsRes{},
		).Endpoint(),
		listAllSubjects: kitgrpc.NewClient(
			conn,
			svcName,
			"ListAllSubjects",
			encodeListSubjectsRequest,
			decodeListSubjectsResponse,
			mainflux.ListSubjectsRes{},
		).Endpoint(),
		countSubjects: kitgrpc.NewClient(
			conn,
			svcName,
			"CountSubjects",
			encodeCountSubjectsRequest,
			decodeCountSubjectsResponse,
			mainflux.CountSubjectsRes{},
		).Endpoint(),

		timeout: timeout,
	}
}

func (client grpcClient) Issue(ctx context.Context, req *mainflux.IssueReq, _ ...grpc.CallOption) (*mainflux.Token, error) {
	ctx, close := context.WithTimeout(ctx, client.timeout)
	defer close()

	res, err := client.issue(ctx, issueReq{id: req.GetId(), keyType: auth.KeyType(req.Type)})
	if err != nil {
		return nil, err
	}
	return res.(*mainflux.Token), nil
}

func encodeIssueRequest(_ context.Context, grpcReq interface{}) (interface{}, error) {
	req := grpcReq.(issueReq)
	return &mainflux.IssueReq{Id: req.id, Type: uint32(req.keyType)}, nil
}

func decodeIssueResponse(_ context.Context, grpcRes interface{}) (interface{}, error) {
	return grpcRes, nil
}

func (client grpcClient) Login(ctx context.Context, req *mainflux.LoginReq, _ ...grpc.CallOption) (*mainflux.Token, error) {
	ctx, close := context.WithTimeout(ctx, client.timeout)
	defer close()

	res, err := client.login(ctx, issueReq{id: req.GetId(), keyType: auth.APIKey})
	if err != nil {
		return nil, err
	}
	return res.(*mainflux.Token), nil
}

func encodeLoginRequest(_ context.Context, grpcReq interface{}) (interface{}, error) {
	req := grpcReq.(issueReq)
	return &mainflux.LoginReq{Id: req.id}, nil
}

func decodeLoginResponse(_ context.Context, grpcRes interface{}) (interface{}, error) {
	return grpcRes, nil
}

func (client grpcClient) Refresh(ctx context.Context, req *mainflux.RefreshReq, _ ...grpc.CallOption) (*mainflux.Token, error) {
	ctx, close := context.WithTimeout(ctx, client.timeout)
	defer close()

	res, err := client.refresh(ctx, refreshReq{value: req.GetValue()})
	if err != nil {
		return nil, err
	}
	return res.(*mainflux.Token), nil
}

func encodeRefreshRequest(_ context.Context, grpcReq interface{}) (interface{}, error) {
	req := grpcReq.(refreshReq)
	return &mainflux.RefreshReq{Value: req.value}, nil
}

func decodeRefreshResponse(_ context.Context, grpcRes interface{}) (interface{}, error) {
	return grpcRes, nil
}

func (client grpcClient) Identify(ctx context.Context, token *mainflux.IdentityReq, _ ...grpc.CallOption) (*mainflux.IdentityRes, error) {
	ctx, close := context.WithTimeout(ctx, client.timeout)
	defer close()

	res, err := client.identify(ctx, identityReq{token: token.GetToken()})
	if err != nil {
		return nil, err
	}

	ir := res.(identityRes)
	return &mainflux.IdentityRes{Id: ir.id}, nil
}

func encodeIdentifyRequest(_ context.Context, grpcReq interface{}) (interface{}, error) {
	req := grpcReq.(identityReq)
	return &mainflux.IdentityReq{Token: req.token}, nil
}

func decodeIdentifyResponse(_ context.Context, grpcRes interface{}) (interface{}, error) {
	res := grpcRes.(*mainflux.IdentityRes)
	return identityRes{id: res.GetId()}, nil
}

func (client grpcClient) Authorize(ctx context.Context, req *mainflux.AuthorizeReq, _ ...grpc.CallOption) (r *mainflux.AuthorizeRes, err error) {
	ctx, close := context.WithTimeout(ctx, client.timeout)
	defer close()

	res, err := client.authorize(ctx, authReq{
		Namespace:   req.GetNamespace(),
		SubjectType: req.GetSubjectType(),
		Subject:     req.GetSubject(),
		SubjectKind: req.GetSubjectKind(),
		Relation:    req.GetRelation(),
		Permission:  req.GetPermission(),
		ObjectType:  req.GetObjectType(),
		Object:      req.GetObject(),
	})
	if err != nil {
		return &mainflux.AuthorizeRes{}, err
	}

	ar := res.(authorizeRes)
	return &mainflux.AuthorizeRes{Authorized: ar.authorized, Id: ar.id}, err
}

func decodeAuthorizeResponse(_ context.Context, grpcRes interface{}) (interface{}, error) {
	res := grpcRes.(*mainflux.AuthorizeRes)
	return authorizeRes{authorized: res.Authorized, id: res.Id}, nil
}

func encodeAuthorizeRequest(_ context.Context, grpcReq interface{}) (interface{}, error) {
	req := grpcReq.(authReq)
	return &mainflux.AuthorizeReq{
		Namespace:   req.Namespace,
		SubjectType: req.SubjectType,
		Subject:     req.Subject,
		SubjectKind: req.SubjectKind,
		Relation:    req.Relation,
		Permission:  req.Permission,
		ObjectType:  req.ObjectType,
		Object:      req.Object,
	}, nil
}

func (client grpcClient) AddPolicy(ctx context.Context, in *mainflux.AddPolicyReq, opts ...grpc.CallOption) (*mainflux.AddPolicyRes, error) {
	ctx, close := context.WithTimeout(ctx, client.timeout)
	defer close()

	res, err := client.addPolicy(ctx, policyReq{
		Namespace:   in.GetNamespace(),
		SubjectType: in.GetSubjectType(),
		Subject:     in.GetSubject(),
		Relation:    in.GetRelation(),
		Permission:  in.GetPermission(),
		ObjectType:  in.GetObjectType(),
		Object:      in.GetObject(),
	})
	if err != nil {
		return &mainflux.AddPolicyRes{}, err
	}

	apr := res.(addPolicyRes)
	return &mainflux.AddPolicyRes{Authorized: apr.authorized}, err
}

func decodeAddPolicyResponse(_ context.Context, grpcRes interface{}) (interface{}, error) {
	res := grpcRes.(*mainflux.AddPolicyRes)
	return addPolicyRes{authorized: res.Authorized}, nil
}

func encodeAddPolicyRequest(_ context.Context, grpcReq interface{}) (interface{}, error) {
	req := grpcReq.(policyReq)
	return &mainflux.AddPolicyReq{
		Namespace:   req.Namespace,
		SubjectType: req.SubjectType,
		Subject:     req.Subject,
		Relation:    req.Relation,
		Permission:  req.Permission,
		ObjectType:  req.ObjectType,
		Object:      req.Object,
	}, nil
}

func (client grpcClient) DeletePolicy(ctx context.Context, in *mainflux.DeletePolicyReq, opts ...grpc.CallOption) (*mainflux.DeletePolicyRes, error) {
	ctx, close := context.WithTimeout(ctx, client.timeout)
	defer close()

	res, err := client.deletePolicy(ctx, policyReq{
		Namespace:   in.GetNamespace(),
		SubjectType: in.GetSubjectType(),
		Subject:     in.GetSubject(),
		Relation:    in.GetRelation(),
		Permission:  in.GetPermission(),
		ObjectType:  in.GetObjectType(),
		Object:      in.GetObject(),
	})
	if err != nil {
		return &mainflux.DeletePolicyRes{}, err
	}

	dpr := res.(deletePolicyRes)
	return &mainflux.DeletePolicyRes{Deleted: dpr.deleted}, err
}

func decodeDeletePolicyResponse(_ context.Context, grpcRes interface{}) (interface{}, error) {
	res := grpcRes.(*mainflux.DeletePolicyRes)
	return deletePolicyRes{deleted: res.GetDeleted()}, nil
}

func encodeDeletePolicyRequest(_ context.Context, grpcReq interface{}) (interface{}, error) {
	req := grpcReq.(policyReq)
	return &mainflux.DeletePolicyReq{
		Namespace:   req.Namespace,
		SubjectType: req.SubjectType,
		Subject:     req.Subject,
		Relation:    req.Relation,
		Permission:  req.Permission,
		ObjectType:  req.ObjectType,
		Object:      req.Object,
	}, nil
}

func (client grpcClient) ListObjects(ctx context.Context, in *mainflux.ListObjectsReq, opts ...grpc.CallOption) (*mainflux.ListObjectsRes, error) {
	ctx, close := context.WithTimeout(ctx, client.timeout)
	defer close()

	res, err := client.listObjects(ctx, listObjectsReq{
		Namespace:   in.GetNamespace(),
		SubjectType: in.GetSubjectType(),
		Subject:     in.GetSubject(),
		Relation:    in.GetRelation(),
		Permission:  in.GetPermission(),
		ObjectType:  in.GetObjectType(),
		Object:      in.GetObject(),
	})
	if err != nil {
		return &mainflux.ListObjectsRes{}, err
	}

	lpr := res.(listObjectsRes)
	return &mainflux.ListObjectsRes{Policies: lpr.policies}, err
}

func decodeListObjectsResponse(_ context.Context, grpcRes interface{}) (interface{}, error) {
	res := grpcRes.(*mainflux.ListObjectsRes)
	return listObjectsRes{policies: res.GetPolicies()}, nil
}

func encodeListObjectsRequest(_ context.Context, grpcReq interface{}) (interface{}, error) {
	req := grpcReq.(listObjectsReq)
	return &mainflux.ListObjectsReq{
		Namespace:   req.Namespace,
		SubjectType: req.SubjectType,
		Subject:     req.Subject,
		Relation:    req.Relation,
		Permission:  req.Permission,
		ObjectType:  req.ObjectType,
		Object:      req.Object,
	}, nil
}

func (client grpcClient) ListAllObjects(ctx context.Context, in *mainflux.ListObjectsReq, opts ...grpc.CallOption) (*mainflux.ListObjectsRes, error) {
	ctx, close := context.WithTimeout(ctx, client.timeout)
	defer close()

	res, err := client.listAllObjects(ctx, listObjectsReq{
		Namespace:   in.GetNamespace(),
		SubjectType: in.GetSubjectType(),
		Subject:     in.GetSubject(),
		Relation:    in.GetRelation(),
		Permission:  in.GetPermission(),
		ObjectType:  in.GetObjectType(),
		Object:      in.GetObject(),
	})
	if err != nil {
		return &mainflux.ListObjectsRes{}, err
	}

	lpr := res.(listObjectsRes)
	return &mainflux.ListObjectsRes{Policies: lpr.policies}, err
}

func (client grpcClient) CountObjects(ctx context.Context, in *mainflux.CountObjectsReq, opts ...grpc.CallOption) (*mainflux.CountObjectsRes, error) {
	ctx, close := context.WithTimeout(ctx, client.timeout)
	defer close()

	res, err := client.countObjects(ctx, listObjectsReq{
		Namespace:   in.GetNamespace(),
		SubjectType: in.GetSubjectType(),
		Subject:     in.GetSubject(),
		Relation:    in.GetRelation(),
		Permission:  in.GetPermission(),
		ObjectType:  in.GetObjectType(),
		Object:      in.GetObject(),
	})
	if err != nil {
		return &mainflux.CountObjectsRes{}, err
	}

	cp := res.(countObjectsRes)
	return &mainflux.CountObjectsRes{Count: int64(cp.count)}, err
}

func decodeCountObjectsResponse(_ context.Context, grpcRes interface{}) (interface{}, error) {
	res := grpcRes.(*mainflux.CountObjectsRes)
	return countObjectsRes{count: int(res.GetCount())}, nil
}

func encodeCountObjectsRequest(_ context.Context, grpcReq interface{}) (interface{}, error) {
	req := grpcReq.(countObjectsReq)
	return &mainflux.CountObjectsReq{
		Namespace:   req.Namespace,
		SubjectType: req.SubjectType,
		Subject:     req.Subject,
		Relation:    req.Relation,
		Permission:  req.Permission,
		ObjectType:  req.ObjectType,
		Object:      req.Object,
	}, nil
}

func (client grpcClient) ListSubjects(ctx context.Context, in *mainflux.ListSubjectsReq, opts ...grpc.CallOption) (*mainflux.ListSubjectsRes, error) {
	ctx, close := context.WithTimeout(ctx, client.timeout)
	defer close()

	res, err := client.listSubjects(ctx, listSubjectsReq{
		Namespace:     in.GetNamespace(),
		SubjectType:   in.GetSubjectType(),
		Subject:       in.GetSubject(),
		Relation:      in.GetRelation(),
		Permission:    in.GetPermission(),
		ObjectType:    in.GetObjectType(),
		Object:        in.GetObject(),
		NextPageToken: in.GetNextPageToken()})
	if err != nil {
		return &mainflux.ListSubjectsRes{}, err
	}

	lpr := res.(listSubjectsRes)
	return &mainflux.ListSubjectsRes{Policies: lpr.policies}, err
}

func decodeListSubjectsResponse(_ context.Context, grpcRes interface{}) (interface{}, error) {
	res := grpcRes.(*mainflux.ListSubjectsRes)
	return listSubjectsRes{policies: res.GetPolicies()}, nil
}

func encodeListSubjectsRequest(_ context.Context, grpcReq interface{}) (interface{}, error) {
	req := grpcReq.(listSubjectsReq)
	return &mainflux.ListSubjectsReq{
		Namespace:   req.Namespace,
		SubjectType: req.SubjectType,
		Subject:     req.Subject,
		Relation:    req.Relation,
		Permission:  req.Permission,
		ObjectType:  req.ObjectType,
		Object:      req.Object,
	}, nil
}

func (client grpcClient) ListAllSubjects(ctx context.Context, in *mainflux.ListSubjectsReq, opts ...grpc.CallOption) (*mainflux.ListSubjectsRes, error) {
	ctx, close := context.WithTimeout(ctx, client.timeout)
	defer close()

	res, err := client.listAllSubjects(ctx, listSubjectsReq{
		Namespace:   in.GetNamespace(),
		SubjectType: in.GetSubjectType(),
		Subject:     in.GetSubject(),
		Relation:    in.GetRelation(),
		Permission:  in.GetPermission(),
		ObjectType:  in.GetObjectType(),
		Object:      in.GetObject(),
	})
	if err != nil {
		return &mainflux.ListSubjectsRes{}, err
	}

	lpr := res.(listSubjectsRes)
	return &mainflux.ListSubjectsRes{Policies: lpr.policies}, err
}

func (client grpcClient) CountSubjects(ctx context.Context, in *mainflux.CountSubjectsReq, opts ...grpc.CallOption) (*mainflux.CountSubjectsRes, error) {
	ctx, close := context.WithTimeout(ctx, client.timeout)
	defer close()

	res, err := client.countSubjects(ctx, countSubjectsReq{
		Namespace:   in.GetNamespace(),
		SubjectType: in.GetSubjectType(),
		Subject:     in.GetSubject(),
		Relation:    in.GetRelation(),
		Permission:  in.GetPermission(),
		ObjectType:  in.GetObjectType(),
		Object:      in.GetObject(),
	})
	if err != nil {
		return &mainflux.CountSubjectsRes{}, err
	}

	cp := res.(countSubjectsRes)
	return &mainflux.CountSubjectsRes{Count: int64(cp.count)}, err
}

func decodeCountSubjectsResponse(_ context.Context, grpcRes interface{}) (interface{}, error) {
	res := grpcRes.(*mainflux.CountSubjectsRes)
	return countSubjectsRes{count: int(res.GetCount())}, nil
}

func encodeCountSubjectsRequest(_ context.Context, grpcReq interface{}) (interface{}, error) {
	req := grpcReq.(countSubjectsReq)
	return &mainflux.CountSubjectsReq{
		Namespace:   req.Namespace,
		SubjectType: req.SubjectType,
		Subject:     req.Subject,
		Relation:    req.Relation,
		Permission:  req.Permission,
		ObjectType:  req.ObjectType,
		Object:      req.Object,
	}, nil
}
