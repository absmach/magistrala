// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0
package things

import (
	"context"
	"time"

	"github.com/absmach/magistrala"
	mgauth "github.com/absmach/magistrala/auth"
	"github.com/absmach/magistrala/pkg/authn"
	"github.com/absmach/magistrala/pkg/errors"
	svcerr "github.com/absmach/magistrala/pkg/errors/service"
	"github.com/absmach/magistrala/pkg/policies"
	"golang.org/x/sync/errgroup"
)

type service struct {
	evaluator   policies.Evaluator
	policysvc   policies.Service
	clients     Repository
	clientCache Cache
	idProvider  magistrala.IDProvider
}

// NewService returns a new Things service implementation.
func NewService(policyEvaluator policies.Evaluator, policyService policies.Service, c Repository, tcache Cache, idp magistrala.IDProvider) Service {
	return service{
		evaluator:   policyEvaluator,
		policysvc:   policyService,
		clients:     c,
		clientCache: tcache,
		idProvider:  idp,
	}
}

func (svc service) Authorize(ctx context.Context, req AuthzReq) (string, error) {
	clientID, err := svc.Identify(ctx, req.ClientKey)
	if err != nil {
		return "", err
	}

	r := policies.Policy{
		SubjectType: policies.GroupType,
		Subject:     req.ChannelID,
		ObjectType:  policies.ThingType,
		Object:      clientID,
		Permission:  req.Permission,
	}
	err = svc.evaluator.CheckPolicy(ctx, r)
	if err != nil {
		return "", errors.Wrap(svcerr.ErrAuthorization, err)
	}

	return clientID, nil
}

