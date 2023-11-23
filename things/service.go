// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0
package things

import (
	"context"
	"time"

	"github.com/absmach/magistrala"
	"github.com/absmach/magistrala/auth"
	mgclients "github.com/absmach/magistrala/pkg/clients"
	"github.com/absmach/magistrala/pkg/errors"
	repoerr "github.com/absmach/magistrala/pkg/errors/repository"
	svcerr "github.com/absmach/magistrala/pkg/errors/service"
	mggroups "github.com/absmach/magistrala/pkg/groups"
	"github.com/absmach/magistrala/things/postgres"
)

var (
	errAddPolicies    = errors.New("failed to add policies")
	errRemovePolicies = errors.New("failed to remove the policies")
)

type service struct {
	auth        magistrala.AuthServiceClient
	clients     postgres.Repository
	clientCache Cache
	idProvider  magistrala.IDProvider
	grepo       mggroups.Repository
}

// NewService returns a new Clients service implementation.
func NewService(uauth magistrala.AuthServiceClient, c postgres.Repository, grepo mggroups.Repository, tcache Cache, idp magistrala.IDProvider) Service {
	return service{
		auth:        uauth,
		clients:     c,
		grepo:       grepo,
		clientCache: tcache,
		idProvider:  idp,
	}
}

func (svc service) Authorize(ctx context.Context, req *magistrala.AuthorizeReq) (string, error) {
	thingID, err := svc.Identify(ctx, req.GetSubject())
	if err != nil {
		return "", errors.Wrap(svcerr.ErrAuthentication, err)
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
		return "", errors.Wrap(errors.ErrAuthorization, err)
	}
	if !resp.GetAuthorized() {
		return "", errors.Wrap(errors.ErrAuthorization, err)
	}

	return thingID, nil
}

func (svc service) CreateThings(ctx context.Context, token string, cls ...mgclients.Client) ([]mgclients.Client, error) {
	user, err := svc.identify(ctx, token)
	if err != nil {
		return []mgclients.Client{}, errors.Wrap(svcerr.ErrAuthentication, err)
	}
	var clients []mgclients.Client
	for _, c := range cls {
		if c.ID == "" {
			clientID, err := svc.idProvider.ID()
			if err != nil {
				return []mgclients.Client{}, errors.Wrap(svcerr.ErrUniqueID, err)
			}
			c.ID = clientID
		}
		if c.Credentials.Secret == "" {
			key, err := svc.idProvider.ID()
			if err != nil {
				return []mgclients.Client{}, errors.Wrap(svcerr.ErrUniqueID, err)
			}
			c.Credentials.Secret = key
		}
		if c.Status != mgclients.DisabledStatus && c.Status != mgclients.EnabledStatus {
			return []mgclients.Client{}, svcerr.ErrInvalidStatus
		}
		c.CreatedAt = time.Now()
		clients = append(clients, c)
	}

	saved, err := svc.clients.Save(ctx, clients...)
	if err != nil {
		return nil, errors.Wrap(repoerr.ErrCreateEntity, err)
	}

	policies := magistrala.AddPoliciesReq{}
	for _, c := range saved {
		policies.AddPoliciesReq = append(policies.AddPoliciesReq, &magistrala.AddPolicyReq{
			Domain:      user.GetDomainId(),
			SubjectType: auth.UserType,
			Subject:     user.GetId(),
			Relation:    auth.AdministratorRelation,
			ObjectKind:  auth.NewThingKind,
			ObjectType:  auth.ThingType,
			Object:      c.ID,
		})
		policies.AddPoliciesReq = append(policies.AddPoliciesReq, &magistrala.AddPolicyReq{
			Domain:      user.GetDomainId(),
			SubjectType: auth.DomainType,
			Subject:     user.GetDomainId(),
			Relation:    auth.DomainRelation,
			ObjectType:  auth.ThingType,
			Object:      c.ID,
		})
	}
	if _, err := svc.auth.AddPolicies(ctx, &policies); err != nil {
		return nil, errors.Wrap(errAddPolicies, err)
	}

	return saved, nil
}

func (svc service) ViewClient(ctx context.Context, token string, id string) (mgclients.Client, error) {
	_, err := svc.authorize(ctx, auth.UserType, auth.TokenKind, token, auth.ViewPermission, auth.ThingType, id)
	if err != nil {
		return mgclients.Client{}, errors.Wrap(svcerr.ErrAuthorization, err)
	}

	return svc.clients.RetrieveByID(ctx, id)
}

