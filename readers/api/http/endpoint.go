// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package http

import (
	"context"

	grpcChannelsV1 "github.com/absmach/magistrala/api/grpc/channels/v1"
	grpcClientsV1 "github.com/absmach/magistrala/api/grpc/clients/v1"
	apiutil "github.com/absmach/magistrala/api/http/util"
	smqauthn "github.com/absmach/magistrala/pkg/authn"
	"github.com/absmach/magistrala/pkg/errors"
	svcerr "github.com/absmach/magistrala/pkg/errors/service"
	"github.com/absmach/magistrala/readers"
	"github.com/go-kit/kit/endpoint"
)

func listMessagesEndpoint(svc readers.MessageRepository, authn smqauthn.Authentication, clients grpcClientsV1.ClientsServiceClient, channels grpcChannelsV1.ChannelsServiceClient, readAuthz *readAuthorizer) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req := request.(listMessagesReq)
		if err := req.validate(); err != nil {
			return nil, errors.Wrap(apiutil.ErrValidation, err)
		}

		pageMeta, noAccess, err := authnAuthz(ctx, req, authn, clients, channels, readAuthz)
		if err != nil {
			return nil, errors.Wrap(svcerr.ErrAuthorization, err)
		}
		if noAccess {
			return pageRes{
				PageMetadata: pageMeta,
				Messages:     []readers.Message{},
			}, nil
		}

		page, err := svc.ReadAll(req.chanID, pageMeta)
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

// listGatewayDevicesEndpoint lists the devices observed publishing through a
// gateway (MG-15). The gateway's publisher id named in the URL is the source
// of the roster, not a filter the caller must hold: a gateway can relay for
// devices belonging to more than one customer, so the caller's own DeviceScope
// narrows the returned rows to their devices (see ListGatewayDevices in
// readers/messages.go). noAccess — the caller holds no per-device grant at all
// — is reported as an empty page, matching how a filtered-out publisher or
// device_id behaves on a message read, not as an error.
func listGatewayDevicesEndpoint(svc readers.MessageRepository, authn smqauthn.Authentication, clients grpcClientsV1.ClientsServiceClient, channels grpcChannelsV1.ChannelsServiceClient, readAuthz *readAuthorizer) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req := request.(deviceViewReq)
		if err := req.validate(); err != nil {
			return nil, errors.Wrap(apiutil.ErrValidation, err)
		}

		scope, noAccess, err := authzDeviceView(ctx, req.chanID, req.domain, req.token, req.key, authn, clients, channels, readAuthz, nil, nil)
		if err != nil {
			return nil, errors.Wrap(svcerr.ErrAuthorization, err)
		}
		if noAccess {
			return gatewayDevicesRes{Offset: req.pageMeta.Offset, Limit: req.pageMeta.Limit, Devices: []gatewayDeviceRes{}}, nil
		}

		pm := req.pageMeta
		pm.DeviceScope = scope

		page, err := svc.ListGatewayDevices(req.chanID, req.filterVal, pm)
		if err != nil {
			return nil, err
		}

		devices := make([]gatewayDeviceRes, len(page.Stats))
		for i, s := range page.Stats {
			devices[i] = gatewayDeviceRes{DeviceID: s.ID, LastSeen: s.LastSeen, MessageCount: s.MessageCount}
		}

		return gatewayDevicesRes{
			Total:   page.Total,
			Offset:  page.PageMetadata.Offset,
			Limit:   page.PageMetadata.Limit,
			Devices: devices,
		}, nil
	}
}

// listDeviceGatewaysEndpoint lists the gateways observed relaying for a
// device (MG-15). Unlike listGatewayDevicesEndpoint, the resolved scope is
// not applied to the query: deviceID is itself the authorization boundary
// here (see the DeviceScope comment on readers.MessageRepository's
// ListDeviceGateways), so once it has passed authzDeviceView every row the
// query can produce already belongs to that one authorized device.
func listDeviceGatewaysEndpoint(svc readers.MessageRepository, authn smqauthn.Authentication, clients grpcClientsV1.ClientsServiceClient, channels grpcChannelsV1.ChannelsServiceClient, readAuthz *readAuthorizer) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req := request.(deviceViewReq)
		if err := req.validate(); err != nil {
			return nil, errors.Wrap(apiutil.ErrValidation, err)
		}

		_, noAccess, err := authzDeviceView(ctx, req.chanID, req.domain, req.token, req.key, authn, clients, channels, readAuthz, nil, []string{req.filterVal})
		if err != nil {
			return nil, errors.Wrap(svcerr.ErrAuthorization, err)
		}
		if noAccess {
			return deviceGatewaysRes{Offset: req.pageMeta.Offset, Limit: req.pageMeta.Limit, Publishers: []deviceGatewayRes{}}, nil
		}

		page, err := svc.ListDeviceGateways(req.chanID, req.filterVal, req.pageMeta)
		if err != nil {
			return nil, err
		}

		publishers := make([]deviceGatewayRes, len(page.Stats))
		for i, s := range page.Stats {
			publishers[i] = deviceGatewayRes{Publisher: s.ID, LastSeen: s.LastSeen, MessageCount: s.MessageCount}
		}

		return deviceGatewaysRes{
			Total:      page.Total,
			Offset:     page.PageMetadata.Offset,
			Limit:      page.PageMetadata.Limit,
			Publishers: publishers,
		}, nil
	}
}
