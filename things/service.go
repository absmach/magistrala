// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0
package things

import (
	"context"
	"time"

	"github.com/absmach/magistrala"
	"github.com/absmach/magistrala/auth"
	grpcclient "github.com/absmach/magistrala/auth/api/grpc"
	mgclients "github.com/absmach/magistrala/pkg/clients"
	"github.com/absmach/magistrala/pkg/errors"
	svcerr "github.com/absmach/magistrala/pkg/errors/service"
	mggroups "github.com/absmach/magistrala/pkg/groups"
	"github.com/absmach/magistrala/pkg/policy"
	"github.com/absmach/magistrala/things/postgres"
	"golang.org/x/sync/errgroup"
)

type service struct {
	auth        grpcclient.AuthServiceClient
	policy      policy.PolicyService
	clients     postgres.Repository
	clientCache Cache
	idProvider  magistrala.IDProvider
	grepo       mggroups.Repository
}

// NewService returns a new Clients service implementation.
func NewService(auth grpcclient.AuthServiceClient, policyService policy.PolicyService, c postgres.Repository, grepo mggroups.Repository, tcache Cache, idp magistrala.IDProvider) Service {
	return service{
		auth:        auth,
		policy:      policyService,
		clients:     c,
		grepo:       grepo,
		clientCache: tcache,
		idProvider:  idp,
	}
}

func (svc service) Authorize(ctx context.Context, req *magistrala.AuthorizeReq) (string, error) {
	thingID, err := svc.Identify(ctx, req.GetSubject())
	if err != nil {
		return "", err
	}

	r := &magistrala.AuthorizeReq{
		SubjectType: auth.GroupType,
		Subject:     req.GetObject(),
		ObjectType:  auth.ThingType,
		Object:      thingID,
		Permission:  req.GetPermission(),
	}
	resp, err := svc.auth.Authorize(ctx, r)
	if err != nil {
		return "", errors.Wrap(svcerr.ErrAuthorization, err)
	}
	if !resp.GetAuthorized() {
		return "", svcerr.ErrAuthorization
	}

	return thingID, nil
}

func (svc service) CreateThings(ctx context.Context, token string, cls ...mgclients.Client) ([]mgclients.Client, error) {
	user, err := svc.identify(ctx, token)
	if err != nil {
		return []mgclients.Client{}, err
	}
	// If domain is disabled , then this authorization will fail for all non-admin domain users
	if _, err := svc.authorize(ctx, "", auth.UserType, auth.UsersKind, user.GetId(), auth.CreatePermission, auth.DomainType, user.GetDomainId()); err != nil {
		return []mgclients.Client{}, err
	}

	var clients []mgclients.Client
	for _, c := range cls {
		if c.ID == "" {
			clientID, err := svc.idProvider.ID()
			if err != nil {
				return []mgclients.Client{}, err
			}
			c.ID = clientID
		}
		if c.Credentials.Secret == "" {
			key, err := svc.idProvider.ID()
			if err != nil {
				return []mgclients.Client{}, err
			}
			c.Credentials.Secret = key
		}
		if c.Status != mgclients.DisabledStatus && c.Status != mgclients.EnabledStatus {
			return []mgclients.Client{}, svcerr.ErrInvalidStatus
		}
		c.Domain = user.GetDomainId()
		c.CreatedAt = time.Now()
		clients = append(clients, c)
	}

	if err := svc.addThingPolicies(ctx, user.GetId(), user.GetDomainId(), clients); err != nil {
		return []mgclients.Client{}, err
	}
	defer func() {
		if err != nil {
			if errRollback := svc.addThingPoliciesRollback(ctx, user.GetId(), user.GetDomainId(), clients); errRollback != nil {
				err = errors.Wrap(errors.Wrap(errors.ErrRollbackTx, errRollback), err)
			}
		}
	}()

	saved, err := svc.clients.Save(ctx, clients...)
	if err != nil {
		return nil, errors.Wrap(svcerr.ErrCreateEntity, err)
	}

	return saved, nil
}

