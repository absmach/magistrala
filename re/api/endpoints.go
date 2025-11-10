// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"context"

	"github.com/absmach/magistrala/re"
	apiutil "github.com/absmach/supermq/api/http/util"
	"github.com/absmach/supermq/pkg/authn"
	"github.com/absmach/supermq/pkg/errors"
	svcerr "github.com/absmach/supermq/pkg/errors/service"
	"github.com/go-kit/kit/endpoint"
)

func addRuleEndpoint(s re.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		session, ok := ctx.Value(authn.SessionKey).(authn.Session)
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
	return func(ctx context.Context, request any) (any, error) {
		session, ok := ctx.Value(authn.SessionKey).(authn.Session)
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
	return func(ctx context.Context, request any) (any, error) {
		session, ok := ctx.Value(authn.SessionKey).(authn.Session)
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

func updateRuleTagsEndpoint(svc re.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req := request.(updateRuleTagsReq)
		if err := req.validate(); err != nil {
			return nil, errors.Wrap(apiutil.ErrValidation, err)
		}

		session, ok := ctx.Value(authn.SessionKey).(authn.Session)
		if !ok {
			return nil, svcerr.ErrAuthentication
		}

		r := re.Rule{
			ID:   req.id,
			Tags: req.Tags,
		}
		res, err := svc.UpdateRuleTags(ctx, session, r)
		if err != nil {
			return nil, err
		}

		return updateRuleRes{Rule: res}, nil
	}
}

func updateRuleScheduleEndpoint(s re.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		session, ok := ctx.Value(authn.SessionKey).(authn.Session)
		if !ok {
			return nil, svcerr.ErrAuthorization
		}

		req := request.(updateRuleScheduleReq)
		if err := req.validate(); err != nil {
			return updateRuleRes{}, err
		}

		rule := re.Rule{
			ID:       req.id,
			Schedule: req.Schedule,
		}

		updatedRule, err := s.UpdateRuleSchedule(ctx, session, rule)
		if err != nil {
			return updateRuleRes{}, err
		}
		return updateRuleRes{Rule: updatedRule}, nil
	}
}

func listRulesEndpoint(s re.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		session, ok := ctx.Value(authn.SessionKey).(authn.Session)
		if !ok {
			return nil, svcerr.ErrAuthorization
		}

		req := request.(listRulesReq)
		if err := req.validate(); err != nil {
			return pageRes{}, err
		}
		page, err := s.ListRules(ctx, session, req.PageMeta)
		if err != nil {
			return rulesPageRes{}, err
		}
		ret := rulesPageRes{
			Page: page,
		}
		return ret, nil
	}
}

func deleteRuleEndpoint(s re.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		session, ok := ctx.Value(authn.SessionKey).(authn.Session)
		if !ok {
			return nil, svcerr.ErrAuthorization
		}

		req := request.(deleteRuleReq)
		if err := req.validate(); err != nil {
			return deleteRuleRes{}, err
		}
		err := s.RemoveRule(ctx, session, req.id)
		if err != nil {
			return deleteRuleRes{false}, err
		}
		return deleteRuleRes{true}, nil
	}
}

func enableRuleEndpoint(s re.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		session, ok := ctx.Value(authn.SessionKey).(authn.Session)
		if !ok {
			return nil, svcerr.ErrAuthorization
		}

		req := request.(updateRuleStatusReq)
		if err := req.validate(); err != nil {
			return updateRuleStatusRes{}, err
		}

		rule, err := s.EnableRule(ctx, session, req.id)
		if err != nil {
			return updateRuleStatusRes{}, err
		}

		return updateRuleStatusRes{Rule: rule}, err
	}
}

func disableRuleEndpoint(s re.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		session, ok := ctx.Value(authn.SessionKey).(authn.Session)
		if !ok {
			return nil, svcerr.ErrAuthorization
		}

		req := request.(updateRuleStatusReq)
		if err := req.validate(); err != nil {
			return updateRuleStatusRes{}, err
		}

		rule, err := s.DisableRule(ctx, session, req.id)
		if err != nil {
			return updateRuleStatusRes{}, err
		}

		return updateRuleStatusRes{Rule: rule}, err
	}
}
