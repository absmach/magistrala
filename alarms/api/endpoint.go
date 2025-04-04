// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"context"

	"github.com/absmach/magistrala/alarms"
	api "github.com/absmach/supermq/api/http"
	apiutil "github.com/absmach/supermq/api/http/util"
	"github.com/absmach/supermq/pkg/authn"
	"github.com/absmach/supermq/pkg/errors"
	svcerr "github.com/absmach/supermq/pkg/errors/service"
	"github.com/go-kit/kit/endpoint"
)

func createRuleEndpoint(svc alarms.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(createRuleReq)
		if err := req.validate(); err != nil {
			return nil, errors.Wrap(apiutil.ErrValidation, err)
		}

		session, ok := ctx.Value(api.SessionKey).(authn.Session)
		if !ok {
			return nil, svcerr.ErrAuthorization
		}

		rule, err := svc.CreateRule(ctx, session, req.Rule)
		if err != nil {
			return nil, err
		}

		return ruleRes{
			Rule: rule,
		}, nil
	}
}

func updateRuleEndpoint(svc alarms.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(createRuleReq)
		if err := req.validate(); err != nil {
			return nil, errors.Wrap(apiutil.ErrValidation, err)
		}

		session, ok := ctx.Value(api.SessionKey).(authn.Session)
		if !ok {
			return nil, svcerr.ErrAuthorization
		}

		rule, err := svc.UpdateRule(ctx, session, req.Rule)
		if err != nil {
			return nil, err
		}

		return ruleRes{
			Rule: rule,
		}, nil
	}
}

func viewRuleEndpoint(svc alarms.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(entityReq)
		if err := req.validate(); err != nil {
			return nil, errors.Wrap(apiutil.ErrValidation, err)
		}

		session, ok := ctx.Value(api.SessionKey).(authn.Session)
		if !ok {
			return nil, svcerr.ErrAuthorization
		}

		rule, err := svc.ViewRule(ctx, session, req.ID)
		if err != nil {
			return nil, err
		}

		return ruleRes{
			Rule: rule,
		}, nil
	}
}

func listRulesEndpoint(svc alarms.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(alarms.PageMetadata)

		session, ok := ctx.Value(api.SessionKey).(authn.Session)
		if !ok {
			return nil, svcerr.ErrAuthorization
		}

		rules, err := svc.ListRules(ctx, session, req)
		if err != nil {
			return nil, err
		}

		return rulesPageRes{
			RulesPage: rules,
		}, nil
	}
}

func deleteRuleEndpoint(svc alarms.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(entityReq)
		if err := req.validate(); err != nil {
			return nil, errors.Wrap(apiutil.ErrValidation, err)
		}

		session, ok := ctx.Value(api.SessionKey).(authn.Session)
		if !ok {
			return nil, svcerr.ErrAuthorization
		}

		if err := svc.DeleteRule(ctx, session, req.ID); err != nil {
			return nil, err
		}

		return nil, nil
	}
}

func createAlarmEndpoint(svc alarms.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(createAlarmReq)
		if err := req.validate(); err != nil {
			return nil, errors.Wrap(apiutil.ErrValidation, err)
		}

		session, ok := ctx.Value(api.SessionKey).(authn.Session)
		if !ok {
			return nil, svcerr.ErrAuthorization
		}

		alarm, err := svc.CreateAlarm(ctx, session, req.Alarm)
		if err != nil {
			return nil, err
		}

		return alarmRes{
			Alarm: alarm,
		}, nil
	}
}

func updateAlarmEndpoint(svc alarms.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(createAlarmReq)
		if err := req.validate(); err != nil {
			return nil, errors.Wrap(apiutil.ErrValidation, err)
		}

		session, ok := ctx.Value(api.SessionKey).(authn.Session)
		if !ok {
			return nil, svcerr.ErrAuthorization
		}

		alarm, err := svc.UpdateAlarm(ctx, session, req.Alarm)
		if err != nil {
			return nil, err
		}

		return alarmRes{
			Alarm: alarm,
		}, nil
	}
}

func viewAlarmEndpoint(svc alarms.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(entityReq)
		if err := req.validate(); err != nil {
			return nil, errors.Wrap(apiutil.ErrValidation, err)
		}

		session, ok := ctx.Value(api.SessionKey).(authn.Session)
		if !ok {
			return nil, svcerr.ErrAuthorization
		}

		alarm, err := svc.ViewAlarm(ctx, session, req.ID)
		if err != nil {
			return nil, err
		}

		return alarmRes{
			Alarm: alarm,
		}, nil
	}
}

func listAlarmsEndpoint(svc alarms.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(alarms.PageMetadata)

		session, ok := ctx.Value(api.SessionKey).(authn.Session)
		if !ok {
			return nil, svcerr.ErrAuthorization
		}

		alarms, err := svc.ListAlarms(ctx, session, req)
		if err != nil {
			return nil, err
		}

		return alarmsPageRes{
			AlarmsPage: alarms,
		}, nil
	}
}

func deleteAlarmEndpoint(svc alarms.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(entityReq)
		if err := req.validate(); err != nil {
			return nil, errors.Wrap(apiutil.ErrValidation, err)
		}

		session, ok := ctx.Value(api.SessionKey).(authn.Session)
		if !ok {
			return nil, svcerr.ErrAuthorization
		}

		if err := svc.DeleteAlarm(ctx, session, req.ID); err != nil {
			return nil, err
		}

		return nil, nil
	}
}

func assignAlarmEndpoint(svc alarms.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(assignAlarmReq)
		if err := req.validate(); err != nil {
			return nil, errors.Wrap(apiutil.ErrValidation, err)
		}

		session, ok := ctx.Value(api.SessionKey).(authn.Session)
		if !ok {
			return nil, svcerr.ErrAuthorization
		}

		if err := svc.AssignAlarm(ctx, session, req.Alarm); err != nil {
			return nil, err
		}

		return nil, nil
	}
}
