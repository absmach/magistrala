// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"context"

	"github.com/go-kit/kit/endpoint"
	"github.com/mainflux/mainflux"
	"github.com/mainflux/mainflux/pkg/errors"
	"github.com/mainflux/mainflux/readers"
)

func listMessagesEndpoint(svc readers.MessageRepository, tc mainflux.ThingsServiceClient, ac mainflux.AuthServiceClient) endpoint.Endpoint {
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
