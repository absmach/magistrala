// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package token

import (
	"context"

	"github.com/absmach/supermq/auth"
	"github.com/go-kit/kit/endpoint"
)

func issueEndpoint(svc auth.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req := request.(issueReq)
		if err := req.validate(); err != nil {
			return issueRes{}, err
		}

		key := auth.Key{
			Type:        req.keyType,
			Subject:     req.userID,
			Role:        req.userRole,
			Verified:    req.verified,
			Description: req.description,
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
	return func(ctx context.Context, request any) (any, error) {
		req := request.(refreshReq)
		if err := req.validate(); err != nil {
			return issueRes{}, err
		}

		key := auth.Key{Type: auth.RefreshKey, Verified: req.verified}
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

func revokeEndpoint(svc auth.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req := request.(revokeReq)
		if err := req.validate(); err != nil {
			return nil, err
		}
		err := svc.RevokeToken(ctx, req.tokenID)
		if err != nil {
			return nil, err
		}

		return nil, nil
	}
}

func listUserRefreshTokensEndpoint(svc auth.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req := request.(listUserRefreshTokensReq)
		if err := req.validate(); err != nil {
			return listUserRefreshTokensRes{}, err
		}

		refreshTokens, err := svc.ListUserRefreshTokens(ctx, req.userID)
		if err != nil {
			return listUserRefreshTokensRes{}, err
		}

		return listUserRefreshTokensRes{refreshTokens: refreshTokens}, nil
	}
}
