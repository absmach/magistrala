// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package http

import (
	"context"

	apiutil "github.com/absmach/magistrala/api/http/util"
	"github.com/absmach/magistrala/channels"
	"github.com/absmach/magistrala/pkg/authn"
	"github.com/absmach/magistrala/pkg/errors"
	svcerr "github.com/absmach/magistrala/pkg/errors/service"
	"github.com/go-kit/kit/endpoint"
)

func createChannelEndpoint(svc channels.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req := request.(createChannelReq)
		if err := req.validate(); err != nil {
			return nil, errors.Wrap(apiutil.ErrValidation, err)
		}

		session, ok := ctx.Value(authn.SessionKey).(authn.Session)
		if !ok {
			return nil, svcerr.ErrAuthentication
		}

		channels, _, err := svc.CreateChannels(ctx, session, req.Channel)
		if err != nil {
			return nil, err
		}

		return createChannelRes{
			Channel: channels[0],
			created: true,
		}, nil
	}
}

func createChannelsEndpoint(svc channels.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req := request.(createChannelsReq)
		if err := req.validate(); err != nil {
			return nil, errors.Wrap(apiutil.ErrValidation, err)
		}

		session, ok := ctx.Value(authn.SessionKey).(authn.Session)
		if !ok {
			return nil, svcerr.ErrAuthentication
		}

		channels, _, err := svc.CreateChannels(ctx, session, req.Channels...)
		if err != nil {
			return nil, err
		}

		res := channelsPageRes{
			pageRes: pageRes{
				Total: uint64(len(channels)),
			},
			Channels: []viewChannelRes{},
		}
		for _, c := range channels {
			res.Channels = append(res.Channels, viewChannelRes{Channel: c})
		}

		return res, nil
	}
}

func viewChannelEndpoint(svc channels.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req := request.(viewChannelReq)
		if err := req.validate(); err != nil {
			return nil, errors.Wrap(apiutil.ErrValidation, err)
		}

		session, ok := ctx.Value(authn.SessionKey).(authn.Session)
		if !ok {
			return nil, svcerr.ErrAuthentication
		}

		c, err := svc.ViewChannel(ctx, session, req.id, req.roles)
		if err != nil {
			return nil, err
		}

		return viewChannelRes{Channel: c}, nil
	}
}

func listChannelsEndpoint(svc channels.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req := request.(listChannelsReq)
		if err := req.validate(); err != nil {
			return nil, errors.Wrap(apiutil.ErrValidation, err)
		}

		session, ok := ctx.Value(authn.SessionKey).(authn.Session)
		if !ok {
			return nil, svcerr.ErrAuthentication
		}

		var page channels.ChannelsPage
		var err error
		switch req.userID != "" {
		case true:
			page, err = svc.ListUserChannels(ctx, session, req.userID, req.Page)
		default:
			page, err = svc.ListChannels(ctx, session, req.Page)
		}
		if err != nil {
			return channelsPageRes{}, err
		}

		res := channelsPageRes{
			pageRes: pageRes{
				Total:  page.Total,
				Offset: page.Offset,
				Limit:  page.Limit,
			},
			Channels: []viewChannelRes{},
		}
		for _, c := range page.Channels {
			res.Channels = append(res.Channels, viewChannelRes{Channel: c})
		}

		return res, nil
	}
}

func updateChannelEndpoint(svc channels.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req := request.(updateChannelReq)
		if err := req.validate(); err != nil {
			return nil, errors.Wrap(apiutil.ErrValidation, err)
		}

		session, ok := ctx.Value(authn.SessionKey).(authn.Session)
		if !ok {
			return nil, svcerr.ErrAuthentication
		}

		ch := channels.Channel{
			ID:       req.id,
			Name:     req.Name,
			Metadata: req.Metadata,
		}
		ch, err := svc.UpdateChannel(ctx, session, ch)
		if err != nil {
			return nil, err
		}

		return updateChannelRes{Channel: ch}, nil
	}
}

func updateChannelTagsEndpoint(svc channels.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req := request.(updateChannelTagsReq)
		if err := req.validate(); err != nil {
			return nil, errors.Wrap(apiutil.ErrValidation, err)
		}

		session, ok := ctx.Value(authn.SessionKey).(authn.Session)
		if !ok {
			return nil, svcerr.ErrAuthentication
		}

		ch := channels.Channel{
			ID:   req.id,
			Tags: req.Tags,
		}
		ch, err := svc.UpdateChannelTags(ctx, session, ch)
		if err != nil {
			return nil, err
		}

		return updateChannelRes{Channel: ch}, nil
	}
}

