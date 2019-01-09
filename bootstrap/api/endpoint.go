//
// Copyright (c) 2018
// Mainflux
//
// SPDX-License-Identifier: Apache-2.0
//

package api

import (
	"context"

	"github.com/go-kit/kit/endpoint"
	"github.com/mainflux/mainflux/bootstrap"
)

func addEndpoint(svc bootstrap.Service) endpoint.Endpoint {
	return func(_ context.Context, request interface{}) (interface{}, error) {
		req := request.(addReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		config := bootstrap.Config{
			ExternalID:  req.ExternalID,
			ExternalKey: req.ExternalKey,
			MFChannels:  req.Channels,
			Content:     req.Content,
		}

		saved, err := svc.Add(req.key, config)
		if err != nil {
			return nil, err
		}

		res := configRes{
			id:      saved.MFThing,
			created: true,
		}

		return res, nil
	}
}

func viewEndpoint(svc bootstrap.Service) endpoint.Endpoint {
	return func(_ context.Context, request interface{}) (interface{}, error) {
		req := request.(entityReq)

		if err := req.validate(); err != nil {
			return nil, err
		}

		config, err := svc.View(req.key, req.id)
		if err != nil {
			return nil, err
		}

		res := viewRes{
			MFThing:     config.MFThing,
			MFKey:       config.MFKey,
			Channels:    config.MFChannels,
			ExternalID:  config.ExternalID,
			ExternalKey: config.ExternalKey,
			Content:     config.Content,
			State:       config.State,
		}

		return res, nil
	}
}

func updateEndpoint(svc bootstrap.Service) endpoint.Endpoint {
	return func(_ context.Context, request interface{}) (interface{}, error) {
		req := request.(updateReq)

		if err := req.validate(); err != nil {
			return nil, err
		}

		config := bootstrap.Config{
			MFThing:    req.id,
			MFChannels: req.Channels,
			Content:    req.Content,
			State:      req.State,
		}

		if err := svc.Update(req.key, config); err != nil {
			return nil, err
		}

		res := configRes{
			id:      config.MFThing,
			created: false,
		}

		return res, nil
	}
}

func listEndpoint(svc bootstrap.Service) endpoint.Endpoint {
	return func(_ context.Context, request interface{}) (interface{}, error) {
		req := request.(listReq)

		if err := req.validate(); err != nil {
			return nil, err
		}

		configs, err := svc.List(req.key, req.filter, req.offset, req.limit)
		if err != nil {
			return nil, err
		}

		res := listRes{
			Configs: []viewRes{},
		}

		for _, cfg := range configs {
			view := viewRes{
				MFThing:     cfg.MFThing,
				MFKey:       cfg.MFKey,
				Channels:    cfg.MFChannels,
				ExternalID:  cfg.ExternalID,
				ExternalKey: cfg.ExternalKey,
				Content:     cfg.Content,
				State:       cfg.State,
			}
			res.Configs = append(res.Configs, view)
		}

		return res, nil
	}
}

func removeEndpoint(svc bootstrap.Service) endpoint.Endpoint {
	return func(_ context.Context, request interface{}) (interface{}, error) {
		req := request.(entityReq)

		if err := req.validate(); err != nil {
			return removeRes{}, err
		}

		if err := svc.Remove(req.key, req.id); err != nil {
			return nil, err
		}

		return removeRes{}, nil
	}
}

func bootstrapEndpoint(svc bootstrap.Service, reader bootstrap.ConfigReader) endpoint.Endpoint {
	return func(_ context.Context, request interface{}) (interface{}, error) {
		req := request.(bootstrapReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		cfg, err := svc.Bootstrap(req.key, req.id)
		if err != nil {
			return nil, err
		}

		return reader.ReadConfig(cfg)
	}
}

func stateEndpoint(svc bootstrap.Service) endpoint.Endpoint {
	return func(_ context.Context, request interface{}) (interface{}, error) {
		req := request.(changeStateReq)

		if err := req.validate(); err != nil {
			return nil, err
		}

		if err := svc.ChangeState(req.key, req.id, req.State); err != nil {
			return nil, err
		}

		return stateRes{}, nil
	}
}
