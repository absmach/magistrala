// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package grpc

import (
	"context"

	"github.com/absmach/magistrala"
	"github.com/absmach/magistrala/auth"
	"github.com/absmach/magistrala/internal/apiutil"
	"github.com/absmach/magistrala/pkg/errors"
	svcerr "github.com/absmach/magistrala/pkg/errors/service"
	kitgrpc "github.com/go-kit/kit/transport/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var _ magistrala.AuthServiceServer = (*grpcServer)(nil)

var (
	defThingsFilterPermissions = []string{
		auth.AdminPermission,
		auth.DeletePermission,
		auth.EditPermission,
		auth.ViewPermission,
		auth.SharePermission,
		auth.PublishPermission,
		auth.SubscribePermission,
	}

	defGroupsFilterPermissions = []string{
		auth.AdminPermission,
		auth.DeletePermission,
		auth.EditPermission,
		auth.ViewPermission,
		auth.MembershipPermission,
		auth.SharePermission,
	}

	defDomainsFilterPermissions = []string{
		auth.AdminPermission,
		auth.DeletePermission,
		auth.EditPermission,
		auth.ViewPermission,
		auth.MembershipPermission,
		auth.SharePermission,
	}
	defPlatformFilterPermissions = []string{
		auth.AdminPermission,
		auth.MembershipPermission,
	}
)

type grpcServer struct {
	magistrala.UnimplementedAuthServiceServer
	issue           kitgrpc.Handler
	refresh         kitgrpc.Handler
	identify        kitgrpc.Handler
	authorize       kitgrpc.Handler
	addPolicy       kitgrpc.Handler
	addPolicies     kitgrpc.Handler
	deletePolicy    kitgrpc.Handler
	deletePolicies  kitgrpc.Handler
	listObjects     kitgrpc.Handler
	listAllObjects  kitgrpc.Handler
	countObjects    kitgrpc.Handler
	listSubjects    kitgrpc.Handler
	listAllSubjects kitgrpc.Handler
	countSubjects   kitgrpc.Handler
	listPermissions kitgrpc.Handler
}

// NewServer returns new AuthServiceServer instance.
func NewServer(svc auth.Service) magistrala.AuthServiceServer {
	return &grpcServer{
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
		authorize: kitgrpc.NewServer(
			(authorizeEndpoint(svc)),
			decodeAuthorizeRequest,
			encodeAuthorizeResponse,
		),
		addPolicy: kitgrpc.NewServer(
			(addPolicyEndpoint(svc)),
			decodeAddPolicyRequest,
			encodeAddPolicyResponse,
		),
		addPolicies: kitgrpc.NewServer(
			(addPoliciesEndpoint(svc)),
			decodeAddPoliciesRequest,
			encodeAddPoliciesResponse,
		),
		deletePolicy: kitgrpc.NewServer(
			(deletePolicyEndpoint(svc)),
			decodeDeletePolicyRequest,
			encodeDeletePolicyResponse,
		),
		deletePolicies: kitgrpc.NewServer(
			(deletePoliciesEndpoint(svc)),
			decodeDeletePoliciesRequest,
			encodeDeletePoliciesResponse,
		),
		listObjects: kitgrpc.NewServer(
			(listObjectsEndpoint(svc)),
			decodeListObjectsRequest,
			encodeListObjectsResponse,
		),
		listAllObjects: kitgrpc.NewServer(
			(listAllObjectsEndpoint(svc)),
			decodeListObjectsRequest,
			encodeListObjectsResponse,
		),
		countObjects: kitgrpc.NewServer(
			(countObjectsEndpoint(svc)),
			decodeCountObjectsRequest,
			encodeCountObjectsResponse,
		),
		listSubjects: kitgrpc.NewServer(
			(listSubjectsEndpoint(svc)),
			decodeListSubjectsRequest,
			encodeListSubjectsResponse,
		),
		listAllSubjects: kitgrpc.NewServer(
			(listAllSubjectsEndpoint(svc)),
			decodeListSubjectsRequest,
			encodeListSubjectsResponse,
		),
		countSubjects: kitgrpc.NewServer(
			(countSubjectsEndpoint(svc)),
			decodeCountSubjectsRequest,
			encodeCountSubjectsResponse,
		),
		listPermissions: kitgrpc.NewServer(
			(listPermissionsEndpoint(svc)),
			decodeListPermissionsRequest,
			encodeListPermissionsResponse,
		),
	}
}

func (s *grpcServer) Issue(ctx context.Context, req *magistrala.IssueReq) (*magistrala.Token, error) {
	_, res, err := s.issue.ServeGRPC(ctx, req)
	if err != nil {
		return nil, encodeError(err)
	}
	return res.(*magistrala.Token), nil
}

func (s *grpcServer) Refresh(ctx context.Context, req *magistrala.RefreshReq) (*magistrala.Token, error) {
	_, res, err := s.refresh.ServeGRPC(ctx, req)
	if err != nil {
		return nil, encodeError(err)
	}
	return res.(*magistrala.Token), nil
}

func (s *grpcServer) Identify(ctx context.Context, token *magistrala.IdentityReq) (*magistrala.IdentityRes, error) {
	_, res, err := s.identify.ServeGRPC(ctx, token)
	if err != nil {
		return nil, encodeError(err)
	}
	return res.(*magistrala.IdentityRes), nil
}

func (s *grpcServer) Authorize(ctx context.Context, req *magistrala.AuthorizeReq) (*magistrala.AuthorizeRes, error) {
	_, res, err := s.authorize.ServeGRPC(ctx, req)
	if err != nil {
		return nil, encodeError(err)
	}
	return res.(*magistrala.AuthorizeRes), nil
}

func (s *grpcServer) AddPolicy(ctx context.Context, req *magistrala.AddPolicyReq) (*magistrala.AddPolicyRes, error) {
	_, res, err := s.addPolicy.ServeGRPC(ctx, req)
	if err != nil {
		return nil, encodeError(err)
	}
	return res.(*magistrala.AddPolicyRes), nil
}

func (s *grpcServer) AddPolicies(ctx context.Context, req *magistrala.AddPoliciesReq) (*magistrala.AddPoliciesRes, error) {
	_, res, err := s.addPolicies.ServeGRPC(ctx, req)
	if err != nil {
		return nil, encodeError(err)
	}
	return res.(*magistrala.AddPoliciesRes), nil
}

func (s *grpcServer) DeletePolicy(ctx context.Context, req *magistrala.DeletePolicyReq) (*magistrala.DeletePolicyRes, error) {
	_, res, err := s.deletePolicy.ServeGRPC(ctx, req)
	if err != nil {
		return nil, encodeError(err)
	}
	return res.(*magistrala.DeletePolicyRes), nil
}

func (s *grpcServer) DeletePolicies(ctx context.Context, req *magistrala.DeletePoliciesReq) (*magistrala.DeletePoliciesRes, error) {
	_, res, err := s.deletePolicies.ServeGRPC(ctx, req)
	if err != nil {
		return nil, encodeError(err)
	}
	return res.(*magistrala.DeletePoliciesRes), nil
}

func (s *grpcServer) ListObjects(ctx context.Context, req *magistrala.ListObjectsReq) (*magistrala.ListObjectsRes, error) {
	_, res, err := s.listObjects.ServeGRPC(ctx, req)
	if err != nil {
		return nil, encodeError(err)
	}
	return res.(*magistrala.ListObjectsRes), nil
}

func (s *grpcServer) ListAllObjects(ctx context.Context, req *magistrala.ListObjectsReq) (*magistrala.ListObjectsRes, error) {
	_, res, err := s.listAllObjects.ServeGRPC(ctx, req)
	if err != nil {
		return nil, encodeError(err)
	}
	return res.(*magistrala.ListObjectsRes), nil
}

func (s *grpcServer) CountObjects(ctx context.Context, req *magistrala.CountObjectsReq) (*magistrala.CountObjectsRes, error) {
	_, res, err := s.countObjects.ServeGRPC(ctx, req)
	if err != nil {
		return nil, encodeError(err)
	}
	return res.(*magistrala.CountObjectsRes), nil
}

func (s *grpcServer) ListSubjects(ctx context.Context, req *magistrala.ListSubjectsReq) (*magistrala.ListSubjectsRes, error) {
	_, res, err := s.listSubjects.ServeGRPC(ctx, req)
	if err != nil {
		return nil, encodeError(err)
	}
	return res.(*magistrala.ListSubjectsRes), nil
}

func (s *grpcServer) ListAllSubjects(ctx context.Context, req *magistrala.ListSubjectsReq) (*magistrala.ListSubjectsRes, error) {
	_, res, err := s.listAllSubjects.ServeGRPC(ctx, req)
	if err != nil {
		return nil, encodeError(err)
	}
	return res.(*magistrala.ListSubjectsRes), nil
}

func (s *grpcServer) CountSubjects(ctx context.Context, req *magistrala.CountSubjectsReq) (*magistrala.CountSubjectsRes, error) {
	_, res, err := s.countSubjects.ServeGRPC(ctx, req)
	if err != nil {
		return nil, encodeError(err)
	}
	return res.(*magistrala.CountSubjectsRes), nil
}

func (s *grpcServer) ListPermissions(ctx context.Context, req *magistrala.ListPermissionsReq) (*magistrala.ListPermissionsRes, error) {
	_, res, err := s.listPermissions.ServeGRPC(ctx, req)
	if err != nil {
		return nil, encodeError(err)
	}
	return res.(*magistrala.ListPermissionsRes), nil
}

func decodeIssueRequest(_ context.Context, grpcReq interface{}) (interface{}, error) {
	req := grpcReq.(*magistrala.IssueReq)
	return issueReq{userID: req.GetUserId(), domainID: req.GetDomainId(), keyType: auth.KeyType(req.GetType())}, nil
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

func decodeAddPolicyRequest(_ context.Context, grpcReq interface{}) (interface{}, error) {
	req := grpcReq.(*magistrala.AddPolicyReq)
	return policyReq{
		Domain:      req.GetDomain(),
		SubjectType: req.GetSubjectType(),
		SubjectKind: req.GetSubjectKind(),
		Subject:     req.GetSubject(),
		Relation:    req.GetRelation(),
		Permission:  req.GetPermission(),
		ObjectType:  req.GetObjectType(),
		ObjectKind:  req.GetObjectKind(),
		Object:      req.GetObject(),
	}, nil
}

func encodeAddPolicyResponse(_ context.Context, grpcRes interface{}) (interface{}, error) {
	res := grpcRes.(addPolicyRes)
	return &magistrala.AddPolicyRes{Authorized: res.authorized}, nil
}

func decodeAddPoliciesRequest(_ context.Context, grpcReq interface{}) (interface{}, error) {
	reqs := grpcReq.(*magistrala.AddPoliciesReq)
	r := policiesReq{}
	for _, req := range reqs.AddPoliciesReq {
		r = append(r, policyReq{
			Domain:      req.GetDomain(),
			SubjectType: req.GetSubjectType(),
			SubjectKind: req.GetSubjectKind(),
			Subject:     req.GetSubject(),
			Relation:    req.GetRelation(),
			Permission:  req.GetPermission(),
			ObjectType:  req.GetObjectType(),
			ObjectKind:  req.GetObjectKind(),
			Object:      req.GetObject(),
		})
	}
	return r, nil
}

func encodeAddPoliciesResponse(_ context.Context, grpcRes interface{}) (interface{}, error) {
	res := grpcRes.(addPoliciesRes)
	return &magistrala.AddPoliciesRes{Authorized: res.authorized}, nil
}

func decodeDeletePolicyRequest(_ context.Context, grpcReq interface{}) (interface{}, error) {
	req := grpcReq.(*magistrala.DeletePolicyReq)
	return policyReq{
		Domain:      req.GetDomain(),
		SubjectType: req.GetSubjectType(),
		SubjectKind: req.GetSubjectKind(),
		Subject:     req.GetSubject(),
		Relation:    req.GetRelation(),
		Permission:  req.GetPermission(),
		ObjectType:  req.GetObjectType(),
		ObjectKind:  req.GetObjectKind(),
		Object:      req.GetObject(),
	}, nil
}

func encodeDeletePolicyResponse(_ context.Context, grpcRes interface{}) (interface{}, error) {
	res := grpcRes.(deletePolicyRes)
	return &magistrala.DeletePolicyRes{Deleted: res.deleted}, nil
}

func decodeDeletePoliciesRequest(_ context.Context, grpcReq interface{}) (interface{}, error) {
	reqs := grpcReq.(*magistrala.DeletePoliciesReq)
	r := policiesReq{}
	for _, req := range reqs.DeletePoliciesReq {
		r = append(r, policyReq{
			Domain:      req.GetDomain(),
			SubjectType: req.GetSubjectType(),
			SubjectKind: req.GetSubjectKind(),
			Subject:     req.GetSubject(),
			Relation:    req.GetRelation(),
			Permission:  req.GetPermission(),
			ObjectType:  req.GetObjectType(),
			ObjectKind:  req.GetObjectKind(),
			Object:      req.GetObject(),
		})
	}
	return r, nil
}

func encodeDeletePoliciesResponse(_ context.Context, grpcRes interface{}) (interface{}, error) {
	res := grpcRes.(deletePoliciesRes)
	return &magistrala.DeletePoliciesRes{Deleted: res.deleted}, nil
}

func decodeListObjectsRequest(_ context.Context, grpcReq interface{}) (interface{}, error) {
	req := grpcReq.(*magistrala.ListObjectsReq)
	return listObjectsReq{
		Domain:        req.GetDomain(),
		SubjectType:   req.GetSubjectType(),
		Subject:       req.GetSubject(),
		Relation:      req.GetRelation(),
		Permission:    req.GetPermission(),
		ObjectType:    req.GetObjectType(),
		Object:        req.GetObject(),
		NextPageToken: req.GetNextPageToken(),
		Limit:         req.GetLimit(),
	}, nil
}

func encodeListObjectsResponse(_ context.Context, grpcRes interface{}) (interface{}, error) {
	res := grpcRes.(listObjectsRes)
	return &magistrala.ListObjectsRes{Policies: res.policies}, nil
}

func decodeCountObjectsRequest(_ context.Context, grpcReq interface{}) (interface{}, error) {
	req := grpcReq.(*magistrala.CountObjectsReq)
	return countObjectsReq{
		Domain:      req.GetDomain(),
		SubjectType: req.GetSubjectType(),
		Subject:     req.GetSubject(),
		Relation:    req.GetRelation(),
		Permission:  req.GetPermission(),
		ObjectType:  req.GetObjectType(),
		Object:      req.GetObject(),
	}, nil
}

func encodeCountObjectsResponse(_ context.Context, grpcRes interface{}) (interface{}, error) {
	res := grpcRes.(countObjectsRes)
	return &magistrala.CountObjectsRes{Count: int64(res.count)}, nil
}

func decodeListSubjectsRequest(_ context.Context, grpcReq interface{}) (interface{}, error) {
	req := grpcReq.(*magistrala.ListSubjectsReq)
	return listSubjectsReq{
		Domain:      req.GetDomain(),
		SubjectType: req.GetSubjectType(),
		Subject:     req.GetSubject(),
		Relation:    req.GetRelation(),
		Permission:  req.GetPermission(),
		ObjectType:  req.GetObjectType(),
		Object:      req.GetObject(), NextPageToken: req.GetNextPageToken(), Limit: req.GetLimit(),
	}, nil
}

func encodeListSubjectsResponse(_ context.Context, grpcRes interface{}) (interface{}, error) {
	res := grpcRes.(listSubjectsRes)
	return &magistrala.ListSubjectsRes{Policies: res.policies}, nil
}

func decodeCountSubjectsRequest(_ context.Context, grpcReq interface{}) (interface{}, error) {
	req := grpcReq.(*magistrala.CountSubjectsReq)
	return countSubjectsReq{
		Domain:      req.GetDomain(),
		SubjectType: req.GetSubjectType(),
		Subject:     req.GetSubject(),
		Relation:    req.GetRelation(),
		Permission:  req.GetPermission(),
		ObjectType:  req.GetObjectType(),
		Object:      req.GetObject(),
	}, nil
}

func encodeCountSubjectsResponse(_ context.Context, grpcRes interface{}) (interface{}, error) {
	res := grpcRes.(countObjectsRes)
	return &magistrala.CountObjectsRes{Count: int64(res.count)}, nil
}

func decodeListPermissionsRequest(_ context.Context, grpcReq interface{}) (interface{}, error) {
	req := grpcReq.(*magistrala.ListPermissionsReq)
	var fp []string

	switch req.GetObjectType() {
	case auth.ThingType:
		fp = defThingsFilterPermissions
	case auth.GroupType:
		fp = defGroupsFilterPermissions
	case auth.PlatformType:
		fp = defPlatformFilterPermissions
	case auth.DomainType:
		fp = defDomainsFilterPermissions
	default:
		return nil, apiutil.ErrMalformedPolicy
	}
	if len(req.GetFilterPermissions()) > 0 {
		fp = req.GetFilterPermissions()
	}
	return listPermissionsReq{
		Domain:            req.GetDomain(),
		SubjectType:       req.GetSubjectType(),
		Subject:           req.GetSubject(),
		SubjectRelation:   req.GetSubjectRelation(),
		ObjectType:        req.GetObjectType(),
		Object:            req.GetObject(),
		FilterPermissions: fp,
	}, nil
}

func encodeListPermissionsResponse(_ context.Context, grpcRes interface{}) (interface{}, error) {
	res := grpcRes.(listPermissionsRes)
	return &magistrala.ListPermissionsRes{
		Domain:          res.Domain,
		SubjectType:     res.SubjectType,
		Subject:         res.Subject,
		SubjectRelation: res.SubjectRelation,
		ObjectType:      res.ObjectType,
		Object:          res.Object,
		Permissions:     res.Permissions,
	}, nil
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
	case errors.Contains(err, errors.ErrAuthentication),
		errors.Contains(err, svcerr.ErrAuthentication),
		errors.Contains(err, auth.ErrKeyExpired),
		err == apiutil.ErrMissingEmail,
		err == apiutil.ErrBearerToken:
		return status.Error(codes.Unauthenticated, err.Error())
	case errors.Contains(err, errors.ErrAuthorization),
		errors.Contains(err, svcerr.ErrAuthorization),
		errors.Contains(err, errors.ErrDomainAuthorization):
		return status.Error(codes.PermissionDenied, err.Error())
	case errors.Contains(err, errors.ErrNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Contains(err, errors.ErrConflict):
		return status.Error(codes.AlreadyExists, err.Error())
	default:
		return status.Error(codes.Internal, err.Error())
	}
}
