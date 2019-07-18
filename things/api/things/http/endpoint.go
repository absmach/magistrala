//
// Copyright (c) 2019
// Mainflux
//
// SPDX-License-Identifier: Apache-2.0
//

package http

import (
	"context"

	"github.com/go-kit/kit/endpoint"
	"github.com/mainflux/mainflux/things"
)

func addThingEndpoint(svc things.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(addThingReq)

		if err := req.validate(); err != nil {
			return nil, err
		}

		thing := things.Thing{
			Key:      req.Key,
			Name:     req.Name,
			Metadata: req.Metadata,
		}
		saved, err := svc.AddThing(ctx, req.token, thing)
		if err != nil {
			return nil, err
		}

		res := thingRes{
			id:      saved.ID,
			created: true,
		}
		return res, nil
	}
}

func updateThingEndpoint(svc things.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(updateThingReq)

		if err := req.validate(); err != nil {
			return nil, err
		}

		thing := things.Thing{
			ID:       req.id,
			Name:     req.Name,
			Metadata: req.Metadata,
		}

		if err := svc.UpdateThing(ctx, req.token, thing); err != nil {
			return nil, err
		}

		res := thingRes{id: req.id, created: false}
		return res, nil
	}
}

func updateKeyEndpoint(svc things.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(updateKeyReq)

		if err := req.validate(); err != nil {
			return nil, err
		}

		if err := svc.UpdateKey(ctx, req.token, req.id, req.Key); err != nil {
			return nil, err
		}

		res := thingRes{id: req.id, created: false}
		return res, nil
	}
}

func viewThingEndpoint(svc things.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(viewResourceReq)

		if err := req.validate(); err != nil {
			return nil, err
		}

		thing, err := svc.ViewThing(ctx, req.token, req.id)
		if err != nil {
			return nil, err
		}

		res := viewThingRes{
			ID:       thing.ID,
			Owner:    thing.Owner,
			Name:     thing.Name,
			Key:      thing.Key,
			Metadata: thing.Metadata,
		}
		return res, nil
	}
}

func listThingsEndpoint(svc things.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(listResourcesReq)

		if err := req.validate(); err != nil {
			return nil, err
		}

		page, err := svc.ListThings(ctx, req.token, req.offset, req.limit, req.name)
		if err != nil {
			return nil, err
		}

		res := thingsPageRes{
			pageRes: pageRes{
				Total:  page.Total,
				Offset: page.Offset,
				Limit:  page.Limit,
			},
			Things: []viewThingRes{},
		}
		for _, thing := range page.Things {
			view := viewThingRes{
				ID:       thing.ID,
				Owner:    thing.Owner,
				Name:     thing.Name,
				Key:      thing.Key,
				Metadata: thing.Metadata,
			}
			res.Things = append(res.Things, view)
		}

		return res, nil
	}
}

func listThingsByChannelEndpoint(svc things.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(listByConnectionReq)

		if err := req.validate(); err != nil {
			return nil, err
		}

		page, err := svc.ListThingsByChannel(ctx, req.token, req.id, req.offset, req.limit)
		if err != nil {
			return nil, err
		}

		res := thingsPageRes{
			pageRes: pageRes{
				Total:  page.Total,
				Offset: page.Offset,
				Limit:  page.Limit,
			},
			Things: []viewThingRes{},
		}
		for _, thing := range page.Things {
			view := viewThingRes{
				ID:       thing.ID,
				Owner:    thing.Owner,
				Key:      thing.Key,
				Name:     thing.Name,
				Metadata: thing.Metadata,
			}
			res.Things = append(res.Things, view)
		}

		return res, nil
	}
}

func removeThingEndpoint(svc things.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(viewResourceReq)

		err := req.validate()
		if err == things.ErrNotFound {
			return removeRes{}, nil
		}

		if err != nil {
			return nil, err
		}

		if err := svc.RemoveThing(ctx, req.token, req.id); err != nil {
			return nil, err
		}

		return removeRes{}, nil
	}
}

func createChannelEndpoint(svc things.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(createChannelReq)

		if err := req.validate(); err != nil {
			return nil, err
		}

		channel := things.Channel{Name: req.Name, Metadata: req.Metadata}
		saved, err := svc.CreateChannel(ctx, req.token, channel)
		if err != nil {
			return nil, err
		}

		res := channelRes{
			id:      saved.ID,
			created: true,
		}
		return res, nil
	}
}

func updateChannelEndpoint(svc things.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(updateChannelReq)

		if err := req.validate(); err != nil {
			return nil, err
		}

		channel := things.Channel{
			ID:       req.id,
			Name:     req.Name,
			Metadata: req.Metadata,
		}
		if err := svc.UpdateChannel(ctx, req.token, channel); err != nil {
			return nil, err
		}

		res := channelRes{
			id:      req.id,
			created: false,
		}
		return res, nil
	}
}

func viewChannelEndpoint(svc things.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(viewResourceReq)

		if err := req.validate(); err != nil {
			return nil, err
		}

		channel, err := svc.ViewChannel(ctx, req.token, req.id)
		if err != nil {
			return nil, err
		}

		res := viewChannelRes{
			ID:       channel.ID,
			Owner:    channel.Owner,
			Name:     channel.Name,
			Metadata: channel.Metadata,
		}

		return res, nil
	}
}

func listChannelsEndpoint(svc things.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(listResourcesReq)

		if err := req.validate(); err != nil {
			return nil, err
		}

		page, err := svc.ListChannels(ctx, req.token, req.offset, req.limit, req.name)
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
		// Cast channels
		for _, channel := range page.Channels {
			view := viewChannelRes{
				ID:       channel.ID,
				Owner:    channel.Owner,
				Name:     channel.Name,
				Metadata: channel.Metadata,
			}

			res.Channels = append(res.Channels, view)
		}

		return res, nil
	}
}

func listChannelsByThingEndpoint(svc things.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(listByConnectionReq)

		if err := req.validate(); err != nil {
			return nil, err
		}

		page, err := svc.ListChannelsByThing(ctx, req.token, req.id, req.offset, req.limit)
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
		for _, channel := range page.Channels {
			view := viewChannelRes{
				ID:       channel.ID,
				Owner:    channel.Owner,
				Name:     channel.Name,
				Metadata: channel.Metadata,
			}
			res.Channels = append(res.Channels, view)
		}

		return res, nil
	}
}

func removeChannelEndpoint(svc things.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(viewResourceReq)

		if err := req.validate(); err != nil {
			if err == things.ErrNotFound {
				return removeRes{}, nil
			}
			return nil, err
		}

		if err := svc.RemoveChannel(ctx, req.token, req.id); err != nil {
			return nil, err
		}

		return removeRes{}, nil
	}
}

func connectEndpoint(svc things.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		cr := request.(connectionReq)

		if err := cr.validate(); err != nil {
			return nil, err
		}

		if err := svc.Connect(ctx, cr.token, cr.chanID, cr.thingID); err != nil {
			return nil, err
		}

		return connectionRes{}, nil
	}
}

func disconnectEndpoint(svc things.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		cr := request.(connectionReq)

		if err := cr.validate(); err != nil {
			return nil, err
		}

		if err := svc.Disconnect(ctx, cr.token, cr.chanID, cr.thingID); err != nil {
			return nil, err
		}

		return disconnectionRes{}, nil
	}
}
