// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"context"

	apiutil "github.com/absmach/magistrala/api/http/util"
	"github.com/absmach/magistrala/bootstrap"
	"github.com/absmach/magistrala/pkg/authn"
	"github.com/absmach/magistrala/pkg/errors"
	svcerr "github.com/absmach/magistrala/pkg/errors/service"
	"github.com/go-kit/kit/endpoint"
)

func addEndpoint(svc bootstrap.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req := request.(addReq)
		if err := req.validate(); err != nil {
			return nil, errors.Wrap(apiutil.ErrValidation, err)
		}

		session, ok := ctx.Value(authn.SessionKey).(authn.Session)
		if !ok {
			return nil, svcerr.ErrAuthorization
		}

		config := bootstrap.Config{
			ExternalID:    req.ExternalID,
			ExternalKey:   req.ExternalKey,
			Name:          req.Name,
			ClientCert:    req.ClientCert,
			ClientKey:     req.ClientKey,
			CACert:        req.CACert,
			Content:       req.Content,
			ProfileID:     req.ProfileID,
			RenderContext: req.RenderContext,
		}

		saved, err := svc.Add(ctx, session, req.token, config)
		if err != nil {
			return nil, err
		}

		res := configRes{
			id:      saved.ID,
			created: true,
		}

		return res, nil
	}
}

func updateCertEndpoint(svc bootstrap.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req := request.(updateCertReq)
		if err := req.validate(); err != nil {
			return nil, errors.Wrap(apiutil.ErrValidation, err)
		}

		session, ok := ctx.Value(authn.SessionKey).(authn.Session)
		if !ok {
			return nil, svcerr.ErrAuthorization
		}

		cfg, err := svc.UpdateCert(ctx, session, req.clientID, req.ClientCert, req.ClientKey, req.CACert)
		if err != nil {
			return nil, err
		}

		res := updateConfigRes{
			ID:         cfg.ID,
			ClientCert: cfg.ClientCert,
			CACert:     cfg.CACert,
			ClientKey:  cfg.ClientKey,
		}

		return res, nil
	}
}

func viewEndpoint(svc bootstrap.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req := request.(entityReq)
		if err := req.validate(); err != nil {
			return nil, errors.Wrap(apiutil.ErrValidation, err)
		}

		session, ok := ctx.Value(authn.SessionKey).(authn.Session)
		if !ok {
			return nil, svcerr.ErrAuthorization
		}

		config, err := svc.View(ctx, session, req.id)
		if err != nil {
			return nil, err
		}

		res := viewRes{
			ID:            config.ID,
			ExternalID:    config.ExternalID,
			Name:          config.Name,
			Content:       config.Content,
			Status:        config.Status,
			ProfileID:     config.ProfileID,
			RenderContext: config.RenderContext,
		}

		return res, nil
	}
}

func updateEndpoint(svc bootstrap.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req := request.(updateReq)
		if err := req.validate(); err != nil {
			return nil, errors.Wrap(apiutil.ErrValidation, err)
		}

		session, ok := ctx.Value(authn.SessionKey).(authn.Session)
		if !ok {
			return nil, svcerr.ErrAuthorization
		}

		config := bootstrap.Config{
			ID:      req.id,
			Name:    req.Name,
			Content: req.Content,
		}

		if err := svc.Update(ctx, session, config); err != nil {
			return nil, err
		}

		res := configRes{
			id:      config.ID,
			created: false,
		}

		return res, nil
	}
}

func listEndpoint(svc bootstrap.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req := request.(listReq)
		if err := req.validate(); err != nil {
			return nil, errors.Wrap(apiutil.ErrValidation, err)
		}

		session, ok := ctx.Value(authn.SessionKey).(authn.Session)
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
			view := viewRes{
				ID:            cfg.ID,
				ExternalID:    cfg.ExternalID,
				Name:          cfg.Name,
				Content:       cfg.Content,
				Status:        cfg.Status,
				ProfileID:     cfg.ProfileID,
				RenderContext: cfg.RenderContext,
			}
			res.Configs = append(res.Configs, view)
		}

		return res, nil
	}
}

