// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"context"

	"github.com/go-kit/kit/endpoint"
	"github.com/mainflux/mainflux/pkg/errors"
	"github.com/mainflux/mainflux/readers"
	tpolicies "github.com/mainflux/mainflux/things/policies"
	upolicies "github.com/mainflux/mainflux/users/policies"
)

func listMessagesEndpoint(svc readers.MessageRepository, tc tpolicies.ThingsServiceClient, ac upolicies.AuthServiceClient) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(listMessagesReq)

		if err := req.validate(); err != nil {
			return nil, err
		}
		if err := authorize(ctx, req, tc, ac); err != nil {
			return nil, errors.Wrap(errors.ErrAuthorization, err)
		}
		page, err := svc.ReadAll(req.chanID, req.pageMeta)
		if err != nil {
			return nil, err
		}

		return pageRes{
			PageMetadata: page.PageMetadata,
			Total:        page.Total,
			Messages:     page.Messages,
		}, nil
	}
}
