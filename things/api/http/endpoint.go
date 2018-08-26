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

		saved, err := svc.AddThing(req.key, req.thing)
		if err != nil {
			return nil, err
		}

		return thingRes{id: saved.ID, created: true}, nil
	}
}

func updateThingEndpoint(svc things.Service) endpoint.Endpoint {
	return func(_ context.Context, request interface{}) (interface{}, error) {
		req := request.(updateThingReq)

		if err := req.validate(); err != nil {
			return nil, err
		}

		req.thing.ID = req.id

		if err := svc.UpdateThing(req.key, req.thing); err != nil {
			return nil, err
		}

		return thingRes{id: req.id, created: false}, nil
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

		return viewThingRes{thing}, nil
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

		return listThingsRes{Things: things}, nil
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

		if err = svc.RemoveThing(req.key, req.id); err != nil {
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

		saved, err := svc.CreateChannel(req.key, req.channel)
		if err != nil {
			return nil, err
		}

		return channelRes{id: saved.ID, created: true}, nil
	}
}

func updateChannelEndpoint(svc things.Service) endpoint.Endpoint {
	return func(_ context.Context, request interface{}) (interface{}, error) {
		req := request.(updateChannelReq)

		if err := req.validate(); err != nil {
			return nil, err
		}

		req.channel.ID = req.id

		if err := svc.UpdateChannel(req.key, req.channel); err != nil {
			return nil, err
		}

		return channelRes{id: req.id, created: false}, nil
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

		return viewChannelRes{channel}, nil
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

		return listChannelsRes{Channels: channels}, nil
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
