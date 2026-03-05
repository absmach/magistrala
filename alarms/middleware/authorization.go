// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package middleware

import (
	"context"

	"github.com/absmach/magistrala/alarms"
	"github.com/absmach/magistrala/alarms/operations"
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

func (am *authorizationMiddleware) CreateAlarm(ctx context.Context, alarm alarms.Alarm) error {
	return am.svc.CreateAlarm(ctx, alarm)
}

func (am *authorizationMiddleware) UpdateAlarm(ctx context.Context, session authn.Session, alarm alarms.Alarm) (alarms.Alarm, error) {
	if err := am.authorize(ctx, operations.OpUpdateAlarm, session, operations.EntityType, alarm.ID); err != nil {
		return alarms.Alarm{}, errors.Wrap(errDomainUpdateAlarms, err)
	}

	if alarm.AssigneeID != "" {
		domainUserID := auth.EncodeDomainUserID(session.DomainID, alarm.AssigneeID)
		if err := am.authz.Authorize(ctx, smqauthz.PolicyReq{
			Domain:      session.DomainID,
			SubjectType: policies.UserType,
			SubjectKind: policies.UsersKind,
			Subject:     domainUserID,
			Permission:  policies.MembershipPermission,
			ObjectType:  policies.DomainType,
			Object:      session.DomainID,
		}, nil); err != nil {
			return alarms.Alarm{}, err
		}
	}

	return am.svc.UpdateAlarm(ctx, session, alarm)
}

func (am *authorizationMiddleware) DeleteAlarm(ctx context.Context, session authn.Session, id string) error {
	if err := am.authorize(ctx, operations.OpDeleteAlarm, session, operations.EntityType, id); err != nil {
		return errors.Wrap(errDomainDeleteAlarms, err)
	}

	return am.svc.DeleteAlarm(ctx, session, id)
}

func (am *authorizationMiddleware) ListAlarms(ctx context.Context, session authn.Session, pm alarms.PageMetadata) (alarms.AlarmsPage, error) {
	if pm.DomainID == "" {
		pm.DomainID = session.DomainID
	}

	if err := am.authorize(ctx, operations.OpListAlarms, session, policies.DomainType, session.DomainID); err != nil {
		return alarms.AlarmsPage{}, errors.Wrap(errDomainViewAlarms, err)
	}

	return am.svc.ListAlarms(ctx, session, pm)
}

func (am *authorizationMiddleware) ViewAlarm(ctx context.Context, session authn.Session, id string) (alarms.Alarm, error) {
	if err := am.authorize(ctx, operations.OpViewAlarm, session, operations.EntityType, id); err != nil {
		return alarms.Alarm{}, errors.Wrap(errDomainViewAlarms, err)
	}

	return am.svc.ViewAlarm(ctx, session, id)
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
