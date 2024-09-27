// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0
package things

import (
	"context"
	"time"

	"github.com/absmach/magistrala"
	mgauth "github.com/absmach/magistrala/auth"
	"github.com/absmach/magistrala/pkg/auth"
	mgclients "github.com/absmach/magistrala/pkg/clients"
	"github.com/absmach/magistrala/pkg/errors"
	svcerr "github.com/absmach/magistrala/pkg/errors/service"
	mggroups "github.com/absmach/magistrala/pkg/groups"
	"github.com/absmach/magistrala/pkg/policies"
	"github.com/absmach/magistrala/things/postgres"
	"golang.org/x/sync/errgroup"
)

type service struct {
	policies    policies.PolicyClient
	clients     postgres.Repository
	clientCache Cache
	idProvider  magistrala.IDProvider
	grepo       mggroups.Repository
}

// NewService returns a new Clients service implementation.
func NewService(policyClient policies.PolicyClient, c postgres.Repository, grepo mggroups.Repository, tcache Cache, idp magistrala.IDProvider) Service {
	return service{
		policies:    policyClient,
		clients:     c,
		grepo:       grepo,
		clientCache: tcache,
		idProvider:  idp,
	}
}

func (svc service) CreateThings(ctx context.Context, session auth.Session, cls ...mgclients.Client) ([]mgclients.Client, error) {
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
		c.Domain = session.DomainID
		c.CreatedAt = time.Now()
		clients = append(clients, c)
	}

	err := svc.addThingPolicies(ctx, session.DomainUserID, session.DomainID, clients)
	if err != nil {
		return []mgclients.Client{}, err
	}
	defer func() {
		if err != nil {
			if errRollback := svc.addThingPoliciesRollback(ctx, session.DomainUserID, session.DomainID, clients); errRollback != nil {
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

func (svc service) ViewClient(ctx context.Context, session auth.Session, id string) (mgclients.Client, error) {
	client, err := svc.clients.RetrieveByID(ctx, id)
	if err != nil {
		return mgclients.Client{}, errors.Wrap(svcerr.ErrViewEntity, err)
	}
	return client, nil
}

func (svc service) ViewClientPerms(ctx context.Context, session auth.Session, id string) ([]string, error) {
	permissions, err := svc.listUserThingPermission(ctx, session.DomainUserID, id)
	if err != nil {
		return nil, err
	}
	if len(permissions) == 0 {
		return nil, svcerr.ErrAuthorization
	}
	return permissions, nil
}

func (svc service) ListClients(ctx context.Context, session auth.Session, reqUserID string, pm mgclients.Page) (mgclients.ClientsPage, error) {
	var ids []string
	var err error
	switch {
	case (reqUserID != "" && reqUserID != session.UserID):
		rtids, err := svc.listClientIDs(ctx, mgauth.EncodeDomainUserID(session.DomainID, reqUserID), pm.Permission)
		if err != nil {
			return mgclients.ClientsPage{}, errors.Wrap(svcerr.ErrNotFound, err)
		}
		ids, err = svc.filterAllowedThingIDs(ctx, session.DomainUserID, pm.Permission, rtids)
		if err != nil {
			return mgclients.ClientsPage{}, errors.Wrap(svcerr.ErrNotFound, err)
		}
	default:
		switch session.SuperAdmin {
		case true:
			pm.Domain = session.DomainID
		default:
			ids, err = svc.listClientIDs(ctx, session.DomainUserID, pm.Permission)
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
				return svc.retrievePermissions(ctx, session.DomainUserID, &tp.Clients[iter])
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
	permissions, err := svc.policies.ListPermissions(ctx, policies.PolicyReq{
		SubjectType: policies.UserType,
		Subject:     userID,
		Object:      thingID,
		ObjectType:  policies.ThingType,
	}, []string{})
	if err != nil {
		return []string{}, errors.Wrap(svcerr.ErrAuthorization, err)
	}
	return permissions, nil
}

func (svc service) listClientIDs(ctx context.Context, userID, permission string) ([]string, error) {
	tids, err := svc.policies.ListAllObjects(ctx, policies.PolicyReq{
		SubjectType: policies.UserType,
		Subject:     userID,
		Permission:  permission,
		ObjectType:  policies.ThingType,
	})
	if err != nil {
		return nil, errors.Wrap(svcerr.ErrNotFound, err)
	}
	return tids.Policies, nil
}

func (svc service) filterAllowedThingIDs(ctx context.Context, userID, permission string, thingIDs []string) ([]string, error) {
	var ids []string
	tids, err := svc.policies.ListAllObjects(ctx, policies.PolicyReq{
		SubjectType: policies.UserType,
		Subject:     userID,
		Permission:  permission,
		ObjectType:  policies.ThingType,
	})
	if err != nil {
		return nil, errors.Wrap(svcerr.ErrNotFound, err)
	}
	for _, thingID := range thingIDs {
		for _, tid := range tids.Policies {
			if thingID == tid {
				ids = append(ids, thingID)
			}
		}
	}
	return ids, nil
}

func (svc service) UpdateClient(ctx context.Context, session auth.Session, cli mgclients.Client) (mgclients.Client, error) {
	client := mgclients.Client{
		ID:        cli.ID,
		Name:      cli.Name,
		Metadata:  cli.Metadata,
		UpdatedAt: time.Now(),
		UpdatedBy: session.UserID,
	}
	client, err := svc.clients.Update(ctx, client)
	if err != nil {
		return mgclients.Client{}, errors.Wrap(svcerr.ErrUpdateEntity, err)
	}
	return client, nil
}

func (svc service) UpdateClientTags(ctx context.Context, session auth.Session, cli mgclients.Client) (mgclients.Client, error) {
	client := mgclients.Client{
		ID:        cli.ID,
		Tags:      cli.Tags,
		UpdatedAt: time.Now(),
		UpdatedBy: session.UserID,
	}
	client, err := svc.clients.UpdateTags(ctx, client)
	if err != nil {
		return mgclients.Client{}, errors.Wrap(svcerr.ErrUpdateEntity, err)
	}
	return client, nil
}

func (svc service) UpdateClientSecret(ctx context.Context, session auth.Session, id, key string) (mgclients.Client, error) {
	client := mgclients.Client{
		ID: id,
		Credentials: mgclients.Credentials{
			Secret: key,
		},
		UpdatedAt: time.Now(),
		UpdatedBy: session.UserID,
		Status:    mgclients.EnabledStatus,
	}
	client, err := svc.clients.UpdateSecret(ctx, client)
	if err != nil {
		return mgclients.Client{}, errors.Wrap(svcerr.ErrUpdateEntity, err)
	}
	return client, nil
}

func (svc service) EnableClient(ctx context.Context, session auth.Session, id string) (mgclients.Client, error) {
	client := mgclients.Client{
		ID:        id,
		Status:    mgclients.EnabledStatus,
		UpdatedAt: time.Now(),
	}
	client, err := svc.changeClientStatus(ctx, session, client)
	if err != nil {
		return mgclients.Client{}, errors.Wrap(mgclients.ErrEnableClient, err)
	}

	return client, nil
}

func (svc service) DisableClient(ctx context.Context, session auth.Session, id string) (mgclients.Client, error) {
	client := mgclients.Client{
		ID:        id,
		Status:    mgclients.DisabledStatus,
		UpdatedAt: time.Now(),
	}
	client, err := svc.changeClientStatus(ctx, session, client)
	if err != nil {
		return mgclients.Client{}, errors.Wrap(mgclients.ErrDisableClient, err)
	}

	if err := svc.clientCache.Remove(ctx, client.ID); err != nil {
		return client, errors.Wrap(svcerr.ErrRemoveEntity, err)
	}

	return client, nil
}

func (svc service) Share(ctx context.Context, session auth.Session, id, relation string, userids ...string) error {
	policyList := []policies.PolicyReq{}
	for _, userid := range userids {
		policyList = append(policyList, policies.PolicyReq{
			SubjectType: policies.UserType,
			Subject:     mgauth.EncodeDomainUserID(session.DomainID, userid),
			Relation:    relation,
			ObjectType:  policies.ThingType,
			Object:      id,
		})
	}
	if err := svc.policies.AddPolicies(ctx, policyList); err != nil {
		return errors.Wrap(svcerr.ErrUpdateEntity, err)
	}

	return nil
}

func (svc service) Unshare(ctx context.Context, session auth.Session, id, relation string, userids ...string) error {
	policyList := []policies.PolicyReq{}
	for _, userid := range userids {
		policyList = append(policyList, policies.PolicyReq{
			SubjectType: policies.UserType,
			Subject:     mgauth.EncodeDomainUserID(session.DomainID, userid),
			Relation:    relation,
			ObjectType:  policies.ThingType,
			Object:      id,
		})
	}
	if err := svc.policies.DeletePolicies(ctx, policyList); err != nil {
		return errors.Wrap(svcerr.ErrUpdateEntity, err)
	}

	return nil
}

func (svc service) DeleteClient(ctx context.Context, session auth.Session, id string) error {
	if err := svc.clientCache.Remove(ctx, id); err != nil {
		return errors.Wrap(svcerr.ErrRemoveEntity, err)
	}

	req := policies.PolicyReq{
		Object:     id,
		ObjectType: policies.ThingType,
	}

	if err := svc.policies.DeletePolicyFilter(ctx, req); err != nil {
		return errors.Wrap(svcerr.ErrRemoveEntity, err)
	}

	if err := svc.clients.Delete(ctx, id); err != nil {
		return errors.Wrap(svcerr.ErrRemoveEntity, err)
	}

	return nil
}

func (svc service) changeClientStatus(ctx context.Context, session auth.Session, client mgclients.Client) (mgclients.Client, error) {
	dbClient, err := svc.clients.RetrieveByID(ctx, client.ID)
	if err != nil {
		return mgclients.Client{}, errors.Wrap(svcerr.ErrViewEntity, err)
	}
	if dbClient.Status == client.Status {
		return mgclients.Client{}, errors.ErrStatusAlreadyAssigned
	}

	client.UpdatedBy = session.UserID

	client, err = svc.clients.ChangeStatus(ctx, client)
	if err != nil {
		return mgclients.Client{}, errors.Wrap(svcerr.ErrUpdateEntity, err)
	}
	return client, nil
}

func (svc service) ListClientsByGroup(ctx context.Context, session auth.Session, groupID string, pm mgclients.Page) (mgclients.MembersPage, error) {
	tids, err := svc.policies.ListAllObjects(ctx, policies.PolicyReq{
		SubjectType: policies.GroupType,
		Subject:     groupID,
		Permission:  policies.GroupRelation,
		ObjectType:  policies.ThingType,
	})
	if err != nil {
		return mgclients.MembersPage{}, errors.Wrap(svcerr.ErrNotFound, err)
	}

	pm.IDs = tids.Policies

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
				return svc.retrievePermissions(ctx, session.DomainUserID, &cp.Clients[iter])
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

func (svc service) addThingPolicies(ctx context.Context, userID, domainID string, things []mgclients.Client) error {
	policyList := []policies.PolicyReq{}
	for _, thing := range things {
		policyList = append(policyList, policies.PolicyReq{
			Domain:      domainID,
			SubjectType: policies.UserType,
			Subject:     userID,
			Relation:    policies.AdministratorRelation,
			ObjectKind:  policies.NewThingKind,
			ObjectType:  policies.ThingType,
			Object:      thing.ID,
		})
		policyList = append(policyList, policies.PolicyReq{
			Domain:      domainID,
			SubjectType: policies.DomainType,
			Subject:     domainID,
			Relation:    policies.DomainRelation,
			ObjectType:  policies.ThingType,
			Object:      thing.ID,
		})
	}
	if err := svc.policies.AddPolicies(ctx, policyList); err != nil {
		return errors.Wrap(svcerr.ErrCreateEntity, err)
	}

	return nil
}

func (svc service) addThingPoliciesRollback(ctx context.Context, userID, domainID string, things []mgclients.Client) error {
	policyList := []policies.PolicyReq{}
	for _, thing := range things {
		policyList = append(policyList, policies.PolicyReq{
			Domain:      domainID,
			SubjectType: policies.UserType,
			Subject:     userID,
			Relation:    policies.AdministratorRelation,
			ObjectKind:  policies.NewThingKind,
			ObjectType:  policies.ThingType,
			Object:      thing.ID,
		})
		policyList = append(policyList, policies.PolicyReq{
			Domain:      domainID,
			SubjectType: policies.DomainType,
			Subject:     domainID,
			Relation:    policies.DomainRelation,
			ObjectType:  policies.ThingType,
			Object:      thing.ID,
		})
	}
	if err := svc.policies.DeletePolicies(ctx, policyList); err != nil {
		return errors.Wrap(svcerr.ErrRemoveEntity, err)
	}

	return nil
}
