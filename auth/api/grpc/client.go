// Copyright (c) Magistrala
// SPDX-License-Identifier: Apache-2.0

package grpc

import (
	"context"
	"time"

	"github.com/absmach/magistrala"
	"github.com/absmach/magistrala/auth"
	"github.com/go-kit/kit/endpoint"
	kitgrpc "github.com/go-kit/kit/transport/grpc"
	"google.golang.org/grpc"
)

const svcName = "magistrala.AuthService"

var _ magistrala.AuthServiceClient = (*grpcClient)(nil)

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
func NewClient(conn *grpc.ClientConn, timeout time.Duration) magistrala.AuthServiceClient {
	return &grpcClient{
		issue: kitgrpc.NewClient(
			conn,
			svcName,
			"Issue",
			encodeIssueRequest,
			decodeIssueResponse,
			magistrala.Token{},
		).Endpoint(),
		login: kitgrpc.NewClient(
			conn,
			svcName,
			"Login",
			encodeLoginRequest,
			decodeLoginResponse,
			magistrala.Token{},
		).Endpoint(),
		refresh: kitgrpc.NewClient(
			conn,
			svcName,
			"Refresh",
			encodeRefreshRequest,
			decodeRefreshResponse,
			magistrala.Token{},
		).Endpoint(),
		identify: kitgrpc.NewClient(
			conn,
			svcName,
			"Identify",
			encodeIdentifyRequest,
			decodeIdentifyResponse,
			magistrala.IdentityRes{},
		).Endpoint(),
		authorize: kitgrpc.NewClient(
			conn,
			svcName,
			"Authorize",
			encodeAuthorizeRequest,
			decodeAuthorizeResponse,
			magistrala.AuthorizeRes{},
		).Endpoint(),
		addPolicy: kitgrpc.NewClient(
			conn,
			svcName,
			"AddPolicy",
			encodeAddPolicyRequest,
			decodeAddPolicyResponse,
			magistrala.AddPolicyRes{},
		).Endpoint(),
		deletePolicy: kitgrpc.NewClient(
			conn,
			svcName,
			"DeletePolicy",
			encodeDeletePolicyRequest,
			decodeDeletePolicyResponse,
			magistrala.DeletePolicyRes{},
		).Endpoint(),
		listObjects: kitgrpc.NewClient(
			conn,
			svcName,
			"ListObjects",
			encodeListObjectsRequest,
			decodeListObjectsResponse,
			magistrala.ListObjectsRes{},
		).Endpoint(),
		listAllObjects: kitgrpc.NewClient(
			conn,
			svcName,
			"ListAllObjects",
			encodeListObjectsRequest,
			decodeListObjectsResponse,
			magistrala.ListObjectsRes{},
		).Endpoint(),
		countObjects: kitgrpc.NewClient(
			conn,
			svcName,
			"CountObjects",
			encodeCountObjectsRequest,
			decodeCountObjectsResponse,
			magistrala.CountObjectsRes{},
		).Endpoint(),
		listSubjects: kitgrpc.NewClient(
			conn,
			svcName,
			"ListSubjects",
			encodeListSubjectsRequest,
			decodeListSubjectsResponse,
			magistrala.ListSubjectsRes{},
		).Endpoint(),
		listAllSubjects: kitgrpc.NewClient(
			conn,
			svcName,
			"ListAllSubjects",
			encodeListSubjectsRequest,
			decodeListSubjectsResponse,
			magistrala.ListSubjectsRes{},
		).Endpoint(),
		countSubjects: kitgrpc.NewClient(
			conn,
			svcName,
			"CountSubjects",
			encodeCountSubjectsRequest,
			decodeCountSubjectsResponse,
			magistrala.CountSubjectsRes{},
		).Endpoint(),

		timeout: timeout,
	}
}

func (client grpcClient) Issue(ctx context.Context, req *magistrala.IssueReq, _ ...grpc.CallOption) (*magistrala.Token, error) {
	ctx, close := context.WithTimeout(ctx, client.timeout)
	defer close()

	res, err := client.issue(ctx, issueReq{id: req.GetId(), keyType: auth.KeyType(req.Type)})
	if err != nil {
		return nil, err
	}
	return res.(*magistrala.Token), nil
}

func encodeIssueRequest(_ context.Context, grpcReq interface{}) (interface{}, error) {
	req := grpcReq.(issueReq)
	return &magistrala.IssueReq{Id: req.id, Type: uint32(req.keyType)}, nil
}

func decodeIssueResponse(_ context.Context, grpcRes interface{}) (interface{}, error) {
	return grpcRes, nil
}

func (client grpcClient) Login(ctx context.Context, req *magistrala.LoginReq, _ ...grpc.CallOption) (*magistrala.Token, error) {
	ctx, close := context.WithTimeout(ctx, client.timeout)
	defer close()

	res, err := client.login(ctx, issueReq{id: req.GetId(), keyType: auth.APIKey})
	if err != nil {
		return nil, err
	}
	return res.(*magistrala.Token), nil
}

