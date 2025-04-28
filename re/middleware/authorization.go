// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package middleware

import (
	"context"

	"github.com/absmach/magistrala/re"
	"github.com/absmach/supermq/pkg/authn"
	smqauthz "github.com/absmach/supermq/pkg/authz"
	"github.com/absmach/supermq/pkg/errors"
	"github.com/absmach/supermq/pkg/messaging"
	"github.com/absmach/supermq/pkg/policies"
)

var (
	errDomainCreateConfigs   = errors.New("not authorized to create report configs in domain")
	errDomainViewConfigs     = errors.New("not authorized to view report configs in domain")
	errDomainUpdateConfigs   = errors.New("not authorized to update report configs in domain")
	errDomainDeleteConfigs   = errors.New("not authorized to delete report configs in domain")
	errDomainCreateRules     = errors.New("not authorized to create rules in domain")
	errDomainViewRules       = errors.New("not authorized to view rules in domain")
	errDomainUpdateRules     = errors.New("not authorized to update rules in domain")
	errDomainDeleteRules     = errors.New("not authorized to delete rules in domain")
	errDomainGenerateReports = errors.New("not authorized to generate reports in domain")
)

type authorizationMiddleware struct {
	svc   re.Service
	authz smqauthz.Authorization
}

// AuthorizationMiddleware adds authorization to the re service.
func AuthorizationMiddleware(svc re.Service, authz smqauthz.Authorization) (re.Service, error) {
	return &authorizationMiddleware{
		svc:   svc,
		authz: authz,
	}, nil
}

func (am *authorizationMiddleware) AddRule(ctx context.Context, session authn.Session, r re.Rule) (re.Rule, error) {
	if err := am.authorize(ctx, smqauthz.PolicyReq{
		Domain:      session.DomainID,
		SubjectType: policies.UserType,
		SubjectKind: policies.UsersKind,
		Subject:     session.DomainUserID,
		Object:      session.DomainID,
		ObjectType:  policies.DomainType,
		Permission:  policies.MembershipPermission,
	}); err != nil {
		return re.Rule{}, errors.Wrap(errDomainCreateRules, err)
	}

	return am.svc.AddRule(ctx, session, r)
}

func (am *authorizationMiddleware) ViewRule(ctx context.Context, session authn.Session, id string) (re.Rule, error) {
	if err := am.authorize(ctx, smqauthz.PolicyReq{
		Domain:      session.DomainID,
		SubjectType: policies.UserType,
		SubjectKind: policies.UsersKind,
		Subject:     session.DomainUserID,
		Object:      session.DomainID,
		ObjectType:  policies.DomainType,
		Permission:  policies.MembershipPermission,
	}); err != nil {
		return re.Rule{}, errors.Wrap(errDomainViewRules, err)
	}

	return am.svc.ViewRule(ctx, session, id)
}

func (am *authorizationMiddleware) UpdateRule(ctx context.Context, session authn.Session, r re.Rule) (re.Rule, error) {
	if err := am.authorize(ctx, smqauthz.PolicyReq{
		Domain:      session.DomainID,
		SubjectType: policies.UserType,
		SubjectKind: policies.UsersKind,
		Subject:     session.DomainUserID,
		Object:      session.DomainID,
		ObjectType:  policies.DomainType,
		Permission:  policies.MembershipPermission,
	}); err != nil {
		return re.Rule{}, errors.Wrap(errDomainUpdateRules, err)
	}

	return am.svc.UpdateRule(ctx, session, r)
}

func (am *authorizationMiddleware) UpdateRuleSchedule(ctx context.Context, session authn.Session, r re.Rule) (re.Rule, error) {
	if err := am.authorize(ctx, smqauthz.PolicyReq{
		Domain:      session.DomainID,
		SubjectType: policies.UserType,
		SubjectKind: policies.UsersKind,
		Subject:     session.DomainUserID,
		Object:      session.DomainID,
		ObjectType:  policies.DomainType,
		Permission:  policies.MembershipPermission,
	}); err != nil {
		return re.Rule{}, errors.Wrap(errDomainUpdateRules, err)
	}

	return am.svc.UpdateRuleSchedule(ctx, session, r)
}

