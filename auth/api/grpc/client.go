// Copyright (c) Abstract Machines
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
	refresh         endpoint.Endpoint
	identify        endpoint.Endpoint
	authorize       endpoint.Endpoint
	addPolicy       endpoint.Endpoint
	addPolicies     endpoint.Endpoint
	deletePolicy    endpoint.Endpoint
	deletePolicies  endpoint.Endpoint
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
		addPolicies: kitgrpc.NewClient(
			conn,
			svcName,
			"AddPolicies",
			encodeAddPoliciesRequest,
			decodeAddPoliciesResponse,
			magistrala.AddPoliciesRes{},
		).Endpoint(),
		deletePolicy: kitgrpc.NewClient(
			conn,
			svcName,
			"DeletePolicy",
			encodeDeletePolicyRequest,
			decodeDeletePolicyResponse,
			magistrala.DeletePolicyRes{},
		).Endpoint(),
		deletePolicies: kitgrpc.NewClient(
			conn,
			svcName,
			"DeletePolicies",
			encodeDeletePoliciesRequest,
			decodeDeletePoliciesResponse,
			magistrala.DeletePoliciesRes{},
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

	res, err := client.issue(ctx, issueReq{userID: req.GetUserId(), domainID: req.GetDomainId(), keyType: auth.KeyType(req.Type)})
	if err != nil {
		return nil, err
	}
	return res.(*magistrala.Token), nil
}

func encodeIssueRequest(_ context.Context, grpcReq interface{}) (interface{}, error) {
	req := grpcReq.(issueReq)
	return &magistrala.IssueReq{UserId: req.userID, DomainId: &req.domainID, Type: uint32(req.keyType)}, nil
}

func decodeIssueResponse(_ context.Context, grpcRes interface{}) (interface{}, error) {
	return grpcRes, nil
}

func (client grpcClient) Refresh(ctx context.Context, req *magistrala.RefreshReq, _ ...grpc.CallOption) (*magistrala.Token, error) {
	ctx, close := context.WithTimeout(ctx, client.timeout)
	defer close()

	res, err := client.refresh(ctx, refreshReq{refreshToken: req.GetRefreshToken(), domainID: req.GetDomainId()})
	if err != nil {
		return nil, err
	}
	return res.(*magistrala.Token), nil
}

func encodeRefreshRequest(_ context.Context, grpcReq interface{}) (interface{}, error) {
	req := grpcReq.(refreshReq)
	return &magistrala.RefreshReq{RefreshToken: req.refreshToken, DomainId: &req.domainID}, nil
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
	return &magistrala.IdentityRes{Id: ir.id, UserId: ir.userID, DomainId: ir.domainID}, nil
}

func encodeIdentifyRequest(_ context.Context, grpcReq interface{}) (interface{}, error) {
	req := grpcReq.(identityReq)
	return &magistrala.IdentityReq{Token: req.token}, nil
}

func decodeIdentifyResponse(_ context.Context, grpcRes interface{}) (interface{}, error) {
	res := grpcRes.(*magistrala.IdentityRes)
	return identityRes{id: res.GetId(), userID: res.GetUserId(), domainID: res.GetDomainId()}, nil
}

