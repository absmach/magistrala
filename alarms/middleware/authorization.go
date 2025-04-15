// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package middleware

import (
	"context"

	"github.com/absmach/magistrala/alarms"
	"github.com/absmach/supermq/auth"
	"github.com/absmach/supermq/pkg/authn"
	smqauthz "github.com/absmach/supermq/pkg/authz"
	"github.com/absmach/supermq/pkg/policies"
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

func (am *authorizationMiddleware) CreateAlarm(ctx context.Context, alarm alarms.Alarm) (err error) {
	return am.svc.CreateAlarm(ctx, alarm)
}

func (am *authorizationMiddleware) UpdateAlarm(ctx context.Context, session authn.Session, alarm alarms.Alarm) (dba alarms.Alarm, err error) {
	// if assignee is present check if assignee is member of domain

	req := smqauthz.PolicyReq{
		Domain:      session.DomainID,
		SubjectType: policies.UserType,
		SubjectKind: policies.UsersKind,
		Subject:     session.DomainUserID,
		Permission:  policies.AdminPermission,
		ObjectType:  policies.DomainType,
		Object:      session.DomainID,
	}

	if err := am.authz.Authorize(ctx, req); err != nil {
		return alarms.Alarm{}, err
	}

	if alarm.AssigneeID != "" {
		domainUserId := auth.EncodeDomainUserID(session.DomainID, alarm.AssigneeID)
		if err := am.authz.Authorize(ctx, smqauthz.PolicyReq{
			Domain:      session.DomainID,
			SubjectType: policies.UserType,
			SubjectKind: policies.UsersKind,
			Subject:     domainUserId,
			Permission:  policies.MembershipPermission,
			ObjectType:  policies.DomainType,
			Object:      session.DomainID,
		}); err != nil {
			return alarms.Alarm{}, err
		}
	}

	return am.svc.UpdateAlarm(ctx, session, alarm)
}

func (am *authorizationMiddleware) DeleteAlarm(ctx context.Context, session authn.Session, id string) error {
	req := smqauthz.PolicyReq{
		SubjectType: policies.UserType,
		SubjectKind: policies.UsersKind,
		Subject:     session.DomainUserID,
		Permission:  policies.AdminPermission,
		ObjectType:  policies.DomainType,
		Object:      session.DomainID,
	}

	if err := am.authz.Authorize(ctx, req); err != nil {
		return err
	}

	return am.svc.DeleteAlarm(ctx, session, id)
}

func (am *authorizationMiddleware) ListAlarms(ctx context.Context, session authn.Session, pm alarms.PageMetadata) (alarms.AlarmsPage, error) {
	if pm.DomainID == "" {
		pm.DomainID = session.DomainID
	}

	req := smqauthz.PolicyReq{
		Domain:      session.DomainID,
		SubjectType: policies.UserType,
		SubjectKind: policies.UsersKind,
		Subject:     session.DomainUserID,
		Permission:  policies.MembershipPermission,
		ObjectType:  policies.DomainType,
		Object:      session.DomainID,
	}

	if err := am.authz.Authorize(ctx, req); err != nil {
		return alarms.AlarmsPage{}, err
	}

	return am.svc.ListAlarms(ctx, session, pm)
}

func (am *authorizationMiddleware) ViewAlarm(ctx context.Context, session authn.Session, id string) (alarms.Alarm, error) {
	req := smqauthz.PolicyReq{
		Domain:      session.DomainID,
		SubjectType: policies.UserType,
		SubjectKind: policies.UsersKind,
		Subject:     session.DomainUserID,
		Permission:  policies.MembershipPermission,
		ObjectType:  policies.DomainType,
		Object:      session.DomainID,
	}

	if err := am.authz.Authorize(ctx, req); err != nil {
		return alarms.Alarm{}, err
	}

	return am.svc.ViewAlarm(ctx, session, id)
}
