// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package middleware

import (
	"context"

	"github.com/absmach/magistrala/alarms"
	"github.com/absmach/magistrala/alarms/operations"
	"github.com/absmach/magistrala/auth"
	"github.com/absmach/magistrala/internal/atom"
	"github.com/absmach/magistrala/pkg/authn"
	smqauthz "github.com/absmach/magistrala/pkg/authz"
	"github.com/absmach/magistrala/pkg/errors"
	svcerr "github.com/absmach/magistrala/pkg/errors/service"
	"github.com/absmach/magistrala/pkg/permissions"
	"github.com/absmach/magistrala/pkg/policies"
)

var (
	errDomainUpdateAlarms = errors.New("not authorized to update alarms in domain")
	errDomainDeleteAlarms = errors.New("not authorized to delete alarms in domain")
	errDomainViewAlarms   = errors.New("not authorized to view alarms in domain")
)

type authorizationMiddleware struct {
	svc         alarms.Service
	authz       smqauthz.Authorization
	atomAuthz   atom.Authorizer
	entitiesOps permissions.EntitiesOperations[permissions.Operation]
}

var _ alarms.Service = (*authorizationMiddleware)(nil)

const (
	atomObjectKindResource      = "resource"
	atomObjectTypeResourceRule  = "resource:" + atom.KindRule
	atomAuthorizedRulePageLimit = 100
)

type atomAuthorizedObjectLister interface {
	AuthorizedObjectIDs(ctx context.Context, q atom.AuthorizedObjectIDsQuery) (atom.AuthorizedObjectIDs, error)
}

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

func NewAtomAuthorizationMiddleware(svc alarms.Service, authz atom.Authorizer, entitiesOps permissions.EntitiesOperations[permissions.Operation]) (alarms.Service, error) {
	if err := entitiesOps.Validate(); err != nil {
		return nil, err
	}

	return &authorizationMiddleware{
		svc:         svc,
		atomAuthz:   authz,
		entitiesOps: entitiesOps,
	}, nil
}

func (am *authorizationMiddleware) CreateAlarm(ctx context.Context, alarm alarms.Alarm) (alarms.Alarm, error) {
	return am.svc.CreateAlarm(ctx, alarm)
}

