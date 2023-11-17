// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"context"

	"github.com/absmach/magistrala"
	"github.com/absmach/magistrala/internal/apiutil"
	"github.com/absmach/magistrala/pkg/errors"
	"github.com/absmach/magistrala/readers"
	"github.com/go-kit/kit/endpoint"
)

func listMessagesEndpoint(svc readers.MessageRepository, uauth magistrala.AuthServiceClient, taauth magistrala.AuthzServiceClient) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(listMessagesReq)
		if err := req.validate(); err != nil {
			return nil, errors.Wrap(apiutil.ErrValidation, err)
		}

		if err := authorize(ctx, req, uauth, taauth); err != nil {
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
