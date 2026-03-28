// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package middleware

import (
	"context"
	"time"

	"github.com/absmach/supermq/domains"
	"github.com/absmach/supermq/domains/operations"
	"github.com/absmach/supermq/pkg/authn"
	"github.com/absmach/supermq/pkg/callout"
	"github.com/absmach/supermq/pkg/permissions"
	"github.com/absmach/supermq/pkg/policies"
	"github.com/absmach/supermq/pkg/roles"
	rolemgr "github.com/absmach/supermq/pkg/roles/rolemanager/middleware"
)

var _ domains.Service = (*calloutMiddleware)(nil)

type calloutMiddleware struct {
	svc         domains.Service
	callout     callout.Callout
	entitiesOps permissions.EntitiesOperations[permissions.Operation]
	rolemgr.RoleManagerCalloutMiddleware
}

func NewCallout(svc domains.Service, entitiesOps permissions.EntitiesOperations[permissions.Operation], roleOps permissions.Operations[permissions.RoleOperation], callout callout.Callout) (domains.Service, error) {
	call, err := rolemgr.NewCallout(policies.DomainType, svc, callout, roleOps)
	if err != nil {
		return nil, err
	}

	if err := entitiesOps.Validate(); err != nil {
		return nil, err
	}

	return &calloutMiddleware{
		svc:                          svc,
		callout:                      callout,
		entitiesOps:                  entitiesOps,
		RoleManagerCalloutMiddleware: call,
	}, nil
}

func (cm *calloutMiddleware) CreateDomain(ctx context.Context, session authn.Session, d domains.Domain) (domains.Domain, []roles.RoleProvision, error) {
	params := map[string]any{
		"entity_id": d.ID,
	}

	if err := cm.callOut(ctx, session, policies.DomainType, operations.OpCreateDomain, params); err != nil {
		return domains.Domain{}, nil, err
	}

	return cm.svc.CreateDomain(ctx, session, d)
}

func (cm *calloutMiddleware) RetrieveDomain(ctx context.Context, session authn.Session, id string, withRoles bool) (domains.Domain, error) {
	params := map[string]any{
		"entity_id":  id,
		"with_roles": withRoles,
	}

	if err := cm.callOut(ctx, session, policies.DomainType, operations.OpRetrieveDomain, params); err != nil {
		return domains.Domain{}, err
	}

	return cm.svc.RetrieveDomain(ctx, session, id, withRoles)
}

func (cm *calloutMiddleware) UpdateDomain(ctx context.Context, session authn.Session, id string, d domains.DomainReq) (domains.Domain, error) {
	params := map[string]any{
		"entity_id":  id,
		"domain_req": d,
	}

	if err := cm.callOut(ctx, session, policies.DomainType, operations.OpUpdateDomain, params); err != nil {
		return domains.Domain{}, err
	}

	return cm.svc.UpdateDomain(ctx, session, id, d)
}

func (cm *calloutMiddleware) EnableDomain(ctx context.Context, session authn.Session, id string) (domains.Domain, error) {
	params := map[string]any{
		"entity_id": id,
	}

	if err := cm.callOut(ctx, session, policies.DomainType, operations.OpEnableDomain, params); err != nil {
		return domains.Domain{}, err
	}

	return cm.svc.EnableDomain(ctx, session, id)
}

func (cm *calloutMiddleware) DisableDomain(ctx context.Context, session authn.Session, id string) (domains.Domain, error) {
	params := map[string]any{
		"entity_id": id,
	}

	if err := cm.callOut(ctx, session, policies.DomainType, operations.OpDisableDomain, params); err != nil {
		return domains.Domain{}, err
	}

	return cm.svc.DisableDomain(ctx, session, id)
}

func (cm *calloutMiddleware) FreezeDomain(ctx context.Context, session authn.Session, id string) (domains.Domain, error) {
	params := map[string]any{
		"entity_id": id,
	}

	if err := cm.callOut(ctx, session, policies.DomainType, operations.OpFreezeDomain, params); err != nil {
		return domains.Domain{}, err
	}

	return cm.svc.FreezeDomain(ctx, session, id)
}

func (cm *calloutMiddleware) ListDomains(ctx context.Context, session authn.Session, page domains.Page) (domains.DomainsPage, error) {
	params := map[string]any{
		"page": page,
	}

	if err := cm.callOut(ctx, session, policies.DomainType, operations.OpListDomains, params); err != nil {
		return domains.DomainsPage{}, err
	}

	return cm.svc.ListDomains(ctx, session, page)
}

