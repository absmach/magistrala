// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package grpc

import (
	"context"
	"fmt"
	"time"

	"github.com/absmach/magistrala"
	"github.com/absmach/magistrala/auth"
	"github.com/absmach/magistrala/pkg/errors"
	"github.com/go-kit/kit/endpoint"
	kitgrpc "github.com/go-kit/kit/transport/grpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
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
	listPermissions endpoint.Endpoint
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
		listPermissions: kitgrpc.NewClient(
			conn,
			svcName,
			"ListPermissions",
			encodeListPermissionsRequest,
			decodeListPermissionsResponse,
			magistrala.ListPermissionsRes{},
		).Endpoint(),

		timeout: timeout,
	}
}

func (client grpcClient) Issue(ctx context.Context, req *magistrala.IssueReq, _ ...grpc.CallOption) (*magistrala.Token, error) {
	ctx, cancel := context.WithTimeout(ctx, client.timeout)
	defer cancel()

	res, err := client.issue(ctx, issueReq{userID: req.GetUserId(), domainID: req.GetDomainId(), keyType: auth.KeyType(req.Type)})
	if err != nil {
		return &magistrala.Token{}, decodeError(err)
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
	ctx, cancel := context.WithTimeout(ctx, client.timeout)
	defer cancel()

	res, err := client.refresh(ctx, refreshReq{refreshToken: req.GetRefreshToken(), domainID: req.GetDomainId()})
	if err != nil {
		return &magistrala.Token{}, decodeError(err)
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
	ctx, cancel := context.WithTimeout(ctx, client.timeout)
	defer cancel()

	res, err := client.identify(ctx, identityReq{token: token.GetToken()})
	if err != nil {
		return &magistrala.IdentityRes{}, decodeError(err)
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
		return &magistrala.AuthorizeRes{}, decodeError(err)
	}

	ar := res.(authorizeRes)
	return &magistrala.AuthorizeRes{Authorized: ar.authorized, Id: ar.id}, nil
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
	ctx, cancel := context.WithTimeout(ctx, client.timeout)
	defer cancel()

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
		return &magistrala.AddPolicyRes{}, decodeError(err)
	}

	apr := res.(addPolicyRes)
	return &magistrala.AddPolicyRes{Authorized: apr.authorized}, nil
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
	ctx, cancel := context.WithTimeout(ctx, client.timeout)
	defer cancel()
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
		return &magistrala.AddPoliciesRes{}, decodeError(err)
	}

	apr := res.(addPoliciesRes)
	return &magistrala.AddPoliciesRes{Authorized: apr.authorized}, nil
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
	ctx, cancel := context.WithTimeout(ctx, client.timeout)
	defer cancel()

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
		return &magistrala.DeletePolicyRes{}, decodeError(err)
	}

	dpr := res.(deletePolicyRes)
	return &magistrala.DeletePolicyRes{Deleted: dpr.deleted}, nil
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
	ctx, cancel := context.WithTimeout(ctx, client.timeout)
	defer cancel()
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
		return &magistrala.DeletePoliciesRes{}, decodeError(err)
	}

	dpr := res.(deletePoliciesRes)
	return &magistrala.DeletePoliciesRes{Deleted: dpr.deleted}, nil
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
	ctx, cancel := context.WithTimeout(ctx, client.timeout)
	defer cancel()

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
		return &magistrala.ListObjectsRes{}, decodeError(err)
	}

	lpr := res.(listObjectsRes)
	return &magistrala.ListObjectsRes{Policies: lpr.policies}, nil
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
	ctx, cancel := context.WithTimeout(ctx, client.timeout)
	defer cancel()

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
		return &magistrala.ListObjectsRes{}, decodeError(err)
	}

	lpr := res.(listObjectsRes)
	return &magistrala.ListObjectsRes{Policies: lpr.policies}, nil
}

