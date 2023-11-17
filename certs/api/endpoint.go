// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"context"

	"github.com/absmach/magistrala/certs"
	"github.com/absmach/magistrala/internal/apiutil"
	"github.com/absmach/magistrala/pkg/errors"
	"github.com/go-kit/kit/endpoint"
)

func issueCert(svc certs.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(addCertsReq)
		if err := req.validate(); err != nil {
			return nil, errors.Wrap(apiutil.ErrValidation, err)
		}
		res, err := svc.IssueCert(ctx, req.token, req.ThingID, req.TTL)
		if err != nil {
			return certsRes{}, errors.Wrap(apiutil.ErrValidation, err)
		}

		return certsRes{
			CertSerial: res.Serial,
			ThingID:    res.ThingID,
			ClientCert: res.ClientCert,
			ClientKey:  res.ClientKey,
			Expiration: res.Expire,
			created:    true,
		}, nil
	}
}

func listSerials(svc certs.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(listReq)
		if err := req.validate(); err != nil {
			return nil, errors.Wrap(apiutil.ErrValidation, err)
		}

		page, err := svc.ListSerials(ctx, req.token, req.thingID, req.offset, req.limit)
		if err != nil {
			return certsPageRes{}, errors.Wrap(apiutil.ErrValidation, err)
		}
		res := certsPageRes{
			pageRes: pageRes{
				Total:  page.Total,
				Offset: page.Offset,
				Limit:  page.Limit,
			},
			Certs: []certsRes{},
		}

		for _, cert := range page.Certs {
			cr := certsRes{
				CertSerial: cert.Serial,
			}
			res.Certs = append(res.Certs, cr)
		}
		return res, nil
	}
}

func viewCert(svc certs.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(viewReq)
		if err := req.validate(); err != nil {
			return nil, errors.Wrap(apiutil.ErrValidation, err)
		}

		cert, err := svc.ViewCert(ctx, req.token, req.serialID)
		if err != nil {
			return certsPageRes{}, errors.Wrap(apiutil.ErrValidation, err)
		}

		certRes := certsRes{
			CertSerial: cert.Serial,
			ThingID:    cert.ThingID,
			ClientCert: cert.ClientCert,
			Expiration: cert.Expire,
		}

		return certRes, nil
	}
}

func revokeCert(svc certs.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(revokeReq)
		if err := req.validate(); err != nil {
			return nil, errors.Wrap(apiutil.ErrValidation, err)
		}
		res, err := svc.RevokeCert(ctx, req.token, req.certID)
		if err != nil {
			return nil, err
		}
		return revokeCertsRes{
			RevocationTime: res.RevocationTime,
		}, nil
	}
}
