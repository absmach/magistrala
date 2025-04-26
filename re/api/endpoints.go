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

func updateRuleScheduleEndpoint(s re.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		session, ok := ctx.Value(api.SessionKey).(authn.Session)
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
			Page: page,
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

func generateReportEndpoint(svc re.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		session, ok := ctx.Value(api.SessionKey).(authn.Session)
		if !ok {
			return nil, svcerr.ErrAuthorization
		}

		req := request.(generateReportReq)
		if err := req.validate(); err != nil {
			return generateReportResp{}, err
		}

		res, err := svc.GenerateReport(ctx, session, re.ReportConfig{
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
		case re.DownloadReport:
			return downloadReportResp{
				File: res.File,
			}, nil
		case re.EmailReport:
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

func listReportsConfigEndpoint(svc re.Service) endpoint.Endpoint {
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

func deleteReportConfigEndpoint(svc re.Service) endpoint.Endpoint {
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

func updateReportConfigEndpoint(svc re.Service) endpoint.Endpoint {
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

func updateReportScheduleEndpoint(s re.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		session, ok := ctx.Value(api.SessionKey).(authn.Session)
		if !ok {
			return nil, svcerr.ErrAuthorization
		}

		req := request.(updateReportScheduleReq)
		if err := req.validate(); err != nil {
			return updateReportConfigRes{}, err
		}

		rpt := re.ReportConfig{
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

func viewReportConfigEndpoint(svc re.Service) endpoint.Endpoint {
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

func addReportConfigEndpoint(svc re.Service) endpoint.Endpoint {
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

func enableReportConfigEndpoint(svc re.Service) endpoint.Endpoint {
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

func disableReportConfigEndpoint(svc re.Service) endpoint.Endpoint {
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
