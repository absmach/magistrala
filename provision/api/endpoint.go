// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"context"

	"github.com/absmach/magistrala/provision"
	apiutil "github.com/absmach/supermq/api/http/util"
	"github.com/absmach/supermq/pkg/authn"
	"github.com/absmach/supermq/pkg/errors"
	svcerr "github.com/absmach/supermq/pkg/errors/service"
	"github.com/go-kit/kit/endpoint"
)

func doProvision(svc provision.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		session, ok := ctx.Value(authn.SessionKey).(authn.Session)
		if !ok {
			return nil, svcerr.ErrAuthorization
		}
		req := request.(provisionReq)
		if err := req.validate(); err != nil {
			return nil, errors.Wrap(apiutil.ErrValidation, err)
		}

		res, err := svc.Provision(ctx, session.DomainID, req.token, req.Name, req.ExternalID, req.ExternalKey)
		if err != nil {
			return nil, err
		}

		provisionResponse := provisionRes{
			Clients:     res.Clients,
			Channels:    res.Channels,
			ClientCert:  res.ClientCert,
			ClientKey:   res.ClientKey,
			CACert:      res.CACert,
			Whitelisted: res.Whitelisted,
		}

		return provisionResponse, nil
	}
}

func getMapping(svc provision.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		res, err := svc.Mapping()
		if err != nil {
			return nil, err
		}

		return mappingRes{Data: res}, nil
	}
}

func issueCert(svc provision.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		session, ok := ctx.Value(authn.SessionKey).(authn.Session)
		if !ok {
			return nil, svcerr.ErrAuthorization
		}
		req := request.(certReq)
		if err := req.validate(); err != nil {
			return nil, errors.Wrap(apiutil.ErrValidation, err)
		}

		cert, key, err := svc.Cert(ctx, session.DomainID, req.token, req.ClientID, req.TTL)
		if err != nil {
			return nil, err
		}

		return certRes{
			Certificate: cert,
			Key:         key,
		}, nil
	}
}