func (am *authorizationMiddleware) ListRules(ctx context.Context, session authn.Session, pm re.PageMeta) (re.Page, error) {
	if err := am.authorize(ctx, smqauthz.PolicyReq{
		Domain:      session.DomainID,
		SubjectType: policies.UserType,
		SubjectKind: policies.UsersKind,
		Subject:     session.DomainUserID,
		Object:      session.DomainID,
		ObjectType:  policies.DomainType,
		Permission:  policies.MembershipPermission,
	}); err != nil {
		return re.Page{}, errors.Wrap(errDomainViewRules, err)
	}

	return am.svc.ListRules(ctx, session, pm)
}

func (am *authorizationMiddleware) RemoveRule(ctx context.Context, session authn.Session, id string) error {
	if err := am.authorize(ctx, smqauthz.PolicyReq{
		Domain:      session.DomainID,
		SubjectType: policies.UserType,
		SubjectKind: policies.UsersKind,
		Subject:     session.DomainUserID,
		Object:      session.DomainID,
		ObjectType:  policies.DomainType,
		Permission:  policies.MembershipPermission,
	}); err != nil {
		return errors.Wrap(errDomainDeleteRules, err)
	}

	return am.svc.RemoveRule(ctx, session, id)
}

func (am *authorizationMiddleware) EnableRule(ctx context.Context, session authn.Session, id string) (re.Rule, error) {
	if err := am.authorize(ctx, smqauthz.PolicyReq{
		Domain:      session.DomainID,
		SubjectType: policies.UserType,
		SubjectKind: policies.UsersKind,
		Subject:     session.DomainUserID,
		Object:      session.DomainID,
		ObjectType:  policies.DomainType,
		Permission:  policies.MembershipPermission,
	}); err != nil {
		return re.Rule{}, errors.Wrap(errDomainUpdateRules, err)
	}

	return am.svc.EnableRule(ctx, session, id)
}

func (am *authorizationMiddleware) DisableRule(ctx context.Context, session authn.Session, id string) (re.Rule, error) {
	if err := am.authorize(ctx, smqauthz.PolicyReq{
		Domain:      session.DomainID,
		SubjectType: policies.UserType,
		SubjectKind: policies.UsersKind,
		Subject:     session.DomainUserID,
		Object:      session.DomainID,
		ObjectType:  policies.DomainType,
		Permission:  policies.MembershipPermission,
	}); err != nil {
		return re.Rule{}, errors.Wrap(errDomainUpdateRules, err)
	}

	return am.svc.DisableRule(ctx, session, id)
}

func (am *authorizationMiddleware) AddReportConfig(ctx context.Context, session authn.Session, cfg re.ReportConfig) (re.ReportConfig, error) {
	if err := am.authorize(ctx, smqauthz.PolicyReq{
		Domain:      session.DomainID,
		SubjectType: policies.UserType,
		SubjectKind: policies.UsersKind,
		Subject:     session.DomainUserID,
		Object:      session.DomainID,
		ObjectType:  policies.DomainType,
		Permission:  policies.MembershipPermission,
	}); err != nil {
		return re.ReportConfig{}, errors.Wrap(errDomainCreateConfigs, err)
	}

	return am.svc.AddReportConfig(ctx, session, cfg)
}

func (am *authorizationMiddleware) ViewReportConfig(ctx context.Context, session authn.Session, id string) (re.ReportConfig, error) {
	if err := am.authorize(ctx, smqauthz.PolicyReq{
		Domain:      session.DomainID,
		SubjectType: policies.UserType,
		SubjectKind: policies.UsersKind,
		Subject:     session.DomainUserID,
		Object:      session.DomainID,
		ObjectType:  policies.DomainType,
		Permission:  policies.MembershipPermission,
	}); err != nil {
		return re.ReportConfig{}, errors.Wrap(errDomainViewConfigs, err)
	}

	return am.svc.ViewReportConfig(ctx, session, id)
}

func (am *authorizationMiddleware) UpdateReportConfig(ctx context.Context, session authn.Session, cfg re.ReportConfig) (re.ReportConfig, error) {
	if err := am.authorize(ctx, smqauthz.PolicyReq{
		Domain:      session.DomainID,
		SubjectType: policies.UserType,
		SubjectKind: policies.UsersKind,
		Subject:     session.DomainUserID,
		Object:      session.DomainID,
		ObjectType:  policies.DomainType,
		Permission:  policies.MembershipPermission,
	}); err != nil {
		return re.ReportConfig{}, errors.Wrap(errDomainUpdateConfigs, err)
	}

	return am.svc.UpdateReportConfig(ctx, session, cfg)
}