func (svc service) ViewClient(ctx context.Context, token, id string) (mgclients.Client, error) {
	_, err := svc.authorize(ctx, "", auth.UserType, auth.TokenKind, token, auth.ViewPermission, auth.ThingType, id)
	if err != nil {
		return mgclients.Client{}, err
	}
	client, err := svc.clients.RetrieveByID(ctx, id)
	if err != nil {
		return mgclients.Client{}, errors.Wrap(svcerr.ErrViewEntity, err)
	}
	return client, nil
}

func (svc service) ViewClientPerms(ctx context.Context, token, id string) ([]string, error) {
	res, err := svc.identify(ctx, token)
	if err != nil {
		return nil, err
	}

	permissions, err := svc.listUserThingPermission(ctx, res.GetId(), id)
	if err != nil {
		return nil, err
	}
	if len(permissions) == 0 {
		return nil, svcerr.ErrAuthorization
	}
	return permissions, nil
}

func (svc service) ListClients(ctx context.Context, token, reqUserID string, pm mgclients.Page) (mgclients.ClientsPage, error) {
	var ids []string

	res, err := svc.identify(ctx, token)
	if err != nil {
		return mgclients.ClientsPage{}, err
	}

	switch {
	case (reqUserID != "" && reqUserID != res.GetUserId()):
		// Check user is admin of domain, if yes then show listing on domain context
		if _, err := svc.authorize(ctx, "", auth.UserType, auth.UsersKind, res.GetId(), auth.AdminPermission, auth.DomainType, res.GetDomainId()); err != nil {
			return mgclients.ClientsPage{}, err
		}
		rtids, err := svc.listClientIDs(ctx, auth.EncodeDomainUserID(res.GetDomainId(), reqUserID), pm.Permission)
		if err != nil {
			return mgclients.ClientsPage{}, errors.Wrap(svcerr.ErrNotFound, err)
		}
		ids, err = svc.filterAllowedThingIDs(ctx, res.GetId(), pm.Permission, rtids)
		if err != nil {
			return mgclients.ClientsPage{}, errors.Wrap(svcerr.ErrNotFound, err)
		}
	default:
		err := svc.checkSuperAdmin(ctx, res.GetUserId())
		switch {
		case err == nil:
			pm.Domain = res.GetDomainId()
		default:
			// If domain is disabled , then this authorization will fail for all non-admin domain users
			if _, err := svc.authorize(ctx, "", auth.UserType, auth.UsersKind, res.GetId(), auth.MembershipPermission, auth.DomainType, res.GetDomainId()); err != nil {
				return mgclients.ClientsPage{}, err
			}
			ids, err = svc.listClientIDs(ctx, res.GetId(), pm.Permission)
			if err != nil {
				return mgclients.ClientsPage{}, errors.Wrap(svcerr.ErrNotFound, err)
			}
		}
	}

	if len(ids) == 0 && pm.Domain == "" {
		return mgclients.ClientsPage{}, nil
	}
	pm.IDs = ids
	tp, err := svc.clients.SearchClients(ctx, pm)
	if err != nil {
		return mgclients.ClientsPage{}, errors.Wrap(svcerr.ErrViewEntity, err)
	}

	if pm.ListPerms && len(tp.Clients) > 0 {
		g, ctx := errgroup.WithContext(ctx)

		for i := range tp.Clients {
			// Copying loop variable "i" to avoid "loop variable captured by func literal"
			iter := i
			g.Go(func() error {
				return svc.retrievePermissions(ctx, res.GetId(), &tp.Clients[iter])
			})
		}

		if err := g.Wait(); err != nil {
			return mgclients.ClientsPage{}, err
		}
	}
	return tp, nil
}

