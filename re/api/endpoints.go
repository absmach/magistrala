// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"context"

	"github.com/absmach/magistrala/re"
	api "github.com/absmach/supermq/api/http"
	"github.com/absmach/supermq/pkg/authn"
	svcerr "github.com/absmach/supermq/pkg/errors/service"
	"github.com/go-kit/kit/endpoint"
)

func addRuleEndpoint(s re.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		session, ok := ctx.Value(api.SessionKey).(authn.Session)
		if !ok {
			return nil, svcerr.ErrAuthorization
		}

		req := request.(addRuleReq)
		if err := req.validate(); err != nil {
			return addRuleRes{}, err
		}
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
		if err := req.validate(); err != nil {
			return viewRuleRes{}, err
		}
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
		if err := req.validate(); err != nil {
			return updateRuleRes{}, err
		}
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
		if err := req.validate(); err != nil {
			return pageRes{}, err
		}
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
		if err := req.validate(); err != nil {
			return updateRoleStatusRes{}, err
		}
		err := s.RemoveRule(ctx, session, req.id)
		if err != nil {
			return updateRoleStatusRes{false}, err
		}
		return updateRoleStatusRes{true}, nil
	}
}
