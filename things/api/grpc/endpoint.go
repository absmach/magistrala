package grpc

import (
	"github.com/go-kit/kit/endpoint"
	"github.com/mainflux/mainflux/things"
	context "golang.org/x/net/context"
)

func canAccessEndpoint(svc things.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(accessReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		id, err := svc.CanAccess(req.thingKey, req.chanID)
		if err != nil {
			return identityRes{id: "", err: err}, err
		}
		return identityRes{id: id, err: nil}, nil
	}
}

func identifyEndpoint(svc things.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(identifyReq)
		id, err := svc.Identify(req.key)
		if err != nil {
			return identityRes{id: "", err: err}, err
		}
		return identityRes{id: id, err: nil}, nil
	}
}