// Experimental functions used for async calling of svc.listUserThingPermission. This might be helpful during listing of large number of entities.
func (svc service) retrievePermissions(ctx context.Context, userID string, client *mgclients.Client) error {
	permissions, err := svc.listUserThingPermission(ctx, userID, client.ID)
	if err != nil {
		return err
	}
	client.Permissions = permissions
	return nil
}

func (svc service) listUserThingPermission(ctx context.Context, userID, thingID string) ([]string, error) {
	permissions, err := svc.policy.ListPermissions(ctx, &magistrala.ListPermissionsReq{
		SubjectType: auth.UserType,
		Subject:     userID,
		Object:      thingID,
		ObjectType:  auth.ThingType,
	})
	if err != nil {
		return []string{}, errors.Wrap(svcerr.ErrAuthorization, err)
	}
	return permissions, nil
}

func (svc service) listClientIDs(ctx context.Context, userID, permission string) ([]string, error) {
	tids, err := svc.policy.ListAllObjects(ctx, &magistrala.ListObjectsReq{
		SubjectType: auth.UserType,
		Subject:     userID,
		Permission:  permission,
		ObjectType:  auth.ThingType,
	})
	if err != nil {
		return nil, errors.Wrap(svcerr.ErrNotFound, err)
	}
	return tids, nil
}

func (svc service) filterAllowedThingIDs(ctx context.Context, userID, permission string, thingIDs []string) ([]string, error) {
	var ids []string
	tids, err := svc.policy.ListAllObjects(ctx, &magistrala.ListObjectsReq{
		SubjectType: auth.UserType,
		Subject:     userID,
		Permission:  permission,
		ObjectType:  auth.ThingType,
	})
	if err != nil {
		return nil, errors.Wrap(svcerr.ErrNotFound, err)
	}
	for _, thingID := range thingIDs {
		for _, tid := range tids {
			if thingID == tid {
				ids = append(ids, thingID)
			}
		}
	}
	return ids, nil
}

func (svc service) checkSuperAdmin(ctx context.Context, userID string) error {
	res, err := svc.auth.Authorize(ctx, &magistrala.AuthorizeReq{
		SubjectType: auth.UserType,
		Subject:     userID,
		Permission:  auth.AdminPermission,
		ObjectType:  auth.PlatformType,
		Object:      auth.MagistralaObject,
	})
	if err != nil {
		return err
	}
	if !res.Authorized {
		return svcerr.ErrAuthorization
	}
	return nil
}

func (svc service) UpdateClient(ctx context.Context, token string, cli mgclients.Client) (mgclients.Client, error) {
	userID, err := svc.authorize(ctx, "", auth.UserType, auth.TokenKind, token, auth.EditPermission, auth.ThingType, cli.ID)
	if err != nil {
		return mgclients.Client{}, errors.Wrap(svcerr.ErrAuthorization, err)
	}

	client := mgclients.Client{
		ID:        cli.ID,
		Name:      cli.Name,
		Metadata:  cli.Metadata,
		UpdatedAt: time.Now(),
		UpdatedBy: userID,
	}
	client, err = svc.clients.Update(ctx, client)
	if err != nil {
		return mgclients.Client{}, errors.Wrap(svcerr.ErrUpdateEntity, err)
	}
	return client, nil
}

func (svc service) UpdateClientTags(ctx context.Context, token string, cli mgclients.Client) (mgclients.Client, error) {
	userID, err := svc.authorize(ctx, "", auth.UserType, auth.TokenKind, token, auth.EditPermission, auth.ThingType, cli.ID)
	if err != nil {
		return mgclients.Client{}, errors.Wrap(svcerr.ErrAuthorization, err)
	}

	client := mgclients.Client{
		ID:        cli.ID,
		Tags:      cli.Tags,
		UpdatedAt: time.Now(),
		UpdatedBy: userID,
	}
	client, err = svc.clients.UpdateTags(ctx, client)
	if err != nil {
		return mgclients.Client{}, errors.Wrap(svcerr.ErrUpdateEntity, err)
	}
	return client, nil
}

