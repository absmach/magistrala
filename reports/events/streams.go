// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package events

import (
	"context"

	"github.com/absmach/magistrala/pkg/authn"
	"github.com/absmach/magistrala/pkg/events"
	"github.com/absmach/magistrala/pkg/events/store"
	rmEvents "github.com/absmach/magistrala/pkg/roles/rolemanager/events"
	"github.com/absmach/magistrala/reports"
	"github.com/go-chi/chi/v5/middleware"
)

const (
	magistralaPrefix = "magistrala."
	CreateStream     = magistralaPrefix + reportCreate
	RemoveStream     = magistralaPrefix + reportRemove
)

var _ reports.Service = (*eventStore)(nil)

type eventStore struct {
	events.Publisher
	svc reports.Service
	rmEvents.RoleManagerEventStore
}

func NewEventStoreMiddleware(ctx context.Context, svc reports.Service, url string) (reports.Service, error) {
	publisher, err := store.NewPublisher(ctx, url, "reports-es-pub")
	if err != nil {
		return nil, err
	}

	res := rmEvents.NewRoleManagerEventStore("reports", reportPrefix, svc, publisher)

	return &eventStore{
		svc:                   svc,
		Publisher:             publisher,
		RoleManagerEventStore: res,
	}, nil
}

func (es *eventStore) AddReportConfig(ctx context.Context, session authn.Session, cfg reports.ReportConfig) (reports.ReportConfig, error) {
	reportCfg, err := es.svc.AddReportConfig(ctx, session, cfg)
	if err != nil {
		return reportCfg, err
	}
	event := createReportConfigEvent{
		cfg:             reportCfg,
		baseReportEvent: newBaseReportEvent(session, middleware.GetReqID(ctx)),
	}
	if err := es.Publish(ctx, CreateStream, event); err != nil {
		return reportCfg, err
	}
	return reportCfg, nil
}

func (es *eventStore) RemoveReportConfig(ctx context.Context, session authn.Session, id string) error {
	if err := es.svc.RemoveReportConfig(ctx, session, id); err != nil {
		return err
	}
	event := removeReportConfigEvent{
		id:              id,
		baseReportEvent: newBaseReportEvent(session, middleware.GetReqID(ctx)),
	}
	return es.Publish(ctx, RemoveStream, event)
}

func (es *eventStore) ViewReportConfig(ctx context.Context, session authn.Session, id string, withRoles bool) (reports.ReportConfig, error) {
	return es.svc.ViewReportConfig(ctx, session, id, withRoles)
}

func (es *eventStore) UpdateReportConfig(ctx context.Context, session authn.Session, cfg reports.ReportConfig) (reports.ReportConfig, error) {
	return es.svc.UpdateReportConfig(ctx, session, cfg)
}

func (es *eventStore) UpdateReportSchedule(ctx context.Context, session authn.Session, cfg reports.ReportConfig) (reports.ReportConfig, error) {
	return es.svc.UpdateReportSchedule(ctx, session, cfg)
}

func (es *eventStore) ListReportsConfig(ctx context.Context, session authn.Session, pm reports.PageMeta) (reports.ReportConfigPage, error) {
	return es.svc.ListReportsConfig(ctx, session, pm)
}

func (es *eventStore) EnableReportConfig(ctx context.Context, session authn.Session, id string) (reports.ReportConfig, error) {
	return es.svc.EnableReportConfig(ctx, session, id)
}

func (es *eventStore) DisableReportConfig(ctx context.Context, session authn.Session, id string) (reports.ReportConfig, error) {
	return es.svc.DisableReportConfig(ctx, session, id)
}

func (es *eventStore) UpdateReportTemplate(ctx context.Context, session authn.Session, cfg reports.ReportConfig) error {
	return es.svc.UpdateReportTemplate(ctx, session, cfg)
}

func (es *eventStore) ViewReportTemplate(ctx context.Context, session authn.Session, id string) (reports.ReportTemplate, error) {
	return es.svc.ViewReportTemplate(ctx, session, id)
}

func (es *eventStore) DeleteReportTemplate(ctx context.Context, session authn.Session, id string) error {
	return es.svc.DeleteReportTemplate(ctx, session, id)
}

func (es *eventStore) GenerateReport(ctx context.Context, session authn.Session, config reports.ReportConfig, action reports.ReportAction) (reports.ReportPage, error) {
	return es.svc.GenerateReport(ctx, session, config, action)
}

func (es *eventStore) StartScheduler(ctx context.Context) error {
	return es.svc.StartScheduler(ctx)
}
