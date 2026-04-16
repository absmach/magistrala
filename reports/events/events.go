// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package events

import (
	"maps"

	"github.com/absmach/magistrala/pkg/authn"
	"github.com/absmach/magistrala/pkg/events"
	"github.com/absmach/magistrala/reports"
)

const (
	reportPrefix = "report."
	reportCreate = reportPrefix + "create"
	reportRemove = reportPrefix + "remove"
)

var (
	_ events.Event = (*createReportConfigEvent)(nil)
	_ events.Event = (*removeReportConfigEvent)(nil)
)

type baseReportEvent struct {
	session   authn.Session
	requestID string
}

func newBaseReportEvent(session authn.Session, requestID string) baseReportEvent {
	return baseReportEvent{
		session:   session,
		requestID: requestID,
	}
}

func (bre baseReportEvent) Encode() map[string]any {
	return map[string]any{
		"domain":      bre.session.DomainID,
		"user_id":     bre.session.UserID,
		"token_type":  bre.session.Type.String(),
		"super_admin": bre.session.SuperAdmin,
		"request_id":  bre.requestID,
	}
}

type createReportConfigEvent struct {
	cfg reports.ReportConfig
	baseReportEvent
}

func (e createReportConfigEvent) Encode() (map[string]any, error) {
	val := map[string]any{
		"id":   e.cfg.ID,
		"name": e.cfg.Name,
	}
	maps.Copy(val, e.baseReportEvent.Encode())
	val["operation"] = reportCreate
	return val, nil
}

type removeReportConfigEvent struct {
	id string
	baseReportEvent
}

func (e removeReportConfigEvent) Encode() (map[string]any, error) {
	val := e.baseReportEvent.Encode()
	val["id"] = e.id
	val["operation"] = reportRemove
	return val, nil
}
