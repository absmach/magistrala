// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package http

import (
	"context"

	"github.com/absmach/magistrala/pkg/apiutil"
	"github.com/absmach/magistrala/pkg/channels"
	"github.com/absmach/magistrala/pkg/errors"
	"github.com/go-kit/kit/endpoint"
)

func createChannelEndpoint(svc channels.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(createChannelReq)
		if err := req.validate(); err != nil {
			return nil, errors.Wrap(apiutil.ErrValidation, err)
		}

		channels, err := svc.CreateChannels(ctx, req.token, req.Channel)
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
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(createChannelsReq)
		if err := req.validate(); err != nil {
			return nil, errors.Wrap(apiutil.ErrValidation, err)
		}

		channels, err := svc.CreateChannels(ctx, req.token, req.Channels...)
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
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(viewChannelReq)
		if err := req.validate(); err != nil {
			return nil, errors.Wrap(apiutil.ErrValidation, err)
		}

		c, err := svc.ViewChannel(ctx, req.token, req.id)
		if err != nil {
			return nil, err
		}

		return viewChannelRes{Channel: c}, nil
	}
}

func listChannelsEndpoint(svc channels.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(listChannelsReq)
		if err := req.validate(); err != nil {
			return nil, errors.Wrap(apiutil.ErrValidation, err)
		}

		pm := channels.PageMetadata{
			Status:     req.status,
			Offset:     req.offset,
			Limit:      req.limit,
			Name:       req.name,
			Tag:        req.tag,
			Permission: req.permission,
			Metadata:   req.metadata,
			ListPerms:  req.listPerms,
			Id:         req.id,
		}
		page, err := svc.ListChannels(ctx, req.token, pm)
		if err != nil {
			return nil, err
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
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(updateChannelReq)
		if err := req.validate(); err != nil {
			return nil, errors.Wrap(apiutil.ErrValidation, err)
		}

		ch := channels.Channel{
			ID:       req.id,
			Name:     req.Name,
			Metadata: req.Metadata,
		}
		ch, err := svc.UpdateChannel(ctx, req.token, ch)
		if err != nil {
			return nil, err
		}

		return updateChannelRes{Channel: ch}, nil
	}
}

func updateChannelTagsEndpoint(svc channels.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(updateChannelTagsReq)
		if err := req.validate(); err != nil {
			return nil, errors.Wrap(apiutil.ErrValidation, err)
		}

		ch := channels.Channel{
			ID:   req.id,
			Tags: req.Tags,
		}
		ch, err := svc.UpdateChannelTags(ctx, req.token, ch)
		if err != nil {
			return nil, err
		}

		return updateChannelRes{Channel: ch}, nil
	}
}

func enableChannelEndpoint(svc channels.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(changeChannelStatusReq)
		if err := req.validate(); err != nil {
			return nil, errors.Wrap(apiutil.ErrValidation, err)
		}

		ch, err := svc.EnableChannel(ctx, req.token, req.id)
		if err != nil {
			return nil, err
		}

		return changeChannelStatusRes{Channel: ch}, nil
	}
}

func disableChannelEndpoint(svc channels.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(changeChannelStatusReq)
		if err := req.validate(); err != nil {
			return nil, errors.Wrap(apiutil.ErrValidation, err)
		}

		ch, err := svc.DisableChannel(ctx, req.token, req.id)
		if err != nil {
			return nil, err
		}

		return changeChannelStatusRes{Channel: ch}, nil
	}
}

func connectChannelThingEndpoint(svc channels.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(connectChannelThingRequest)
		if err := req.validate(); err != nil {
			return nil, errors.Wrap(apiutil.ErrValidation, err)
		}

		return connectChannelThingRes{}, nil
	}
}

func disconnectChannelThingEndpoint(svc channels.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(disconnectChannelThingRequest)
		if err := req.validate(); err != nil {
			return nil, errors.Wrap(apiutil.ErrValidation, err)
		}

		return disconnectChannelThingRes{}, nil
	}
}

func connectEndpoint(svc channels.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(connectChannelThingRequest)
		if err := req.validate(); err != nil {
			return nil, errors.Wrap(apiutil.ErrValidation, err)
		}

		return connectChannelThingRes{}, nil
	}
}

func disconnectEndpoint(svc channels.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(disconnectChannelThingRequest)
		if err := req.validate(); err != nil {
			return nil, errors.Wrap(apiutil.ErrValidation, err)
		}

		return disconnectChannelThingRes{}, nil
	}
}

func deleteChannelEndpoint(svc channels.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(deleteChannelReq)
		if err := req.validate(); err != nil {
			return nil, errors.Wrap(apiutil.ErrValidation, err)
		}

		if err := svc.RemoveChannel(ctx, req.token, req.id); err != nil {
			return nil, err
		}

		return deleteChannelRes{}, nil
	}
}
