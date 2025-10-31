// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package middleware

import (
	"context"
	"time"

	"github.com/absmach/magistrala/re"
	"github.com/absmach/supermq/pkg/authn"
	smqauthz "github.com/absmach/supermq/pkg/authz"
	"github.com/absmach/supermq/pkg/callout"
	"github.com/absmach/supermq/pkg/errors"
	"github.com/absmach/supermq/pkg/messaging"
	"github.com/absmach/supermq/pkg/policies"
)

var (
	errDomainCreateRules = errors.New("not authorized to create rules in domain")
	errDomainViewRules   = errors.New("not authorized to view rules in domain")
	errDomainUpdateRules = errors.New("not authorized to update rules in domain")
	errDomainDeleteRules = errors.New("not authorized to delete rules in domain")
)

const entityType = "rule"

type authorizationMiddleware struct {
	svc     re.Service
	authz   smqauthz.Authorization
	callout callout.Callout
}

// AuthorizationMiddleware adds authorization to the re service.
func AuthorizationMiddleware(svc re.Service, authz smqauthz.Authorization, callout callout.Callout) (re.Service, error) {
	return &authorizationMiddleware{
		svc:     svc,
		authz:   authz,
		callout: callout,
	}, nil
}

func (am *authorizationMiddleware) AddRule(ctx context.Context, session authn.Session, r re.Rule) (re.Rule, error) {
	if err := am.authorize(ctx, smqauthz.PolicyReq{
		Domain:      session.DomainID,
		SubjectType: policies.UserType,
		SubjectKind: policies.UsersKind,
		Subject:     session.DomainUserID,
		Object:      session.DomainID,
		ObjectType:  policies.DomainType,
		Permission:  policies.MembershipPermission,
	}); err != nil {
		return re.Rule{}, errors.Wrap(errDomainCreateRules, err)
	}

	params := map[string]any{
		"entities": r,
		"count":    1,
	}

	if err := am.callOut(ctx, session, re.OpAddRule, params); err != nil {
		return re.Rule{}, err
	}

	return am.svc.AddRule(ctx, session, r)
}

func (am *authorizationMiddleware) ViewRule(ctx context.Context, session authn.Session, id string) (re.Rule, error) {
	if err := am.authorize(ctx, smqauthz.PolicyReq{
		Domain:      session.DomainID,
		SubjectType: policies.UserType,
		SubjectKind: policies.UsersKind,
		Subject:     session.DomainUserID,
		Object:      session.DomainID,
		ObjectType:  policies.DomainType,
		Permission:  policies.MembershipPermission,
	}); err != nil {
		return re.Rule{}, errors.Wrap(errDomainViewRules, err)
	}

	params := map[string]any{
		"entity_id": id,
	}

	if err := am.callOut(ctx, session, re.OpViewRule, params); err != nil {
		return re.Rule{}, err
	}

	return am.svc.ViewRule(ctx, session, id)
}

func (am *authorizationMiddleware) UpdateRule(ctx context.Context, session authn.Session, r re.Rule) (re.Rule, error) {
	if err := am.authorize(ctx, smqauthz.PolicyReq{
		Domain:      session.DomainID,
		SubjectType: policies.UserType,
		SubjectKind: policies.UsersKind,
		Subject:     session.DomainUserID,
		Object:      session.DomainID,
		ObjectType:  policies.DomainType,
		Permission:  policies.MembershipPermission,
	}); err != nil {
		return re.Rule{}, errors.Wrap(errDomainUpdateRules, err)
	}

	params := map[string]any{
		"entity_id": r.ID,
	}

	if err := am.callOut(ctx, session, re.OpUpdateRule, params); err != nil {
		return re.Rule{}, err
	}

	return am.svc.UpdateRule(ctx, session, r)
}

func (am *authorizationMiddleware) UpdateRuleTags(ctx context.Context, session authn.Session, r re.Rule) (re.Rule, error) {
	if err := am.authorize(ctx, smqauthz.PolicyReq{
		Domain:      session.DomainID,
		SubjectType: policies.UserType,
		SubjectKind: policies.UsersKind,
		Subject:     session.DomainUserID,
		Object:      session.DomainID,
		ObjectType:  policies.DomainType,
		Permission:  policies.MembershipPermission,
	}); err != nil {
		return re.Rule{}, errors.Wrap(errDomainUpdateRules, err)
	}

	params := map[string]any{
		"entity_id": r.ID,
	}

	if err := am.callOut(ctx, session, re.OpUpdateRuleTags, params); err != nil {
		return re.Rule{}, err
	}

	return am.svc.UpdateRuleTags(ctx, session, r)
}

