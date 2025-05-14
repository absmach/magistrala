// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"context"

	"github.com/absmach/magistrala/reports"
	api "github.com/absmach/supermq/api/http"
	"github.com/absmach/supermq/pkg/authn"
	svcerr "github.com/absmach/supermq/pkg/errors/service"
	"github.com/go-kit/kit/endpoint"
)

func generateReportEndpoint(svc reports.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		session, ok := ctx.Value(api.SessionKey).(authn.Session)
		if !ok {
			return nil, svcerr.ErrAuthorization
		}

		req := request.(generateReportReq)
		if err := req.validate(); err != nil {
			return generateReportResp{}, err
		}

		res, err := svc.GenerateReport(ctx, session, reports.ReportConfig{
			Name:     req.Name,
			DomainID: req.DomainID,
			Config:   req.Config,
			Metrics:  req.Metrics,
			Email:    req.Email,
		}, req.action)
		if err != nil {
			return generateReportResp{}, err
		}

		switch req.action {
		case reports.DownloadReport:
			return downloadReportResp{
				File: res.File,
			}, nil
		case reports.EmailReport:
			return emailReportResp{}, nil
		default:
			return generateReportResp{
				Total:       res.Total,
				From:        res.From,
				To:          res.To,
				Aggregation: res.Aggregation,
				Reports:     res.Reports,
			}, nil
		}
	}
}

func listReportsConfigEndpoint(svc reports.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		session, ok := ctx.Value(api.SessionKey).(authn.Session)
		if !ok {
			return nil, svcerr.ErrAuthorization
		}

		req := request.(listReportsConfigReq)
		if err := req.validate(); err != nil {
			return listReportsConfigRes{}, err
		}

		page, err := svc.ListReportsConfig(ctx, session, req.PageMeta)
		if err != nil {
			return listReportsConfigRes{}, err
		}

		return listReportsConfigRes{
			pageRes: pageRes{
				Limit:  page.Limit,
				Offset: page.Offset,
				Total:  page.Total,
			},
			ReportConfigs: page.ReportConfigs,
		}, nil
	}
}

func deleteReportConfigEndpoint(svc reports.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		session, ok := ctx.Value(api.SessionKey).(authn.Session)
		if !ok {
			return nil, svcerr.ErrAuthorization
		}

		req := request.(deleteReportConfigReq)
		if err := req.validate(); err != nil {
			return deleteReportConfigRes{}, err
		}

		err := svc.RemoveReportConfig(ctx, session, req.ID)
		if err != nil {
			return deleteReportConfigRes{false}, err
		}

		return deleteReportConfigRes{true}, nil
	}
}

func updateReportConfigEndpoint(svc reports.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		session, ok := ctx.Value(api.SessionKey).(authn.Session)
		if !ok {
			return nil, svcerr.ErrAuthorization
		}

		req := request.(updateReportConfigReq)
		if err := req.validate(); err != nil {
			return updateReportConfigRes{}, err
		}

		cfg, err := svc.UpdateReportConfig(ctx, session, req.ReportConfig)
		if err != nil {
			return updateReportConfigRes{}, err
		}

		return updateReportConfigRes{ReportConfig: cfg}, nil
	}
}

func updateReportScheduleEndpoint(s reports.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		session, ok := ctx.Value(api.SessionKey).(authn.Session)
		if !ok {
			return nil, svcerr.ErrAuthorization
		}

		req := request.(updateReportScheduleReq)
		if err := req.validate(); err != nil {
			return updateReportConfigRes{}, err
		}

		rpt := reports.ReportConfig{
			ID:       req.id,
			Schedule: req.Schedule,
		}

		updatedReport, err := s.UpdateReportSchedule(ctx, session, rpt)
		if err != nil {
			return updateReportConfigRes{}, err
		}
		return updateReportConfigRes{ReportConfig: updatedReport}, nil
	}
}

func viewReportConfigEndpoint(svc reports.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		session, ok := ctx.Value(api.SessionKey).(authn.Session)
		if !ok {
			return nil, svcerr.ErrAuthorization
		}

		req := request.(viewReportConfigReq)
		if err := req.validate(); err != nil {
			return viewReportConfigRes{}, err
		}

		cfg, err := svc.ViewReportConfig(ctx, session, req.ID)
		if err != nil {
			return viewReportConfigRes{}, err
		}

		return viewReportConfigRes{ReportConfig: cfg}, nil
	}
}

func addReportConfigEndpoint(svc reports.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		session, ok := ctx.Value(api.SessionKey).(authn.Session)
		if !ok {
			return nil, svcerr.ErrAuthorization
		}

		req := request.(addReportConfigReq)
		if err := req.validate(); err != nil {
			return addReportConfigRes{}, err
		}

		cfg, err := svc.AddReportConfig(ctx, session, req.ReportConfig)
		if err != nil {
			return addReportConfigRes{}, err
		}

		return addReportConfigRes{
			ReportConfig: cfg,
			created:      true,
		}, nil
	}
}

func enableReportConfigEndpoint(svc reports.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		session, ok := ctx.Value(api.SessionKey).(authn.Session)
		if !ok {
			return nil, svcerr.ErrAuthorization
		}

		req := request.(updateReportStatusReq)
		if err := req.validate(); err != nil {
			return updateReportConfigRes{}, err
		}

		cfg, err := svc.EnableReportConfig(ctx, session, req.id)
		if err != nil {
			return updateReportConfigRes{}, err
		}

		return updateReportConfigRes{ReportConfig: cfg}, nil
	}
}

func disableReportConfigEndpoint(svc reports.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		session, ok := ctx.Value(api.SessionKey).(authn.Session)
		if !ok {
			return nil, svcerr.ErrAuthorization
		}

		req := request.(updateReportStatusReq)
		if err := req.validate(); err != nil {
			return updateReportConfigRes{}, err
		}

		cfg, err := svc.DisableReportConfig(ctx, session, req.id)
		if err != nil {
			return updateReportConfigRes{}, err
		}

		return updateReportConfigRes{ReportConfig: cfg}, nil
	}
}
