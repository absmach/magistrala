// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"context"

	"github.com/absmach/magistrala/internal/apiutil"
	"github.com/absmach/magistrala/invitations"
	"github.com/absmach/magistrala/pkg/errors"
	"github.com/go-kit/kit/endpoint"
)

// InvitationSent is the message returned when an invitation is sent.
const InvitationSent = "invitation sent"

func sendInvitationEndpoint(svc invitations.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(sendInvitationReq)
		if err := req.validate(); err != nil {
			return nil, errors.Wrap(apiutil.ErrValidation, err)
		}

		invitation := invitations.Invitation{
			UserID:   req.UserID,
			DomainID: req.DomainID,
			Relation: req.Relation,
			Resend:   req.Resend,
		}

		if err := svc.SendInvitation(ctx, req.token, invitation); err != nil {
			return nil, err
		}

		return sendInvitationRes{
			Message: InvitationSent,
		}, nil
	}
}

func viewInvitationEndpoint(svc invitations.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(invitationReq)
		if err := req.validate(); err != nil {
			return nil, errors.Wrap(apiutil.ErrValidation, err)
		}

		invitation, err := svc.ViewInvitation(ctx, req.token, req.userID, req.domainID)
		if err != nil {
			return nil, err
		}

		return viewInvitationRes{
			Invitation: invitation,
		}, nil
	}
}

func listInvitationsEndpoint(svc invitations.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(listInvitationsReq)
		if err := req.validate(); err != nil {
			return nil, errors.Wrap(apiutil.ErrValidation, err)
		}

		page, err := svc.ListInvitations(ctx, req.token, req.Page)
		if err != nil {
			return nil, err
		}

		return listInvitationsRes{
			page,
		}, nil
	}
}

func acceptInvitationEndpoint(svc invitations.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(acceptInvitationReq)
		if err := req.validate(); err != nil {
			return nil, errors.Wrap(apiutil.ErrValidation, err)
		}

		if err := svc.AcceptInvitation(ctx, req.token, req.DomainID); err != nil {
			return nil, err
		}

		return acceptInvitationRes{}, nil
	}
}

func deleteInvitationEndpoint(svc invitations.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(invitationReq)
		if err := req.validate(); err != nil {
			return nil, errors.Wrap(apiutil.ErrValidation, err)
		}

		if err := svc.DeleteInvitation(ctx, req.token, req.userID, req.domainID); err != nil {
			return nil, err
		}

		return deleteInvitationRes{}, nil
	}
}
