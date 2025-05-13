// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package middleware

import (
	"context"

	"github.com/absmach/magistrala/reports"
	"github.com/absmach/supermq/pkg/authn"
	smqauthz "github.com/absmach/supermq/pkg/authz"
	"github.com/absmach/supermq/pkg/errors"
	"github.com/absmach/supermq/pkg/policies"
)

var (
	errDomainCreateConfigs   = errors.New("not authorized to create report configs in domain")
	errDomainViewConfigs     = errors.New("not authorized to view report configs in domain")
	errDomainUpdateConfigs   = errors.New("not authorized to update report configs in domain")
	errDomainDeleteConfigs   = errors.New("not authorized to delete report configs in domain")
	errDomainGenerateReports = errors.New("not authorized to generate reports in domain")
)

type authorizationMiddleware struct {
	svc   reports.Service
	authz smqauthz.Authorization
}

// AuthorizationMiddleware adds authorization to the reports service.
func AuthorizationMiddleware(svc reports.Service, authz smqauthz.Authorization) (reports.Service, error) {
	return &authorizationMiddleware{
		svc:   svc,
		authz: authz,
	}, nil
}

func (am *authorizationMiddleware) AddReportConfig(ctx context.Context, session authn.Session, cfg reports.ReportConfig) (reports.ReportConfig, error) {
	if err := am.authorize(ctx, smqauthz.PolicyReq{
		Domain:      session.DomainID,
		SubjectType: policies.UserType,
		SubjectKind: policies.UsersKind,
		Subject:     session.DomainUserID,
		Object:      session.DomainID,
		ObjectType:  policies.DomainType,
		Permission:  policies.MembershipPermission,
	}); err != nil {
		return reports.ReportConfig{}, errors.Wrap(errDomainCreateConfigs, err)
	}

	return am.svc.AddReportConfig(ctx, session, cfg)
}

func (am *authorizationMiddleware) ViewReportConfig(ctx context.Context, session authn.Session, id string) (reports.ReportConfig, error) {
	if err := am.authorize(ctx, smqauthz.PolicyReq{
		Domain:      session.DomainID,
		SubjectType: policies.UserType,
		SubjectKind: policies.UsersKind,
		Subject:     session.DomainUserID,
		Object:      session.DomainID,
		ObjectType:  policies.DomainType,
		Permission:  policies.MembershipPermission,
	}); err != nil {
		return reports.ReportConfig{}, errors.Wrap(errDomainViewConfigs, err)
	}

	return am.svc.ViewReportConfig(ctx, session, id)
}

func (am *authorizationMiddleware) UpdateReportConfig(ctx context.Context, session authn.Session, cfg reports.ReportConfig) (reports.ReportConfig, error) {
	if err := am.authorize(ctx, smqauthz.PolicyReq{
		Domain:      session.DomainID,
		SubjectType: policies.UserType,
		SubjectKind: policies.UsersKind,
		Subject:     session.DomainUserID,
		Object:      session.DomainID,
		ObjectType:  policies.DomainType,
		Permission:  policies.MembershipPermission,
	}); err != nil {
		return reports.ReportConfig{}, errors.Wrap(errDomainUpdateConfigs, err)
	}

	return am.svc.UpdateReportConfig(ctx, session, cfg)
}

func (am *authorizationMiddleware) UpdateReportSchedule(ctx context.Context, session authn.Session, cfg reports.ReportConfig) (reports.ReportConfig, error) {
	if err := am.authorize(ctx, smqauthz.PolicyReq{
		Domain:      session.DomainID,
		SubjectType: policies.UserType,
		SubjectKind: policies.UsersKind,
		Subject:     session.DomainUserID,
		Object:      session.DomainID,
		ObjectType:  policies.DomainType,
		Permission:  policies.MembershipPermission,
	}); err != nil {
		return reports.ReportConfig{}, errors.Wrap(errDomainDeleteConfigs, err)
	}

	return am.svc.UpdateReportSchedule(ctx, session, cfg)
}

