// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package auth

import (
	"context"
	"fmt"
	"time"

	"github.com/absmach/magistrala"
	mgauth "github.com/absmach/magistrala/auth"
	"github.com/absmach/magistrala/pkg/auth"
	"github.com/absmach/magistrala/pkg/errors"
	svcerr "github.com/absmach/magistrala/pkg/errors/service"
	"github.com/go-kit/kit/endpoint"
	kitgrpc "github.com/go-kit/kit/transport/grpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	authzSvcName = "magistrala.AuthzService"
	authnSvcName = "magistrala.AuthnService"
)

type authGrpcClient struct {
	issue     endpoint.Endpoint
	refresh   endpoint.Endpoint
	identify  endpoint.Endpoint
	authorize endpoint.Endpoint
	timeout   time.Duration
}

// NewAuthClient returns new auth gRPC client instance.
func NewAuthClient(conn *grpc.ClientConn, timeout time.Duration) auth.AuthClient {
	return &authGrpcClient{
		issue: kitgrpc.NewClient(
			conn,
			authnSvcName,
			"Issue",
			encodeIssueRequest,
			decodeIssueResponse,
			magistrala.Token{},
		).Endpoint(),
		refresh: kitgrpc.NewClient(
			conn,
			authnSvcName,
			"Refresh",
			encodeRefreshRequest,
			decodeRefreshResponse,
			magistrala.Token{},
		).Endpoint(),
		identify: kitgrpc.NewClient(
			conn,
			authnSvcName,
			"Identify",
			encodeIdentifyRequest,
			decodeIdentifyResponse,
			magistrala.IdentityRes{},
		).Endpoint(),
		authorize: kitgrpc.NewClient(
			conn,
			authzSvcName,
			"Authorize",
			encodeAuthorizeRequest,
			decodeAuthorizeResponse,
			magistrala.AuthorizeRes{},
		).Endpoint(),
		timeout: timeout,
	}
}

func (client authGrpcClient) Issue(ctx context.Context, req *magistrala.IssueReq, _ ...grpc.CallOption) (*magistrala.Token, error) {
	ctx, cancel := context.WithTimeout(ctx, client.timeout)
	defer cancel()

	res, err := client.issue(ctx, issueReq{
		userID:   req.GetUserId(),
		domainID: req.GetDomainId(),
		keyType:  mgauth.KeyType(req.GetType()),
	})
	if err != nil {
		return &magistrala.Token{}, decodeError(err)
	}
	return res.(*magistrala.Token), nil
}

func encodeIssueRequest(_ context.Context, grpcReq interface{}) (interface{}, error) {
	req := grpcReq.(issueReq)
	return &magistrala.IssueReq{
		UserId:   req.userID,
		DomainId: &req.domainID,
		Type:     uint32(req.keyType),
	}, nil
}

func decodeIssueResponse(_ context.Context, grpcRes interface{}) (interface{}, error) {
	return grpcRes, nil
}

func (client authGrpcClient) Refresh(ctx context.Context, req *magistrala.RefreshReq, _ ...grpc.CallOption) (*magistrala.Token, error) {
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

func (client authGrpcClient) Identify(ctx context.Context, token *magistrala.IdentityReq, _ ...grpc.CallOption) (*magistrala.IdentityRes, error) {
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

func (client authGrpcClient) Authorize(ctx context.Context, req *magistrala.AuthorizeReq, _ ...grpc.CallOption) (r *magistrala.AuthorizeRes, err error) {
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

func decodeError(err error) error {
	if st, ok := status.FromError(err); ok {
		switch st.Code() {
		case codes.NotFound:
			return errors.Wrap(svcerr.ErrNotFound, errors.New(st.Message()))
		case codes.InvalidArgument:
			return errors.Wrap(errors.ErrMalformedEntity, errors.New(st.Message()))
		case codes.AlreadyExists:
			return errors.Wrap(svcerr.ErrConflict, errors.New(st.Message()))
		case codes.Unauthenticated:
			return errors.Wrap(svcerr.ErrAuthentication, errors.New(st.Message()))
		case codes.OK:
			if msg := st.Message(); msg != "" {
				return errors.Wrap(errors.ErrUnidentified, errors.New(msg))
			}
			return nil
		case codes.FailedPrecondition:
			return errors.Wrap(errors.ErrMalformedEntity, errors.New(st.Message()))
		case codes.PermissionDenied:
			return errors.Wrap(svcerr.ErrAuthorization, errors.New(st.Message()))
		default:
			return errors.Wrap(fmt.Errorf("unexpected gRPC status: %s (status code:%v)", st.Code().String(), st.Code()), errors.New(st.Message()))
		}
	}
	return err
}