func (client grpcClient) Authorize(ctx context.Context, req *magistrala.AuthorizeReq, _ ...grpc.CallOption) (r *magistrala.AuthorizeRes, err error) {
	ctx, close := context.WithTimeout(ctx, client.timeout)
	defer close()

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

func (client grpcClient) AddPolicy(ctx context.Context, in *magistrala.AddPolicyReq, opts ...grpc.CallOption) (*magistrala.AddPolicyRes, error) {
	ctx, close := context.WithTimeout(ctx, client.timeout)
	defer close()

	res, err := client.addPolicy(ctx, policyReq{
		Domain:      in.GetDomain(),
		SubjectType: in.GetSubjectType(),
		SubjectKind: in.GetSubjectKind(),
		Subject:     in.GetSubject(),
		Relation:    in.GetRelation(),
		Permission:  in.GetPermission(),
		ObjectType:  in.GetObjectType(),
		ObjectKind:  in.GetObjectKind(),
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
		Domain:      req.Domain,
		SubjectType: req.SubjectType,
		SubjectKind: req.SubjectKind,
		Subject:     req.Subject,
		Relation:    req.Relation,
		Permission:  req.Permission,
		ObjectType:  req.ObjectType,
		ObjectKind:  req.ObjectKind,
		Object:      req.Object,
	}, nil
}

func (client grpcClient) AddPolicies(ctx context.Context, in *magistrala.AddPoliciesReq, opts ...grpc.CallOption) (*magistrala.AddPoliciesRes, error) {
	ctx, close := context.WithTimeout(ctx, client.timeout)
	defer close()
	r := policiesReq{}
	if in.GetAddPoliciesReq() != nil {
		for _, mgApr := range in.GetAddPoliciesReq() {
			r = append(r, policyReq{
				Domain:      mgApr.GetDomain(),
				SubjectType: mgApr.GetSubjectType(),
				SubjectKind: mgApr.GetSubjectKind(),
				Subject:     mgApr.GetSubject(),
				Relation:    mgApr.GetRelation(),
				Permission:  mgApr.GetPermission(),
				ObjectType:  mgApr.GetObjectType(),
				ObjectKind:  mgApr.GetObjectKind(),
				Object:      mgApr.GetObject(),
			})
		}
	}

	res, err := client.addPolicies(ctx, r)
	if err != nil {
		return &magistrala.AddPoliciesRes{}, err
	}

	apr := res.(addPoliciesRes)
	return &magistrala.AddPoliciesRes{Authorized: apr.authorized}, err
}

func decodeAddPoliciesResponse(_ context.Context, grpcRes interface{}) (interface{}, error) {
	res := grpcRes.(*magistrala.AddPoliciesRes)
	return addPoliciesRes{authorized: res.Authorized}, nil
}

func encodeAddPoliciesRequest(_ context.Context, grpcReq interface{}) (interface{}, error) {
	reqs := grpcReq.(policiesReq)

	addPolicies := []*magistrala.AddPolicyReq{}

	for _, req := range reqs {
		addPolicies = append(addPolicies, &magistrala.AddPolicyReq{
			Domain:      req.Domain,
			SubjectType: req.SubjectType,
			SubjectKind: req.SubjectKind,
			Subject:     req.Subject,
			Relation:    req.Relation,
			Permission:  req.Permission,
			ObjectType:  req.ObjectType,
			ObjectKind:  req.ObjectKind,
			Object:      req.Object,
		})
	}
	return &magistrala.AddPoliciesReq{AddPoliciesReq: addPolicies}, nil
}

func (client grpcClient) DeletePolicy(ctx context.Context, in *magistrala.DeletePolicyReq, opts ...grpc.CallOption) (*magistrala.DeletePolicyRes, error) {
	ctx, close := context.WithTimeout(ctx, client.timeout)
	defer close()

	res, err := client.deletePolicy(ctx, policyReq{
		Domain:      in.GetDomain(),
		SubjectType: in.GetSubjectType(),
		SubjectKind: in.GetSubjectKind(),
		Subject:     in.GetSubject(),
		Relation:    in.GetRelation(),
		Permission:  in.GetPermission(),
		ObjectType:  in.GetObjectType(),
		ObjectKind:  in.GetObjectKind(),
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
		Domain:      req.Domain,
		SubjectType: req.SubjectType,
		SubjectKind: req.SubjectKind,
		Subject:     req.Subject,
		Relation:    req.Relation,
		Permission:  req.Permission,
		ObjectType:  req.ObjectType,
		ObjectKind:  req.ObjectKind,
		Object:      req.Object,
	}, nil
}

func (client grpcClient) DeletePolicies(ctx context.Context, in *magistrala.DeletePoliciesReq, opts ...grpc.CallOption) (*magistrala.DeletePoliciesRes, error) {
	ctx, close := context.WithTimeout(ctx, client.timeout)
	defer close()
	r := policiesReq{}

	if in.GetDeletePoliciesReq() != nil {
		for _, mgApr := range in.GetDeletePoliciesReq() {
			r = append(r, policyReq{
				Domain:      mgApr.GetDomain(),
				SubjectType: mgApr.GetSubjectType(),
				SubjectKind: mgApr.GetSubjectKind(),
				Subject:     mgApr.GetSubject(),
				Relation:    mgApr.GetRelation(),
				Permission:  mgApr.GetPermission(),
				ObjectType:  mgApr.GetObjectType(),
				ObjectKind:  mgApr.GetObjectKind(),
				Object:      mgApr.GetObject(),
			})
		}
	}
	res, err := client.deletePolicies(ctx, r)
	if err != nil {
		return &magistrala.DeletePoliciesRes{}, err
	}

	dpr := res.(deletePoliciesRes)
	return &magistrala.DeletePoliciesRes{Deleted: dpr.deleted}, err
}

func decodeDeletePoliciesResponse(_ context.Context, grpcRes interface{}) (interface{}, error) {
	res := grpcRes.(*magistrala.DeletePoliciesRes)
	return deletePoliciesRes{deleted: res.GetDeleted()}, nil
}

func encodeDeletePoliciesRequest(_ context.Context, grpcReq interface{}) (interface{}, error) {
	reqs := grpcReq.(policiesReq)

	deletePolicies := []*magistrala.DeletePolicyReq{}

	for _, req := range reqs {
		deletePolicies = append(deletePolicies, &magistrala.DeletePolicyReq{
			Domain:      req.Domain,
			SubjectType: req.SubjectType,
			SubjectKind: req.SubjectKind,
			Subject:     req.Subject,
			Relation:    req.Relation,
			Permission:  req.Permission,
			ObjectType:  req.ObjectType,
			ObjectKind:  req.ObjectKind,
			Object:      req.Object,
		})
	}
	return &magistrala.DeletePoliciesReq{DeletePoliciesReq: deletePolicies}, nil
}

func (client grpcClient) ListObjects(ctx context.Context, in *magistrala.ListObjectsReq, opts ...grpc.CallOption) (*magistrala.ListObjectsRes, error) {
	ctx, close := context.WithTimeout(ctx, client.timeout)
	defer close()

	res, err := client.listObjects(ctx, listObjectsReq{
		Domain:      in.GetDomain(),
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
		Domain:      req.Domain,
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
		Domain:      in.GetDomain(),
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
		Domain:      in.GetDomain(),
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
		Domain:      req.Domain,
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
		Domain:        in.GetDomain(),
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
		Domain:      req.Domain,
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
		Domain:      in.GetDomain(),
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
		Domain:      in.GetDomain(),
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
		Domain:      req.Domain,
		SubjectType: req.SubjectType,
		Subject:     req.Subject,
		Relation:    req.Relation,
		Permission:  req.Permission,
		ObjectType:  req.ObjectType,
		Object:      req.Object,
	}, nil
}
