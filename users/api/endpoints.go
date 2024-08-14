// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"context"

	"github.com/absmach/magistrala/pkg/apiutil"
	mgclients "github.com/absmach/magistrala/pkg/clients"
	"github.com/absmach/magistrala/pkg/errors"
	"github.com/absmach/magistrala/users"
	"github.com/go-kit/kit/endpoint"
)

func registrationEndpoint(svc users.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(createClientReq)
		if err := req.validate(); err != nil {
			return nil, errors.Wrap(apiutil.ErrValidation, err)
		}

		client, err := svc.RegisterClient(ctx, req.token, req.client)
		if err != nil {
			return nil, err
		}

		return createClientRes{
			Client:  client,
			created: true,
		}, nil
	}
}

func viewClientEndpoint(svc users.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(viewClientReq)
		if err := req.validate(); err != nil {
			return nil, errors.Wrap(apiutil.ErrValidation, err)
		}

		client, err := svc.ViewClient(ctx, req.token, req.id)
		if err != nil {
			return nil, err
		}

		return viewClientRes{Client: client}, nil
	}
}

func viewProfileEndpoint(svc users.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(viewProfileReq)
		if err := req.validate(); err != nil {
			return nil, errors.Wrap(apiutil.ErrValidation, err)
		}

		client, err := svc.ViewProfile(ctx, req.token)
		if err != nil {
			return nil, err
		}

		return viewClientRes{Client: client}, nil
	}
}

func listClientsEndpoint(svc users.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(listClientsReq)
		if err := req.validate(); err != nil {
			return nil, errors.Wrap(apiutil.ErrValidation, err)
		}

		page, err := svc.ListClients(ctx, req.token, req.page)
		if err != nil {
			return nil, err
		}

		res := clientsPageRes{
			pageRes: pageRes{
				Total:  page.Total,
				Offset: page.Offset,
				Limit:  page.Limit,
			},
			Clients: []viewClientRes{},
		}
		for _, client := range page.Clients {
			res.Clients = append(res.Clients, viewClientRes{Client: client})
		}

		return res, nil
	}
}

func searchClientsEndpoint(svc users.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(searchClientsReq)
		if err := req.validate(); err != nil {
			return nil, errors.Wrap(apiutil.ErrValidation, err)
		}

		pm := mgclients.Page{
			Offset: req.Offset,
			Limit:  req.Limit,
			Name:   req.Name,
			Id:     req.Id,
			Order:  req.Order,
			Dir:    req.Dir,
		}
		page, err := svc.SearchUsers(ctx, req.token, pm)
		if err != nil {
			return nil, err
		}

		res := clientsPageRes{
			pageRes: pageRes{
				Total:  page.Total,
				Offset: page.Offset,
				Limit:  page.Limit,
			},
			Clients: []viewClientRes{},
		}
		for _, client := range page.Clients {
			res.Clients = append(res.Clients, viewClientRes{Client: client})
		}

		return res, nil
	}
}

func updateClientEndpoint(svc users.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(updateClientReq)
		if err := req.validate(); err != nil {
			return nil, errors.Wrap(apiutil.ErrValidation, err)
		}

		client := mgclients.Client{
			ID:       req.id,
			Name:     req.Name,
			Metadata: req.Metadata,
		}
		client, err := svc.UpdateClient(ctx, req.token, client)
		if err != nil {
			return nil, err
		}

		return updateClientRes{Client: client}, nil
	}
}

func updateClientTagsEndpoint(svc users.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(updateClientTagsReq)
		if err := req.validate(); err != nil {
			return nil, errors.Wrap(apiutil.ErrValidation, err)
		}

		client := mgclients.Client{
			ID:   req.id,
			Tags: req.Tags,
		}
		client, err := svc.UpdateClientTags(ctx, req.token, client)
		if err != nil {
			return nil, err
		}

		return updateClientRes{Client: client}, nil
	}
}

func updateClientIdentityEndpoint(svc users.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(updateClientIdentityReq)
		if err := req.validate(); err != nil {
			return nil, errors.Wrap(apiutil.ErrValidation, err)
		}

		client, err := svc.UpdateClientIdentity(ctx, req.token, req.id, req.Identity)
		if err != nil {
			return nil, err
		}

		return updateClientRes{Client: client}, nil
	}
}

