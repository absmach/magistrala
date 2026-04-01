// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package middleware

import (
	"context"

	"github.com/absmach/supermq/alarms"
	"github.com/absmach/supermq/alarms/operations"
	"github.com/absmach/supermq/auth"
	"github.com/absmach/supermq/pkg/authn"
	smqauthz "github.com/absmach/supermq/pkg/authz"
	"github.com/absmach/supermq/pkg/errors"
	svcerr "github.com/absmach/supermq/pkg/errors/service"
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
	if len(alarm.Metadata) > 0 {
		if err := am.authorize(ctx, operations.OpUpdateAlarm, session, policies.DomainType, session.DomainID); err != nil {
			return alarms.Alarm{}, errors.Wrap(errDomainUpdateAlarms, err)
		}
	}

	if alarm.AssigneeID != "" {
		if err := am.authorize(ctx, operations.OpAssignAlarm, session, policies.DomainType, session.DomainID); err != nil {
			return alarms.Alarm{}, errors.Wrap(errDomainUpdateAlarms, err)
		}
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

	if alarm.AcknowledgedBy != "" {
		if err := am.authorize(ctx, operations.OpAcknowledgeAlarm, session, policies.DomainType, session.DomainID); err != nil {
			return alarms.Alarm{}, errors.Wrap(errDomainUpdateAlarms, err)
		}
	}

	if alarm.ResolvedBy != "" {
		if err := am.authorize(ctx, operations.OpResolveAlarm, session, policies.DomainType, session.DomainID); err != nil {
			return alarms.Alarm{}, errors.Wrap(errDomainUpdateAlarms, err)
		}
	}

	return am.svc.UpdateAlarm(ctx, session, alarm)
}

func (am *authorizationMiddleware) DeleteAlarm(ctx context.Context, session authn.Session, id string) error {
	if err := am.authorize(ctx, operations.OpDeleteAlarm, session, policies.DomainType, session.DomainID); err != nil {
		return errors.Wrap(errDomainDeleteAlarms, err)
	}

	return am.svc.DeleteAlarm(ctx, session, id)
}

func (am *authorizationMiddleware) ListAlarms(ctx context.Context, session authn.Session, pm alarms.PageMetadata) (alarms.AlarmsPage, error) {
	if pm.DomainID == "" {
		pm.DomainID = session.DomainID
	}

	switch err := am.checkSuperAdmin(ctx, session); {
	case err == nil:
		session.SuperAdmin = true
	case errors.Contains(err, svcerr.ErrSuperAdminAction):
	default:
		return alarms.AlarmsPage{}, err
	}

	return am.svc.ListAlarms(ctx, session, pm)
}

func (am *authorizationMiddleware) ViewAlarm(ctx context.Context, session authn.Session, id string) (alarms.Alarm, error) {
	if err := am.authorize(ctx, operations.OpViewAlarm, session, policies.DomainType, session.DomainID); err != nil {
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