func setChannelParentGroupEndpoint(svc channels.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req := request.(setChannelParentGroupReq)
		if err := req.validate(); err != nil {
			return nil, errors.Wrap(apiutil.ErrValidation, err)
		}

		session, ok := ctx.Value(authn.SessionKey).(authn.Session)
		if !ok {
			return nil, svcerr.ErrAuthentication
		}

		if err := svc.SetParentGroup(ctx, session, req.ParentGroupID, req.id); err != nil {
			return nil, err
		}

		return setChannelParentGroupRes{}, nil
	}
}

func removeChannelParentGroupEndpoint(svc channels.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req := request.(removeChannelParentGroupReq)
		if err := req.validate(); err != nil {
			return nil, errors.Wrap(apiutil.ErrValidation, err)
		}

		session, ok := ctx.Value(authn.SessionKey).(authn.Session)
		if !ok {
			return nil, svcerr.ErrAuthentication
		}

		if err := svc.RemoveParentGroup(ctx, session, req.id); err != nil {
			return nil, err
		}

		return removeChannelParentGroupRes{}, nil
	}
}

func enableChannelEndpoint(svc channels.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req := request.(changeChannelStatusReq)
		if err := req.validate(); err != nil {
			return nil, errors.Wrap(apiutil.ErrValidation, err)
		}

		session, ok := ctx.Value(authn.SessionKey).(authn.Session)
		if !ok {
			return nil, svcerr.ErrAuthentication
		}

		ch, err := svc.EnableChannel(ctx, session, req.id)
		if err != nil {
			return nil, err
		}

		return changeChannelStatusRes{Channel: ch}, nil
	}
}

func disableChannelEndpoint(svc channels.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req := request.(changeChannelStatusReq)
		if err := req.validate(); err != nil {
			return nil, errors.Wrap(apiutil.ErrValidation, err)
		}

		session, ok := ctx.Value(authn.SessionKey).(authn.Session)
		if !ok {
			return nil, svcerr.ErrAuthentication
		}

		ch, err := svc.DisableChannel(ctx, session, req.id)
		if err != nil {
			return nil, err
		}

		return changeChannelStatusRes{Channel: ch}, nil
	}
}

func connectChannelClientEndpoint(svc channels.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req := request.(connectChannelClientsRequest)
		if err := req.validate(); err != nil {
			return nil, errors.Wrap(apiutil.ErrValidation, err)
		}

		session, ok := ctx.Value(authn.SessionKey).(authn.Session)
		if !ok {
			return nil, svcerr.ErrAuthentication
		}

		if err := svc.Connect(ctx, session, []string{req.channelID}, req.ClientIDs, req.Types); err != nil {
			return nil, err
		}

		return connectChannelClientsRes{}, nil
	}
}

func disconnectChannelClientsEndpoint(svc channels.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req := request.(disconnectChannelClientsRequest)
		if err := req.validate(); err != nil {
			return nil, errors.Wrap(apiutil.ErrValidation, err)
		}

		session, ok := ctx.Value(authn.SessionKey).(authn.Session)
		if !ok {
			return nil, svcerr.ErrAuthentication
		}

		if err := svc.Disconnect(ctx, session, []string{req.channelID}, req.ClientIds, req.Types); err != nil {
			return nil, err
		}

		return disconnectChannelClientsRes{}, nil
	}
}

func connectEndpoint(svc channels.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req := request.(connectRequest)
		if err := req.validate(); err != nil {
			return nil, errors.Wrap(apiutil.ErrValidation, err)
		}

		session, ok := ctx.Value(authn.SessionKey).(authn.Session)
		if !ok {
			return nil, svcerr.ErrAuthentication
		}

		if err := svc.Connect(ctx, session, req.ChannelIds, req.ClientIds, req.Types); err != nil {
			return nil, err
		}

		return connectRes{}, nil
	}
}

func disconnectEndpoint(svc channels.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req := request.(disconnectRequest)
		if err := req.validate(); err != nil {
			return nil, errors.Wrap(apiutil.ErrValidation, err)
		}

		session, ok := ctx.Value(authn.SessionKey).(authn.Session)
		if !ok {
			return nil, svcerr.ErrAuthentication
		}

		if err := svc.Disconnect(ctx, session, req.ChannelIds, req.ClientIds, req.Types); err != nil {
			return nil, err
		}

		return disconnectRes{}, nil
	}
}

func deleteChannelEndpoint(svc channels.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req := request.(deleteChannelReq)
		if err := req.validate(); err != nil {
			return nil, errors.Wrap(apiutil.ErrValidation, err)
		}

		session, ok := ctx.Value(authn.SessionKey).(authn.Session)
		if !ok {
			return nil, svcerr.ErrAuthentication
		}

		if err := svc.RemoveChannel(ctx, session, req.id); err != nil {
			return nil, err
		}

		return deleteChannelRes{}, nil
	}
}