func (am *authorizationMiddleware) UpdateAlarm(ctx context.Context, session authn.Session, alarm alarms.Alarm) (alarms.Alarm, error) {
	current, err := am.svc.ViewAlarm(ctx, session, alarm.ID)
	if err != nil {
		return alarms.Alarm{}, err
	}

	if len(alarm.Metadata) > 0 {
		if err := am.authorizeAlarmOrRule(ctx, operations.OpUpdateAlarm, session, current); err != nil {
			return alarms.Alarm{}, errors.Wrap(errDomainUpdateAlarms, err)
		}
	}

	if alarm.AssigneeID != "" {
		if err := am.authorizeAlarmOrRule(ctx, operations.OpAssignAlarm, session, current); err != nil {
			return alarms.Alarm{}, errors.Wrap(errDomainUpdateAlarms, err)
		}
		if am.atomAuthz == nil {
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
	}

	if alarm.AcknowledgedBy != "" {
		if err := am.authorizeAlarmOrRule(ctx, operations.OpAcknowledgeAlarm, session, current); err != nil {
			return alarms.Alarm{}, errors.Wrap(errDomainUpdateAlarms, err)
		}
	}

	if alarm.ResolvedBy != "" {
		if err := am.authorizeAlarmOrRule(ctx, operations.OpResolveAlarm, session, current); err != nil {
			return alarms.Alarm{}, errors.Wrap(errDomainUpdateAlarms, err)
		}
	}

	return am.svc.UpdateAlarm(ctx, session, alarm)
}

func (am *authorizationMiddleware) DeleteAlarm(ctx context.Context, session authn.Session, id string) error {
	alarm, err := am.svc.ViewAlarm(ctx, session, id)
	if err != nil {
		return err
	}
	if err := am.authorizeAlarmOrRule(ctx, operations.OpDeleteAlarm, session, alarm); err != nil {
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
		if err := am.authorizeTenantAlarm(ctx, operations.OpViewAlarm, session); err != nil {
			if pm.RuleID != "" {
				if ruleErr := am.authorizeRuleAlarmRead(ctx, session, pm.RuleID); ruleErr != nil {
					return alarms.AlarmsPage{}, errors.Wrap(errDomainViewAlarms, err)
				}
				break
			}
			ruleIDs, ruleErr := am.authorizedReadableRuleIDs(ctx, session)
			if ruleErr != nil {
				return alarms.AlarmsPage{}, errors.Wrap(errDomainViewAlarms, err)
			}
			if len(ruleIDs) == 0 {
				return alarms.AlarmsPage{
					Offset: pm.Offset,
					Limit:  pm.Limit,
					Alarms: []alarms.Alarm{},
				}, nil
			}
			pm.RuleIDs = ruleIDs
		}
	default:
		return alarms.AlarmsPage{}, err
	}

	return am.svc.ListAlarms(ctx, session, pm)
}

func (am *authorizationMiddleware) ViewAlarm(ctx context.Context, session authn.Session, id string) (alarms.Alarm, error) {
	alarm, err := am.svc.ViewAlarm(ctx, session, id)
	if err != nil {
		return alarms.Alarm{}, err
	}
	if err := am.authorizeViewAlarm(ctx, session, alarm); err != nil {
		return alarms.Alarm{}, errors.Wrap(errDomainViewAlarms, err)
	}

	return alarm, nil
}

func (am *authorizationMiddleware) authorizeAlarmOrRule(ctx context.Context, op permissions.Operation, session authn.Session, alarm alarms.Alarm) error {
	tenantErr := am.authorizeTenantAlarm(ctx, op, session)
	if tenantErr == nil {
		return nil
	}
	if alarm.RuleID == "" {
		return tenantErr
	}
	if err := am.authorize(ctx, op, session, policies.RulesType, alarm.RuleID, atom.KindRule); err != nil {
		return tenantErr
	}
	return nil
}

func (am *authorizationMiddleware) authorizeViewAlarm(ctx context.Context, session authn.Session, alarm alarms.Alarm) error {
	tenantErr := am.authorizeTenantAlarm(ctx, operations.OpViewAlarm, session)
	if tenantErr == nil {
		return nil
	}
	if alarm.RuleID == "" {
		return tenantErr
	}
	if err := am.authorizeRuleAlarmRead(ctx, session, alarm.RuleID); err != nil {
		return tenantErr
	}
	return nil
}

func (am *authorizationMiddleware) authorizeTenantAlarm(ctx context.Context, op permissions.Operation, session authn.Session) error {
	return am.authorize(ctx, op, session, policies.DomainType, session.DomainID, atom.KindAlarm)
}

func (am *authorizationMiddleware) authorizeRuleAlarmRead(ctx context.Context, session authn.Session, ruleID string) error {
	if am.atomAuthz != nil {
		return am.authorize(ctx, operations.OpViewAlarm, session, policies.RulesType, ruleID, atom.KindRule)
	}
	perm, err := am.entitiesOps.GetPermission(operations.EntityType, operations.OpViewAlarm)
	if err != nil {
		return err
	}
	pr := smqauthz.PolicyReq{
		Domain:      session.DomainID,
		SubjectType: policies.UserType,
		SubjectKind: policies.UsersKind,
		Subject:     session.DomainUserID,
		Object:      ruleID,
		ObjectType:  policies.RulesType,
		Permission:  perm.String(),
	}
	return am.authz.Authorize(ctx, pr, nil)
}

func (am *authorizationMiddleware) authorizedReadableRuleIDs(ctx context.Context, session authn.Session) ([]string, error) {
	lister, ok := am.atomAuthz.(atomAuthorizedObjectLister)
	if !ok {
		return nil, errors.ErrAuthorization
	}
	perm, err := am.entitiesOps.GetPermission(operations.EntityType, operations.OpViewAlarm)
	if err != nil {
		return nil, err
	}

	var ids []string
	for offset := uint64(0); ; offset += atomAuthorizedRulePageLimit {
		page, err := lister.AuthorizedObjectIDs(ctx, atom.AuthorizedObjectIDsQuery{
			SubjectID:  atom.SubjectID(session),
			Action:     atom.CapabilityName(perm.String()),
			ObjectKind: atomObjectKindResource,
			ObjectType: atomObjectTypeResourceRule,
			TenantID:   session.DomainID,
			Limit:      atomAuthorizedRulePageLimit,
			Offset:     offset,
		})
		if err != nil {
			return nil, err
		}

		ids = append(ids, page.IDs...)
		if uint64(len(page.IDs)) < atomAuthorizedRulePageLimit || offset+uint64(len(page.IDs)) >= page.Total {
			break
		}
	}
	return ids, nil
}

func (am *authorizationMiddleware) authorize(ctx context.Context, op permissions.Operation, session authn.Session, objType, obj, resourceKind string) error {
	perm, err := am.entitiesOps.GetPermission(operations.EntityType, op)
	if err != nil {
		return err
	}
	if am.atomAuthz != nil {
		return atom.Authorize(ctx, am.atomAuthz, session, perm.String(), objType, obj, resourceKind)
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
			EntityID:   auth.AnyIDs,
			EntityType: auth.RulesType.String(),
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
	if am.atomAuthz != nil {
		return atom.Authorize(ctx, am.atomAuthz, session, policies.AdminPermission, policies.PlatformType, policies.MagistralaObject, policies.PlatformType)
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
