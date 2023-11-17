// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"context"

	"github.com/absmach/magistrala/bootstrap"
	"github.com/absmach/magistrala/internal/apiutil"
	"github.com/absmach/magistrala/pkg/errors"
	"github.com/go-kit/kit/endpoint"
)

func addEndpoint(svc bootstrap.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(addReq)
		if err := req.validate(); err != nil {
			return nil, errors.Wrap(apiutil.ErrValidation, err)
		}

		channels := []bootstrap.Channel{}
		for _, c := range req.Channels {
			channels = append(channels, bootstrap.Channel{ID: c})
		}

		config := bootstrap.Config{
			ThingID:     req.ThingID,
			ExternalID:  req.ExternalID,
			ExternalKey: req.ExternalKey,
			Channels:    channels,
			Name:        req.Name,
			ClientCert:  req.ClientCert,
			ClientKey:   req.ClientKey,
			CACert:      req.CACert,
			Content:     req.Content,
		}

		saved, err := svc.Add(ctx, req.token, config)
		if err != nil {
			return nil, err
		}

		res := configRes{
			id:      saved.ThingID,
			created: true,
		}

		return res, nil
	}
}

func updateCertEndpoint(svc bootstrap.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(updateCertReq)
		if err := req.validate(); err != nil {
			return nil, errors.Wrap(apiutil.ErrValidation, err)
		}

		cfg, err := svc.UpdateCert(ctx, req.token, req.thingID, req.ClientCert, req.ClientKey, req.CACert)
		if err != nil {
			return nil, err
		}

		res := updateConfigRes{
			ThingID:    cfg.ThingID,
			ClientCert: cfg.ClientCert,
			CACert:     cfg.CACert,
			ClientKey:  cfg.ClientKey,
		}

		return res, nil
	}
}

func viewEndpoint(svc bootstrap.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(entityReq)

		if err := req.validate(); err != nil {
			return nil, errors.Wrap(apiutil.ErrValidation, err)
		}

		config, err := svc.View(ctx, req.token, req.id)
		if err != nil {
			return nil, err
		}

		var channels []channelRes
		for _, ch := range config.Channels {
			channels = append(channels, channelRes{
				ID:       ch.ID,
				Name:     ch.Name,
				Metadata: ch.Metadata,
			})
		}

		res := viewRes{
			ThingID:     config.ThingID,
			ThingKey:    config.ThingKey,
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
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(updateReq)

		if err := req.validate(); err != nil {
			return nil, errors.Wrap(apiutil.ErrValidation, err)
		}

		config := bootstrap.Config{
			ThingID: req.id,
			Name:    req.Name,
			Content: req.Content,
		}

		if err := svc.Update(ctx, req.token, config); err != nil {
			return nil, err
		}

		res := configRes{
			id:      config.ThingID,
			created: false,
		}

		return res, nil
	}
}

func updateConnEndpoint(svc bootstrap.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(updateConnReq)

		if err := req.validate(); err != nil {
			return nil, errors.Wrap(apiutil.ErrValidation, err)
		}

		if err := svc.UpdateConnections(ctx, req.token, req.id, req.Channels); err != nil {
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
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(listReq)

		if err := req.validate(); err != nil {
			return nil, errors.Wrap(apiutil.ErrValidation, err)
		}

		page, err := svc.List(ctx, req.token, req.filter, req.offset, req.limit)
		if err != nil {
			return nil, err
		}
		res := listRes{
			Total:   page.Total,
			Offset:  page.Offset,
			Limit:   page.Limit,
			Configs: []viewRes{},
		}

		for _, cfg := range page.Configs {
			var channels []channelRes
			for _, ch := range cfg.Channels {
				channels = append(channels, channelRes{
					ID:       ch.ID,
					Name:     ch.Name,
					Metadata: ch.Metadata,
				})
			}

			view := viewRes{
				ThingID:     cfg.ThingID,
				ThingKey:    cfg.ThingKey,
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

func removeEndpoint(svc bootstrap.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(entityReq)

		if err := req.validate(); err != nil {
			return removeRes{}, errors.Wrap(apiutil.ErrValidation, err)
		}

		if err := svc.Remove(ctx, req.token, req.id); err != nil {
			return nil, err
		}

		return removeRes{}, nil
	}
}

func bootstrapEndpoint(svc bootstrap.Service, reader bootstrap.ConfigReader, secure bool) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(bootstrapReq)
		if err := req.validate(); err != nil {
			return nil, errors.Wrap(apiutil.ErrValidation, err)
		}

		cfg, err := svc.Bootstrap(ctx, req.key, req.id, secure)
		if err != nil {
			return nil, err
		}

		return reader.ReadConfig(cfg, secure)
	}
}

func stateEndpoint(svc bootstrap.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(changeStateReq)

		if err := req.validate(); err != nil {
			return nil, errors.Wrap(apiutil.ErrValidation, err)
		}

		if err := svc.ChangeState(ctx, req.token, req.id, req.State); err != nil {
			return nil, err
		}

		return stateRes{}, nil
	}
}
