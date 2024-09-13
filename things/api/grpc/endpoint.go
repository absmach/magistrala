// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package grpc

import (
	"context"

	"github.com/absmach/magistrala"
	"github.com/absmach/magistrala/pkg/auth"
	"github.com/absmach/magistrala/pkg/errors"
	svcerr "github.com/absmach/magistrala/pkg/errors/service"
	"github.com/absmach/magistrala/pkg/policy"
	"github.com/absmach/magistrala/things"
	"github.com/go-kit/kit/endpoint"
)

func authorizeEndpoint(svc things.Service, authClient auth.AuthClient) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(*magistrala.AuthorizeReq)

		thingID, err := svc.Identify(ctx, req.GetSubject())
		if err != nil {
			return authorizeRes{}, err
		}
		r := &magistrala.AuthorizeReq{
			SubjectType: policy.GroupType,
			Subject:     req.GetObject(),
			ObjectType:  policy.ThingType,
			Object:      thingID,
			Permission:  req.GetPermission(),
		}
		resp, err := authClient.Authorize(ctx, r)
		if err != nil {
			return authorizeRes{}, errors.Wrap(svcerr.ErrAuthorization, err)
		}
		if !resp.GetAuthorized() {
			return authorizeRes{}, svcerr.ErrAuthorization
		}

		return authorizeRes{
			authorized: true,
			id:         thingID,
		}, err
	}
}