func removeEndpoint(svc bootstrap.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req := request.(entityReq)
		if err := req.validate(); err != nil {
			return removeRes{}, errors.Wrap(apiutil.ErrValidation, err)
		}

		session, ok := ctx.Value(authn.SessionKey).(authn.Session)
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
	return func(ctx context.Context, request any) (any, error) {
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

func enableConfigEndpoint(svc bootstrap.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req := request.(changeConfigStatusReq)
		if err := req.validate(); err != nil {
			return nil, errors.Wrap(apiutil.ErrValidation, err)
		}

		session, ok := ctx.Value(authn.SessionKey).(authn.Session)
		if !ok {
			return nil, svcerr.ErrAuthorization
		}

		cfg, err := svc.EnableConfig(ctx, session, req.id)
		if err != nil {
			return nil, err
		}

		return changeConfigStatusRes{Config: cfg}, nil
	}
}

func disableConfigEndpoint(svc bootstrap.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req := request.(changeConfigStatusReq)
		if err := req.validate(); err != nil {
			return nil, errors.Wrap(apiutil.ErrValidation, err)
		}

		session, ok := ctx.Value(authn.SessionKey).(authn.Session)
		if !ok {
			return nil, svcerr.ErrAuthorization
		}

		cfg, err := svc.DisableConfig(ctx, session, req.id)
		if err != nil {
			return nil, err
		}

		return changeConfigStatusRes{Config: cfg}, nil
	}
}

func createProfileEndpoint(svc bootstrap.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req := request.(createProfileReq)
		if err := req.validate(); err != nil {
			return nil, errors.Wrap(apiutil.ErrValidation, err)
		}
		session, ok := ctx.Value(authn.SessionKey).(authn.Session)
		if !ok {
			return nil, svcerr.ErrAuthorization
		}
		saved, err := svc.CreateProfile(ctx, session, req.Profile)
		if err != nil {
			return nil, err
		}
		return profileRes{Profile: saved, created: true}, nil
	}
}

func uploadProfileEndpoint(svc bootstrap.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req := request.(uploadProfileReq)
		if err := req.validate(); err != nil {
			return nil, errors.Wrap(apiutil.ErrValidation, err)
		}
		session, ok := ctx.Value(authn.SessionKey).(authn.Session)
		if !ok {
			return nil, svcerr.ErrAuthorization
		}
		saved, err := svc.CreateProfile(ctx, session, req.Profile)
		if err != nil {
			return nil, err
		}
		return profileRes{Profile: saved, created: true}, nil
	}
}

func viewProfileEndpoint(svc bootstrap.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req := request.(viewProfileReq)
		if err := req.validate(); err != nil {
			return nil, errors.Wrap(apiutil.ErrValidation, err)
		}
		session, ok := ctx.Value(authn.SessionKey).(authn.Session)
		if !ok {
			return nil, svcerr.ErrAuthorization
		}
		p, err := svc.ViewProfile(ctx, session, req.profileID)
		if err != nil {
			return nil, err
		}
		return profileRes{Profile: p}, nil
	}
}

func profileSlotsEndpoint(svc bootstrap.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req := request.(viewProfileReq)
		if err := req.validate(); err != nil {
			return nil, errors.Wrap(apiutil.ErrValidation, err)
		}
		session, ok := ctx.Value(authn.SessionKey).(authn.Session)
		if !ok {
			return nil, svcerr.ErrAuthorization
		}
		p, err := svc.ViewProfile(ctx, session, req.profileID)
		if err != nil {
			return nil, err
		}
		return profileSlotsRes{BindingSlots: p.BindingSlots}, nil
	}
}

func renderPreviewEndpoint(svc bootstrap.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req := request.(renderPreviewReq)
		if err := req.validate(); err != nil {
			return nil, errors.Wrap(apiutil.ErrValidation, err)
		}
		session, ok := ctx.Value(authn.SessionKey).(authn.Session)
		if !ok {
			return nil, svcerr.ErrAuthorization
		}
		p, err := svc.ViewProfile(ctx, session, req.profileID)
		if err != nil {
			return nil, err
		}

		cfg := req.Config
		cfg.DomainID = session.DomainID
		cfg.ProfileID = p.ID
		if cfg.RenderContext == nil {
			cfg.RenderContext = req.RenderContext
		}

		rendered, err := bootstrap.NewRenderer().Render(p, cfg, req.Bindings)
		if err != nil {
			return nil, err
		}

		return renderPreviewRes{Content: string(rendered)}, nil
	}
}

