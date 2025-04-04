// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package middleware

import (
	"context"

	"github.com/absmach/magistrala/alarms"
	"github.com/absmach/supermq/pkg/authn"
	smqauthz "github.com/absmach/supermq/pkg/authz"
)

type authorizationMiddleware struct {
	svc   alarms.Service
	authz smqauthz.Authorization
}

var _ alarms.Service = (*authorizationMiddleware)(nil)

func NewAuthorizationMiddleware(svc alarms.Service, authz smqauthz.Authorization) alarms.Service {
	return &authorizationMiddleware{
		svc:   svc,
		authz: authz,
	}
}

func (am *authorizationMiddleware) CreateAlarm(ctx context.Context, session authn.Session, alarm alarms.Alarm) (dba alarms.Alarm, err error) {
	return am.svc.CreateAlarm(ctx, session, alarm)
}

func (am *authorizationMiddleware) UpdateAlarm(ctx context.Context, session authn.Session, alarm alarms.Alarm) (dba alarms.Alarm, err error) {
	return am.svc.UpdateAlarm(ctx, session, alarm)
}

func (am *authorizationMiddleware) DeleteAlarm(ctx context.Context, session authn.Session, id string) error {
	return am.svc.DeleteAlarm(ctx, session, id)
}

func (am *authorizationMiddleware) ListAlarms(ctx context.Context, session authn.Session, pm alarms.PageMetadata) (alarms.AlarmsPage, error) {
	return am.svc.ListAlarms(ctx, session, pm)
}

func (am *authorizationMiddleware) ViewAlarm(ctx context.Context, session authn.Session, id string) (alarms.Alarm, error) {
	return am.svc.ViewAlarm(ctx, session, id)
}

func (am *authorizationMiddleware) AssignAlarm(ctx context.Context, session authn.Session, alarm alarms.Alarm) (err error) {
	return am.svc.AssignAlarm(ctx, session, alarm)
}
