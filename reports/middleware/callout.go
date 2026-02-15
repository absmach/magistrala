// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package middleware

import (
	"context"
	"time"

	mgPolicies "github.com/absmach/magistrala/pkg/policies"
	"github.com/absmach/magistrala/reports"
	"github.com/absmach/magistrala/reports/operations"
	"github.com/absmach/supermq/pkg/authn"
	"github.com/absmach/supermq/pkg/callout"
	"github.com/absmach/supermq/pkg/permissions"
	"github.com/absmach/supermq/pkg/policies"
	rolemw "github.com/absmach/supermq/pkg/roles/rolemanager/middleware"
)

var _ reports.Service = (*calloutMiddleware)(nil)

type calloutMiddleware struct {
	svc         reports.Service
	callout     callout.Callout
	entitiesOps permissions.EntitiesOperations[permissions.Operation]
	rolemw.RoleManagerCalloutMiddleware
}

const entityType = "report"

func NewCallout(svc reports.Service, callout callout.Callout, entitiesOps permissions.EntitiesOperations[permissions.Operation], roleOps permissions.Operations[permissions.RoleOperation]) (reports.Service, error) {
	call, err := rolemw.NewCallout(mgPolicies.ReportsType, svc, callout, roleOps)
	if err != nil {
		return nil, err
	}

	if err := entitiesOps.Validate(); err != nil {
		return nil, err
	}

	return &calloutMiddleware{
		svc:                          svc,
		callout:                      callout,
		entitiesOps:                  entitiesOps,
		RoleManagerCalloutMiddleware: call,
	}, nil
}

func (cm *calloutMiddleware) AddReportConfig(ctx context.Context, session authn.Session, cfg reports.ReportConfig) (reports.ReportConfig, error) {
	params := map[string]any{
		"entities": cfg,
		"count":    1,
	}

	if err := cm.callOut(ctx, session, operations.OpAddReportConfig, params); err != nil {
		return reports.ReportConfig{}, err
	}

	return cm.svc.AddReportConfig(ctx, session, cfg)
}

func (cm *calloutMiddleware) ViewReportConfig(ctx context.Context, session authn.Session, id string, withRoles bool) (reports.ReportConfig, error) {
	params := map[string]any{
		"entity_id": id,
	}

	if err := cm.callOut(ctx, session, operations.OpViewReportConfig, params); err != nil {
		return reports.ReportConfig{}, err
	}

	return cm.svc.ViewReportConfig(ctx, session, id, withRoles)
}

func (cm *calloutMiddleware) UpdateReportConfig(ctx context.Context, session authn.Session, cfg reports.ReportConfig) (reports.ReportConfig, error) {
	params := map[string]any{
		"entity_id": cfg.ID,
	}

	if err := cm.callOut(ctx, session, operations.OpUpdateReportConfig, params); err != nil {
		return reports.ReportConfig{}, err
	}

	return cm.svc.UpdateReportConfig(ctx, session, cfg)
}

func (cm *calloutMiddleware) UpdateReportSchedule(ctx context.Context, session authn.Session, cfg reports.ReportConfig) (reports.ReportConfig, error) {
	params := map[string]any{
		"entity_id": cfg.ID,
	}

	if err := cm.callOut(ctx, session, operations.OpUpdateReportSchedule, params); err != nil {
		return reports.ReportConfig{}, err
	}

	return cm.svc.UpdateReportSchedule(ctx, session, cfg)
}

func (cm *calloutMiddleware) RemoveReportConfig(ctx context.Context, session authn.Session, id string) error {
	params := map[string]any{
		"entity_id": id,
	}

	if err := cm.callOut(ctx, session, operations.OpRemoveReportConfig, params); err != nil {
		return err
	}

	return cm.svc.RemoveReportConfig(ctx, session, id)
}

func (cm *calloutMiddleware) ListReportsConfig(ctx context.Context, session authn.Session, pm reports.PageMeta) (reports.ReportConfigPage, error) {
	params := map[string]any{
		"pagemeta": pm,
	}

	if err := cm.callOut(ctx, session, operations.OpListReportsConfig, params); err != nil {
		return reports.ReportConfigPage{}, err
	}

	return cm.svc.ListReportsConfig(ctx, session, pm)
}

func (cm *calloutMiddleware) EnableReportConfig(ctx context.Context, session authn.Session, id string) (reports.ReportConfig, error) {
	params := map[string]any{
		"entity_id": id,
	}

	if err := cm.callOut(ctx, session, operations.OpEnableReportConfig, params); err != nil {
		return reports.ReportConfig{}, err
	}

	return cm.svc.EnableReportConfig(ctx, session, id)
}

func (cm *calloutMiddleware) DisableReportConfig(ctx context.Context, session authn.Session, id string) (reports.ReportConfig, error) {
	params := map[string]any{
		"entity_id": id,
	}

	if err := cm.callOut(ctx, session, operations.OpDisableReportConfig, params); err != nil {
		return reports.ReportConfig{}, err
	}

	return cm.svc.DisableReportConfig(ctx, session, id)
}

func (cm *calloutMiddleware) GenerateReport(ctx context.Context, session authn.Session, config reports.ReportConfig, action reports.ReportAction) (reports.ReportPage, error) {
	params := map[string]any{
		"entity_id": config.ID,
	}

	if err := cm.callOut(ctx, session, operations.OpGenerateReport, params); err != nil {
		return reports.ReportPage{}, err
	}

	return cm.svc.GenerateReport(ctx, session, config, action)
}

func (cm *calloutMiddleware) UpdateReportTemplate(ctx context.Context, session authn.Session, cfg reports.ReportConfig) error {
	params := map[string]any{
		"entity_id": cfg.ID,
	}

	if err := cm.callOut(ctx, session, operations.OpUpdateReportTemplate, params); err != nil {
		return err
	}

	return cm.svc.UpdateReportTemplate(ctx, session, cfg)
}

func (cm *calloutMiddleware) ViewReportTemplate(ctx context.Context, session authn.Session, id string) (reports.ReportTemplate, error) {
	params := map[string]any{
		"entity_id": id,
	}

	if err := cm.callOut(ctx, session, operations.OpViewReportTemplate, params); err != nil {
		return "", err
	}

	return cm.svc.ViewReportTemplate(ctx, session, id)
}

func (cm *calloutMiddleware) DeleteReportTemplate(ctx context.Context, session authn.Session, id string) error {
	params := map[string]any{
		"entity_id": id,
	}

	if err := cm.callOut(ctx, session, operations.OpDeleteReportTemplate, params); err != nil {
		return err
	}

	return cm.svc.DeleteReportTemplate(ctx, session, id)
}

func (cm *calloutMiddleware) StartScheduler(ctx context.Context) error {
	return cm.svc.StartScheduler(ctx)
}

func (cm *calloutMiddleware) callOut(ctx context.Context, session authn.Session, op permissions.Operation, pld map[string]any) error {
	var entityID string
	if id, ok := pld["entity_id"].(string); ok {
		entityID = id
	}

	req := callout.Request{
		BaseRequest: callout.BaseRequest{
			Operation:  cm.entitiesOps.OperationName(entityType, op),
			EntityType: entityType,
			EntityID:   entityID,
			CallerID:   session.UserID,
			CallerType: policies.UserType,
			DomainID:   session.DomainID,
			Time:       time.Now().UTC(),
		},
		Payload: pld,
	}

	if err := cm.callout.Callout(ctx, req); err != nil {
		return err
	}

	return nil
}