func (svc service) UpdateClientSecret(ctx context.Context, token, id, key string) (mgclients.Client, error) {
	userID, err := svc.authorize(ctx, "", auth.UserType, auth.TokenKind, token, auth.EditPermission, auth.ThingType, id)
	if err != nil {
		return mgclients.Client{}, errors.Wrap(svcerr.ErrAuthorization, err)
	}

	client := mgclients.Client{
		ID: id,
		Credentials: mgclients.Credentials{
			Secret: key,
		},
		UpdatedAt: time.Now(),
		UpdatedBy: userID,
		Status:    mgclients.EnabledStatus,
	}
	client, err = svc.clients.UpdateSecret(ctx, client)
	if err != nil {
		return mgclients.Client{}, errors.Wrap(svcerr.ErrUpdateEntity, err)
	}
	return client, nil
}

func (svc service) EnableClient(ctx context.Context, token, id string) (mgclients.Client, error) {
	client := mgclients.Client{
		ID:        id,
		Status:    mgclients.EnabledStatus,
		UpdatedAt: time.Now(),
	}
	client, err := svc.changeClientStatus(ctx, token, client)
	if err != nil {
		return mgclients.Client{}, errors.Wrap(mgclients.ErrEnableClient, err)
	}

	return client, nil
}

func (svc service) DisableClient(ctx context.Context, token, id string) (mgclients.Client, error) {
	client := mgclients.Client{
		ID:        id,
		Status:    mgclients.DisabledStatus,
		UpdatedAt: time.Now(),
	}
	client, err := svc.changeClientStatus(ctx, token, client)
	if err != nil {
		return mgclients.Client{}, errors.Wrap(mgclients.ErrDisableClient, err)
	}

	if err := svc.clientCache.Remove(ctx, client.ID); err != nil {
		return client, errors.Wrap(svcerr.ErrRemoveEntity, err)
	}

	return client, nil
}

func (svc service) Share(ctx context.Context, token, id, relation string, userids ...string) error {
	user, err := svc.identify(ctx, token)
	if err != nil {
		return err
	}
	if _, err := svc.authorize(ctx, user.GetDomainId(), auth.UserType, auth.UsersKind, user.GetId(), auth.DeletePermission, auth.ThingType, id); err != nil {
		return errors.Wrap(svcerr.ErrAuthorization, err)
	}

	policies := magistrala.AddPoliciesReq{}
	for _, userid := range userids {
		policies.AddPoliciesReq = append(policies.AddPoliciesReq, &magistrala.AddPolicyReq{
			SubjectType: auth.UserType,
			Subject:     auth.EncodeDomainUserID(user.GetDomainId(), userid),
			Relation:    relation,
			ObjectType:  auth.ThingType,
			Object:      id,
		})
	}
	added, err := svc.policy.AddPolicies(ctx, &policies)
	if err != nil {
		return errors.Wrap(svcerr.ErrUpdateEntity, err)
	}
	if !added {
		return errors.Wrap(svcerr.ErrUpdateEntity, err)
	}
	return nil
}

func (svc service) Unshare(ctx context.Context, token, id, relation string, userids ...string) error {
	user, err := svc.identify(ctx, token)
	if err != nil {
		return err
	}
	if _, err := svc.authorize(ctx, user.GetDomainId(), auth.UserType, auth.UsersKind, user.GetId(), auth.DeletePermission, auth.ThingType, id); err != nil {
		return errors.Wrap(svcerr.ErrAuthorization, err)
	}

	policies := magistrala.DeletePoliciesReq{}
	for _, userid := range userids {
		policies.DeletePoliciesReq = append(policies.DeletePoliciesReq, &magistrala.DeletePolicyReq{
			SubjectType: auth.UserType,
			Subject:     auth.EncodeDomainUserID(user.GetDomainId(), userid),
			Relation:    relation,
			ObjectType:  auth.ThingType,
			Object:      id,
		})
	}
	deleted, err := svc.policy.DeletePolicies(ctx, &policies)
	if err != nil {
		return errors.Wrap(svcerr.ErrUpdateEntity, err)
	}
	if !deleted {
		return err
	}
	return nil
}