func (svc service) ListClients(ctx context.Context, token string, reqUserID string, pm mgclients.Page) (mgclients.ClientsPage, error) {
	var ids []string

	res, err := svc.identify(ctx, token)
	if err != nil {
		return mgclients.ClientsPage{}, errors.Wrap(svcerr.ErrAuthentication, err)
	}

	switch {
	case (reqUserID != "" && reqUserID != res.GetUserId()):
		// Check user is admin of domain, if yes then show listing on domain context
		if _, err := svc.authorize(ctx, auth.UserType, auth.UsersKind, res.GetId(), auth.AdminPermission, auth.DomainType, res.GetDomainId()); err != nil {
			return mgclients.ClientsPage{}, err
		}
		rtids, err := svc.listClientIDs(ctx, auth.EncodeDomainUserID(res.GetDomainId(), reqUserID), pm.Permission)
		if err != nil {
			return mgclients.ClientsPage{}, errors.Wrap(repoerr.ErrNotFound, err)
		}
		ids, err = svc.filterAllowedThingIDs(ctx, res.GetId(), pm.Permission, rtids)
		if err != nil {
			return mgclients.ClientsPage{}, errors.Wrap(repoerr.ErrNotFound, err)
		}
	default:
		ids, err = svc.listClientIDs(ctx, res.GetId(), pm.Permission)
		if err != nil {
			return mgclients.ClientsPage{}, errors.Wrap(repoerr.ErrNotFound, err)
		}
	}

	if len(ids) == 0 {
		return mgclients.ClientsPage{
			Page: mgclients.Page{Total: 0, Limit: pm.Limit, Offset: pm.Offset},
		}, nil
	}

	pm.IDs = ids

	return svc.clients.RetrieveAllByIDs(ctx, pm)
}

func (svc service) listClientIDs(ctx context.Context, userID, permission string) ([]string, error) {
	tids, err := svc.auth.ListAllObjects(ctx, &magistrala.ListObjectsReq{
		SubjectType: auth.UserType,
		Subject:     userID,
		Permission:  permission,
		ObjectType:  auth.ThingType,
	})
	if err != nil {
		return nil, errors.Wrap(repoerr.ErrNotFound, err)
	}
	return tids.Policies, nil
}

