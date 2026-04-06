// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package middleware

import (
	"context"
	"time"

	"github.com/absmach/magistrala/clients"
	"github.com/absmach/magistrala/clients/operations"
	dOperations "github.com/absmach/magistrala/domains/operations"
	"github.com/absmach/magistrala/pkg/authn"
	"github.com/absmach/magistrala/pkg/callout"
	"github.com/absmach/magistrala/pkg/permissions"
	"github.com/absmach/magistrala/pkg/policies"
	"github.com/absmach/magistrala/pkg/roles"
	rolemgr "github.com/absmach/magistrala/pkg/roles/rolemanager/middleware"
)

var _ clients.Service = (*calloutMiddleware)(nil)

type calloutMiddleware struct {
	svc         clients.Service
	repo        clients.Repository
	callout     callout.Callout
	entitiesOps permissions.EntitiesOperations[permissions.Operation]
	rolemgr.RoleManagerCalloutMiddleware
}

func NewCallout(svc clients.Service, repo clients.Repository, entitiesOps permissions.EntitiesOperations[permissions.Operation], roleOps permissions.Operations[permissions.RoleOperation], callout callout.Callout) (clients.Service, error) {
	call, err := rolemgr.NewCallout(policies.ClientType, svc, callout, roleOps)
	if err != nil {
		return nil, err
	}

	if err := entitiesOps.Validate(); err != nil {
		return nil, err
	}

	return &calloutMiddleware{
		svc:                          svc,
		repo:                         repo,
		callout:                      callout,
		entitiesOps:                  entitiesOps,
		RoleManagerCalloutMiddleware: call,
	}, nil
}

func (cm *calloutMiddleware) CreateClients(ctx context.Context, session authn.Session, client ...clients.Client) ([]clients.Client, []roles.RoleProvision, error) {
	params := map[string]any{
		"entities": client,
		"count":    len(client),
	}

	if err := cm.callOut(ctx, session, policies.DomainType, dOperations.OpCreateDomainClients, params); err != nil {
		return []clients.Client{}, []roles.RoleProvision{}, err
	}

	return cm.svc.CreateClients(ctx, session, client...)
}

func (cm *calloutMiddleware) View(ctx context.Context, session authn.Session, id string, withRoles bool) (clients.Client, error) {
	params := map[string]any{
		"entity_id": id,
	}
	if err := cm.callOut(ctx, session, policies.ClientType, operations.OpViewClient, params); err != nil {
		return clients.Client{}, err
	}

	return cm.svc.View(ctx, session, id, withRoles)
}

func (cm *calloutMiddleware) ListClients(ctx context.Context, session authn.Session, pm clients.Page) (clients.ClientsPage, error) {
	params := map[string]any{
		"pagemeta": pm,
	}

	if err := cm.callOut(ctx, session, policies.DomainType, dOperations.OpListDomainClients, params); err != nil {
		return clients.ClientsPage{}, err
	}

	return cm.svc.ListClients(ctx, session, pm)
}

func (cm *calloutMiddleware) ListUserClients(ctx context.Context, session authn.Session, userID string, pm clients.Page) (clients.ClientsPage, error) {
	params := map[string]any{
		"user_id":  userID,
		"pagemeta": pm,
	}

	if err := cm.callOut(ctx, session, policies.ClientType, operations.OpListUserClients, params); err != nil {
		return clients.ClientsPage{}, err
	}

	return cm.svc.ListUserClients(ctx, session, userID, pm)
}

func (cm *calloutMiddleware) Update(ctx context.Context, session authn.Session, client clients.Client) (clients.Client, error) {
	params := map[string]any{
		"entity_id": client.ID,
	}

	if err := cm.callOut(ctx, session, policies.ClientType, operations.OpUpdateClient, params); err != nil {
		return clients.Client{}, err
	}

	return cm.svc.Update(ctx, session, client)
}

func (cm *calloutMiddleware) UpdateTags(ctx context.Context, session authn.Session, client clients.Client) (clients.Client, error) {
	params := map[string]any{
		"entity_id": client.ID,
	}

	if err := cm.callOut(ctx, session, policies.ClientType, operations.OpUpdateClientTags, params); err != nil {
		return clients.Client{}, err
	}

	return cm.svc.UpdateTags(ctx, session, client)
}

func (cm *calloutMiddleware) UpdateSecret(ctx context.Context, session authn.Session, id, key string) (clients.Client, error) {
	params := map[string]any{
		"entity_id": id,
	}

	if err := cm.callOut(ctx, session, policies.ClientType, operations.OpUpdateClientSecret, params); err != nil {
		return clients.Client{}, err
	}

	return cm.svc.UpdateSecret(ctx, session, id, key)
}

func (cm *calloutMiddleware) Enable(ctx context.Context, session authn.Session, id string) (clients.Client, error) {
	params := map[string]any{
		"entity_id": id,
	}

	if err := cm.callOut(ctx, session, policies.ClientType, operations.OpEnableClient, params); err != nil {
		return clients.Client{}, err
	}

	return cm.svc.Enable(ctx, session, id)
}

func (cm *calloutMiddleware) Disable(ctx context.Context, session authn.Session, id string) (clients.Client, error) {
	params := map[string]any{
		"entity_id": id,
	}

	if err := cm.callOut(ctx, session, policies.ClientType, operations.OpDisableClient, params); err != nil {
		return clients.Client{}, err
	}

	return cm.svc.Disable(ctx, session, id)
}

func (cm *calloutMiddleware) Delete(ctx context.Context, session authn.Session, id string) error {
	params := map[string]any{
		"entity_id": id,
	}

	if err := cm.callOut(ctx, session, policies.ClientType, operations.OpDeleteClient, params); err != nil {
		return err
	}

	return cm.svc.Delete(ctx, session, id)
}

func (cm *calloutMiddleware) SetParentGroup(ctx context.Context, session authn.Session, parentGroupID string, id string) error {
	params := map[string]any{
		"entity_id": id,
		"parent_id": parentGroupID,
	}

	if err := cm.callOut(ctx, session, policies.ClientType, operations.OpSetParentGroup, params); err != nil {
		return err
	}

	return cm.svc.SetParentGroup(ctx, session, parentGroupID, id)
}

func (cm *calloutMiddleware) RemoveParentGroup(ctx context.Context, session authn.Session, id string) error {
	th, err := cm.repo.RetrieveByID(ctx, id)
	if err != nil {
		return err
	}

	if th.ParentGroup != "" {
		params := map[string]any{
			"entity_id": id,
			"parent_id": th.ParentGroup,
		}

		if err := cm.callOut(ctx, session, policies.ClientType, operations.OpRemoveParentGroup, params); err != nil {
			return err
		}
	}

	return cm.svc.RemoveParentGroup(ctx, session, id)
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
			DomainID:   session.DomainID,
			Time:       time.Now().UTC(),
		},
		Payload: pld,
	}

	if err := cm.callout.Callout(ctx, req); err != nil {
		return err
	}

	return nil
}
