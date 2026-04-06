// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package middleware

import (
	"context"

	"github.com/absmach/magistrala/pkg/authn"
	smqauthz "github.com/absmach/magistrala/pkg/authz"
	"github.com/absmach/magistrala/pkg/errors"
	svcerr "github.com/absmach/magistrala/pkg/errors/service"
	"github.com/absmach/magistrala/pkg/permissions"
	"github.com/absmach/magistrala/pkg/policies"
	rolemgr "github.com/absmach/magistrala/pkg/roles/rolemanager/middleware"
	"github.com/absmach/magistrala/reports"
	"github.com/absmach/magistrala/reports/operations"
)

var (
	errDomainCreateConfigs   = errors.New("not authorized to create report configs in domain")
	errDomainViewConfigs     = errors.New("not authorized to view report configs in domain")
	errDomainUpdateConfigs   = errors.New("not authorized to update report configs in domain")
	errDomainDeleteConfigs   = errors.New("not authorized to delete report configs in domain")
	errDomainGenerateReports = errors.New("not authorized to generate reports in domain")

	errDomainUpdateTemplates = errors.New("not authorized to update report templates in domain")
	errDomainRemoveTemplates = errors.New("not authorized to delete report templates in domain")
	errDomainViewTemplates   = errors.New("not authorized to view report templates in domain")
)

type authorizationMiddleware struct {
	svc         reports.Service
	authz       smqauthz.Authorization
	entitiesOps permissions.EntitiesOperations[permissions.Operation]
	rolemgr.RoleManagerAuthorizationMiddleware
}

// AuthorizationMiddleware adds authorization to the reports service.
func AuthorizationMiddleware(svc reports.Service, authz smqauthz.Authorization, entitiesOps permissions.EntitiesOperations[permissions.Operation], roleOps permissions.Operations[permissions.RoleOperation]) (reports.Service, error) {
	if err := entitiesOps.Validate(); err != nil {
		return nil, err
	}
	ram, err := rolemgr.NewAuthorization(operations.EntityType, svc, authz, roleOps)
	if err != nil {
		return nil, err
	}
	return &authorizationMiddleware{
		svc:                                svc,
		authz:                              authz,
		entitiesOps:                        entitiesOps,
		RoleManagerAuthorizationMiddleware: ram,
	}, nil
}

func (am *authorizationMiddleware) AddReportConfig(ctx context.Context, session authn.Session, cfg reports.ReportConfig) (reports.ReportConfig, error) {
	if err := am.authorize(ctx, operations.OpAddReportConfig, session, policies.DomainType, session.DomainID); err != nil {
		return reports.ReportConfig{}, errors.Wrap(errDomainCreateConfigs, err)
	}

	return am.svc.AddReportConfig(ctx, session, cfg)
}

func (am *authorizationMiddleware) ViewReportConfig(ctx context.Context, session authn.Session, id string, withRoles bool) (reports.ReportConfig, error) {
	if err := am.authorize(ctx, operations.OpViewReportConfig, session, operations.EntityType, id); err != nil {
		return reports.ReportConfig{}, errors.Wrap(errDomainViewConfigs, err)
	}

	return am.svc.ViewReportConfig(ctx, session, id, withRoles)
}

func (am *authorizationMiddleware) UpdateReportConfig(ctx context.Context, session authn.Session, cfg reports.ReportConfig) (reports.ReportConfig, error) {
	if err := am.authorize(ctx, operations.OpUpdateReportConfig, session, operations.EntityType, cfg.ID); err != nil {
		return reports.ReportConfig{}, errors.Wrap(errDomainUpdateConfigs, err)
	}

	return am.svc.UpdateReportConfig(ctx, session, cfg)
}

func (am *authorizationMiddleware) UpdateReportSchedule(ctx context.Context, session authn.Session, cfg reports.ReportConfig) (reports.ReportConfig, error) {
	if err := am.authorize(ctx, operations.OpUpdateReportSchedule, session, operations.EntityType, cfg.ID); err != nil {
		return reports.ReportConfig{}, errors.Wrap(errDomainUpdateConfigs, err)
	}

	return am.svc.UpdateReportSchedule(ctx, session, cfg)
}

func (am *authorizationMiddleware) RemoveReportConfig(ctx context.Context, session authn.Session, id string) error {
	if err := am.authorize(ctx, operations.OpRemoveReportConfig, session, operations.EntityType, id); err != nil {
		return errors.Wrap(errDomainDeleteConfigs, err)
	}

	return am.svc.RemoveReportConfig(ctx, session, id)
}