func (svc service) filterAllowedThingIDs(ctx context.Context, userID, permission string, thingIDs []string) ([]string, error) {
	var ids []string
	tids, err := svc.auth.ListAllObjects(ctx, &magistrala.ListObjectsReq{
		SubjectType: auth.UserType,
		Subject:     userID,
		Permission:  permission,
		ObjectType:  auth.ThingType,
	})
	if err != nil {
		return nil, errors.Wrap(repoerr.ErrNotFound, err)
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

func (svc service) UpdateClient(ctx context.Context, token string, cli mgclients.Client) (mgclients.Client, error) {
	userID, err := svc.authorize(ctx, auth.UserType, auth.TokenKind, token, auth.EditPermission, auth.ThingType, cli.ID)
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
	return svc.clients.Update(ctx, client)
}

func (svc service) UpdateClientTags(ctx context.Context, token string, cli mgclients.Client) (mgclients.Client, error) {
	userID, err := svc.authorize(ctx, auth.UserType, auth.TokenKind, token, auth.EditPermission, auth.ThingType, cli.ID)
	if err != nil {
		return mgclients.Client{}, errors.Wrap(svcerr.ErrAuthorization, err)
	}

	client := mgclients.Client{
		ID:        cli.ID,
		Tags:      cli.Tags,
		UpdatedAt: time.Now(),
		UpdatedBy: userID,
	}
	return svc.clients.UpdateTags(ctx, client)
}

func (svc service) UpdateClientSecret(ctx context.Context, token, id, key string) (mgclients.Client, error) {
	userID, err := svc.authorize(ctx, auth.UserType, auth.TokenKind, token, auth.EditPermission, auth.ThingType, id)
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
	return svc.clients.UpdateSecret(ctx, client)
}

func (svc service) UpdateClientOwner(ctx context.Context, token string, cli mgclients.Client) (mgclients.Client, error) {
	userID, err := svc.authorize(ctx, auth.UserType, auth.TokenKind, token, auth.EditPermission, auth.ThingType, cli.ID)
	if err != nil {
		return mgclients.Client{}, errors.Wrap(svcerr.ErrAuthorization, err)
	}

	client := mgclients.Client{
		ID:        cli.ID,
		Owner:     cli.Owner,
		UpdatedAt: time.Now(),
		UpdatedBy: userID,
		Status:    mgclients.EnabledStatus,
	}
	return svc.clients.UpdateOwner(ctx, client)
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
		return client, errors.Wrap(repoerr.ErrRemoveEntity, err)
	}

	return client, nil
}

func (svc service) Share(ctx context.Context, token, id, relation string, userids ...string) error {
	user, err := svc.identify(ctx, token)
	if err != nil {
		return nil
	}
	if _, err := svc.authorize(ctx, auth.UserType, auth.UsersKind, user.GetId(), auth.DeletePermission, auth.ThingType, id); err != nil {
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
	res, err := svc.auth.AddPolicies(ctx, &policies)
	if err != nil {
		return errors.Wrap(errAddPolicies, err)
	}
	if !res.Authorized {
		return err
	}
	return nil
}

func (svc service) Unshare(ctx context.Context, token, id, relation string, userids ...string) error {
	user, err := svc.identify(ctx, token)
	if err != nil {
		return nil
	}
	if _, err := svc.authorize(ctx, auth.UserType, auth.UsersKind, user.GetId(), auth.DeletePermission, auth.ThingType, id); err != nil {
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
	res, err := svc.auth.DeletePolicies(ctx, &policies)
	if err != nil {
		return errors.Wrap(errRemovePolicies, err)
	}
	if !res.Deleted {
		return err
	}
	return nil
}

func (svc service) changeClientStatus(ctx context.Context, token string, client mgclients.Client) (mgclients.Client, error) {
	userID, err := svc.authorize(ctx, auth.UserType, auth.TokenKind, token, auth.DeletePermission, auth.ThingType, client.ID)
	if err != nil {
		return mgclients.Client{}, errors.Wrap(svcerr.ErrAuthorization, err)
	}
	dbClient, err := svc.clients.RetrieveByID(ctx, client.ID)
	if err != nil {
		return mgclients.Client{}, errors.Wrap(repoerr.ErrNotFound, err)
	}
	if dbClient.Status == client.Status {
		return mgclients.Client{}, mgclients.ErrStatusAlreadyAssigned
	}

	client.UpdatedBy = userID
	return svc.clients.ChangeStatus(ctx, client)
}

func (svc service) ListClientsByGroup(ctx context.Context, token, groupID string, pm mgclients.Page) (mgclients.MembersPage, error) {
	if _, err := svc.authorize(ctx, auth.UserType, auth.TokenKind, token, pm.Permission, auth.GroupType, groupID); err != nil {
		return mgclients.MembersPage{}, err
	}

	tids, err := svc.auth.ListAllObjects(ctx, &magistrala.ListObjectsReq{
		SubjectType: auth.GroupType,
		Subject:     groupID,
		Permission:  auth.GroupRelation,
		ObjectType:  auth.ThingType,
	})
	if err != nil {
		return mgclients.MembersPage{}, errors.Wrap(repoerr.ErrNotFound, err)
	}

	pm.IDs = tids.Policies

	cp, err := svc.clients.RetrieveAllByIDs(ctx, pm)
	if err != nil {
		return mgclients.MembersPage{}, errors.Wrap(repoerr.ErrNotFound, err)
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
		return "", errors.Wrap(repoerr.ErrNotFound, err)
	}
	if err := svc.clientCache.Save(ctx, key, client.ID); err != nil {
		return "", errors.Wrap(repoerr.ErrUpdateEntity, err)
	}

	return client.ID, nil
}

func (svc service) identify(ctx context.Context, token string) (*magistrala.IdentityRes, error) {
	res, err := svc.auth.Identify(ctx, &magistrala.IdentityReq{Token: token})
	if err != nil {
		return nil, errors.Wrap(errors.ErrAuthentication, err)
	}
	if res.GetId() == "" || res.GetDomainId() == "" {
		return nil, errors.ErrDomainAuthorization
	}
	return res, nil
}

func (svc *service) authorize(ctx context.Context, subjType, subjKind, subj, perm, objType, obj string) (string, error) {
	req := &magistrala.AuthorizeReq{
		SubjectType: subjType,
		SubjectKind: subjKind,
		Subject:     subj,
		Permission:  perm,
		ObjectType:  objType,
		Object:      obj,
	}
	res, err := svc.auth.Authorize(ctx, req)
	if err != nil {
		return "", errors.Wrap(errors.ErrAuthorization, err)
	}
	if !res.GetAuthorized() {
		return "", errors.Wrap(errors.ErrAuthorization, err)
	}

	return res.GetId(), nil
}
