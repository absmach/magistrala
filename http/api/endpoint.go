// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"context"

	"github.com/go-kit/kit/endpoint"
	"github.com/mainflux/mainflux/http"
)

func sendMessageEndpoint(svc http.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(publishReq)

		if err := req.validate(); err != nil {
			return nil, err
		}

		err := svc.Publish(ctx, req.token, req.msg)
		return nil, err
	}
}
