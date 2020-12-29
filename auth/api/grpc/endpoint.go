// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package grpc

import (
	"context"
	"time"

	"github.com/go-kit/kit/endpoint"
	"github.com/mainflux/mainflux/auth"
)

func issueEndpoint(svc auth.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(issueReq)
		if err := req.validate(); err != nil {
			return issueRes{}, err
		}

		key := auth.Key{
			Type:     req.keyType,
			Subject:  req.email,
			IssuerID: req.id,
			IssuedAt: time.Now().UTC(),
		}

		_, secret, err := svc.Issue(ctx, "", key)
		if err != nil {
			return issueRes{}, err
		}

		return issueRes{secret}, nil
	}
}

func identifyEndpoint(svc auth.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(identityReq)
		if err := req.validate(); err != nil {
			return identityRes{}, err
		}

		id, err := svc.Identify(ctx, req.token)
		if err != nil {
			return identityRes{}, err
		}

		ret := identityRes{
			id:    id.ID,
			email: id.Email,
		}
		return ret, nil
	}
}

func authorizeEndpoint(svc auth.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(authReq)

		if err := req.validate(); err != nil {
			return authorizeRes{}, err
		}

		_, err := svc.Identify(ctx, req.token)
		if err != nil {
			return authorizeRes{}, err
		}

		// TODO implement authorization
		authorized, err := svc.Authorize(ctx, req.token, req.Sub, req.Obj, req.Obj)
		if err != nil {
			return authorizeRes{}, err
		}

		return authorizeRes{authorized: authorized}, err
	}
}

func assignEndpoint(svc auth.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(assignReq)

		if err := req.validate(); err != nil {
			return emptyRes{}, err
		}

		_, err := svc.Identify(ctx, req.token)
		if err != nil {
			return emptyRes{}, err
		}

		err = svc.Assign(ctx, req.token, req.memberID, req.groupID)
		if err != nil {
			return emptyRes{}, err
		}
		return emptyRes{}, nil

	}
}

func membersEndpoint(svc auth.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(membersReq)

		if err := req.validate(); err != nil {
			return membersRes{}, err
		}

		_, err := svc.Identify(ctx, req.token)
		if err != nil {
			return membersRes{}, err
		}

		mp, err := svc.ListMembers(ctx, req.token, req.groupID, req.offset, req.limit, nil)
		if err != nil {
			return membersRes{}, err
		}
		memberIDs := []string{}
		for _, m := range mp.Members {
			memberIDs = append(memberIDs, m.(string))
		}
		return membersRes{
			offset:  req.offset,
			limit:   req.limit,
			total:   mp.PageMetadata.Total,
			members: memberIDs,
		}, nil

	}
}
