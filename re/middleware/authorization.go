// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package middleware

import (
	"context"

	"github.com/absmach/magistrala/re"
	"github.com/absmach/magistrala/re/operations"
	"github.com/absmach/supermq/pkg/authn"
	smqauthz "github.com/absmach/supermq/pkg/authz"
	"github.com/absmach/supermq/pkg/errors"
	"github.com/absmach/supermq/pkg/messaging"
	"github.com/absmach/supermq/pkg/permissions"
	mgPolicies "github.com/absmach/magistrala/pkg/policies"
	"github.com/absmach/supermq/pkg/policies"
	rolemgr "github.com/absmach/supermq/pkg/roles/rolemanager/middleware"
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
	ram, err := rolemgr.NewAuthorization(mgPolicies.RuleType, svc, authz, roleOps)
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

func (am *authorizationMiddleware) AddRule(ctx context.Context, session authn.Session, r re.Rule) (re.Rule, error) {
	if err := am.authorize(ctx, session, mgPolicies.RuleType, operations.OpAddRule, smqauthz.PolicyReq{
		Domain:      session.DomainID,
		SubjectType: policies.UserType,
		SubjectKind: policies.UsersKind,
		Subject:     session.DomainUserID,
		Object:      session.DomainID,
		ObjectType:  policies.DomainType,
	}); err != nil {
		return re.Rule{}, errors.Wrap(errDomainCreateRules, err)
	}

	return am.svc.AddRule(ctx, session, r)
}

func (am *authorizationMiddleware) ViewRule(ctx context.Context, session authn.Session, id string) (re.Rule, error) {
	if err := am.authorize(ctx, session, mgPolicies.RuleType, operations.OpViewRule, smqauthz.PolicyReq{
		Domain:      session.DomainID,
		SubjectType: policies.UserType,
		SubjectKind: policies.UsersKind,
		Subject:     session.DomainUserID,
		Object:      session.DomainID,
		ObjectType:  policies.DomainType,
	}); err != nil {
		return re.Rule{}, errors.Wrap(errDomainViewRules, err)
	}

	return am.svc.ViewRule(ctx, session, id)
}

func (am *authorizationMiddleware) UpdateRule(ctx context.Context, session authn.Session, r re.Rule) (re.Rule, error) {
	if err := am.authorize(ctx, session, mgPolicies.RuleType, operations.OpUpdateRule, smqauthz.PolicyReq{
		Domain:      session.DomainID,
		SubjectType: policies.UserType,
		SubjectKind: policies.UsersKind,
		Subject:     session.DomainUserID,
		Object:      session.DomainID,
		ObjectType:  policies.DomainType,
	}); err != nil {
		return re.Rule{}, errors.Wrap(errDomainUpdateRules, err)
	}

	return am.svc.UpdateRule(ctx, session, r)
}

func (am *authorizationMiddleware) UpdateRuleTags(ctx context.Context, session authn.Session, r re.Rule) (re.Rule, error) {
	if err := am.authorize(ctx, session, mgPolicies.RuleType, operations.OpUpdateRuleTags, smqauthz.PolicyReq{
		Domain:      session.DomainID,
		SubjectType: policies.UserType,
		SubjectKind: policies.UsersKind,
		Subject:     session.DomainUserID,
		Object:      session.DomainID,
		ObjectType:  policies.DomainType,
	}); err != nil {
		return re.Rule{}, errors.Wrap(errDomainUpdateRules, err)
	}

	return am.svc.UpdateRuleTags(ctx, session, r)
}

func (am *authorizationMiddleware) UpdateRuleSchedule(ctx context.Context, session authn.Session, r re.Rule) (re.Rule, error) {
	if err := am.authorize(ctx, session, mgPolicies.RuleType, operations.OpUpdateRuleSchedule, smqauthz.PolicyReq{
		Domain:      session.DomainID,
		SubjectType: policies.UserType,
		SubjectKind: policies.UsersKind,
		Subject:     session.DomainUserID,
		Object:      session.DomainID,
		ObjectType:  policies.DomainType,
	}); err != nil {
		return re.Rule{}, errors.Wrap(errDomainUpdateRules, err)
	}

	return am.svc.UpdateRuleSchedule(ctx, session, r)
}

func (am *authorizationMiddleware) ListRules(ctx context.Context, session authn.Session, pm re.PageMeta) (re.Page, error) {
	if err := am.authorize(ctx, session, mgPolicies.RuleType, operations.OpListRules, smqauthz.PolicyReq{
		Domain:      session.DomainID,
		SubjectType: policies.UserType,
		SubjectKind: policies.UsersKind,
		Subject:     session.DomainUserID,
		Object:      session.DomainID,
		ObjectType:  policies.DomainType,
	}); err != nil {
		return re.Page{}, errors.Wrap(errDomainViewRules, err)
	}

	return am.svc.ListRules(ctx, session, pm)
}

func (am *authorizationMiddleware) RemoveRule(ctx context.Context, session authn.Session, id string) error {
	if err := am.authorize(ctx, session, mgPolicies.RuleType, operations.OpRemoveRule, smqauthz.PolicyReq{
		Domain:      session.DomainID,
		SubjectType: policies.UserType,
		SubjectKind: policies.UsersKind,
		Subject:     session.DomainUserID,
		Object:      session.DomainID,
		ObjectType:  policies.DomainType,
	}); err != nil {
		return errors.Wrap(errDomainDeleteRules, err)
	}

	return am.svc.RemoveRule(ctx, session, id)
}

func (am *authorizationMiddleware) EnableRule(ctx context.Context, session authn.Session, id string) (re.Rule, error) {
	if err := am.authorize(ctx, session, mgPolicies.RuleType, operations.OpEnableRule, smqauthz.PolicyReq{
		Domain:      session.DomainID,
		SubjectType: policies.UserType,
		SubjectKind: policies.UsersKind,
		Subject:     session.DomainUserID,
		Object:      session.DomainID,
		ObjectType:  policies.DomainType,
	}); err != nil {
		return re.Rule{}, errors.Wrap(errDomainUpdateRules, err)
	}

	return am.svc.EnableRule(ctx, session, id)
}

func (am *authorizationMiddleware) DisableRule(ctx context.Context, session authn.Session, id string) (re.Rule, error) {
	if err := am.authorize(ctx, session, mgPolicies.RuleType, operations.OpDisableRule, smqauthz.PolicyReq{
		Domain:      session.DomainID,
		SubjectType: policies.UserType,
		SubjectKind: policies.UsersKind,
		Subject:     session.DomainUserID,
		Object:      session.DomainID,
		ObjectType:  policies.DomainType,
	}); err != nil {
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

func (am *authorizationMiddleware) authorize(ctx context.Context, session authn.Session, entityType string, op permissions.Operation, req smqauthz.PolicyReq) error {
	req.TokenType = session.Type
	req.UserID = session.UserID
	req.PatID = session.PatID
	req.OptionalDomainID = session.DomainID

	perm, err := am.entitiesOps.GetPermission(entityType, op)
	if err != nil {
		return err
	}

	req.Permission = perm.String()

	if err := am.authz.Authorize(ctx, req); err != nil {
		return err
	}

	return nil
}
