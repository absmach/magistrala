//
// Copyright (c) 2018
// Mainflux
//
// SPDX-License-Identifier: Apache-2.0
//

package api

import (
	"context"

	"github.com/go-kit/kit/endpoint"
	"github.com/mainflux/mainflux/readers"
)

func listMessagesEndpoint(svc readers.MessageRepository) endpoint.Endpoint {
	return func(_ context.Context, request interface{}) (interface{}, error) {
		req := request.(listMessagesReq)

		if err := req.validate(); err != nil {
			return nil, err
		}

		page := svc.ReadAll(req.chanID, req.offset, req.limit, req.query)

		return pageRes{
			Total:    page.Total,
			Offset:   page.Offset,
			Limit:    page.Limit,
			Messages: page.Messages,
		}, nil
	}
}
