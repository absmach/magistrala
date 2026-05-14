// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package reports

import (
	"context"

	"github.com/absmach/magistrala/internal/atom"
	"github.com/absmach/magistrala/pkg/authn"
)

type atomService struct {
	Service
	projector atom.Projector
}

func WithAtom(svc Service, projector atom.Projector) Service {
	if projector == nil {
		return svc
	}
	return atomService{Service: svc, projector: projector}
}

func (svc atomService) AddReportConfig(ctx context.Context, session authn.Session, cfg ReportConfig) (ReportConfig, error) {
	report, err := svc.Service.AddReportConfig(ctx, session, cfg)
	if err != nil {
		return report, err
	}
	if err := svc.projector.UpsertResource(ctx, reportProjection(report)); err != nil {
		return report, nil
	}
	return report, nil
}

func (svc atomService) UpdateReportConfig(ctx context.Context, session authn.Session, cfg ReportConfig) (ReportConfig, error) {
	report, err := svc.Service.UpdateReportConfig(ctx, session, cfg)
	return svc.upsertAfterReportChange(ctx, report, err)
}

func (svc atomService) UpdateReportSchedule(ctx context.Context, session authn.Session, cfg ReportConfig) (ReportConfig, error) {
	report, err := svc.Service.UpdateReportSchedule(ctx, session, cfg)
	return svc.upsertAfterReportChange(ctx, report, err)
}

func (svc atomService) EnableReportConfig(ctx context.Context, session authn.Session, id string) (ReportConfig, error) {
	report, err := svc.Service.EnableReportConfig(ctx, session, id)
	return svc.upsertAfterReportChange(ctx, report, err)
}

func (svc atomService) DisableReportConfig(ctx context.Context, session authn.Session, id string) (ReportConfig, error) {
	report, err := svc.Service.DisableReportConfig(ctx, session, id)
	return svc.upsertAfterReportChange(ctx, report, err)
}

func (svc atomService) RemoveReportConfig(ctx context.Context, session authn.Session, id string) error {
	if err := svc.Service.RemoveReportConfig(ctx, session, id); err != nil {
		return err
	}
	_ = svc.projector.DeleteResource(ctx, id)
	return nil
}

func (svc atomService) upsertAfterReportChange(ctx context.Context, report ReportConfig, err error) (ReportConfig, error) {
	if err != nil {
		return report, err
	}
	if err := svc.projector.UpsertResource(ctx, reportProjection(report)); err != nil {
		return report, nil
	}
	return report, nil
}

func reportProjection(r ReportConfig) atom.Resource {
	res := atom.ResourceFromFields(atom.ObjectFields{
		ID:          r.ID,
		Kind:        atom.KindReport,
		Name:        r.Name,
		TenantID:    r.DomainID,
		OwnerID:     r.CreatedBy,
		Status:      r.Status.String(),
		Metadata:    map[string]any{"description": r.Description},
		CreatedBy:   r.CreatedBy,
		UpdatedBy:   r.UpdatedBy,
		CreatedAt:   r.CreatedAt,
		UpdatedAt:   r.UpdatedAt,
		Description: r.Description,
	})
	res.Attributes["scheduled_at"] = r.Schedule.Time
	return res
}