func (svc service) CreateClients(ctx context.Context, session authn.Session, cli ...Client) ([]Client, error) {
	var clients []Client
	for _, c := range cli {
		if c.ID == "" {
			clientID, err := svc.idProvider.ID()
			if err != nil {
				return []Client{}, err
			}
			c.ID = clientID
		}
		if c.Credentials.Secret == "" {
			key, err := svc.idProvider.ID()
			if err != nil {
				return []Client{}, err
			}
			c.Credentials.Secret = key
		}
		if c.Status != DisabledStatus && c.Status != EnabledStatus {
			return []Client{}, svcerr.ErrInvalidStatus
		}
		c.Domain = session.DomainID
		c.CreatedAt = time.Now()
		clients = append(clients, c)
	}

	err := svc.addClientPolicies(ctx, session.DomainUserID, session.DomainID, clients)
	if err != nil {
		return []Client{}, err
	}
	defer func() {
		if err != nil {
			if errRollback := svc.addClientPoliciesRollback(ctx, session.DomainUserID, session.DomainID, clients); errRollback != nil {
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

func (svc service) View(ctx context.Context, session authn.Session, id string) (Client, error) {
	client, err := svc.clients.RetrieveByID(ctx, id)
	if err != nil {
		return Client{}, errors.Wrap(svcerr.ErrViewEntity, err)
	}
	return client, nil
}

func (svc service) ViewPerms(ctx context.Context, session authn.Session, id string) ([]string, error) {
	permissions, err := svc.listUserClientPermission(ctx, session.DomainUserID, id)
	if err != nil {
		return nil, err
	}
	if len(permissions) == 0 {
		return nil, svcerr.ErrAuthorization
	}
	return permissions, nil
}

func (svc service) ListClients(ctx context.Context, session authn.Session, reqUserID string, pm Page) (ClientsPage, error) {
	var ids []string
	var err error
	switch {
	case (reqUserID != "" && reqUserID != session.UserID):
		rtids, err := svc.listClientIDs(ctx, mgauth.EncodeDomainUserID(session.DomainID, reqUserID), pm.Permission)
		if err != nil {
			return ClientsPage{}, errors.Wrap(svcerr.ErrNotFound, err)
		}
		ids, err = svc.filterAllowedClientIDs(ctx, session.DomainUserID, pm.Permission, rtids)
		if err != nil {
			return ClientsPage{}, errors.Wrap(svcerr.ErrNotFound, err)
		}
	default:
		switch session.SuperAdmin {
		case true:
			pm.Domain = session.DomainID
		default:
			ids, err = svc.listClientIDs(ctx, session.DomainUserID, pm.Permission)
			if err != nil {
				return ClientsPage{}, errors.Wrap(svcerr.ErrNotFound, err)
			}
		}
	}

	if len(ids) == 0 && pm.Domain == "" {
		return ClientsPage{}, nil
	}
	pm.IDs = ids
	tp, err := svc.clients.SearchClients(ctx, pm)
	if err != nil {
		return ClientsPage{}, errors.Wrap(svcerr.ErrViewEntity, err)
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
			return ClientsPage{}, err
		}
	}
	return tp, nil
}

// Experimental functions used for async calling of svc.listUserClientPermission. This might be helpful during listing of large number of entities.
func (svc service) retrievePermissions(ctx context.Context, userID string, client *Client) error {
	permissions, err := svc.listUserClientPermission(ctx, userID, client.ID)
	if err != nil {
		return err
	}
	client.Permissions = permissions
	return nil
}

func (svc service) listUserClientPermission(ctx context.Context, userID, clientID string) ([]string, error) {
	permissions, err := svc.policysvc.ListPermissions(ctx, policies.Policy{
		SubjectType: policies.UserType,
		Subject:     userID,
		Object:      clientID,
		ObjectType:  policies.ThingType,
	}, []string{})
	if err != nil {
		return []string{}, errors.Wrap(svcerr.ErrAuthorization, err)
	}
	return permissions, nil
}

func (svc service) listClientIDs(ctx context.Context, userID, permission string) ([]string, error) {
	tids, err := svc.policysvc.ListAllObjects(ctx, policies.Policy{
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

func (svc service) filterAllowedClientIDs(ctx context.Context, userID, permission string, clientIDs []string) ([]string, error) {
	var ids []string
	tids, err := svc.policysvc.ListAllObjects(ctx, policies.Policy{
		SubjectType: policies.UserType,
		Subject:     userID,
		Permission:  permission,
		ObjectType:  policies.ThingType,
	})
	if err != nil {
		return nil, errors.Wrap(svcerr.ErrNotFound, err)
	}
	for _, clientID := range clientIDs {
		for _, tid := range tids.Policies {
			if clientID == tid {
				ids = append(ids, clientID)
			}
		}
	}
	return ids, nil
}

func (svc service) Update(ctx context.Context, session authn.Session, thi Client) (Client, error) {
	client := Client{
		ID:        thi.ID,
		Name:      thi.Name,
		Metadata:  thi.Metadata,
		UpdatedAt: time.Now(),
		UpdatedBy: session.UserID,
	}
	client, err := svc.clients.Update(ctx, client)
	if err != nil {
		return Client{}, errors.Wrap(svcerr.ErrUpdateEntity, err)
	}
	return client, nil
}

func (svc service) UpdateTags(ctx context.Context, session authn.Session, thi Client) (Client, error) {
	client := Client{
		ID:        thi.ID,
		Tags:      thi.Tags,
		UpdatedAt: time.Now(),
		UpdatedBy: session.UserID,
	}
	client, err := svc.clients.UpdateTags(ctx, client)
	if err != nil {
		return Client{}, errors.Wrap(svcerr.ErrUpdateEntity, err)
	}
	return client, nil
}

func (svc service) UpdateSecret(ctx context.Context, session authn.Session, id, key string) (Client, error) {
	client := Client{
		ID: id,
		Credentials: Credentials{
			Secret: key,
		},
		UpdatedAt: time.Now(),
		UpdatedBy: session.UserID,
		Status:    EnabledStatus,
	}
	client, err := svc.clients.UpdateSecret(ctx, client)
	if err != nil {
		return Client{}, errors.Wrap(svcerr.ErrUpdateEntity, err)
	}
	return client, nil
}

func (svc service) Enable(ctx context.Context, session authn.Session, id string) (Client, error) {
	client := Client{
		ID:        id,
		Status:    EnabledStatus,
		UpdatedAt: time.Now(),
	}
	client, err := svc.changeClientStatus(ctx, session, client)
	if err != nil {
		return Client{}, errors.Wrap(ErrEnableClient, err)
	}

	return client, nil
}

func (svc service) Disable(ctx context.Context, session authn.Session, id string) (Client, error) {
	client := Client{
		ID:        id,
		Status:    DisabledStatus,
		UpdatedAt: time.Now(),
	}
	client, err := svc.changeClientStatus(ctx, session, client)
	if err != nil {
		return Client{}, errors.Wrap(ErrDisableClient, err)
	}

	if err := svc.clientCache.Remove(ctx, client.ID); err != nil {
		return client, errors.Wrap(svcerr.ErrRemoveEntity, err)
	}

	return client, nil
}

func (svc service) Share(ctx context.Context, session authn.Session, id, relation string, userids ...string) error {
	policyList := []policies.Policy{}
	for _, userid := range userids {
		policyList = append(policyList, policies.Policy{
			SubjectType: policies.UserType,
			Subject:     mgauth.EncodeDomainUserID(session.DomainID, userid),
			Relation:    relation,
			ObjectType:  policies.ThingType,
			Object:      id,
		})
	}
	if err := svc.policysvc.AddPolicies(ctx, policyList); err != nil {
		return errors.Wrap(svcerr.ErrUpdateEntity, err)
	}

	return nil
}

func (svc service) Unshare(ctx context.Context, session authn.Session, id, relation string, userids ...string) error {
	policyList := []policies.Policy{}
	for _, userid := range userids {
		policyList = append(policyList, policies.Policy{
			SubjectType: policies.UserType,
			Subject:     mgauth.EncodeDomainUserID(session.DomainID, userid),
			Relation:    relation,
			ObjectType:  policies.ThingType,
			Object:      id,
		})
	}
	if err := svc.policysvc.DeletePolicies(ctx, policyList); err != nil {
		return errors.Wrap(svcerr.ErrUpdateEntity, err)
	}

	return nil
}

func (svc service) Delete(ctx context.Context, session authn.Session, id string) error {
	if err := svc.clientCache.Remove(ctx, id); err != nil {
		return errors.Wrap(svcerr.ErrRemoveEntity, err)
	}

	req := policies.Policy{
		Object:     id,
		ObjectType: policies.ThingType,
	}

	if err := svc.policysvc.DeletePolicyFilter(ctx, req); err != nil {
		return errors.Wrap(svcerr.ErrRemoveEntity, err)
	}

	if err := svc.clients.Delete(ctx, id); err != nil {
		return errors.Wrap(svcerr.ErrRemoveEntity, err)
	}

	return nil
}

func (svc service) changeClientStatus(ctx context.Context, session authn.Session, client Client) (Client, error) {
	dbClient, err := svc.clients.RetrieveByID(ctx, client.ID)
	if err != nil {
		return Client{}, errors.Wrap(svcerr.ErrViewEntity, err)
	}
	if dbClient.Status == client.Status {
		return Client{}, errors.ErrStatusAlreadyAssigned
	}

	client.UpdatedBy = session.UserID

	client, err = svc.clients.ChangeStatus(ctx, client)
	if err != nil {
		return Client{}, errors.Wrap(svcerr.ErrUpdateEntity, err)
	}
	return client, nil
}

func (svc service) ListClientsByGroup(ctx context.Context, session authn.Session, groupID string, pm Page) (MembersPage, error) {
	tids, err := svc.policysvc.ListAllObjects(ctx, policies.Policy{
		SubjectType: policies.GroupType,
		Subject:     groupID,
		Permission:  policies.GroupRelation,
		ObjectType:  policies.ThingType,
	})
	if err != nil {
		return MembersPage{}, errors.Wrap(svcerr.ErrNotFound, err)
	}

	pm.IDs = tids.Policies

	cp, err := svc.clients.RetrieveAllByIDs(ctx, pm)
	if err != nil {
		return MembersPage{}, errors.Wrap(svcerr.ErrViewEntity, err)
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
			return MembersPage{}, err
		}
	}

	return MembersPage{
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

func (svc service) addClientPolicies(ctx context.Context, userID, domainID string, clients []Client) error {
	policyList := []policies.Policy{}
	for _, client := range clients {
		policyList = append(policyList, policies.Policy{
			Domain:      domainID,
			SubjectType: policies.UserType,
			Subject:     userID,
			Relation:    policies.AdministratorRelation,
			ObjectKind:  policies.NewThingKind,
			ObjectType:  policies.ThingType,
			Object:      client.ID,
		})
		policyList = append(policyList, policies.Policy{
			Domain:      domainID,
			SubjectType: policies.DomainType,
			Subject:     domainID,
			Relation:    policies.DomainRelation,
			ObjectType:  policies.ThingType,
			Object:      client.ID,
		})
	}
	if err := svc.policysvc.AddPolicies(ctx, policyList); err != nil {
		return errors.Wrap(svcerr.ErrCreateEntity, err)
	}

	return nil
}

func (svc service) addClientPoliciesRollback(ctx context.Context, userID, domainID string, clients []Client) error {
	policyList := []policies.Policy{}
	for _, client := range clients {
		policyList = append(policyList, policies.Policy{
			Domain:      domainID,
			SubjectType: policies.UserType,
			Subject:     userID,
			Relation:    policies.AdministratorRelation,
			ObjectKind:  policies.NewThingKind,
			ObjectType:  policies.ThingType,
			Object:      client.ID,
		})
		policyList = append(policyList, policies.Policy{
			Domain:      domainID,
			SubjectType: policies.DomainType,
			Subject:     domainID,
			Relation:    policies.DomainRelation,
			ObjectType:  policies.ThingType,
			Object:      client.ID,
		})
	}
	if err := svc.policysvc.DeletePolicies(ctx, policyList); err != nil {
		return errors.Wrap(svcerr.ErrRemoveEntity, err)
	}

	return nil
}
