// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"context"

	"github.com/go-kit/kit/endpoint"
	"github.com/mainflux/mainflux/internal/apiutil"
	mfclients "github.com/mainflux/mainflux/pkg/clients"
	"github.com/mainflux/mainflux/pkg/errors"
	"github.com/mainflux/mainflux/things/clients"
)

func createClientEndpoint(svc clients.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(createClientReq)
		if err := req.validate(); err != nil {
			return createClientRes{}, errors.Wrap(apiutil.ErrValidation, err)
		}
		client, err := svc.CreateThings(ctx, req.token, req.client)
		if err != nil {
			return createClientRes{}, err
		}
		ucr := createClientRes{
			Client:  client[0],
			created: true,
		}

		return ucr, nil
	}
}

func createClientsEndpoint(svc clients.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(createClientsReq)
		if err := req.validate(); err != nil {
			return clientsPageRes{}, errors.Wrap(apiutil.ErrValidation, err)
		}
		page, err := svc.CreateThings(ctx, req.token, req.Clients...)
		if err != nil {
			return clientsPageRes{}, err
		}
		res := clientsPageRes{
			pageRes: pageRes{
				Total: uint64(len(page)),
			},
			Clients: []viewClientRes{},
		}
		for _, c := range page {
			res.Clients = append(res.Clients, viewClientRes{Client: c})
		}
		return res, nil
	}
}

func viewClientEndpoint(svc clients.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(viewClientReq)
		if err := req.validate(); err != nil {
			return nil, errors.Wrap(apiutil.ErrValidation, err)
		}

		c, err := svc.ViewClient(ctx, req.token, req.id)
		if err != nil {
			return nil, err
		}
		return viewClientRes{Client: c}, nil
	}
}

func listClientsEndpoint(svc clients.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(listClientsReq)
		if err := req.validate(); err != nil {
			return mfclients.ClientsPage{}, errors.Wrap(apiutil.ErrValidation, err)
		}

		pm := mfclients.Page{
			SharedBy: req.sharedBy,
			Status:   req.status,
			Offset:   req.offset,
			Limit:    req.limit,
			Owner:    req.owner,
			Name:     req.name,
			Tag:      req.tag,
			Metadata: req.metadata,
		}
		page, err := svc.ListClients(ctx, req.token, pm)
		if err != nil {
			return mfclients.ClientsPage{}, err
		}

		res := clientsPageRes{
			pageRes: pageRes{
				Total:  page.Total,
				Offset: page.Offset,
				Limit:  page.Limit,
			},
			Clients: []viewClientRes{},
		}
		for _, c := range page.Clients {
			res.Clients = append(res.Clients, viewClientRes{Client: c})
		}

		return res, nil
	}
}

func listMembersEndpoint(svc clients.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(listMembersReq)
		if err := req.validate(); err != nil {
			return memberPageRes{}, errors.Wrap(apiutil.ErrValidation, err)
		}
		page, err := svc.ListClientsByGroup(ctx, req.token, req.groupID, req.Page)
		if err != nil {
			return memberPageRes{}, err
		}
		return buildMembersResponse(page), nil
	}
}

func updateClientEndpoint(svc clients.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(updateClientReq)
		if err := req.validate(); err != nil {
			return nil, errors.Wrap(apiutil.ErrValidation, err)
		}

		cli := mfclients.Client{
			ID:       req.id,
			Name:     req.Name,
			Metadata: req.Metadata,
		}
		client, err := svc.UpdateClient(ctx, req.token, cli)
		if err != nil {
			return nil, err
		}
		return updateClientRes{Client: client}, nil
	}
}

func updateClientTagsEndpoint(svc clients.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(updateClientTagsReq)
		if err := req.validate(); err != nil {
			return nil, errors.Wrap(apiutil.ErrValidation, err)
		}

		cli := mfclients.Client{
			ID:   req.id,
			Tags: req.Tags,
		}
		client, err := svc.UpdateClientTags(ctx, req.token, cli)
		if err != nil {
			return nil, err
		}
		return updateClientRes{Client: client}, nil
	}
}

func updateClientSecretEndpoint(svc clients.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(updateClientCredentialsReq)
		if err := req.validate(); err != nil {
			return nil, errors.Wrap(apiutil.ErrValidation, err)
		}
		client, err := svc.UpdateClientSecret(ctx, req.token, req.id, req.Secret)
		if err != nil {
			return nil, err
		}
		return updateClientRes{Client: client}, nil
	}
}

func updateClientOwnerEndpoint(svc clients.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(updateClientOwnerReq)
		if err := req.validate(); err != nil {
			return nil, errors.Wrap(apiutil.ErrValidation, err)
		}

		cli := mfclients.Client{
			ID:    req.id,
			Owner: req.Owner,
		}

		client, err := svc.UpdateClientOwner(ctx, req.token, cli)
		if err != nil {
			return nil, err
		}
		return updateClientRes{Client: client}, nil
	}
}

func enableClientEndpoint(svc clients.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(changeClientStatusReq)
		if err := req.validate(); err != nil {
			return nil, errors.Wrap(apiutil.ErrValidation, err)
		}
		client, err := svc.EnableClient(ctx, req.token, req.id)
		if err != nil {
			return nil, err
		}
		return deleteClientRes{Client: client}, nil
	}
}

func disableClientEndpoint(svc clients.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(changeClientStatusReq)
		if err := req.validate(); err != nil {
			return nil, errors.Wrap(apiutil.ErrValidation, err)
		}
		client, err := svc.DisableClient(ctx, req.token, req.id)
		if err != nil {
			return nil, err
		}
		return deleteClientRes{Client: client}, nil
	}
}

func buildMembersResponse(cp mfclients.MembersPage) memberPageRes {
	res := memberPageRes{
		pageRes: pageRes{
			Total:  cp.Total,
			Offset: cp.Offset,
			Limit:  cp.Limit,
		},
		Members: []viewMembersRes{},
	}
	for _, c := range cp.Members {
		res.Members = append(res.Members, viewMembersRes{Client: c})
	}
	return res
}
