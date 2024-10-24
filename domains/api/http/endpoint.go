// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package http

import (
	"context"

	"github.com/absmach/magistrala/domains"
	"github.com/absmach/magistrala/internal/api"
	"github.com/absmach/magistrala/pkg/apiutil"
	"github.com/absmach/magistrala/pkg/authn"
	"github.com/absmach/magistrala/pkg/errors"
	svcerr "github.com/absmach/magistrala/pkg/errors/service"
	"github.com/go-kit/kit/endpoint"
)

func createDomainEndpoint(svc domains.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(createDomainReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		session, ok := ctx.Value(api.SessionKey).(authn.Session)
		if !ok {
			return nil, svcerr.ErrAuthorization
		}

		d := domains.Domain{
			Name:     req.Name,
			Metadata: req.Metadata,
			Tags:     req.Tags,
			Alias:    req.Alias,
		}
		domain, err := svc.CreateDomain(ctx, session, d)
		if err != nil {
			return nil, err
		}

		return createDomainRes{domain}, nil
	}
}

func retrieveDomainEndpoint(svc domains.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(retrieveDomainRequest)
		if err := req.validate(); err != nil {
			return nil, err
		}

		session, ok := ctx.Value(api.SessionKey).(authn.Session)
		if !ok {
			return nil, svcerr.ErrAuthorization
		}

		domain, err := svc.RetrieveDomain(ctx, session, req.domainID)
		if err != nil {
			return nil, err
		}
		return retrieveDomainRes{domain}, nil
	}
}

func updateDomainEndpoint(svc domains.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(updateDomainReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		session, ok := ctx.Value(api.SessionKey).(authn.Session)
		if !ok {
			return nil, svcerr.ErrAuthorization
		}

		var metadata domains.Metadata
		if req.Metadata != nil {
			metadata = *req.Metadata
		}
		d := domains.DomainReq{
			Name:     req.Name,
			Metadata: &metadata,
			Tags:     req.Tags,
			Alias:    req.Alias,
		}
		domain, err := svc.UpdateDomain(ctx, session, req.domainID, d)
		if err != nil {
			return nil, err
		}

		return updateDomainRes{domain}, nil
	}
}

func listDomainsEndpoint(svc domains.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(listDomainsReq)
		if err := req.validate(); err != nil {
			return nil, errors.Wrap(apiutil.ErrValidation, err)
		}

		session, ok := ctx.Value(api.SessionKey).(authn.Session)
		if !ok {
			return nil, svcerr.ErrAuthorization
		}

		page := domains.Page{
			Offset:     req.offset,
			Limit:      req.limit,
			Name:       req.name,
			Metadata:   req.metadata,
			Order:      req.order,
			Dir:        req.dir,
			Tag:        req.tag,
			Permission: req.permission,
			Status:     req.status,
		}
		dp, err := svc.ListDomains(ctx, session, page)
		if err != nil {
			return nil, err
		}
		return listDomainsRes{dp}, nil
	}
}

func enableDomainEndpoint(svc domains.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(enableDomainReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		session, ok := ctx.Value(api.SessionKey).(authn.Session)
		if !ok {
			return nil, svcerr.ErrAuthorization
		}

		if _, err := svc.EnableDomain(ctx, session, req.domainID); err != nil {
			return nil, err
		}
		return enableDomainRes{}, nil
	}
}

func disableDomainEndpoint(svc domains.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(disableDomainReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		session, ok := ctx.Value(api.SessionKey).(authn.Session)
		if !ok {
			return nil, svcerr.ErrAuthorization
		}

		if _, err := svc.DisableDomain(ctx, session, req.domainID); err != nil {
			return nil, err
		}
		return disableDomainRes{}, nil
	}
}

func freezeDomainEndpoint(svc domains.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(freezeDomainReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		session, ok := ctx.Value(api.SessionKey).(authn.Session)
		if !ok {
			return nil, svcerr.ErrAuthorization
		}

		if _, err := svc.FreezeDomain(ctx, session, req.domainID); err != nil {
			return nil, err
		}
		return freezeDomainRes{}, nil
	}
}