func (am *authorizationMiddleware) UpdateReportSchedule(ctx context.Context, session authn.Session, cfg re.ReportConfig) (re.ReportConfig, error) {
	if err := am.authorize(ctx, smqauthz.PolicyReq{
		Domain:      session.DomainID,
		SubjectType: policies.UserType,
		SubjectKind: policies.UsersKind,
		Subject:     session.DomainUserID,
		Object:      session.DomainID,
		ObjectType:  policies.DomainType,
		Permission:  policies.MembershipPermission,
	}); err != nil {
		return re.ReportConfig{}, errors.Wrap(errDomainDeleteConfigs, err)
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

func (am *authorizationMiddleware) ListReportsConfig(ctx context.Context, session authn.Session, pm re.PageMeta) (re.ReportConfigPage, error) {
	if err := am.authorize(ctx, smqauthz.PolicyReq{
		Domain:      session.DomainID,
		SubjectType: policies.UserType,
		SubjectKind: policies.UsersKind,
		Subject:     session.DomainUserID,
		Object:      session.DomainID,
		ObjectType:  policies.DomainType,
		Permission:  policies.MembershipPermission,
	}); err != nil {
		return re.ReportConfigPage{}, errors.Wrap(errDomainViewConfigs, err)
	}

	return am.svc.ListReportsConfig(ctx, session, pm)
}

func (am *authorizationMiddleware) EnableReportConfig(ctx context.Context, session authn.Session, id string) (re.ReportConfig, error) {
	if err := am.authorize(ctx, smqauthz.PolicyReq{
		Domain:      session.DomainID,
		SubjectType: policies.UserType,
		SubjectKind: policies.UsersKind,
		Subject:     session.DomainUserID,
		Object:      session.DomainID,
		ObjectType:  policies.DomainType,
		Permission:  policies.MembershipPermission,
	}); err != nil {
		return re.ReportConfig{}, errors.Wrap(errDomainUpdateConfigs, err)
	}

	return am.svc.EnableReportConfig(ctx, session, id)
}

func (am *authorizationMiddleware) DisableReportConfig(ctx context.Context, session authn.Session, id string) (re.ReportConfig, error) {
	if err := am.authorize(ctx, smqauthz.PolicyReq{
		Domain:      session.DomainID,
		SubjectType: policies.UserType,
		SubjectKind: policies.UsersKind,
		Subject:     session.DomainUserID,
		Object:      session.DomainID,
		ObjectType:  policies.DomainType,
		Permission:  policies.MembershipPermission,
	}); err != nil {
		return re.ReportConfig{}, errors.Wrap(errDomainUpdateConfigs, err)
	}

	return am.svc.DisableReportConfig(ctx, session, id)
}

func (am *authorizationMiddleware) GenerateReport(ctx context.Context, session authn.Session, config re.ReportConfig, action re.ReportAction) (re.ReportPage, error) {
	if err := am.authorize(ctx, smqauthz.PolicyReq{
		Domain:      session.DomainID,
		SubjectType: policies.UserType,
		SubjectKind: policies.UsersKind,
		Subject:     session.DomainUserID,
		Object:      session.DomainID,
		ObjectType:  policies.DomainType,
		Permission:  policies.MembershipPermission,
	}); err != nil {
		return re.ReportPage{}, errors.Wrap(errDomainGenerateReports, err)
	}

	return am.svc.GenerateReport(ctx, session, config, action)
}

func (am *authorizationMiddleware) StartScheduler(ctx context.Context) error {
	return am.svc.StartScheduler(ctx)
}

func (am *authorizationMiddleware) Handle(msg *messaging.Message) error {
	return am.svc.Handle(msg)
}

func (am *authorizationMiddleware) Cancel() error {
	return am.svc.Cancel()
}

func (am *authorizationMiddleware) authorize(ctx context.Context, pr smqauthz.PolicyReq) error {
	if err := am.authz.Authorize(ctx, pr); err != nil {
		return err
	}
	return nil
}