func (client grpcClient) CountObjects(ctx context.Context, in *magistrala.CountObjectsReq, opts ...grpc.CallOption) (*magistrala.CountObjectsRes, error) {
	ctx, cancel := context.WithTimeout(ctx, client.timeout)
	defer cancel()

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
		return &magistrala.CountObjectsRes{}, decodeError(err)
	}

	cp := res.(countObjectsRes)
	return &magistrala.CountObjectsRes{Count: int64(cp.count)}, nil
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
	ctx, cancel := context.WithTimeout(ctx, client.timeout)
	defer cancel()

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
		return &magistrala.ListSubjectsRes{}, decodeError(err)
	}

	lpr := res.(listSubjectsRes)
	return &magistrala.ListSubjectsRes{Policies: lpr.policies}, nil
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
	ctx, cancel := context.WithTimeout(ctx, client.timeout)
	defer cancel()

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
		return &magistrala.ListSubjectsRes{}, decodeError(err)
	}

	lpr := res.(listSubjectsRes)
	return &magistrala.ListSubjectsRes{Policies: lpr.policies}, nil
}

func (client grpcClient) CountSubjects(ctx context.Context, in *magistrala.CountSubjectsReq, opts ...grpc.CallOption) (*magistrala.CountSubjectsRes, error) {
	ctx, cancel := context.WithTimeout(ctx, client.timeout)
	defer cancel()

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

func (client grpcClient) ListPermissions(ctx context.Context, in *magistrala.ListPermissionsReq, opts ...grpc.CallOption) (*magistrala.ListPermissionsRes, error) {
	ctx, cancel := context.WithTimeout(ctx, client.timeout)
	defer cancel()

	res, err := client.listPermissions(ctx, listPermissionsReq{
		Domain:            in.GetDomain(),
		SubjectType:       in.GetSubjectType(),
		Subject:           in.GetSubject(),
		SubjectRelation:   in.GetSubjectRelation(),
		ObjectType:        in.GetObjectType(),
		Object:            in.GetObject(),
		FilterPermissions: in.GetFilterPermissions(),
	})
	if err != nil {
		return &magistrala.ListPermissionsRes{}, decodeError(err)
	}

	lp := res.(listPermissionsRes)
	return &magistrala.ListPermissionsRes{
		Domain:          lp.Domain,
		SubjectType:     lp.SubjectType,
		Subject:         lp.Subject,
		SubjectRelation: lp.SubjectRelation,
		ObjectType:      lp.ObjectType,
		Object:          lp.Object,
		Permissions:     lp.Permissions,
	}, nil
}

func decodeListPermissionsResponse(_ context.Context, grpcRes interface{}) (interface{}, error) {
	res := grpcRes.(*magistrala.ListPermissionsRes)
	return listPermissionsRes{
		Domain:          res.GetDomain(),
		SubjectType:     res.GetSubjectType(),
		Subject:         res.GetSubject(),
		SubjectRelation: res.GetSubjectRelation(),
		ObjectType:      res.GetObjectType(),
		Object:          res.GetObject(),
		Permissions:     res.GetPermissions(),
	}, nil
}

func encodeListPermissionsRequest(_ context.Context, grpcReq interface{}) (interface{}, error) {
	req := grpcReq.(listPermissionsReq)
	return &magistrala.ListPermissionsReq{
		Domain:      req.Domain,
		SubjectType: req.SubjectType,
		Subject:     req.Subject,
		ObjectType:  req.ObjectType,
		Object:      req.Object,
	}, nil
}

func decodeError(err error) error {
	if st, ok := status.FromError(err); ok {
		switch st.Code() {
		case codes.NotFound:
			return errors.Wrap(errors.ErrNotFound, errors.New(st.Message()))
		case codes.InvalidArgument:
			return errors.Wrap(errors.ErrMalformedEntity, errors.New(st.Message()))
		case codes.AlreadyExists:
			return errors.Wrap(errors.ErrConflict, errors.New(st.Message()))
		case codes.Unauthenticated:
			return errors.Wrap(errors.ErrAuthentication, errors.New(st.Message()))
		case codes.OK:
			if msg := st.Message(); msg != "" {
				return errors.Wrap(errors.ErrUnidentified, errors.New(msg))
			}
			return nil
		case codes.FailedPrecondition:
			return errors.Wrap(errors.ErrMalformedEntity, errors.New(st.Message()))
		case codes.PermissionDenied:
			return errors.Wrap(errors.ErrAuthorization, errors.New(st.Message()))
		default:
			return errors.Wrap(fmt.Errorf("unexpected gRPC status: %s (status code:%v)", st.Code().String(), st.Code()), errors.New(st.Message()))
		}
	}
	return err
}
