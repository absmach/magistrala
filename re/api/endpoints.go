// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"context"

	"github.com/absmach/magistrala/internal/api"
	"github.com/absmach/magistrala/pkg/authn"
	svcerr "github.com/absmach/magistrala/pkg/errors/service"
	"github.com/absmach/magistrala/re"
	"github.com/go-kit/kit/endpoint"
)

func addRuleEndpoint(s re.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		session, ok := ctx.Value(api.SessionKey).(authn.Session)
		if !ok {
			return nil, svcerr.ErrAuthorization
		}

		req := request.(addRuleReq)
		rule, err := s.AddRule(ctx, session, req.Rule)
		if err != nil {
			return addRuleRes{}, err
		}
		return addRuleRes{Rule: rule, created: true}, nil
	}
}

func viewRuleEndpoint(s re.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		session, ok := ctx.Value(api.SessionKey).(authn.Session)
		if !ok {
			return nil, svcerr.ErrAuthorization
		}

		req := request.(viewRuleReq)
		rule, err := s.ViewRule(ctx, session, req.id)
		if err != nil {
			return viewRuleRes{}, err
		}
		return viewRuleRes{Rule: rule}, nil
	}
}

func updateRuleEndpoint(s re.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		session, ok := ctx.Value(api.SessionKey).(authn.Session)
		if !ok {
			return nil, svcerr.ErrAuthorization
		}

		req := request.(updateRuleReq)
		rule, err := s.UpdateRule(ctx, session, req.Rule)
		if err != nil {
			return updateRuleRes{}, err
		}
		return updateRuleRes{Rule: rule}, nil
	}
}

func listRulesEndpoint(s re.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		session, ok := ctx.Value(api.SessionKey).(authn.Session)
		if !ok {
			return nil, svcerr.ErrAuthorization
		}

		req := request.(listRulesReq)
		page, err := s.ListRules(ctx, session, req.PageMeta)
		if err != nil {
			return rulesPageRes{}, nil
		}
		ret := rulesPageRes{
			Rules: page.Rules,
		}
		return ret, nil
	}
}

func upadateRuleStatusEndpoint(s re.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		session, ok := ctx.Value(api.SessionKey).(authn.Session)
		if !ok {
			return nil, svcerr.ErrAuthorization
		}

		req := request.(changeRuleStatusReq)
		err := s.RemoveRule(ctx, session, req.id)
		if err != nil {
			return changeRoleStatusRes{false}, err
		}
		return changeRoleStatusRes{true}, nil
	}
}