func encodeLoginRequest(_ context.Context, grpcReq interface{}) (interface{}, error) {
	req := grpcReq.(issueReq)
	return &magistrala.LoginReq{Id: req.id}, nil
}

func decodeLoginResponse(_ context.Context, grpcRes interface{}) (interface{}, error) {
	return grpcRes, nil
}

func (client grpcClient) Refresh(ctx context.Context, req *magistrala.RefreshReq, _ ...grpc.CallOption) (*magistrala.Token, error) {
	ctx, close := context.WithTimeout(ctx, client.timeout)
	defer close()

	res, err := client.refresh(ctx, refreshReq{value: req.GetValue()})
	if err != nil {
		return nil, err
	}
	return res.(*magistrala.Token), nil
}

func encodeRefreshRequest(_ context.Context, grpcReq interface{}) (interface{}, error) {
	req := grpcReq.(refreshReq)
	return &magistrala.RefreshReq{Value: req.value}, nil
}

func decodeRefreshResponse(_ context.Context, grpcRes interface{}) (interface{}, error) {
	return grpcRes, nil
}

func (client grpcClient) Identify(ctx context.Context, token *magistrala.IdentityReq, _ ...grpc.CallOption) (*magistrala.IdentityRes, error) {
	ctx, close := context.WithTimeout(ctx, client.timeout)
	defer close()

	res, err := client.identify(ctx, identityReq{token: token.GetToken()})
	if err != nil {
		return nil, err
	}

	ir := res.(identityRes)
	return &magistrala.IdentityRes{Id: ir.id}, nil
}

func encodeIdentifyRequest(_ context.Context, grpcReq interface{}) (interface{}, error) {
	req := grpcReq.(identityReq)
	return &magistrala.IdentityReq{Token: req.token}, nil
}

func decodeIdentifyResponse(_ context.Context, grpcRes interface{}) (interface{}, error) {
	res := grpcRes.(*magistrala.IdentityRes)
	return identityRes{id: res.GetId()}, nil
}

func (client grpcClient) Authorize(ctx context.Context, req *magistrala.AuthorizeReq, _ ...grpc.CallOption) (r *magistrala.AuthorizeRes, err error) {
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
		return &magistrala.AuthorizeRes{}, err
	}

	ar := res.(authorizeRes)
	return &magistrala.AuthorizeRes{Authorized: ar.authorized, Id: ar.id}, err
}

func decodeAuthorizeResponse(_ context.Context, grpcRes interface{}) (interface{}, error) {
	res := grpcRes.(*magistrala.AuthorizeRes)
	return authorizeRes{authorized: res.Authorized, id: res.Id}, nil
}

func encodeAuthorizeRequest(_ context.Context, grpcReq interface{}) (interface{}, error) {
	req := grpcReq.(authReq)
	return &magistrala.AuthorizeReq{
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

func (client grpcClient) AddPolicy(ctx context.Context, in *magistrala.AddPolicyReq, opts ...grpc.CallOption) (*magistrala.AddPolicyRes, error) {
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
		return &magistrala.AddPolicyRes{}, err
	}

	apr := res.(addPolicyRes)
	return &magistrala.AddPolicyRes{Authorized: apr.authorized}, err
}

func decodeAddPolicyResponse(_ context.Context, grpcRes interface{}) (interface{}, error) {
	res := grpcRes.(*magistrala.AddPolicyRes)
	return addPolicyRes{authorized: res.Authorized}, nil
}

func encodeAddPolicyRequest(_ context.Context, grpcReq interface{}) (interface{}, error) {
	req := grpcReq.(policyReq)
	return &magistrala.AddPolicyReq{
		Namespace:   req.Namespace,
		SubjectType: req.SubjectType,
		Subject:     req.Subject,
		Relation:    req.Relation,
		Permission:  req.Permission,
		ObjectType:  req.ObjectType,
		Object:      req.Object,
	}, nil
}

func (client grpcClient) DeletePolicy(ctx context.Context, in *magistrala.DeletePolicyReq, opts ...grpc.CallOption) (*magistrala.DeletePolicyRes, error) {
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
		return &magistrala.DeletePolicyRes{}, err
	}

	dpr := res.(deletePolicyRes)
	return &magistrala.DeletePolicyRes{Deleted: dpr.deleted}, err
}

func decodeDeletePolicyResponse(_ context.Context, grpcRes interface{}) (interface{}, error) {
	res := grpcRes.(*magistrala.DeletePolicyRes)
	return deletePolicyRes{deleted: res.GetDeleted()}, nil
}

func encodeDeletePolicyRequest(_ context.Context, grpcReq interface{}) (interface{}, error) {
	req := grpcReq.(policyReq)
	return &magistrala.DeletePolicyReq{
		Namespace:   req.Namespace,
		SubjectType: req.SubjectType,
		Subject:     req.Subject,
		Relation:    req.Relation,
		Permission:  req.Permission,
		ObjectType:  req.ObjectType,
		Object:      req.Object,
	}, nil
}

