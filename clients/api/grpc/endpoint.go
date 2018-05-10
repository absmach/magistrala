package grpc

import (
	"github.com/go-kit/kit/endpoint"
	"github.com/mainflux/mainflux/clients"
	context "golang.org/x/net/context"
)

func canAccessEndpoint(svc clients.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(accessReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		id, err := svc.CanAccess(req.clientKey, req.chanID)
		if err != nil {
			return accessRes{"", err}, err
		}
		return accessRes{id, nil}, nil
	}
}
