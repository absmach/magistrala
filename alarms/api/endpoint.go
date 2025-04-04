// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"context"

	"github.com/absmach/magistrala/alarms"
	sapi "github.com/absmach/supermq/api/http"
	apiutil "github.com/absmach/supermq/api/http/util"
	"github.com/absmach/supermq/pkg/authn"
	"github.com/absmach/supermq/pkg/errors"
	svcerr "github.com/absmach/supermq/pkg/errors/service"
	"github.com/go-kit/kit/endpoint"
)

func createAlarmEndpoint(svc alarms.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(createAlarmReq)
		if err := req.validate(); err != nil {
			return nil, errors.Wrap(apiutil.ErrValidation, err)
		}

		session, ok := ctx.Value(sapi.SessionKey).(authn.Session)
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

		session, ok := ctx.Value(sapi.SessionKey).(authn.Session)
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

		session, ok := ctx.Value(sapi.SessionKey).(authn.Session)
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
		req := request.(listAlarmsReq)
		if err := req.validate(); err != nil {
			return nil, errors.Wrap(apiutil.ErrValidation, err)
		}

		session, ok := ctx.Value(sapi.SessionKey).(authn.Session)
		if !ok {
			return nil, svcerr.ErrAuthorization
		}

		alarms, err := svc.ListAlarms(ctx, session, req.PageMetadata)
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

		session, ok := ctx.Value(sapi.SessionKey).(authn.Session)
		if !ok {
			return nil, svcerr.ErrAuthorization
		}

		if err := svc.DeleteAlarm(ctx, session, req.ID); err != nil {
			return nil, err
		}

		return nil, nil
	}
}
