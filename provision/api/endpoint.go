package api

import (
	"context"

	"github.com/go-kit/kit/endpoint"
	"github.com/mainflux/mainflux/provision"
)

func doProvision(svc provision.Service) endpoint.Endpoint {
	return func(_ context.Context, request interface{}) (interface{}, error) {
		req := request.(provisionReq)
		if err := req.validate(); err != nil {
			return nil, err
		}
		token := req.token

		res, err := svc.Provision(token, req.Name, req.ExternalID, req.ExternalKey)

		if err != nil {
			return provisionRes{Error: err.Error()}, nil
		}

		provisionResponse := provisionRes{
			Things:      res.Things,
			Channels:    res.Channels,
			ClientCert:  res.ClientCert,
			ClientKey:   res.ClientKey,
			CACert:      res.CACert,
			Whitelisted: res.Whitelisted,
		}

		return provisionResponse, nil

	}
}

func getMapping(svc provision.Service) endpoint.Endpoint {
	return func(_ context.Context, request interface{}) (interface{}, error) {
		req := request.(mappingReq)
		if err := req.validate(); err != nil {
			return nil, err
		}
		return svc.Mapping(req.token)
	}
}
