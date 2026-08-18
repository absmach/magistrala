// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package grpc

import (
	"context"

	readers "github.com/absmach/magistrala/readers"
	"github.com/go-kit/kit/endpoint"
)

func readMessagesEndpoint(svc readers.MessageRepository) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req := request.(readMessagesReq)
		if err := req.validate(); err != nil {
			return readMessagesRes{}, err
		}

		page, err := svc.ReadAll(req.chanID, req.pageMeta)
		if err != nil {
			return readMessagesRes{}, err
		}

		return readMessagesRes{
			PageMetadata: page.PageMetadata,
			Total:        page.Total,
			Messages:     page.Messages,
		}, nil
	}
}

func listGatewayDevicesEndpoint(svc readers.MessageRepository) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req := request.(deviceViewReq)
		if err := req.validate(); err != nil {
			return deviceStatsRes{}, err
		}

		page, err := svc.ListGatewayDevices(req.chanID, req.filterVal, req.pageMeta)
		if err != nil {
			return deviceStatsRes{}, err
		}

		return deviceStatsRes{
			PageMetadata: page.PageMetadata,
			Total:        page.Total,
			Stats:        page.Stats,
		}, nil
	}
}

func listDeviceGatewaysEndpoint(svc readers.MessageRepository) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req := request.(deviceViewReq)
		if err := req.validate(); err != nil {
			return deviceStatsRes{}, err
		}

		page, err := svc.ListDeviceGateways(req.chanID, req.filterVal, req.pageMeta)
		if err != nil {
			return deviceStatsRes{}, err
		}

		return deviceStatsRes{
			PageMetadata: page.PageMetadata,
			Total:        page.Total,
			Stats:        page.Stats,
		}, nil
	}
}
