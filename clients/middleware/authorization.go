// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package middleware

import (
	"context"

	"github.com/absmach/magistrala/clients"
	"github.com/absmach/magistrala/pkg/authn"
	"github.com/absmach/magistrala/pkg/authz"
	"github.com/absmach/magistrala/pkg/errors"
	svcerr "github.com/absmach/magistrala/pkg/errors/service"
	"github.com/absmach/magistrala/pkg/policies"
	rmMW "github.com/absmach/magistrala/pkg/roles/rolemanager/middleware"
	"github.com/absmach/magistrala/pkg/svcutil"
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
	svc    clients.Service
	repo   clients.Repository
	authz  authz.Authorization
	opp    svcutil.OperationPerm
	extOpp svcutil.ExternalOperationPerm
	rmMW.RoleManagerAuthorizationMiddleware
}

// AuthorizationMiddleware adds authorization to the clients service.
func AuthorizationMiddleware(entityType string, svc clients.Service, authz authz.Authorization, repo clients.Repository, clientsOpPerm, rolesOpPerm map[svcutil.Operation]svcutil.Permission, extOpPerm map[svcutil.ExternalOperation]svcutil.Permission) (clients.Service, error) {
	opp := clients.NewOperationPerm()
	if err := opp.AddOperationPermissionMap(clientsOpPerm); err != nil {
		return nil, err
	}
	if err := opp.Validate(); err != nil {
		return nil, err
	}
	ram, err := rmMW.NewRoleManagerAuthorizationMiddleware(policies.ClientType, svc, authz, rolesOpPerm)
	if err != nil {
		return nil, err
	}
	extOpp := clients.NewExternalOperationPerm()
	if err := extOpp.AddOperationPermissionMap(extOpPerm); err != nil {
		return nil, err
	}
	if err := extOpp.Validate(); err != nil {
		return nil, err
	}
	return &authorizationMiddleware{
		svc:                                svc,
		authz:                              authz,
		repo:                               repo,
		opp:                                opp,
		extOpp:                             extOpp,
		RoleManagerAuthorizationMiddleware: ram,
	}, nil
}

func (am *authorizationMiddleware) CreateClients(ctx context.Context, session authn.Session, client ...clients.Client) ([]clients.Client, error) {
	if err := am.extAuthorize(ctx, clients.DomainOpCreateClient, authz.PolicyReq{
		Domain:      session.DomainID,
		SubjectType: policies.UserType,
		Subject:     session.DomainUserID,
		ObjectType:  policies.DomainType,
		Object:      session.DomainID,
	}); err != nil {
		return []clients.Client{}, errors.Wrap(err, errDomainCreateClients)
	}

	return am.svc.CreateClients(ctx, session, client...)
}

func (am *authorizationMiddleware) View(ctx context.Context, session authn.Session, id string) (clients.Client, error) {
	if err := am.authorize(ctx, clients.OpViewClient, authz.PolicyReq{
		Domain:      session.DomainID,
		SubjectType: policies.UserType,
		Subject:     session.DomainUserID,
		ObjectType:  policies.ClientType,
		Object:      id,
	}); err != nil {
		return clients.Client{}, errors.Wrap(err, errView)
	}
	return am.svc.View(ctx, session, id)
}

func (am *authorizationMiddleware) ListClients(ctx context.Context, session authn.Session, reqUserID string, pm clients.Page) (clients.ClientsPage, error) {
	if err := am.checkSuperAdmin(ctx, session.UserID); err != nil {
		session.SuperAdmin = true
	}

	return am.svc.ListClients(ctx, session, reqUserID, pm)
}

func (am *authorizationMiddleware) Update(ctx context.Context, session authn.Session, client clients.Client) (clients.Client, error) {
	if err := am.authorize(ctx, clients.OpUpdateClient, authz.PolicyReq{
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
	if err := am.authorize(ctx, clients.OpUpdateClientTags, authz.PolicyReq{
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
	if err := am.authorize(ctx, clients.OpUpdateClientSecret, authz.PolicyReq{
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
	if err := am.authorize(ctx, clients.OpEnableClient, authz.PolicyReq{
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
	if err := am.authorize(ctx, clients.OpDisableClient, authz.PolicyReq{
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
	if err := am.authorize(ctx, clients.OpDeleteClient, authz.PolicyReq{
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
	if err := am.authorize(ctx, clients.OpSetParentGroup, authz.PolicyReq{
		Domain:      session.DomainID,
		SubjectType: policies.UserType,
		Subject:     session.DomainUserID,
		ObjectType:  policies.ClientType,
		Object:      id,
	}); err != nil {
		return errors.Wrap(err, errSetParentGroup)
	}

	if err := am.extAuthorize(ctx, clients.GroupOpSetChildClient, authz.PolicyReq{
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
	if err := am.authorize(ctx, clients.OpRemoveParentGroup, authz.PolicyReq{
		Domain:      session.DomainID,
		SubjectType: policies.UserType,
		Subject:     session.DomainUserID,
		ObjectType:  policies.ClientType,
		Object:      id,
	}); err != nil {
		return errors.Wrap(err, errRemoveParentGroup)
	}

	cli, err := am.repo.RetrieveByID(ctx, id)
	if err != nil {
		return errors.Wrap(svcerr.ErrRemoveEntity, err)
	}

	if cli.ParentGroup != "" {
		if err := am.extAuthorize(ctx, clients.GroupOpSetChildClient, authz.PolicyReq{
			Domain:      session.DomainID,
			SubjectType: policies.UserType,
			Subject:     session.DomainUserID,
			ObjectType:  policies.GroupType,
			Object:      cli.ParentGroup,
		}); err != nil {
			return errors.Wrap(err, errGroupRemoveChildClients)
		}
		return am.svc.RemoveParentGroup(ctx, session, id)
	}
	return nil
}

func (am *authorizationMiddleware) authorize(ctx context.Context, op svcutil.Operation, req authz.PolicyReq) error {
	perm, err := am.opp.GetPermission(op)
	if err != nil {
		return err
	}

	req.Permission = perm.String()

	if err := am.authz.Authorize(ctx, req); err != nil {
		return err
	}

	return nil
}

func (am *authorizationMiddleware) extAuthorize(ctx context.Context, extOp svcutil.ExternalOperation, req authz.PolicyReq) error {
	perm, err := am.extOpp.GetPermission(extOp)
	if err != nil {
		return err
	}
	req.Permission = perm.String()

	if err := am.authz.Authorize(ctx, req); err != nil {
		return err
	}

	return nil
}

func (am *authorizationMiddleware) checkSuperAdmin(ctx context.Context, userID string) error {
	if err := am.authz.Authorize(ctx, authz.PolicyReq{
		SubjectType: policies.UserType,
		Subject:     userID,
		Permission:  policies.AdminPermission,
		ObjectType:  policies.PlatformType,
		Object:      policies.MagistralaObject,
	}); err != nil {
		return err
	}
	return nil
}