func (svc service) DeleteClient(ctx context.Context, token, id string) error {
	res, err := svc.identify(ctx, token)
	if err != nil {
		return err
	}
	if _, err := svc.authorize(ctx, res.GetDomainId(), auth.UserType, auth.UsersKind, res.GetId(), auth.DeletePermission, auth.ThingType, id); err != nil {
		return err
	}

	if err := svc.clientCache.Remove(ctx, id); err != nil {
		return errors.Wrap(svcerr.ErrRemoveEntity, err)
	}

	deleted, err := svc.policy.DeleteEntityPolicies(ctx, &magistrala.DeleteEntityPoliciesReq{
		EntityType: auth.ThingType,
		Id:         id,
	})
	if err != nil {
		return errors.Wrap(svcerr.ErrRemoveEntity, err)
	}
	if !deleted {
		return svcerr.ErrAuthorization
	}

	if err := svc.clients.Delete(ctx, id); err != nil {
		return errors.Wrap(svcerr.ErrRemoveEntity, err)
	}

	return nil
}

func (svc service) changeClientStatus(ctx context.Context, token string, client mgclients.Client) (mgclients.Client, error) {
	userID, err := svc.authorize(ctx, "", auth.UserType, auth.TokenKind, token, auth.DeletePermission, auth.ThingType, client.ID)
	if err != nil {
		return mgclients.Client{}, errors.Wrap(svcerr.ErrAuthorization, err)
	}
	dbClient, err := svc.clients.RetrieveByID(ctx, client.ID)
	if err != nil {
		return mgclients.Client{}, errors.Wrap(svcerr.ErrViewEntity, err)
	}
	if dbClient.Status == client.Status {
		return mgclients.Client{}, errors.ErrStatusAlreadyAssigned
	}

	client.UpdatedBy = userID

	client, err = svc.clients.ChangeStatus(ctx, client)
	if err != nil {
		return mgclients.Client{}, errors.Wrap(svcerr.ErrUpdateEntity, err)
	}
	return client, nil
}

func (svc service) ListClientsByGroup(ctx context.Context, token, groupID string, pm mgclients.Page) (mgclients.MembersPage, error) {
	res, err := svc.identify(ctx, token)
	if err != nil {
		return mgclients.MembersPage{}, err
	}
	if _, err := svc.authorize(ctx, res.GetDomainId(), auth.UserType, auth.UsersKind, res.GetId(), pm.Permission, auth.GroupType, groupID); err != nil {
		return mgclients.MembersPage{}, err
	}

	tids, err := svc.policy.ListAllObjects(ctx, &magistrala.ListObjectsReq{
		SubjectType: auth.GroupType,
		Subject:     groupID,
		Permission:  auth.GroupRelation,
		ObjectType:  auth.ThingType,
	})
	if err != nil {
		return mgclients.MembersPage{}, errors.Wrap(svcerr.ErrNotFound, err)
	}

	pm.IDs = tids

	cp, err := svc.clients.RetrieveAllByIDs(ctx, pm)
	if err != nil {
		return mgclients.MembersPage{}, errors.Wrap(svcerr.ErrViewEntity, err)
	}

	if pm.ListPerms && len(cp.Clients) > 0 {
		g, ctx := errgroup.WithContext(ctx)

		for i := range cp.Clients {
			// Copying loop variable "i" to avoid "loop variable captured by func literal"
			iter := i
			g.Go(func() error {
				return svc.retrievePermissions(ctx, res.GetId(), &cp.Clients[iter])
			})
		}

		if err := g.Wait(); err != nil {
			return mgclients.MembersPage{}, err
		}
	}

	return mgclients.MembersPage{
		Page:    cp.Page,
		Members: cp.Clients,
	}, nil
}

