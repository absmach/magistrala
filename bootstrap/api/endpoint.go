// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"context"

	"github.com/absmach/magistrala/bootstrap"
	"github.com/absmach/magistrala/internal/api"
	"github.com/absmach/magistrala/pkg/apiutil"
	"github.com/absmach/magistrala/pkg/authn"
	"github.com/absmach/magistrala/pkg/errors"
	svcerr "github.com/absmach/magistrala/pkg/errors/service"
	"github.com/go-kit/kit/endpoint"
)

func addEndpoint(svc bootstrap.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(addReq)
		if err := req.validate(); err != nil {
			return nil, errors.Wrap(apiutil.ErrValidation, err)
		}

		session, ok := ctx.Value(api.SessionKey).(authn.Session)
		if !ok {
			return nil, svcerr.ErrAuthorization
		}

		channels := []bootstrap.Channel{}
		for _, c := range req.Channels {
			channels = append(channels, bootstrap.Channel{ID: c})
		}

		config := bootstrap.Config{
			ClientID:    req.ClientID,
			ExternalID:  req.ExternalID,
			ExternalKey: req.ExternalSecret,
			Channels:    channels,
			Name:        req.Name,
			ClientCert:  req.ClientCert,
			ClientKey:   req.ClientKey,
			CACert:      req.CACert,
			Content:     req.Content,
		}

		saved, err := svc.Add(ctx, session, req.token, config)
		if err != nil {
			return nil, err
		}

		res := configRes{
			id:      saved.ClientID,
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

		session, ok := ctx.Value(api.SessionKey).(authn.Session)
		if !ok {
			return nil, svcerr.ErrAuthorization
		}

		cfg, err := svc.UpdateCert(ctx, session, req.clientID, req.ClientCert, req.ClientKey, req.CACert)
		if err != nil {
			return nil, err
		}

		res := updateConfigRes{
			ClientID:   cfg.ClientID,
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

		session, ok := ctx.Value(api.SessionKey).(authn.Session)
		if !ok {
			return nil, svcerr.ErrAuthorization
		}

		config, err := svc.View(ctx, session, req.id)
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
			ClientID:     config.ClientID,
			CLientSecret: config.ClientSecret,
			Channels:     channels,
			ExternalID:   config.ExternalID,
			ExternalKey:  config.ExternalKey,
			Name:         config.Name,
			Content:      config.Content,
			State:        config.State,
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

		session, ok := ctx.Value(api.SessionKey).(authn.Session)
		if !ok {
			return nil, svcerr.ErrAuthorization
		}

		config := bootstrap.Config{
			ClientID: req.id,
			Name:     req.Name,
			Content:  req.Content,
		}

		if err := svc.Update(ctx, session, config); err != nil {
			return nil, err
		}

		res := configRes{
			id:      config.ClientID,
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

		session, ok := ctx.Value(api.SessionKey).(authn.Session)
		if !ok {
			return nil, svcerr.ErrAuthorization
		}

		if err := svc.UpdateConnections(ctx, session, req.token, req.id, req.Channels); err != nil {
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

		session, ok := ctx.Value(api.SessionKey).(authn.Session)
		if !ok {
			return nil, svcerr.ErrAuthorization
		}

		page, err := svc.List(ctx, session, req.filter, req.offset, req.limit)
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
				ClientID:     cfg.ClientID,
				CLientSecret: cfg.ClientSecret,
				Channels:     channels,
				ExternalID:   cfg.ExternalID,
				ExternalKey:  cfg.ExternalKey,
				Name:         cfg.Name,
				Content:      cfg.Content,
				State:        cfg.State,
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

		session, ok := ctx.Value(api.SessionKey).(authn.Session)
		if !ok {
			return nil, svcerr.ErrAuthorization
		}

		if err := svc.Remove(ctx, session, req.id); err != nil {
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

		session, ok := ctx.Value(api.SessionKey).(authn.Session)
		if !ok {
			return nil, svcerr.ErrAuthorization
		}

		if err := svc.ChangeState(ctx, session, req.token, req.id, req.State); err != nil {
			return nil, err
		}

		return stateRes{}, nil
	}
}
