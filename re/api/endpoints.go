// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"context"
	"time"

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

		rule, err := s.AddRule(ctx, session, toRules(req))
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
		rule, err := s.UpdateRule(ctx, session, toRules(req.Rule))
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
			pageRes: pageRes{
				Limit:  page.Limit,
				Offset: page.Offset,
				Total:  page.Total,
			},
			Rules: page.Rules,
		}
		return ret, nil
	}
}

func deleteRuleEndpoint(s re.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		session, ok := ctx.Value(api.SessionKey).(authn.Session)
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
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		session, ok := ctx.Value(api.SessionKey).(authn.Session)
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
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		session, ok := ctx.Value(api.SessionKey).(authn.Session)
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

func toRules(req addRuleReq) re.Rule {
	var startDateTime time.Time
	var schTime time.Time

	if req.Schedule.StartDateTime != 0 {
		startDateTime = time.Unix(req.Schedule.StartDateTime, 0)
	}
	if req.Schedule.Time != 0 {
		schTime = time.Unix(req.Schedule.Time, 0)
	}
	return re.Rule{
		ID:            req.ID,
		Name:          req.Name,
		Metadata:      req.Metadata,
		InputChannel:  req.InputChannel,
		InputTopic:    req.InputTopic,
		Logic:         req.Logic,
		OutputChannel: req.OutputChannel,
		OutputTopic:   req.OutputTopic,
		Schedule: re.Schedule{
			Recurring:       req.Schedule.Recurring,
			RecurringPeriod: req.Schedule.RecurringPeriod,
			StartDateTime:   startDateTime,
			Time:            schTime,
		},
	}
}