func (am *authorizationMiddleware) ListReportsConfig(ctx context.Context, session authn.Session, pm reports.PageMeta) (reports.ReportConfigPage, error) {
	switch err := am.checkSuperAdmin(ctx, session); {
	case err == nil:
		session.SuperAdmin = true
	case errors.Contains(err, svcerr.ErrSuperAdminAction):
	default:
		return reports.ReportConfigPage{}, err
	}

	return am.svc.ListReportsConfig(ctx, session, pm)
}

func (am *authorizationMiddleware) EnableReportConfig(ctx context.Context, session authn.Session, id string) (reports.ReportConfig, error) {
	if err := am.authorize(ctx, operations.OpEnableReportConfig, session, operations.EntityType, id); err != nil {
		return reports.ReportConfig{}, errors.Wrap(errDomainUpdateConfigs, err)
	}

	return am.svc.EnableReportConfig(ctx, session, id)
}

func (am *authorizationMiddleware) DisableReportConfig(ctx context.Context, session authn.Session, id string) (reports.ReportConfig, error) {
	if err := am.authorize(ctx, operations.OpDisableReportConfig, session, operations.EntityType, id); err != nil {
		return reports.ReportConfig{}, errors.Wrap(errDomainUpdateConfigs, err)
	}

	return am.svc.DisableReportConfig(ctx, session, id)
}

func (am *authorizationMiddleware) GenerateReport(ctx context.Context, session authn.Session, config reports.ReportConfig, action reports.ReportAction) (reports.ReportPage, error) {
	if err := am.authorize(ctx, operations.OpGenerateReport, session, policies.DomainType, session.DomainID); err != nil {
		return reports.ReportPage{}, errors.Wrap(errDomainGenerateReports, err)
	}

	return am.svc.GenerateReport(ctx, session, config, action)
}

func (am *authorizationMiddleware) UpdateReportTemplate(ctx context.Context, session authn.Session, cfg reports.ReportConfig) error {
	if err := am.authorize(ctx, operations.OpUpdateReportTemplate, session, operations.EntityType, cfg.ID); err != nil {
		return errors.Wrap(errDomainUpdateTemplates, err)
	}

	return am.svc.UpdateReportTemplate(ctx, session, cfg)
}

func (am *authorizationMiddleware) ViewReportTemplate(ctx context.Context, session authn.Session, id string) (reports.ReportTemplate, error) {
	if err := am.authorize(ctx, operations.OpViewReportTemplate, session, operations.EntityType, id); err != nil {
		return "", errors.Wrap(errDomainViewTemplates, err)
	}

	return am.svc.ViewReportTemplate(ctx, session, id)
}

func (am *authorizationMiddleware) DeleteReportTemplate(ctx context.Context, session authn.Session, id string) error {
	if err := am.authorize(ctx, operations.OpDeleteReportTemplate, session, operations.EntityType, id); err != nil {
		return errors.Wrap(errDomainRemoveTemplates, err)
	}

	return am.svc.DeleteReportTemplate(ctx, session, id)
}

func (am *authorizationMiddleware) StartScheduler(ctx context.Context) error {
	return am.svc.StartScheduler(ctx)
}

func (am *authorizationMiddleware) authorize(ctx context.Context, op permissions.Operation, session authn.Session, objType, obj string) error {
	perm, err := am.entitiesOps.GetPermission(operations.EntityType, op)
	if err != nil {
		return err
	}

	pr := smqauthz.PolicyReq{
		Domain:      session.DomainID,
		SubjectType: policies.UserType,
		SubjectKind: policies.UsersKind,
		Subject:     session.DomainUserID,
		Object:      obj,
		ObjectType:  objType,
		Permission:  perm.String(),
	}

	var pat *smqauthz.PATReq
	if session.PatID != "" {
		opName := am.entitiesOps.OperationName(operations.EntityType, op)
		pat = &smqauthz.PATReq{
			UserID:     session.UserID,
			PatID:      session.PatID,
			EntityID:   session.DomainID,
			EntityType: operations.EntityType,
			Operation:  opName,
			Domain:     session.DomainID,
		}
	}

	if err := am.authz.Authorize(ctx, pr, pat); err != nil {
		return err
	}

	return nil
}

func (am *authorizationMiddleware) checkSuperAdmin(ctx context.Context, session authn.Session) error {
	if session.Role != authn.SuperAdminRole {
		return svcerr.ErrSuperAdminAction
	}
	if err := am.authz.Authorize(ctx, smqauthz.PolicyReq{
		SubjectType: policies.UserType,
		Subject:     session.UserID,
		Permission:  policies.AdminPermission,
		ObjectType:  policies.PlatformType,
		Object:      policies.MagistralaObject,
	}, nil); err != nil {
		return err
	}
	return nil
}
