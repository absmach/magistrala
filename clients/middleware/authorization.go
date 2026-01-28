// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package middleware

import (
	"context"

	"github.com/absmach/supermq/auth"
	"github.com/absmach/supermq/clients"
	"github.com/absmach/supermq/clients/operations"
	dOperations "github.com/absmach/supermq/domains/operations"
	gOperations "github.com/absmach/supermq/groups/operations"
	"github.com/absmach/supermq/pkg/authn"
	smqauthz "github.com/absmach/supermq/pkg/authz"
	"github.com/absmach/supermq/pkg/errors"
	svcerr "github.com/absmach/supermq/pkg/errors/service"
	"github.com/absmach/supermq/pkg/permissions"
	"github.com/absmach/supermq/pkg/policies"
	"github.com/absmach/supermq/pkg/roles"
	rolemgr "github.com/absmach/supermq/pkg/roles/rolemanager/middleware"
)

var (
	errView                    = errors.New("not authorized to view client")
	errUpdate                  = errors.New("not authorized to update client")
	errUpdateTags              = errors.New("not authorized to update client tags")
	errUpdateSecret            = errors.New("not authorized to update client secret")
	errEnable                  = errors.New("not authorized to enable client")
	errDisable                 = errors.New("not authorized to disable client")
	errDelete                  = errors.New("not authorized to delete client")
	errSetParentGroup          = errors.New("not authorized to set parent group to client")
	errRemoveParentGroup       = errors.New("not authorized to remove parent group from client")
	errDomainCreateClients     = errors.New("not authorized to create client in domain")
	errGroupSetChildClients    = errors.New("not authorized to set child client for group")
	errGroupRemoveChildClients = errors.New("not authorized to remove child client for group")
)

var _ clients.Service = (*authorizationMiddleware)(nil)

type authorizationMiddleware struct {
	svc         clients.Service
	repo        clients.Repository
	authz       smqauthz.Authorization
	entitiesOps permissions.EntitiesOperations[permissions.Operation]
	rolemgr.RoleManagerAuthorizationMiddleware
}

// NewAuthorization adds authorization to the clients service.
func NewAuthorization(
	entityType string,
	svc clients.Service,
	authz smqauthz.Authorization,
	repo clients.Repository,
	entitiesOps permissions.EntitiesOperations[permissions.Operation],
	roleOps permissions.Operations[permissions.RoleOperation],
) (clients.Service, error) {
	if err := entitiesOps.Validate(); err != nil {
		return nil, err
	}
	ram, err := rolemgr.NewAuthorization(policies.ClientType, svc, authz, roleOps)
	if err != nil {
		return nil, err
	}

	return &authorizationMiddleware{
		svc:                                svc,
		authz:                              authz,
		repo:                               repo,
		entitiesOps:                        entitiesOps,
		RoleManagerAuthorizationMiddleware: ram,
	}, nil
}

func (am *authorizationMiddleware) CreateClients(ctx context.Context, session authn.Session, client ...clients.Client) ([]clients.Client, []roles.RoleProvision, error) {
	if err := am.authorize(ctx, session, policies.DomainType, dOperations.OpCreateDomainClients, smqauthz.PolicyReq{
		Domain:      session.DomainID,
		SubjectType: policies.UserType,
		Subject:     session.DomainUserID,
		ObjectType:  policies.DomainType,
		Object:      session.DomainID,
	}); err != nil {
		return []clients.Client{}, []roles.RoleProvision{}, errors.Wrap(err, errDomainCreateClients)
	}

	return am.svc.CreateClients(ctx, session, client...)
}

func (am *authorizationMiddleware) View(ctx context.Context, session authn.Session, id string, withRoles bool) (clients.Client, error) {
	if err := am.authorize(ctx, session, policies.ClientType, operations.OpViewClient, smqauthz.PolicyReq{
		Domain:      session.DomainID,
		SubjectType: policies.UserType,
		Subject:     session.DomainUserID,
		ObjectType:  policies.ClientType,
		Object:      id,
	}); err != nil {
		return clients.Client{}, errors.Wrap(err, errView)
	}

	return am.svc.View(ctx, session, id, withRoles)
}