func (cm *calloutMiddleware) DeleteDomain(ctx context.Context, session authn.Session, id string) error {
	params := map[string]any{
		"entity_id": id,
	}

	if err := cm.callOut(ctx, session, policies.DomainType, domains.OpDeleteDomain, params); err != nil {
		return err
	}

	return cm.svc.DeleteDomain(ctx, session, id)
}

func (cm *calloutMiddleware) SendInvitation(ctx context.Context, session authn.Session, invitation domains.Invitation) (domains.Invitation, error) {
	params := map[string]any{
		"entity_id":  invitation.DomainID,
		"invitation": invitation,
	}

	// While entity here is technically an invitation, Domain is used as
	// the entity in callout since the invitation refers to the domain.
	if err := cm.callOut(ctx, session, policies.DomainType, operations.OpSendDomainInvitation, params); err != nil {
		return domains.Invitation{}, err
	}

	return cm.svc.SendInvitation(ctx, session, invitation)
}

func (cm *calloutMiddleware) ListInvitations(ctx context.Context, session authn.Session, page domains.InvitationPageMeta) (domains.InvitationPage, error) {
	params := map[string]any{
		"page": page,
	}

	if err := cm.callOut(ctx, session, policies.DomainType, operations.OpListInvitations, params); err != nil {
		return domains.InvitationPage{}, err
	}

	return cm.svc.ListInvitations(ctx, session, page)
}

func (cm *calloutMiddleware) ListDomainInvitations(ctx context.Context, session authn.Session, page domains.InvitationPageMeta) (domains.InvitationPage, error) {
	params := map[string]any{
		"entity_id": page.DomainID,
		"page":      page,
	}

	if err := cm.callOut(ctx, session, policies.DomainType, operations.OpListDomainInvitations, params); err != nil {
		return domains.InvitationPage{}, err
	}

	return cm.svc.ListDomainInvitations(ctx, session, page)
}

func (cm *calloutMiddleware) AcceptInvitation(ctx context.Context, session authn.Session, domainID string) (domains.Invitation, error) {
	params := map[string]any{
		"entity_id": domainID,
	}

	// Similar to sending an invitation, Domain is used as the
	// entity in callout since the invitation refers to the domain.
	if err := cm.callOut(ctx, session, policies.DomainType, operations.OpAcceptInvitation, params); err != nil {
		return domains.Invitation{}, err
	}

	return cm.svc.AcceptInvitation(ctx, session, domainID)
}

func (cm *calloutMiddleware) RejectInvitation(ctx context.Context, session authn.Session, domainID string) (domains.Invitation, error) {
	params := map[string]any{
		"entity_id": domainID,
	}

	// Similar to sending and accepting, Domain is used as
	// the entity in callout since the invitation refers to the domain.
	if err := cm.callOut(ctx, session, policies.DomainType, operations.OpRejectInvitation, params); err != nil {
		return domains.Invitation{}, err
	}

	return cm.svc.RejectInvitation(ctx, session, domainID)
}

func (cm *calloutMiddleware) DeleteInvitation(ctx context.Context, session authn.Session, inviteeUserID, domainID string) error {
	params := map[string]any{
		"entity_id":       domainID,
		"invitee_user_id": inviteeUserID,
	}

	if err := cm.callOut(ctx, session, policies.DomainType, operations.OpDeleteDomainInvitation, params); err != nil {
		return err
	}

	return cm.svc.DeleteInvitation(ctx, session, inviteeUserID, domainID)
}

func (cm *calloutMiddleware) callOut(ctx context.Context, session authn.Session, entityType string, op permissions.Operation, pld map[string]any) error {
	var entityID string
	if id, ok := pld["entity_id"].(string); ok {
		entityID = id
	}

	req := callout.Request{
		BaseRequest: callout.BaseRequest{
			Operation:  cm.entitiesOps.OperationName(entityType, op),
			EntityType: entityType,
			EntityID:   entityID,
			CallerID:   session.UserID,
			CallerType: policies.UserType,
			DomainID:   entityID,
			Time:       time.Now().UTC(),
		},
		Payload: pld,
	}

	if err := cm.callout.Callout(ctx, req); err != nil {
		return err
	}

	return nil
}
