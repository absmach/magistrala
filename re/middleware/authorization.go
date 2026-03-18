// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package middleware

import (
	"context"

	"github.com/absmach/supermq/pkg/authn"
	smqauthz "github.com/absmach/supermq/pkg/authz"
	"github.com/absmach/supermq/pkg/errors"
	svcerr "github.com/absmach/supermq/pkg/errors/service"
	"github.com/absmach/supermq/pkg/messaging"
	"github.com/absmach/supermq/pkg/permissions"
	"github.com/absmach/supermq/pkg/policies"
	"github.com/absmach/supermq/pkg/roles"
	rolemgr "github.com/absmach/supermq/pkg/roles/rolemanager/middleware"
	"github.com/absmach/supermq/re"
	"github.com/absmach/supermq/re/operations"
)

var (
	errDomainCreateRules = errors.New("not authorized to create rules in domain")
	errDomainViewRules   = errors.New("not authorized to view rules in domain")
	errDomainUpdateRules = errors.New("not authorized to update rules in domain")
	errDomainDeleteRules = errors.New("not authorized to delete rules in domain")
)

type authorizationMiddleware struct {
	svc         re.Service
	authz       smqauthz.Authorization
	entitiesOps permissions.EntitiesOperations[permissions.Operation]
	rolemgr.RoleManagerAuthorizationMiddleware
}

// AuthorizationMiddleware adds authorization to the re service.
func AuthorizationMiddleware(svc re.Service, authz smqauthz.Authorization, entitiesOps permissions.EntitiesOperations[permissions.Operation], roleOps permissions.Operations[permissions.RoleOperation]) (re.Service, error) {
	if err := entitiesOps.Validate(); err != nil {
		return nil, err
	}
	ram, err := rolemgr.NewAuthorization(operations.EntityType, svc, authz, roleOps)
	if err != nil {
		return nil, err
	}
	return &authorizationMiddleware{
		svc:                                svc,
		authz:                              authz,
		entitiesOps:                        entitiesOps,
		RoleManagerAuthorizationMiddleware: ram,
	}, nil
}

func (am *authorizationMiddleware) AddRule(ctx context.Context, session authn.Session, r re.Rule) (re.Rule, []roles.RoleProvision, error) {
	if err := am.authorize(ctx, operations.OpAddRule, session, policies.DomainType, session.DomainID); err != nil {
		return re.Rule{}, nil, errors.Wrap(errDomainCreateRules, err)
	}

	return am.svc.AddRule(ctx, session, r)
}

func (am *authorizationMiddleware) ViewRule(ctx context.Context, session authn.Session, id string, withRoles bool) (re.Rule, error) {
	if err := am.authorize(ctx, operations.OpViewRule, session, operations.EntityType, id); err != nil {
		return re.Rule{}, errors.Wrap(errDomainViewRules, err)
	}

	return am.svc.ViewRule(ctx, session, id, withRoles)
}

func (am *authorizationMiddleware) UpdateRule(ctx context.Context, session authn.Session, r re.Rule) (re.Rule, error) {
	if err := am.authorize(ctx, operations.OpUpdateRule, session, operations.EntityType, r.ID); err != nil {
		return re.Rule{}, errors.Wrap(errDomainUpdateRules, err)
	}

	return am.svc.UpdateRule(ctx, session, r)
}

func (am *authorizationMiddleware) UpdateRuleTags(ctx context.Context, session authn.Session, r re.Rule) (re.Rule, error) {
	if err := am.authorize(ctx, operations.OpUpdateRuleTags, session, operations.EntityType, r.ID); err != nil {
		return re.Rule{}, errors.Wrap(errDomainUpdateRules, err)
	}

	return am.svc.UpdateRuleTags(ctx, session, r)
}

func (am *authorizationMiddleware) UpdateRuleSchedule(ctx context.Context, session authn.Session, r re.Rule) (re.Rule, error) {
	if err := am.authorize(ctx, operations.OpUpdateRuleSchedule, session, operations.EntityType, r.ID); err != nil {
		return re.Rule{}, errors.Wrap(errDomainUpdateRules, err)
	}

	return am.svc.UpdateRuleSchedule(ctx, session, r)
}

func (am *authorizationMiddleware) ListRules(ctx context.Context, session authn.Session, pm re.PageMeta) (re.Page, error) {
	switch err := am.checkSuperAdmin(ctx, session); {
	case err == nil:
		session.SuperAdmin = true
	case errors.Contains(err, svcerr.ErrSuperAdminAction):
	default:
		return re.Page{}, err
	}

	return am.svc.ListRules(ctx, session, pm)
}

func (am *authorizationMiddleware) RemoveRule(ctx context.Context, session authn.Session, id string) error {
	if err := am.authorize(ctx, operations.OpRemoveRule, session, operations.EntityType, id); err != nil {
		return errors.Wrap(errDomainDeleteRules, err)
	}

	return am.svc.RemoveRule(ctx, session, id)
}

func (am *authorizationMiddleware) EnableRule(ctx context.Context, session authn.Session, id string) (re.Rule, error) {
	if err := am.authorize(ctx, operations.OpEnableRule, session, operations.EntityType, id); err != nil {
		return re.Rule{}, errors.Wrap(errDomainUpdateRules, err)
	}

	return am.svc.EnableRule(ctx, session, id)
}

func (am *authorizationMiddleware) DisableRule(ctx context.Context, session authn.Session, id string) (re.Rule, error) {
	if err := am.authorize(ctx, operations.OpDisableRule, session, operations.EntityType, id); err != nil {
		return re.Rule{}, errors.Wrap(errDomainUpdateRules, err)
	}

	return am.svc.DisableRule(ctx, session, id)
}

func (am *authorizationMiddleware) StartScheduler(ctx context.Context) error {
	return am.svc.StartScheduler(ctx)
}

func (am *authorizationMiddleware) Handle(msg *messaging.Message) error {
	return am.svc.Handle(msg)
}

func (am *authorizationMiddleware) Cancel() error {
	return am.svc.Cancel()
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
		Object:      policies.SuperMQObject,
	}, nil); err != nil {
		return err
	}
	return nil
}