func updateProfileEndpoint(svc bootstrap.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req := request.(updateProfileReq)
		if err := req.validate(); err != nil {
			return nil, errors.Wrap(apiutil.ErrValidation, err)
		}
		session, ok := ctx.Value(authn.SessionKey).(authn.Session)
		if !ok {
			return nil, svcerr.ErrAuthorization
		}
		req.Profile.ID = req.profileID
		if err := svc.UpdateProfile(ctx, session, req.Profile); err != nil {
			return nil, err
		}
		return profileRes{Profile: req.Profile}, nil
	}
}

func deleteProfileEndpoint(svc bootstrap.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req := request.(deleteProfileReq)
		if err := req.validate(); err != nil {
			return nil, errors.Wrap(apiutil.ErrValidation, err)
		}
		session, ok := ctx.Value(authn.SessionKey).(authn.Session)
		if !ok {
			return nil, svcerr.ErrAuthorization
		}
		if err := svc.DeleteProfile(ctx, session, req.profileID); err != nil {
			return nil, err
		}
		return removeRes{}, nil
	}
}

func listProfilesEndpoint(svc bootstrap.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req := request.(listProfilesReq)
		if err := req.validate(); err != nil {
			return nil, errors.Wrap(apiutil.ErrValidation, err)
		}
		session, ok := ctx.Value(authn.SessionKey).(authn.Session)
		if !ok {
			return nil, svcerr.ErrAuthorization
		}
		page, err := svc.ListProfiles(ctx, session, req.offset, req.limit)
		if err != nil {
			return nil, err
		}
		return profilesPageRes{ProfilesPage: page}, nil
	}
}

func assignProfileEndpoint(svc bootstrap.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req := request.(assignProfileReq)
		if err := req.validate(); err != nil {
			return nil, errors.Wrap(apiutil.ErrValidation, err)
		}
		session, ok := ctx.Value(authn.SessionKey).(authn.Session)
		if !ok {
			return nil, svcerr.ErrAuthorization
		}
		if err := svc.AssignProfile(ctx, session, req.configID, req.ProfileID); err != nil {
			return nil, err
		}
		return removeRes{}, nil
	}
}

func bindResourcesEndpoint(svc bootstrap.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req := request.(bindResourcesReq)
		if err := req.validate(); err != nil {
			return nil, errors.Wrap(apiutil.ErrValidation, err)
		}
		session, ok := ctx.Value(authn.SessionKey).(authn.Session)
		if !ok {
			return nil, svcerr.ErrAuthorization
		}
		if err := svc.BindResources(ctx, session, req.token, req.configID, req.Bindings); err != nil {
			return nil, err
		}
		return removeRes{}, nil
	}
}

func listBindingsEndpoint(svc bootstrap.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req := request.(listBindingsReq)
		if err := req.validate(); err != nil {
			return nil, errors.Wrap(apiutil.ErrValidation, err)
		}
		session, ok := ctx.Value(authn.SessionKey).(authn.Session)
		if !ok {
			return nil, svcerr.ErrAuthorization
		}
		snapshots, err := svc.ListBindings(ctx, session, req.configID)
		if err != nil {
			return nil, err
		}
		return bindingsRes{Bindings: snapshots}, nil
	}
}

func refreshBindingsEndpoint(svc bootstrap.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req := request.(refreshBindingsReq)
		if err := req.validate(); err != nil {
			return nil, errors.Wrap(apiutil.ErrValidation, err)
		}
		session, ok := ctx.Value(authn.SessionKey).(authn.Session)
		if !ok {
			return nil, svcerr.ErrAuthorization
		}
		if err := svc.RefreshBindings(ctx, session, req.token, req.configID); err != nil {
			return nil, err
		}
		return removeRes{}, nil
	}
}
