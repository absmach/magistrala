// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package http

import (
	"context"

	api "github.com/absmach/supermq/api/http"
	apiutil "github.com/absmach/supermq/api/http/util"
	"github.com/absmach/supermq/domains"
	"github.com/absmach/supermq/pkg/authn"
	"github.com/absmach/supermq/pkg/errors"
	svcerr "github.com/absmach/supermq/pkg/errors/service"
	"github.com/go-kit/kit/endpoint"
)

// InvitationSent is the message returned when an invitation is sent.
const InvitationSent = "invitation sent"

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
			ID:       req.ID,
			Name:     req.Name,
			Metadata: req.Metadata,
			Tags:     req.Tags,
			Alias:    req.Alias,
		}
		domain, _, err := svc.CreateDomain(ctx, session, d)
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
			Offset:   req.offset,
			Limit:    req.limit,
			Name:     req.name,
			Metadata: req.metadata,
			Order:    req.order,
			Dir:      req.dir,
			Tag:      req.tag,
			RoleID:   req.roleID,
			RoleName: req.roleName,
			Actions:  req.actions,
			Status:   req.status,
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

func sendInvitationEndpoint(svc domains.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(sendInvitationReq)
		if err := req.validate(); err != nil {
			return nil, errors.Wrap(apiutil.ErrValidation, err)
		}
		session, ok := ctx.Value(api.SessionKey).(authn.Session)
		if !ok {
			return nil, svcerr.ErrAuthorization
		}

		invitation := domains.Invitation{
			InviteeUserID: req.InviteeUserID,
			DomainID:      session.DomainID,
			RoleID:        req.RoleID,
		}

		if err := svc.SendInvitation(ctx, session, invitation); err != nil {
			return nil, err
		}

		return sendInvitationRes{
			Message: InvitationSent,
		}, nil
	}
}

func viewInvitationEndpoint(svc domains.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(invitationReq)
		if err := req.validate(); err != nil {
			return nil, errors.Wrap(apiutil.ErrValidation, err)
		}
		session, ok := ctx.Value(api.SessionKey).(authn.Session)
		if !ok {
			return nil, svcerr.ErrAuthorization
		}
		session.DomainID = req.domainID
		invitation, err := svc.ViewInvitation(ctx, session, req.userID, req.domainID)
		if err != nil {
			return nil, err
		}

		return viewInvitationRes{
			Invitation: invitation,
		}, nil
	}
}

func listDomainInvitationsEndpoint(svc domains.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(listInvitationsReq)
		if err := req.validate(); err != nil {
			return nil, errors.Wrap(apiutil.ErrValidation, err)
		}

		session, ok := ctx.Value(api.SessionKey).(authn.Session)
		if !ok {
			return nil, svcerr.ErrAuthorization
		}
		req.InvitationPageMeta.DomainID = session.DomainID

		page, err := svc.ListInvitations(ctx, session, req.InvitationPageMeta)
		if err != nil {
			return nil, err
		}

		return listInvitationsRes{
			page,
		}, nil
	}
}

func listUserInvitationsEndpoint(svc domains.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(listInvitationsReq)
		if err := req.validate(); err != nil {
			return nil, errors.Wrap(apiutil.ErrValidation, err)
		}

		session, ok := ctx.Value(api.SessionKey).(authn.Session)
		if !ok {
			return nil, svcerr.ErrAuthorization
		}
		session.DomainID = req.DomainID

		page, err := svc.ListInvitations(ctx, session, req.InvitationPageMeta)
		if err != nil {
			return nil, err
		}

		return listInvitationsRes{
			page,
		}, nil
	}
}

func acceptInvitationEndpoint(svc domains.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(acceptInvitationReq)
		if err := req.validate(); err != nil {
			return nil, errors.Wrap(apiutil.ErrValidation, err)
		}

		session, ok := ctx.Value(api.SessionKey).(authn.Session)
		if !ok {
			return nil, svcerr.ErrAuthorization
		}

		if err := svc.AcceptInvitation(ctx, session, req.DomainID); err != nil {
			return nil, err
		}

		return acceptInvitationRes{}, nil
	}
}

func rejectInvitationEndpoint(svc domains.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(acceptInvitationReq)
		if err := req.validate(); err != nil {
			return nil, errors.Wrap(apiutil.ErrValidation, err)
		}

		session, ok := ctx.Value(api.SessionKey).(authn.Session)
		if !ok {
			return nil, svcerr.ErrAuthorization
		}

		if err := svc.RejectInvitation(ctx, session, req.DomainID); err != nil {
			return nil, err
		}

		return rejectInvitationRes{}, nil
	}
}

func deleteInvitationEndpoint(svc domains.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(invitationReq)
		if err := req.validate(); err != nil {
			return nil, errors.Wrap(apiutil.ErrValidation, err)
		}

		session, ok := ctx.Value(api.SessionKey).(authn.Session)
		if !ok {
			return nil, svcerr.ErrAuthorization
		}
		session.DomainID = req.domainID

		if err := svc.DeleteInvitation(ctx, session, req.userID, req.domainID); err != nil {
			return nil, err
		}

		return deleteInvitationRes{}, nil
	}
}
