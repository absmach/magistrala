// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"context"
	"fmt"

	"github.com/go-kit/kit/endpoint"
	"github.com/mainflux/mainflux/opcua"
)

func browseEndpoint(svc opcua.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(browseReq)

		if err := req.validate(); err != nil {
			return nil, err
		}

		nodeID := fmt.Sprintf("ns=%s;%s", req.Namespace, req.Identifier)
		nodes, err := svc.Browse(req.ServerURI, nodeID)
		if err != nil {
			return nil, err
		}

		res := browseRes{
			Nodes: nodes,
		}

		return res, nil
	}
}