func (client grpcClient) ListObjects(ctx context.Context, in *magistrala.ListObjectsReq, opts ...grpc.CallOption) (*magistrala.ListObjectsRes, error) {
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
		return &magistrala.ListObjectsRes{}, err
	}

	lpr := res.(listObjectsRes)
	return &magistrala.ListObjectsRes{Policies: lpr.policies}, err
}

func decodeListObjectsResponse(_ context.Context, grpcRes interface{}) (interface{}, error) {
	res := grpcRes.(*magistrala.ListObjectsRes)
	return listObjectsRes{policies: res.GetPolicies()}, nil
}

func encodeListObjectsRequest(_ context.Context, grpcReq interface{}) (interface{}, error) {
	req := grpcReq.(listObjectsReq)
	return &magistrala.ListObjectsReq{
		Namespace:   req.Namespace,
		SubjectType: req.SubjectType,
		Subject:     req.Subject,
		Relation:    req.Relation,
		Permission:  req.Permission,
		ObjectType:  req.ObjectType,
		Object:      req.Object,
	}, nil
}

func (client grpcClient) ListAllObjects(ctx context.Context, in *magistrala.ListObjectsReq, opts ...grpc.CallOption) (*magistrala.ListObjectsRes, error) {
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
		return &magistrala.ListObjectsRes{}, err
	}

	lpr := res.(listObjectsRes)
	return &magistrala.ListObjectsRes{Policies: lpr.policies}, err
}

func (client grpcClient) CountObjects(ctx context.Context, in *magistrala.CountObjectsReq, opts ...grpc.CallOption) (*magistrala.CountObjectsRes, error) {
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
		return &magistrala.CountObjectsRes{}, err
	}

	cp := res.(countObjectsRes)
	return &magistrala.CountObjectsRes{Count: int64(cp.count)}, err
}

func decodeCountObjectsResponse(_ context.Context, grpcRes interface{}) (interface{}, error) {
	res := grpcRes.(*magistrala.CountObjectsRes)
	return countObjectsRes{count: int(res.GetCount())}, nil
}

func encodeCountObjectsRequest(_ context.Context, grpcReq interface{}) (interface{}, error) {
	req := grpcReq.(countObjectsReq)
	return &magistrala.CountObjectsReq{
		Namespace:   req.Namespace,
		SubjectType: req.SubjectType,
		Subject:     req.Subject,
		Relation:    req.Relation,
		Permission:  req.Permission,
		ObjectType:  req.ObjectType,
		Object:      req.Object,
	}, nil
}

func (client grpcClient) ListSubjects(ctx context.Context, in *magistrala.ListSubjectsReq, opts ...grpc.CallOption) (*magistrala.ListSubjectsRes, error) {
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
		NextPageToken: in.GetNextPageToken(),
	})
	if err != nil {
		return &magistrala.ListSubjectsRes{}, err
	}

	lpr := res.(listSubjectsRes)
	return &magistrala.ListSubjectsRes{Policies: lpr.policies}, err
}

func decodeListSubjectsResponse(_ context.Context, grpcRes interface{}) (interface{}, error) {
	res := grpcRes.(*magistrala.ListSubjectsRes)
	return listSubjectsRes{policies: res.GetPolicies()}, nil
}

func encodeListSubjectsRequest(_ context.Context, grpcReq interface{}) (interface{}, error) {
	req := grpcReq.(listSubjectsReq)
	return &magistrala.ListSubjectsReq{
		Namespace:   req.Namespace,
		SubjectType: req.SubjectType,
		Subject:     req.Subject,
		Relation:    req.Relation,
		Permission:  req.Permission,
		ObjectType:  req.ObjectType,
		Object:      req.Object,
	}, nil
}

func (client grpcClient) ListAllSubjects(ctx context.Context, in *magistrala.ListSubjectsReq, opts ...grpc.CallOption) (*magistrala.ListSubjectsRes, error) {
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
		return &magistrala.ListSubjectsRes{}, err
	}

	lpr := res.(listSubjectsRes)
	return &magistrala.ListSubjectsRes{Policies: lpr.policies}, err
}

func (client grpcClient) CountSubjects(ctx context.Context, in *magistrala.CountSubjectsReq, opts ...grpc.CallOption) (*magistrala.CountSubjectsRes, error) {
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
		return &magistrala.CountSubjectsRes{}, err
	}

	cp := res.(countSubjectsRes)
	return &magistrala.CountSubjectsRes{Count: int64(cp.count)}, err
}

func decodeCountSubjectsResponse(_ context.Context, grpcRes interface{}) (interface{}, error) {
	res := grpcRes.(*magistrala.CountSubjectsRes)
	return countSubjectsRes{count: int(res.GetCount())}, nil
}

func encodeCountSubjectsRequest(_ context.Context, grpcReq interface{}) (interface{}, error) {
	req := grpcReq.(countSubjectsReq)
	return &magistrala.CountSubjectsReq{
		Namespace:   req.Namespace,
		SubjectType: req.SubjectType,
		Subject:     req.Subject,
		Relation:    req.Relation,
		Permission:  req.Permission,
		ObjectType:  req.ObjectType,
		Object:      req.Object,
	}, nil
}
