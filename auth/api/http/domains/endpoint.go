// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package domains

import (
	"context"

	"github.com/absmach/magistrala/auth"
	"github.com/absmach/magistrala/internal/apiutil"
	"github.com/absmach/magistrala/pkg/clients"
	"github.com/absmach/magistrala/pkg/errors"
	"github.com/go-kit/kit/endpoint"
)

func createDomainEndpoint(svc auth.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(createDomainReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		d := auth.Domain{
			Name:     req.Name,
			Metadata: req.Metadata,
			Tags:     req.Tags,
			Alias:    req.Alias,
		}
		domain, err := svc.CreateDomain(ctx, req.token, d)
		if err != nil {
			return nil, err
		}

		return createDomainRes{domain}, nil
	}
}

func retrieveDomainEndpoint(svc auth.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(retrieveDomainRequest)
		if err := req.validate(); err != nil {
			return nil, err
		}

		domain, err := svc.RetrieveDomain(ctx, req.token, req.domainID)
		if err != nil {
			return nil, err
		}
		return retrieveDomainRes{domain}, nil
	}
}

func retrieveDomainPermissionsEndpoint(svc auth.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(retrieveDomainPermissionsRequest)
		if err := req.validate(); err != nil {
			return nil, err
		}

		permissions, err := svc.RetrieveDomainPermissions(ctx, req.token, req.domainID)
		if err != nil {
			return nil, err
		}
		return retrieveDomainPermissionsRes{Permissions: permissions}, nil
	}
}

func updateDomainEndpoint(svc auth.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(updateDomainReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		var metadata clients.Metadata
		if req.Metadata != nil {
			metadata = *req.Metadata
		}
		d := auth.DomainReq{
			Name:     req.Name,
			Metadata: &metadata,
			Tags:     req.Tags,
			Alias:    req.Alias,
		}
		domain, err := svc.UpdateDomain(ctx, req.token, req.domainID, d)
		if err != nil {
			return nil, err
		}

		return updateDomainRes{domain}, nil
	}
}

func listDomainsEndpoint(svc auth.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(listDomainsReq)
		if err := req.validate(); err != nil {
			return nil, errors.Wrap(apiutil.ErrValidation, err)
		}

		page := auth.Page{
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

		var dp auth.DomainsPage
		var err error

		switch {
		case req.userID != "":
			dp, err = svc.ListUserDomains(ctx, req.token, req.userID, page)
		default:
			dp, err = svc.ListDomains(ctx, req.token, page)
		}

		if err != nil {
			return nil, err
		}
		return listDomainsRes{dp}, nil
	}
}

func enableDomainEndpoint(svc auth.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(enableDomainReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		enable := auth.EnabledStatus
		d := auth.DomainReq{
			Status: &enable,
		}
		if _, err := svc.ChangeDomainStatus(ctx, req.token, req.domainID, d); err != nil {
			return nil, err
		}
		return enableDomainRes{}, nil
	}
}

func disableDomainEndpoint(svc auth.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(disableDomainReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		disable := auth.DisabledStatus
		d := auth.DomainReq{
			Status: &disable,
		}
		if _, err := svc.ChangeDomainStatus(ctx, req.token, req.domainID, d); err != nil {
			return nil, err
		}
		return disableDomainRes{}, nil
	}
}

func freezeDomainEndpoint(svc auth.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(freezeDomainReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		freeze := auth.FreezeStatus
		d := auth.DomainReq{
			Status: &freeze,
		}
		if _, err := svc.ChangeDomainStatus(ctx, req.token, req.domainID, d); err != nil {
			return nil, err
		}
		return freezeDomainRes{}, nil
	}
}

func assignDomainUsersEndpoint(svc auth.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(assignUsersReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		if err := svc.AssignUsers(ctx, req.token, req.domainID, req.UserIDs, req.Relation); err != nil {
			return nil, err
		}
		return assignUsersRes{}, nil
	}
}

func unassignDomainUsersEndpoint(svc auth.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(unassignUsersReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		if err := svc.UnassignUsers(ctx, req.token, req.domainID, req.UserIDs, req.Relation); err != nil {
			return nil, err
		}
		return unassignUsersRes{}, nil
	}
}