func (svc service) Identify(ctx context.Context, key string) (string, error) {
	id, err := svc.clientCache.ID(ctx, key)
	if err == nil {
		return id, nil
	}

	client, err := svc.clients.RetrieveBySecret(ctx, key)
	if err != nil {
		return "", errors.Wrap(svcerr.ErrAuthorization, err)
	}
	if err := svc.clientCache.Save(ctx, key, client.ID); err != nil {
		return "", errors.Wrap(svcerr.ErrAuthorization, err)
	}

	return client.ID, nil
}

func (svc service) identify(ctx context.Context, token string) (*magistrala.IdentityRes, error) {
	res, err := svc.auth.Identify(ctx, &magistrala.IdentityReq{Token: token})
	if err != nil {
		return nil, errors.Wrap(svcerr.ErrAuthentication, err)
	}
	if res.GetId() == "" || res.GetDomainId() == "" {
		return nil, svcerr.ErrDomainAuthorization
	}
	return res, nil
}

func (svc *service) authorize(ctx context.Context, domainID, subjType, subjKind, subj, perm, objType, obj string) (string, error) {
	req := &magistrala.AuthorizeReq{
		Domain:      domainID,
		SubjectType: subjType,
		SubjectKind: subjKind,
		Subject:     subj,
		Permission:  perm,
		ObjectType:  objType,
		Object:      obj,
	}
	res, err := svc.auth.Authorize(ctx, req)
	if err != nil {
		return "", errors.Wrap(svcerr.ErrAuthorization, err)
	}
	if !res.GetAuthorized() {
		return "", errors.Wrap(svcerr.ErrAuthorization, err)
	}

	return res.GetId(), nil
}

func (svc service) addThingPolicies(ctx context.Context, userID, domainID string, things []mgclients.Client) error {
	policies := magistrala.AddPoliciesReq{}
	for _, thing := range things {
		policies.AddPoliciesReq = append(policies.AddPoliciesReq, &magistrala.AddPolicyReq{
			Domain:      domainID,
			SubjectType: auth.UserType,
			Subject:     userID,
			Relation:    auth.AdministratorRelation,
			ObjectKind:  auth.NewThingKind,
			ObjectType:  auth.ThingType,
			Object:      thing.ID,
		})
		policies.AddPoliciesReq = append(policies.AddPoliciesReq, &magistrala.AddPolicyReq{
			Domain:      domainID,
			SubjectType: auth.DomainType,
			Subject:     domainID,
			Relation:    auth.DomainRelation,
			ObjectType:  auth.ThingType,
			Object:      thing.ID,
		})
	}

	if _, err := svc.policy.AddPolicies(ctx, &policies); err != nil {
		return errors.Wrap(svcerr.ErrCreateEntity, err)
	}

	return nil
}

func (svc service) addThingPoliciesRollback(ctx context.Context, userID, domainID string, things []mgclients.Client) error {
	policies := magistrala.DeletePoliciesReq{}
	for _, thing := range things {
		policies.DeletePoliciesReq = append(policies.DeletePoliciesReq, &magistrala.DeletePolicyReq{
			Domain:      domainID,
			SubjectType: auth.UserType,
			Subject:     userID,
			Relation:    auth.AdministratorRelation,
			ObjectKind:  auth.NewThingKind,
			ObjectType:  auth.ThingType,
			Object:      thing.ID,
		})
		policies.DeletePoliciesReq = append(policies.DeletePoliciesReq, &magistrala.DeletePolicyReq{
			Domain:      domainID,
			SubjectType: auth.DomainType,
			Subject:     domainID,
			Relation:    auth.DomainRelation,
			ObjectType:  auth.ThingType,
			Object:      thing.ID,
		})
	}

	if _, err := svc.policy.DeletePolicies(ctx, &policies); err != nil {
		return errors.Wrap(svcerr.ErrRemoveEntity, err)
	}

	return nil
}
