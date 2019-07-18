//
// Copyright (c) 2019
// Mainflux
//
// SPDX-License-Identifier: Apache-2.0
//

package api

import (
	"context"

	"github.com/go-kit/kit/endpoint"
	"github.com/mainflux/mainflux"
)

func sendMessageEndpoint(svc mainflux.MessagePublisher) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(publishReq)
		err := svc.Publish(ctx, req.token, req.msg)
		return nil, err
	}
}
