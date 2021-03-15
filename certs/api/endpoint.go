// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"context"

	"github.com/go-kit/kit/endpoint"
	"github.com/mainflux/mainflux/certs"
)

func issueCert(svc certs.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(addCertsReq)
		if err := req.validate(); err != nil {
			return nil, err
		}
		res, err := svc.IssueCert(ctx, req.token, req.ThingID, req.Valid, req.KeyBits, req.KeyType)
		if err != nil {
			return certsRes{}, err
		}
		return certsRes{
			CertSerial: res.Serial,
			ThingID:    res.ThingID,
			CertKey:    res.ClientKey,
			Cert:       res.ClientCert,
			CACert:     res.IssuingCA,
		}, nil
	}
}

func listCerts(svc certs.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(listReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		page, err := svc.ListCerts(ctx, req.token, req.thingID, req.offset, req.limit)
		if err != nil {
			return certsPageRes{}, err
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
			view := certsRes{
				CertSerial: cert.Serial,
				ThingID:    cert.ThingID,
				CertKey:    cert.ClientKey,
				Cert:       cert.ClientCert,
				CACert:     cert.IssuingCA,
			}
			res.Certs = append(res.Certs, view)
		}
		return res, nil
	}
}

func revokeCert(svc certs.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(revokeReq)
		if err := req.validate(); err != nil {
			return nil, err
		}
		return svc.RevokeCert(ctx, req.token, req.certID)
	}
}