func (am *authorizationMiddleware) RemoveReportConfig(ctx context.Context, session authn.Session, id string) error {
	if err := am.authorize(ctx, smqauthz.PolicyReq{
		Domain:      session.DomainID,
		SubjectType: policies.UserType,
		SubjectKind: policies.UsersKind,
		Subject:     session.DomainUserID,
		Object:      session.DomainID,
		ObjectType:  policies.DomainType,
		Permission:  policies.MembershipPermission,
	}); err != nil {
		return errors.Wrap(errDomainDeleteConfigs, err)
	}

	return am.svc.RemoveReportConfig(ctx, session, id)
}

func (am *authorizationMiddleware) ListReportsConfig(ctx context.Context, session authn.Session, pm reports.PageMeta) (reports.ReportConfigPage, error) {
	if err := am.authorize(ctx, smqauthz.PolicyReq{
		Domain:      session.DomainID,
		SubjectType: policies.UserType,
		SubjectKind: policies.UsersKind,
		Subject:     session.DomainUserID,
		Object:      session.DomainID,
		ObjectType:  policies.DomainType,
		Permission:  policies.MembershipPermission,
	}); err != nil {
		return reports.ReportConfigPage{}, errors.Wrap(errDomainViewConfigs, err)
	}

	return am.svc.ListReportsConfig(ctx, session, pm)
}

func (am *authorizationMiddleware) EnableReportConfig(ctx context.Context, session authn.Session, id string) (reports.ReportConfig, error) {
	if err := am.authorize(ctx, smqauthz.PolicyReq{
		Domain:      session.DomainID,
		SubjectType: policies.UserType,
		SubjectKind: policies.UsersKind,
		Subject:     session.DomainUserID,
		Object:      session.DomainID,
		ObjectType:  policies.DomainType,
		Permission:  policies.MembershipPermission,
	}); err != nil {
		return reports.ReportConfig{}, errors.Wrap(errDomainUpdateConfigs, err)
	}

	return am.svc.EnableReportConfig(ctx, session, id)
}

func (am *authorizationMiddleware) DisableReportConfig(ctx context.Context, session authn.Session, id string) (reports.ReportConfig, error) {
	if err := am.authorize(ctx, smqauthz.PolicyReq{
		Domain:      session.DomainID,
		SubjectType: policies.UserType,
		SubjectKind: policies.UsersKind,
		Subject:     session.DomainUserID,
		Object:      session.DomainID,
		ObjectType:  policies.DomainType,
		Permission:  policies.MembershipPermission,
	}); err != nil {
		return reports.ReportConfig{}, errors.Wrap(errDomainUpdateConfigs, err)
	}

	return am.svc.DisableReportConfig(ctx, session, id)
}

func (am *authorizationMiddleware) GenerateReport(ctx context.Context, session authn.Session, config reports.ReportConfig, action reports.ReportAction) (reports.ReportPage, error) {
	if err := am.authorize(ctx, smqauthz.PolicyReq{
		Domain:      session.DomainID,
		SubjectType: policies.UserType,
		SubjectKind: policies.UsersKind,
		Subject:     session.DomainUserID,
		Object:      session.DomainID,
		ObjectType:  policies.DomainType,
		Permission:  policies.MembershipPermission,
	}); err != nil {
		return reports.ReportPage{}, errors.Wrap(errDomainGenerateReports, err)
	}

	return am.svc.GenerateReport(ctx, session, config, action)
}

func (am *authorizationMiddleware) StartScheduler(ctx context.Context) error {
	return am.svc.StartScheduler(ctx)
}

func (am *authorizationMiddleware) authorize(ctx context.Context, pr smqauthz.PolicyReq) error {
	if err := am.authz.Authorize(ctx, pr); err != nil {
		return err
	}
	return nil
}
