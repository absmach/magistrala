// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package grpc

import (
	"context"
	"log"

	readers "github.com/absmach/supermq/readers"
	"github.com/go-kit/kit/endpoint"
)

func readMessagesEndpoint(svc readers.MessageRepository) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(readMessagesReq)
		if err := req.validate(); err != nil {
			return readMessagesRes{}, err
		}

		log.Printf("am here in grpc an req is %+v\n", req)

		page, err := svc.ReadAll(req.chanID, req.pageMeta)
		if err != nil {
			return readMessagesRes{}, err
		}
		log.Printf("page is %+v\n", page)


		return readMessagesRes{
			PageMetadata: page.PageMetadata,
			Total:        page.Total,
			Messages:     page.Messages,
		}, nil
	}
}
