// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package token

import (
	"context"

	"github.com/absmach/supermq/auth"
	"github.com/go-kit/kit/endpoint"
)

func issueEndpoint(svc auth.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(issueReq)
		if err := req.validate(); err != nil {
			return issueRes{}, err
		}

		key := auth.Key{
			Type: req.keyType,
			User: req.userID,
		}
		tkn, err := svc.Issue(ctx, "", key)
		if err != nil {
			return issueRes{}, err
		}
		ret := issueRes{
			accessToken:  tkn.AccessToken,
			refreshToken: tkn.RefreshToken,
			accessType:   tkn.AccessType,
		}
		return ret, nil
	}
}

func refreshEndpoint(svc auth.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(refreshReq)
		if err := req.validate(); err != nil {
			return issueRes{}, err
		}

		key := auth.Key{Type: auth.RefreshKey}
		tkn, err := svc.Issue(ctx, req.refreshToken, key)
		if err != nil {
			return issueRes{}, err
		}
		ret := issueRes{
			accessToken:  tkn.AccessToken,
			refreshToken: tkn.RefreshToken,
			accessType:   tkn.AccessType,
		}
		return ret, nil
	}
}