// Password reset request endpoint.
// When successful password reset link is generated.
// Link is generated using MG_TOKEN_RESET_ENDPOINT env.
// and value from Referer header for host.
// {Referer}+{MG_TOKEN_RESET_ENDPOINT}+{token=TOKEN}
// http://magistrala.com/reset-request?token=xxxxxxxxxxx.
// Email with a link is being sent to the user.
// When user clicks on a link it should get the ui with form to
// enter new password, when form is submitted token and new password
// must be sent as PUT request to 'password/reset' passwordResetEndpoint.
func passwordResetRequestEndpoint(svc users.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(passwResetReq)
		if err := req.validate(); err != nil {
			return nil, errors.Wrap(apiutil.ErrValidation, err)
		}

		if err := svc.GenerateResetToken(ctx, req.Email, req.Host); err != nil {
			return nil, err
		}

		return passwResetReqRes{Msg: MailSent}, nil
	}
}

// This is endpoint that actually sets new password in password reset flow.
// When user clicks on a link in email finally ends on this endpoint as explained in
// the comment above.
func passwordResetEndpoint(svc users.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(resetTokenReq)
		if err := req.validate(); err != nil {
			return nil, errors.Wrap(apiutil.ErrValidation, err)
		}

		if err := svc.ResetSecret(ctx, req.Token, req.Password); err != nil {
			return nil, err
		}

		return passwChangeRes{}, nil
	}
}

func updateClientSecretEndpoint(svc users.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(updateClientSecretReq)
		if err := req.validate(); err != nil {
			return nil, errors.Wrap(apiutil.ErrValidation, err)
		}

		client, err := svc.UpdateClientSecret(ctx, req.token, req.OldSecret, req.NewSecret)
		if err != nil {
			return nil, err
		}

		return updateClientRes{Client: client}, nil
	}
}

func updateClientRoleEndpoint(svc users.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(updateClientRoleReq)
		if err := req.validate(); err != nil {
			return nil, errors.Wrap(apiutil.ErrValidation, err)
		}

		client := mgclients.Client{
			ID:   req.id,
			Role: req.role,
		}
		client, err := svc.UpdateClientRole(ctx, req.token, client)
		if err != nil {
			return nil, err
		}

		return updateClientRes{Client: client}, nil
	}
}

func issueTokenEndpoint(svc users.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(loginClientReq)
		if err := req.validate(); err != nil {
			return nil, errors.Wrap(apiutil.ErrValidation, err)
		}

		token, err := svc.IssueToken(ctx, req.Identity, req.Secret, req.DomainID)
		if err != nil {
			return nil, err
		}

		return tokenRes{
			AccessToken:  token.GetAccessToken(),
			RefreshToken: token.GetRefreshToken(),
			AccessType:   token.GetAccessType(),
		}, nil
	}
}

func refreshTokenEndpoint(svc users.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(tokenReq)
		if err := req.validate(); err != nil {
			return nil, errors.Wrap(apiutil.ErrValidation, err)
		}

		token, err := svc.RefreshToken(ctx, req.RefreshToken, req.DomainID)
		if err != nil {
			return nil, err
		}

		return tokenRes{
			AccessToken:  token.GetAccessToken(),
			RefreshToken: token.GetRefreshToken(),
			AccessType:   token.GetAccessType(),
		}, nil
	}
}

func enableClientEndpoint(svc users.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(changeClientStatusReq)
		if err := req.validate(); err != nil {
			return nil, errors.Wrap(apiutil.ErrValidation, err)
		}

		client, err := svc.EnableClient(ctx, req.token, req.id)
		if err != nil {
			return nil, err
		}

		return changeClientStatusClientRes{Client: client}, nil
	}
}

func disableClientEndpoint(svc users.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(changeClientStatusReq)
		if err := req.validate(); err != nil {
			return nil, errors.Wrap(apiutil.ErrValidation, err)
		}

		client, err := svc.DisableClient(ctx, req.token, req.id)
		if err != nil {
			return nil, err
		}

		return changeClientStatusClientRes{Client: client}, nil
	}
}

func deleteClientEndpoint(svc users.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(changeClientStatusReq)
		if err := req.validate(); err != nil {
			return nil, errors.Wrap(apiutil.ErrValidation, err)
		}

		if err := svc.DeleteClient(ctx, req.token, req.id); err != nil {
			return nil, err
		}

		return deleteClientRes{true}, nil
	}
}

func buildClientsResponse(cp mgclients.MembersPage) clientsPageRes {
	res := clientsPageRes{
		pageRes: pageRes{
			Total:  cp.Total,
			Offset: cp.Offset,
			Limit:  cp.Limit,
		},
		Clients: []viewClientRes{},
	}

	for _, client := range cp.Members {
		res.Clients = append(res.Clients, viewClientRes{Client: client})
	}

	return res
}
