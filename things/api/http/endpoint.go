//
// Copyright (c) 2018
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
	return func(_ context.Context, request interface{}) (interface{}, error) {
		req := request.(addThingReq)

		if err := req.validate(); err != nil {
			return nil, err
		}

		thing := things.Thing{
			Type:     req.Type,
			Name:     req.Name,
			Metadata: req.Metadata,
		}
		saved, err := svc.AddThing(req.key, thing)
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
	return func(_ context.Context, request interface{}) (interface{}, error) {
		req := request.(updateThingReq)

		if err := req.validate(); err != nil {
			return nil, err
		}

		thing := things.Thing{
			ID:       req.id,
			Type:     req.Type,
			Name:     req.Name,
			Metadata: req.Metadata,
		}

		if err := svc.UpdateThing(req.key, thing); err != nil {
			return nil, err
		}

		res := thingRes{id: req.id, created: false}
		return res, nil
	}
}

func viewThingEndpoint(svc things.Service) endpoint.Endpoint {
	return func(_ context.Context, request interface{}) (interface{}, error) {
		req := request.(viewResourceReq)

		if err := req.validate(); err != nil {
			return nil, err
		}

		thing, err := svc.ViewThing(req.key, req.id)
		if err != nil {
			return nil, err
		}

		res := viewThingRes{
			ID:       thing.ID,
			Owner:    thing.Owner,
			Type:     thing.Type,
			Name:     thing.Name,
			Key:      thing.Key,
			Metadata: thing.Metadata,
		}
		return res, nil
	}
}

func listThingsEndpoint(svc things.Service) endpoint.Endpoint {
	return func(_ context.Context, request interface{}) (interface{}, error) {
		req := request.(listResourcesReq)

		if err := req.validate(); err != nil {
			return nil, err
		}

		things, err := svc.ListThings(req.key, req.offset, req.limit)
		if err != nil {
			return nil, err
		}

		res := listThingsRes{}
		for _, thing := range things {
			view := viewThingRes{
				ID:       thing.ID,
				Owner:    thing.Owner,
				Type:     thing.Type,
				Name:     thing.Name,
				Key:      thing.Key,
				Metadata: thing.Metadata,
			}
			res.Things = append(res.Things, view)
		}

		return res, nil
	}
}

func removeThingEndpoint(svc things.Service) endpoint.Endpoint {
	return func(_ context.Context, request interface{}) (interface{}, error) {
		req := request.(viewResourceReq)

		err := req.validate()
		if err == things.ErrNotFound {
			return removeRes{}, nil
		}

		if err != nil {
			return nil, err
		}

		if err := svc.RemoveThing(req.key, req.id); err != nil {
			return nil, err
		}

		return removeRes{}, nil
	}
}

func createChannelEndpoint(svc things.Service) endpoint.Endpoint {
	return func(_ context.Context, request interface{}) (interface{}, error) {
		req := request.(createChannelReq)

		if err := req.validate(); err != nil {
			return nil, err
		}

		channel := things.Channel{Name: req.Name, Metadata: req.Metadata}
		saved, err := svc.CreateChannel(req.key, channel)
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
	return func(_ context.Context, request interface{}) (interface{}, error) {
		req := request.(updateChannelReq)

		if err := req.validate(); err != nil {
			return nil, err
		}

		channel := things.Channel{
			ID:       req.id,
			Name:     req.Name,
			Metadata: req.Metadata,
		}
		if err := svc.UpdateChannel(req.key, channel); err != nil {
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
	return func(_ context.Context, request interface{}) (interface{}, error) {
		req := request.(viewResourceReq)

		if err := req.validate(); err != nil {
			return nil, err
		}

		channel, err := svc.ViewChannel(req.key, req.id)
		if err != nil {
			return nil, err
		}

		res := viewChannelRes{
			ID:       channel.ID,
			Owner:    channel.Owner,
			Name:     channel.Name,
			Metadata: channel.Metadata,
		}
		for _, thing := range channel.Things {
			view := viewThingRes{
				ID:       thing.ID,
				Owner:    thing.Owner,
				Type:     thing.Type,
				Name:     thing.Name,
				Key:      thing.Key,
				Metadata: thing.Metadata,
			}
			res.Things = append(res.Things, view)
		}

		return res, nil
	}
}

func listChannelsEndpoint(svc things.Service) endpoint.Endpoint {
	return func(_ context.Context, request interface{}) (interface{}, error) {
		req := request.(listResourcesReq)

		if err := req.validate(); err != nil {
			return nil, err
		}

		channels, err := svc.ListChannels(req.key, req.offset, req.limit)
		if err != nil {
			return nil, err
		}

		res := listChannelsRes{}
		// Cast channels
		for _, channel := range channels {
			cView := viewChannelRes{
				ID:       channel.ID,
				Owner:    channel.Owner,
				Name:     channel.Name,
				Metadata: channel.Metadata,
			}

			// Cast things
			for _, thing := range channel.Things {
				tView := viewThingRes{
					ID:       thing.ID,
					Owner:    thing.Owner,
					Type:     thing.Type,
					Name:     thing.Name,
					Key:      thing.Key,
					Metadata: thing.Metadata,
				}
				cView.Things = append(cView.Things, tView)
			}

			res.Channels = append(res.Channels, cView)
		}

		return res, nil
	}
}

func removeChannelEndpoint(svc things.Service) endpoint.Endpoint {
	return func(_ context.Context, request interface{}) (interface{}, error) {
		req := request.(viewResourceReq)

		if err := req.validate(); err != nil {
			if err == things.ErrNotFound {
				return removeRes{}, nil
			}
			return nil, err
		}

		if err := svc.RemoveChannel(req.key, req.id); err != nil {
			return nil, err
		}

		return removeRes{}, nil
	}
}

func connectEndpoint(svc things.Service) endpoint.Endpoint {
	return func(_ context.Context, request interface{}) (interface{}, error) {
		cr := request.(connectionReq)

		if err := cr.validate(); err != nil {
			return nil, err
		}

		if err := svc.Connect(cr.key, cr.chanID, cr.thingID); err != nil {
			return nil, err
		}

		return connectionRes{}, nil
	}
}

func disconnectEndpoint(svc things.Service) endpoint.Endpoint {
	return func(_ context.Context, request interface{}) (interface{}, error) {
		cr := request.(connectionReq)

		if err := cr.validate(); err != nil {
			return nil, err
		}

		if err := svc.Disconnect(cr.key, cr.chanID, cr.thingID); err != nil {
			return nil, err
		}

		return disconnectionRes{}, nil
	}
}
