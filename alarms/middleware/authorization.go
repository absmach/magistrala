// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package middleware

import (
	"context"

	"github.com/absmach/magistrala/alarms"
	mgPolicies "github.com/absmach/magistrala/pkg/policies"
	"github.com/absmach/supermq/auth"
	"github.com/absmach/supermq/pkg/authn"
	smqauthz "github.com/absmach/supermq/pkg/authz"
	"github.com/absmach/supermq/pkg/errors"
	"github.com/absmach/supermq/pkg/permissions"
	"github.com/absmach/supermq/pkg/policies"
)

var (
	errDomainUpdateAlarms = errors.New("not authorized to update alarms in domain")
	errDomainDeleteAlarms = errors.New("not authorized to delete alarms in domain")
	errDomainViewAlarms   = errors.New("not authorized to view alarms in domain")
)

type authorizationMiddleware struct {
	svc         alarms.Service
	authz       smqauthz.Authorization
	entitiesOps permissions.EntitiesOperations[permissions.Operation]
}

var _ alarms.Service = (*authorizationMiddleware)(nil)

func NewAuthorizationMiddleware(svc alarms.Service, authz smqauthz.Authorization, entitiesOps permissions.EntitiesOperations[permissions.Operation]) (alarms.Service, error) {
	if err := entitiesOps.Validate(); err != nil {
		return nil, err
	}
	return &authorizationMiddleware{
		svc:         svc,
		authz:       authz,
		entitiesOps: entitiesOps,
	}, nil
}

func (am *authorizationMiddleware) CreateAlarm(ctx context.Context, alarm alarms.Alarm) (err error) {
	return am.svc.CreateAlarm(ctx, alarm)
}

func (am *authorizationMiddleware) UpdateAlarm(ctx context.Context, session authn.Session, alarm alarms.Alarm) (dba alarms.Alarm, err error) {
	// If assignee is present, check if assignee is member of domain

	if err := am.authorize(ctx, alarms.OpUpdateAlarm, session, mgPolicies.AlarmType, smqauthz.PolicyReq{
		Domain:      session.DomainID,
		SubjectType: policies.UserType,
		Subject:     session.DomainUserID,
		ObjectType:  mgPolicies.AlarmType,
		Object:      alarm.ID,
	}); err != nil {
		return alarms.Alarm{}, errors.Wrap(errDomainUpdateAlarms, err)
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
	if err := am.authorize(ctx, alarms.OpDeleteAlarm, session, mgPolicies.AlarmType, smqauthz.PolicyReq{
		Domain:      session.DomainID,
		SubjectType: policies.UserType,
		Subject:     session.DomainUserID,
		ObjectType:  mgPolicies.AlarmType,
		Object:      id,
	}); err != nil {
		return errors.Wrap(errDomainDeleteAlarms, err)
	}

	return am.svc.DeleteAlarm(ctx, session, id)
}

func (am *authorizationMiddleware) ListAlarms(ctx context.Context, session authn.Session, pm alarms.PageMetadata) (alarms.AlarmsPage, error) {
	if pm.DomainID == "" {
		pm.DomainID = session.DomainID
	}

	if err := am.authorize(ctx, alarms.OpListAlarms, session, mgPolicies.AlarmType, smqauthz.PolicyReq{
		Domain:      session.DomainID,
		SubjectType: policies.UserType,
		Subject:     session.DomainUserID,
		ObjectType:  policies.DomainType,
		Object:      session.DomainID,
	}); err != nil {
		return alarms.AlarmsPage{}, errors.Wrap(errDomainViewAlarms, err)
	}

	return am.svc.ListAlarms(ctx, session, pm)
}

func (am *authorizationMiddleware) ViewAlarm(ctx context.Context, session authn.Session, id string) (alarms.Alarm, error) {
	if err := am.authorize(ctx, alarms.OpViewAlarm, session, mgPolicies.AlarmType, smqauthz.PolicyReq{
		Domain:      session.DomainID,
		SubjectType: policies.UserType,
		Subject:     session.DomainUserID,
		ObjectType:  mgPolicies.AlarmType,
		Object:      id,
	}); err != nil {
		return alarms.Alarm{}, errors.Wrap(errDomainViewAlarms, err)
	}

	return am.svc.ViewAlarm(ctx, session, id)
}

func (am *authorizationMiddleware) authorize(ctx context.Context, op permissions.Operation, session authn.Session, entityType string, req smqauthz.PolicyReq) error {
	req.Domain = session.DomainID

	perm, err := am.entitiesOps.GetPermission(entityType, op)
	if err != nil {
		return err
	}

	req.Permission = perm

	var pat *smqauthz.PATReq
	if session.PatID != "" {
		entityID := req.Object
		opName := am.entitiesOps.OperationName(entityType, op)
		if op == alarms.OpListAlarms {
			entityID = auth.AnyIDs
		}
		pat = &smqauthz.PATReq{
			UserID:     session.UserID,
			PatID:      session.PatID,
			EntityID:   entityID,
			EntityType: mgPolicies.AlarmType,
			Operation:  opName,
			Domain:     session.DomainID,
		}
	}

	return am.authz.Authorize(ctx, req, pat)
}
