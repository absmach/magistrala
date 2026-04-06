// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package grpc

import (
	"context"

	pusers "github.com/absmach/magistrala/users/private"
	"github.com/go-kit/kit/endpoint"
)

func retrieveUsersEndpoint(svc pusers.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req := request.(retrieveUsersReq)

		if err := req.validate(); err != nil {
			return retrieveUsersRes{}, err
		}

		page, err := svc.RetrieveByIDs(ctx, req.ids, req.offset, req.limit)
		if err != nil {
			return retrieveUsersRes{}, err
		}

		return retrieveUsersRes{
			users:  page.Users,
			total:  page.Total,
			limit:  page.Limit,
			offset: page.Offset,
		}, nil
	}
}
