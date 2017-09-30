package api

import (
	"context"

	"github.com/go-kit/kit/endpoint"
	"github.com/mainflux/mainflux/manager"
)

func registrationEndpoint(svc manager.Service) endpoint.Endpoint {
	return func(_ context.Context, request interface{}) (interface{}, error) {
		req := request.(userReq)

		if err := req.validate(); err != nil {
			return nil, err
		}

		err := svc.Register(req.user)
		return tokenRes{}, err
	}
}

func loginEndpoint(svc manager.Service) endpoint.Endpoint {
	return func(_ context.Context, request interface{}) (interface{}, error) {
		req := request.(userReq)

		if err := req.validate(); err != nil {
			return nil, err
		}

		token, err := svc.Login(req.user)
		if err != nil {
			return nil, err
		}

		return tokenRes{token}, nil
	}
}

func identityEndpoint(svc manager.Service) endpoint.Endpoint {
	return func(_ context.Context, request interface{}) (interface{}, error) {
		req := request.(identityReq)

		if err := req.validate(); err != nil {
			return nil, manager.ErrUnauthorizedAccess
		}

		id, err := svc.Identity(req.key)
		if err != nil {
			return nil, err
		}

		res := identityRes{id: id}
		return res, nil
	}
}

func addClientEndpoint(svc manager.Service) endpoint.Endpoint {
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

func updateClientEndpoint(svc manager.Service) endpoint.Endpoint {
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

func viewClientEndpoint(svc manager.Service) endpoint.Endpoint {
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

func listClientsEndpoint(svc manager.Service) endpoint.Endpoint {
	return func(_ context.Context, request interface{}) (interface{}, error) {
		req := request.(listResourcesReq)

		if err := req.validate(); err != nil {
			return nil, err
		}

		clients, err := svc.ListClients(req.key)
		if err != nil {
			return nil, err
		}

		return listClientsRes{clients, len(clients)}, nil
	}
}

func removeClientEndpoint(svc manager.Service) endpoint.Endpoint {
	return func(_ context.Context, request interface{}) (interface{}, error) {
		req := request.(viewResourceReq)

		err := req.validate()
		if err == manager.ErrNotFound {
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

func createChannelEndpoint(svc manager.Service) endpoint.Endpoint {
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

func updateChannelEndpoint(svc manager.Service) endpoint.Endpoint {
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

func viewChannelEndpoint(svc manager.Service) endpoint.Endpoint {
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

func listChannelsEndpoint(svc manager.Service) endpoint.Endpoint {
	return func(_ context.Context, request interface{}) (interface{}, error) {
		req := request.(listResourcesReq)

		if err := req.validate(); err != nil {
			return nil, err
		}

		channels, err := svc.ListChannels(req.key)
		if err != nil {
			return nil, err
		}

		return listChannelsRes{channels, len(channels)}, nil
	}
}

func removeChannelEndpoint(svc manager.Service) endpoint.Endpoint {
	return func(_ context.Context, request interface{}) (interface{}, error) {
		req := request.(viewResourceReq)

		err := req.validate()
		if err == manager.ErrNotFound {
			return removeRes{}, nil
		}

		if err != nil {
			return nil, err
		}

		if err = svc.RemoveChannel(req.key, req.id); err != nil {
			return nil, err
		}

		return removeRes{}, nil
	}
}

func canAccessEndpoint(svc manager.Service) endpoint.Endpoint {
	return func(_ context.Context, request interface{}) (interface{}, error) {
		req := request.(viewResourceReq)

		if err := req.validate(); err != nil {
			return nil, manager.ErrUnauthorizedAccess
		}

		if allowed := svc.CanAccess(req.key, req.id); !allowed {
			return nil, manager.ErrUnauthorizedAccess
		}

		return accessRes{}, nil
	}
}