func (am *authorizationMiddleware) UpdateRuleSchedule(ctx context.Context, session authn.Session, r re.Rule) (re.Rule, error) {
	if err := am.authorize(ctx, smqauthz.PolicyReq{
		Domain:      session.DomainID,
		SubjectType: policies.UserType,
		SubjectKind: policies.UsersKind,
		Subject:     session.DomainUserID,
		Object:      session.DomainID,
		ObjectType:  policies.DomainType,
		Permission:  policies.MembershipPermission,
	}); err != nil {
		return re.Rule{}, errors.Wrap(errDomainUpdateRules, err)
	}

	params := map[string]any{
		"entity_id": r.ID,
	}

	if err := am.callOut(ctx, session, re.OpUpdateRuleSchedule, params); err != nil {
		return re.Rule{}, err
	}

	return am.svc.UpdateRuleSchedule(ctx, session, r)
}

func (am *authorizationMiddleware) ListRules(ctx context.Context, session authn.Session, pm re.PageMeta) (re.Page, error) {
	if err := am.authorize(ctx, smqauthz.PolicyReq{
		Domain:      session.DomainID,
		SubjectType: policies.UserType,
		SubjectKind: policies.UsersKind,
		Subject:     session.DomainUserID,
		Object:      session.DomainID,
		ObjectType:  policies.DomainType,
		Permission:  policies.MembershipPermission,
	}); err != nil {
		return re.Page{}, errors.Wrap(errDomainViewRules, err)
	}

	params := map[string]any{
		"pagemeta": pm,
	}

	if err := am.callOut(ctx, session, re.OpListRules, params); err != nil {
		return re.Page{}, err
	}

	return am.svc.ListRules(ctx, session, pm)
}

func (am *authorizationMiddleware) RemoveRule(ctx context.Context, session authn.Session, id string) error {
	if err := am.authorize(ctx, smqauthz.PolicyReq{
		Domain:      session.DomainID,
		SubjectType: policies.UserType,
		SubjectKind: policies.UsersKind,
		Subject:     session.DomainUserID,
		Object:      session.DomainID,
		ObjectType:  policies.DomainType,
		Permission:  policies.MembershipPermission,
	}); err != nil {
		return errors.Wrap(errDomainDeleteRules, err)
	}

	params := map[string]any{
		"entity_id": id,
	}

	if err := am.callOut(ctx, session, re.OpRemoveRule, params); err != nil {
		return err
	}

	return am.svc.RemoveRule(ctx, session, id)
}

func (am *authorizationMiddleware) EnableRule(ctx context.Context, session authn.Session, id string) (re.Rule, error) {
	if err := am.authorize(ctx, smqauthz.PolicyReq{
		Domain:      session.DomainID,
		SubjectType: policies.UserType,
		SubjectKind: policies.UsersKind,
		Subject:     session.DomainUserID,
		Object:      session.DomainID,
		ObjectType:  policies.DomainType,
		Permission:  policies.MembershipPermission,
	}); err != nil {
		return re.Rule{}, errors.Wrap(errDomainUpdateRules, err)
	}

	params := map[string]any{
		"entity_id": id,
	}

	if err := am.callOut(ctx, session, re.OpEnableRule, params); err != nil {
		return re.Rule{}, err
	}

	return am.svc.EnableRule(ctx, session, id)
}

func (am *authorizationMiddleware) DisableRule(ctx context.Context, session authn.Session, id string) (re.Rule, error) {
	if err := am.authorize(ctx, smqauthz.PolicyReq{
		Domain:      session.DomainID,
		SubjectType: policies.UserType,
		SubjectKind: policies.UsersKind,
		Subject:     session.DomainUserID,
		Object:      session.DomainID,
		ObjectType:  policies.DomainType,
		Permission:  policies.MembershipPermission,
	}); err != nil {
		return re.Rule{}, errors.Wrap(errDomainUpdateRules, err)
	}

	params := map[string]any{
		"entity_id": id,
	}

	if err := am.callOut(ctx, session, re.OpDisableRule, params); err != nil {
		return re.Rule{}, err
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

func (am *authorizationMiddleware) authorize(ctx context.Context, pr smqauthz.PolicyReq) error {
	if err := am.authz.Authorize(ctx, pr); err != nil {
		return err
	}
	return nil
}

func (am *authorizationMiddleware) callOut(ctx context.Context, session authn.Session, op string, params map[string]any) error {
	req := callout.Request{
		BaseRequest: callout.BaseRequest{
			EntityType: entityType,
			CallerID:   session.UserID,
			CallerType: policies.UserType,
			DomainID:   session.DomainID,
			Time:       time.Now().UTC(),
			Operation:  op,
		},
	}

	if err := am.callout.Callout(ctx, req); err != nil {
		return err
	}

	return nil
}
