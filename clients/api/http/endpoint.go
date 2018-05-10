package http

import (
	"context"

	"github.com/go-kit/kit/endpoint"
	"github.com/mainflux/mainflux/clients"
)

func addClientEndpoint(svc clients.Service) endpoint.Endpoint {
	return func(_ context.Context, request interface{}) (interface{}, error) {
		req := request.(addClientReq)

		if err := req.validate(); err != nil {
			return nil, err
		}

		id, err := svc.AddClient(req.key, req.client)
		if err != nil {
			return nil, err
		}

		return clientRes{id: id, created: true}, nil
	}
}

func updateClientEndpoint(svc clients.Service) endpoint.Endpoint {
	return func(_ context.Context, request interface{}) (interface{}, error) {
		req := request.(updateClientReq)

		if err := req.validate(); err != nil {
			return nil, err
		}

		req.client.ID = req.id

		if err := svc.UpdateClient(req.key, req.client); err != nil {
			return nil, err
		}

		return clientRes{id: req.id, created: false}, nil
	}
}

func viewClientEndpoint(svc clients.Service) endpoint.Endpoint {
	return func(_ context.Context, request interface{}) (interface{}, error) {
		req := request.(viewResourceReq)

		if err := req.validate(); err != nil {
			return nil, err
		}

		client, err := svc.ViewClient(req.key, req.id)
		if err != nil {
			return nil, err
		}

		return viewClientRes{client}, nil
	}
}

func listClientsEndpoint(svc clients.Service) endpoint.Endpoint {
	return func(_ context.Context, request interface{}) (interface{}, error) {
		req := request.(listResourcesReq)

		if err := req.validate(); err != nil {
			return nil, err
		}

		clients, err := svc.ListClients(req.key, req.offset, req.limit)
		if err != nil {
			return nil, err
		}

		return listClientsRes{clients}, nil
	}
}

func removeClientEndpoint(svc clients.Service) endpoint.Endpoint {
	return func(_ context.Context, request interface{}) (interface{}, error) {
		req := request.(viewResourceReq)

		err := req.validate()
		if err == clients.ErrNotFound {
			return removeRes{}, nil
		}

		if err != nil {
			return nil, err
		}

		if err = svc.RemoveClient(req.key, req.id); err != nil {
			return nil, err
		}

		return removeRes{}, nil
	}
}

func createChannelEndpoint(svc clients.Service) endpoint.Endpoint {
	return func(_ context.Context, request interface{}) (interface{}, error) {
		req := request.(createChannelReq)

		if err := req.validate(); err != nil {
			return nil, err
		}

		id, err := svc.CreateChannel(req.key, req.channel)
		if err != nil {
			return nil, err
		}

		return channelRes{id: id, created: true}, nil
	}
}

func updateChannelEndpoint(svc clients.Service) endpoint.Endpoint {
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

func viewChannelEndpoint(svc clients.Service) endpoint.Endpoint {
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

func listChannelsEndpoint(svc clients.Service) endpoint.Endpoint {
	return func(_ context.Context, request interface{}) (interface{}, error) {
		req := request.(listResourcesReq)

		if err := req.validate(); err != nil {
			return nil, err
		}

		channels, err := svc.ListChannels(req.key, req.offset, req.limit)
		if err != nil {
			return nil, err
		}

		return listChannelsRes{channels}, nil
	}
}

func removeChannelEndpoint(svc clients.Service) endpoint.Endpoint {
	return func(_ context.Context, request interface{}) (interface{}, error) {
		req := request.(viewResourceReq)

		if err := req.validate(); err != nil {
			if err == clients.ErrNotFound {
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
func connectEndpoint(svc clients.Service) endpoint.Endpoint {
	return func(_ context.Context, request interface{}) (interface{}, error) {
		cr := request.(connectionReq)

		if err := cr.validate(); err != nil {
			return nil, err
		}

		if err := svc.Connect(cr.key, cr.chanID, cr.clientID); err != nil {
			return nil, err
		}

		return connectionRes{}, nil
	}
}

func disconnectEndpoint(svc clients.Service) endpoint.Endpoint {
	return func(_ context.Context, request interface{}) (interface{}, error) {
		cr := request.(connectionReq)

		if err := cr.validate(); err != nil {
			return nil, err
		}

		if err := svc.Disconnect(cr.key, cr.chanID, cr.clientID); err != nil {
			return nil, err
		}

		return disconnectionRes{}, nil
	}
}