func (am *authorizationMiddleware) ListClients(ctx context.Context, session authn.Session, pm clients.Page) (clients.ClientsPage, error) {
	if err := am.checkSuperAdmin(ctx, session); err == nil {
		session.SuperAdmin = true
	}

	return am.svc.ListClients(ctx, session, pm)
}

func (am *authorizationMiddleware) ListUserClients(ctx context.Context, session authn.Session, userID string, pm clients.Page) (clients.ClientsPage, error) {
	if err := am.checkSuperAdmin(ctx, session); err != nil {
		return clients.ClientsPage{}, err
	}

	return am.svc.ListUserClients(ctx, session, userID, pm)
}

func (am *authorizationMiddleware) Update(ctx context.Context, session authn.Session, client clients.Client) (clients.Client, error) {
	if err := am.authorize(ctx, session, policies.ClientType, operations.OpUpdateClient, smqauthz.PolicyReq{
		Domain:      session.DomainID,
		SubjectType: policies.UserType,
		Subject:     session.DomainUserID,
		ObjectType:  policies.ClientType,
		Object:      client.ID,
	}); err != nil {
		return clients.Client{}, errors.Wrap(err, errUpdate)
	}

	return am.svc.Update(ctx, session, client)
}

func (am *authorizationMiddleware) UpdateTags(ctx context.Context, session authn.Session, client clients.Client) (clients.Client, error) {
	if err := am.authorize(ctx, session, policies.ClientType, operations.OpUpdateClientTags, smqauthz.PolicyReq{
		Domain:      session.DomainID,
		SubjectType: policies.UserType,
		Subject:     session.DomainUserID,
		ObjectType:  policies.ClientType,
		Object:      client.ID,
	}); err != nil {
		return clients.Client{}, errors.Wrap(err, errUpdateTags)
	}

	return am.svc.UpdateTags(ctx, session, client)
}

func (am *authorizationMiddleware) UpdateSecret(ctx context.Context, session authn.Session, id, key string) (clients.Client, error) {
	if err := am.authorize(ctx, session, policies.ClientType, operations.OpUpdateClientSecret, smqauthz.PolicyReq{
		Domain:      session.DomainID,
		SubjectType: policies.UserType,
		Subject:     session.DomainUserID,
		ObjectType:  policies.ClientType,
		Object:      id,
	}); err != nil {
		return clients.Client{}, errors.Wrap(err, errUpdateSecret)
	}

	return am.svc.UpdateSecret(ctx, session, id, key)
}

func (am *authorizationMiddleware) Enable(ctx context.Context, session authn.Session, id string) (clients.Client, error) {
	if err := am.authorize(ctx, session, policies.ClientType, operations.OpEnableClient, smqauthz.PolicyReq{
		Domain:      session.DomainID,
		SubjectType: policies.UserType,
		Subject:     session.DomainUserID,
		ObjectType:  policies.ClientType,
		Object:      id,
	}); err != nil {
		return clients.Client{}, errors.Wrap(err, errEnable)
	}

	return am.svc.Enable(ctx, session, id)
}

func (am *authorizationMiddleware) Disable(ctx context.Context, session authn.Session, id string) (clients.Client, error) {
	if err := am.authorize(ctx, session, policies.ClientType, operations.OpDisableClient, smqauthz.PolicyReq{
		Domain:      session.DomainID,
		SubjectType: policies.UserType,
		Subject:     session.DomainUserID,
		ObjectType:  policies.ClientType,
		Object:      id,
	}); err != nil {
		return clients.Client{}, errors.Wrap(err, errDisable)
	}

	return am.svc.Disable(ctx, session, id)
}

