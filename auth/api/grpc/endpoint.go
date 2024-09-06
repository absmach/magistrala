// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package grpc

import (
	"context"

	"github.com/absmach/magistrala/auth"
	"github.com/go-kit/kit/endpoint"
)

func issueEndpoint(svc auth.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(issueReq)
		if err := req.validate(); err != nil {
			return issueRes{}, err
		}

		key := auth.Key{
			Type:   req.keyType,
			User:   req.userID,
			Domain: req.domainID,
		}
		tkn, err := svc.Issue(ctx, "", key)
		if err != nil {
			return issueRes{}, err
		}
		ret := issueRes{
			accessToken:  tkn.AccessToken,
			refreshToken: tkn.RefreshToken,
			accessType:   tkn.AccessType,
		}
		return ret, nil
	}
}

func refreshEndpoint(svc auth.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(refreshReq)
		if err := req.validate(); err != nil {
			return issueRes{}, err
		}

		key := auth.Key{Domain: req.domainID, Type: auth.RefreshKey}
		tkn, err := svc.Issue(ctx, req.refreshToken, key)
		if err != nil {
			return issueRes{}, err
		}
		ret := issueRes{
			accessToken:  tkn.AccessToken,
			refreshToken: tkn.RefreshToken,
			accessType:   tkn.AccessType,
		}
		return ret, nil
	}
}

func identifyEndpoint(svc auth.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(identityReq)
		if err := req.validate(); err != nil {
			return identityRes{}, err
		}

		key, err := svc.Identify(ctx, req.token)
		if err != nil {
			return identityRes{}, err
		}

		return identityRes{id: key.Subject, userID: key.User, domainID: key.Domain}, nil
	}
}

func authorizeEndpoint(svc auth.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(authReq)

		if err := req.validate(); err != nil {
			return authorizeRes{}, err
		}
		err := svc.Authorize(ctx, auth.PolicyReq{
			Domain:      req.Domain,
			SubjectType: req.SubjectType,
			SubjectKind: req.SubjectKind,
			Subject:     req.Subject,
			Relation:    req.Relation,
			Permission:  req.Permission,
			ObjectType:  req.ObjectType,
			Object:      req.Object,
		})
		if err != nil {
			return authorizeRes{authorized: false}, err
		}
		return authorizeRes{authorized: true}, nil
	}
}

func deleteUserPoliciesEndpoint(svc auth.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(deleteUserPoliciesReq)
		if err := req.validate(); err != nil {
			return deletePolicyRes{}, err
		}

		if err := svc.DeleteUserPolicies(ctx, req.ID); err != nil {
			return deletePolicyRes{}, err
		}

		return deletePolicyRes{deleted: true}, nil
	}
}
