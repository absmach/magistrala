// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package middleware

import (
	"context"

	mgPolicies "github.com/absmach/magistrala/pkg/policies"
	"github.com/absmach/magistrala/re"
	"github.com/absmach/supermq/auth"
	"github.com/absmach/supermq/pkg/authn"
	smqauthz "github.com/absmach/supermq/pkg/authz"
	"github.com/absmach/supermq/pkg/errors"
	"github.com/absmach/supermq/pkg/messaging"
	"github.com/absmach/supermq/pkg/permissions"
	"github.com/absmach/supermq/pkg/policies"
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
}

// AuthorizationMiddleware adds authorization to the re service.
func AuthorizationMiddleware(svc re.Service, authz smqauthz.Authorization, entitiesOps permissions.EntitiesOperations[permissions.Operation]) (re.Service, error) {
	if err := entitiesOps.Validate(); err != nil {
		return nil, err
	}
	return &authorizationMiddleware{
		svc:         svc,
		authz:       authz,
		entitiesOps: entitiesOps,
	}, nil
}

func (am *authorizationMiddleware) AddRule(ctx context.Context, session authn.Session, r re.Rule) (re.Rule, error) {
	if err := am.authorize(ctx, re.OpAddRule, session, mgPolicies.RuleType, smqauthz.PolicyReq{
		Domain:      session.DomainID,
		SubjectType: policies.UserType,
		Subject:     session.DomainUserID,
		ObjectType:  policies.DomainType,
		Object:      session.DomainID,
	}); err != nil {
		return re.Rule{}, errors.Wrap(errDomainCreateRules, err)
	}

	return am.svc.AddRule(ctx, session, r)
}

func (am *authorizationMiddleware) ViewRule(ctx context.Context, session authn.Session, id string) (re.Rule, error) {
	if err := am.authorize(ctx, re.OpViewRule, session, mgPolicies.RuleType, smqauthz.PolicyReq{
		Domain:      session.DomainID,
		SubjectType: policies.UserType,
		Subject:     session.DomainUserID,
		ObjectType:  mgPolicies.RuleType,
		Object:      id,
	}); err != nil {
		return re.Rule{}, errors.Wrap(errDomainViewRules, err)
	}

	return am.svc.ViewRule(ctx, session, id)
}

func (am *authorizationMiddleware) UpdateRule(ctx context.Context, session authn.Session, r re.Rule) (re.Rule, error) {
	if err := am.authorize(ctx, re.OpUpdateRule, session, mgPolicies.RuleType, smqauthz.PolicyReq{
		Domain:      session.DomainID,
		SubjectType: policies.UserType,
		Subject:     session.DomainUserID,
		ObjectType:  mgPolicies.RuleType,
		Object:      r.ID,
	}); err != nil {
		return re.Rule{}, errors.Wrap(errDomainUpdateRules, err)
	}

	return am.svc.UpdateRule(ctx, session, r)
}

func (am *authorizationMiddleware) UpdateRuleTags(ctx context.Context, session authn.Session, r re.Rule) (re.Rule, error) {
	if err := am.authorize(ctx, re.OpUpdateRuleTags, session, mgPolicies.RuleType, smqauthz.PolicyReq{
		Domain:      session.DomainID,
		SubjectType: policies.UserType,
		Subject:     session.DomainUserID,
		ObjectType:  mgPolicies.RuleType,
		Object:      r.ID,
	}); err != nil {
		return re.Rule{}, errors.Wrap(errDomainUpdateRules, err)
	}

	return am.svc.UpdateRuleTags(ctx, session, r)
}

func (am *authorizationMiddleware) UpdateRuleSchedule(ctx context.Context, session authn.Session, r re.Rule) (re.Rule, error) {
	if err := am.authorize(ctx, re.OpUpdateRuleSchedule, session, mgPolicies.RuleType, smqauthz.PolicyReq{
		Domain:      session.DomainID,
		SubjectType: policies.UserType,
		Subject:     session.DomainUserID,
		ObjectType:  mgPolicies.RuleType,
		Object:      r.ID,
	}); err != nil {
		return re.Rule{}, errors.Wrap(errDomainUpdateRules, err)
	}

	return am.svc.UpdateRuleSchedule(ctx, session, r)
}

func (am *authorizationMiddleware) ListRules(ctx context.Context, session authn.Session, pm re.PageMeta) (re.Page, error) {
	if err := am.authorize(ctx, re.OpListRules, session, mgPolicies.RuleType, smqauthz.PolicyReq{
		Domain:      session.DomainID,
		SubjectType: policies.UserType,
		Subject:     session.DomainUserID,
		ObjectType:  policies.DomainType,
		Object:      session.DomainID,
	}); err != nil {
		return re.Page{}, errors.Wrap(errDomainViewRules, err)
	}

	return am.svc.ListRules(ctx, session, pm)
}

func (am *authorizationMiddleware) RemoveRule(ctx context.Context, session authn.Session, id string) error {
	if err := am.authorize(ctx, re.OpRemoveRule, session, mgPolicies.RuleType, smqauthz.PolicyReq{
		Domain:      session.DomainID,
		SubjectType: policies.UserType,
		Subject:     session.DomainUserID,
		ObjectType:  mgPolicies.RuleType,
		Object:      id,
	}); err != nil {
		return errors.Wrap(errDomainDeleteRules, err)
	}

	return am.svc.RemoveRule(ctx, session, id)
}

func (am *authorizationMiddleware) EnableRule(ctx context.Context, session authn.Session, id string) (re.Rule, error) {
	if err := am.authorize(ctx, re.OpEnableRule, session, mgPolicies.RuleType, smqauthz.PolicyReq{
		Domain:      session.DomainID,
		SubjectType: policies.UserType,
		Subject:     session.DomainUserID,
		ObjectType:  mgPolicies.RuleType,
		Object:      id,
	}); err != nil {
		return re.Rule{}, errors.Wrap(errDomainUpdateRules, err)
	}

	return am.svc.EnableRule(ctx, session, id)
}

func (am *authorizationMiddleware) DisableRule(ctx context.Context, session authn.Session, id string) (re.Rule, error) {
	if err := am.authorize(ctx, re.OpDisableRule, session, mgPolicies.RuleType, smqauthz.PolicyReq{
		Domain:      session.DomainID,
		SubjectType: policies.UserType,
		Subject:     session.DomainUserID,
		ObjectType:  mgPolicies.RuleType,
		Object:      id,
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
		if op == re.OpListRules || op == re.OpAddRule {
			entityID = auth.AnyIDs
		}
		pat = &smqauthz.PATReq{
			UserID:     session.UserID,
			PatID:      session.PatID,
			EntityID:   entityID,
			EntityType: mgPolicies.RuleType,
			Operation:  opName,
			Domain:     session.DomainID,
		}
	}

	return am.authz.Authorize(ctx, req, pat)
}