func (am *authorizationMiddleware) Delete(ctx context.Context, session authn.Session, id string) error {
	if err := am.authorize(ctx, session, policies.ClientType, operations.OpDeleteClient, smqauthz.PolicyReq{
		Domain:      session.DomainID,
		SubjectType: policies.UserType,
		Subject:     session.DomainUserID,
		ObjectType:  policies.ClientType,
		Object:      id,
	}); err != nil {
		return errors.Wrap(err, errDelete)
	}

	return am.svc.Delete(ctx, session, id)
}

func (am *authorizationMiddleware) SetParentGroup(ctx context.Context, session authn.Session, parentGroupID string, id string) error {
	if err := am.authorize(ctx, session, policies.ClientType, operations.OpSetParentGroup, smqauthz.PolicyReq{
		Domain:      session.DomainID,
		SubjectType: policies.UserType,
		Subject:     session.DomainUserID,
		ObjectType:  policies.ClientType,
		Object:      id,
	}); err != nil {
		return errors.Wrap(err, errSetParentGroup)
	}

	if err := am.authorize(ctx, session, policies.GroupType, gOperations.OpGroupSetChildClient, smqauthz.PolicyReq{
		Domain:      session.DomainID,
		SubjectType: policies.UserType,
		Subject:     session.DomainUserID,
		ObjectType:  policies.GroupType,
		Object:      parentGroupID,
	}); err != nil {
		return errors.Wrap(err, errGroupSetChildClients)
	}

	return am.svc.SetParentGroup(ctx, session, parentGroupID, id)
}

func (am *authorizationMiddleware) RemoveParentGroup(ctx context.Context, session authn.Session, id string) error {
	if err := am.authorize(ctx, session, policies.ClientType, operations.OpRemoveParentGroup, smqauthz.PolicyReq{
		Domain:      session.DomainID,
		SubjectType: policies.UserType,
		Subject:     session.DomainUserID,
		ObjectType:  policies.ClientType,
		Object:      id,
	}); err != nil {
		return errors.Wrap(err, errRemoveParentGroup)
	}

	th, err := am.repo.RetrieveByID(ctx, id)
	if err != nil {
		return errors.Wrap(svcerr.ErrRemoveEntity, err)
	}

	if th.ParentGroup != "" {
		if err := am.authorize(ctx, session, policies.GroupType, gOperations.OpGroupRemoveChildClient, smqauthz.PolicyReq{
			Domain:      session.DomainID,
			SubjectType: policies.UserType,
			Subject:     session.DomainUserID,
			ObjectType:  policies.GroupType,
			Object:      th.ParentGroup,
		}); err != nil {
			return errors.Wrap(err, errGroupRemoveChildClients)
		}

		return am.svc.RemoveParentGroup(ctx, session, id)
	}
	return nil
}

func (am *authorizationMiddleware) authorize(ctx context.Context, session authn.Session, entityType string, op permissions.Operation, req smqauthz.PolicyReq) error {
	req.Domain = session.DomainID

	perm, err := am.entitiesOps.GetPermission(entityType, op)
	if err != nil {
		return err
	}

	req.Permission = perm.String()

	var pat *smqauthz.PATReq
	if session.PatID != "" {
		entityID := req.Object
		opName := am.entitiesOps.OperationName(entityType, op)
		if op == operations.OpListUserClients || op == dOperations.OpCreateDomainClients || op == dOperations.OpListDomainClients {
			entityID = auth.AnyIDs
		}
		pat = &smqauthz.PATReq{
			UserID:     session.UserID,
			PatID:      session.PatID,
			EntityID:   entityID,
			EntityType: auth.ClientsType.String(),
			Operation:  opName,
			Domain:     session.DomainID,
		}
	}

	if err := am.authz.Authorize(ctx, req, pat); err != nil {
		return err
	}

	return nil
}

func (am *authorizationMiddleware) checkSuperAdmin(ctx context.Context, session authn.Session) error {
	if session.Role != authn.AdminRole {
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
