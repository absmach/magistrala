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

		channels := []bootstrap.Channel{}
		for _, c := range req.Channels {
			channels = append(channels, bootstrap.Channel{ID: c})
		}

		config := bootstrap.Config{
			MFThing:     req.ThingID,
			ExternalID:  req.ExternalID,
			ExternalKey: req.ExternalKey,
			MFChannels:  channels,
			Name:        req.Name,
			ClientCert:  req.ClientCert,
			ClientKey:   req.ClientKey,
			CACert:      req.CACert,
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

func updateCertEndpoint(svc bootstrap.Service) endpoint.Endpoint {
	return func(_ context.Context, request interface{}) (interface{}, error) {
		req := request.(updateCertReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		if err := svc.UpdateCert(req.key, req.thingKey, req.ClientCert, req.ClientKey, req.CACert); err != nil {
			return nil, err
		}

		res := configRes{}

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

		var channels []channelRes
		for _, ch := range config.MFChannels {
			channels = append(channels, channelRes{
				ID:       ch.ID,
				Name:     ch.Name,
				Metadata: ch.Metadata,
			})
		}

		res := viewRes{
			MFThing:     config.MFThing,
			MFKey:       config.MFKey,
			Channels:    channels,
			ExternalID:  config.ExternalID,
			ExternalKey: config.ExternalKey,
			Name:        config.Name,
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
			MFThing: req.id,
			Name:    req.Name,
			Content: req.Content,
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

func updateConnEndpoint(svc bootstrap.Service) endpoint.Endpoint {
	return func(_ context.Context, request interface{}) (interface{}, error) {
		req := request.(updateConnReq)

		if err := req.validate(); err != nil {
			return nil, err
		}

		if err := svc.UpdateConnections(req.key, req.id, req.Channels); err != nil {
			return nil, err
		}

		res := configRes{
			id:      req.id,
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

		page, err := svc.List(req.key, req.filter, req.offset, req.limit)
		if err != nil {
			return nil, err
		}
		switch {
		case req.filter.Unknown:
			res := listUnknownRes{}
			for _, cfg := range page.Configs {
				res.Configs = append(res.Configs, unknownRes{
					ExternalID:  cfg.ExternalID,
					ExternalKey: cfg.ExternalKey,
				})
			}
			return res, nil
		default:
			res := listRes{
				Total:   page.Total,
				Offset:  page.Offset,
				Limit:   page.Limit,
				Configs: []viewRes{},
			}

			for _, cfg := range page.Configs {
				var channels []channelRes
				for _, ch := range cfg.MFChannels {
					channels = append(channels, channelRes{
						ID:       ch.ID,
						Name:     ch.Name,
						Metadata: ch.Metadata,
					})
				}

				view := viewRes{
					MFThing:     cfg.MFThing,
					MFKey:       cfg.MFKey,
					Channels:    channels,
					ExternalID:  cfg.ExternalID,
					ExternalKey: cfg.ExternalKey,
					Name:        cfg.Name,
					Content:     cfg.Content,
					State:       cfg.State,
				}
				res.Configs = append(res.Configs, view)
			}

			return res, nil
		}
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

func bootstrapEndpoint(svc bootstrap.Service, reader bootstrap.ConfigReader, secure bool) endpoint.Endpoint {
	return func(_ context.Context, request interface{}) (interface{}, error) {
		req := request.(bootstrapReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		cfg, err := svc.Bootstrap(req.key, req.id, secure)
		if err != nil {
			return nil, err
		}

		return reader.ReadConfig(cfg, secure)
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
