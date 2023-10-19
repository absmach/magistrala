// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0
package things

import (
	"context"
	"time"

	"github.com/mainflux/mainflux"
	"github.com/mainflux/mainflux/internal/apiutil"
	mfclients "github.com/mainflux/mainflux/pkg/clients"
	"github.com/mainflux/mainflux/pkg/errors"
	mfgroups "github.com/mainflux/mainflux/pkg/groups"
	"github.com/mainflux/mainflux/things/postgres"
)

const (
	ownerRelation = "owner"
	groupRelation = "group"

	ownerPermission  = "delete"
	deletePermission = "delete"
	editPermission   = "edit"
	viewPermission   = "view"

	userType  = "user"
	tokenKind = "token"
	userKind  = "users"
	thingType = "thing"
	groupType = "group"
)

type service struct {
	auth        mainflux.AuthServiceClient
	clients     postgres.Repository
	clientCache Cache
	idProvider  mainflux.IDProvider
	grepo       mfgroups.Repository
}

// NewService returns a new Clients service implementation.
func NewService(uauth mainflux.AuthServiceClient, c postgres.Repository, grepo mfgroups.Repository, tcache Cache, idp mainflux.IDProvider) Service {
	return service{
		auth:        uauth,
		clients:     c,
		grepo:       grepo,
		clientCache: tcache,
		idProvider:  idp,
	}
}

func (svc service) Authorize(ctx context.Context, req *mainflux.AuthorizeReq) (string, error) {
	thingID, err := svc.Identify(ctx, req.GetSubject())
	if err != nil {
		return "", errors.ErrAuthentication
	}

	r := &mainflux.AuthorizeReq{
		SubjectType: groupType,
		Subject:     req.GetObject(),
		ObjectType:  thingType,
		Object:      thingID,
		Permission:  req.GetPermission(),
	}
	resp, err := svc.auth.Authorize(ctx, r)
	if err != nil {
		return "", err
	}
	if !resp.GetAuthorized() {
		return "", errors.ErrAuthorization
	}

	return thingID, nil
}

func (svc service) CreateThings(ctx context.Context, token string, cls ...mfclients.Client) ([]mfclients.Client, error) {
	user, err := svc.auth.Identify(ctx, &mainflux.IdentityReq{Token: token})
	if err != nil {
		return []mfclients.Client{}, errors.Wrap(errors.ErrAuthorization, err)
	}
	var clients []mfclients.Client
	for _, c := range cls {
		if c.ID == "" {
			clientID, err := svc.idProvider.ID()
			if err != nil {
				return []mfclients.Client{}, err
			}
			c.ID = clientID
		}
		if c.Credentials.Secret == "" {
			key, err := svc.idProvider.ID()
			if err != nil {
				return []mfclients.Client{}, err
			}
			c.Credentials.Secret = key
		}
		if c.Owner == "" {
			c.Owner = user.GetId()
		}
		if c.Status != mfclients.DisabledStatus && c.Status != mfclients.EnabledStatus {
			return []mfclients.Client{}, apiutil.ErrInvalidStatus
		}
		c.CreatedAt = time.Now()
		clients = append(clients, c)
	}

	saved, err := svc.clients.Save(ctx, clients...)
	if err != nil {
		return nil, err
	}

	for _, c := range saved {
		policy := mainflux.AddPolicyReq{
			SubjectType: userType,
			Subject:     user.GetId(),
			Relation:    ownerRelation,
			ObjectType:  thingType,
			Object:      c.ID,
		}
		if _, err := svc.auth.AddPolicy(ctx, &policy); err != nil {
			return nil, err
		}
	}

	return saved, nil
}

func (svc service) ViewClient(ctx context.Context, token string, id string) (mfclients.Client, error) {
	_, err := svc.authorize(ctx, userType, tokenKind, token, viewPermission, thingType, id)
	if err != nil {
		return mfclients.Client{}, err
	}

	return svc.clients.RetrieveByID(ctx, id)
}

func (svc service) ListClients(ctx context.Context, token string, reqUserID string, pm mfclients.Page) (mfclients.ClientsPage, error) {
	var ids []string

	userID, err := svc.identify(ctx, token)
	if err != nil {
		return mfclients.ClientsPage{}, err
	}

	switch {
	case (reqUserID != "" && reqUserID != userID):
		if _, err := svc.authorize(ctx, userType, userKind, userID, ownerRelation, userType, reqUserID); err != nil {
			return mfclients.ClientsPage{}, err
		}
		rtids, err := svc.listClientIDs(ctx, reqUserID, pm.Permission)
		if err != nil {
			return mfclients.ClientsPage{}, err
		}
		ids, err = svc.filterAllowedThingIDs(ctx, userID, pm.Permission, rtids)
		if err != nil {
			return mfclients.ClientsPage{}, err
		}
	default:
		ids, err = svc.listClientIDs(ctx, userID, pm.Permission)
		if err != nil {
			return mfclients.ClientsPage{}, err
		}
	}

	if len(ids) == 0 {
		return mfclients.ClientsPage{
			Page: mfclients.Page{Total: 0, Limit: pm.Limit, Offset: pm.Offset},
		}, nil
	}

	pm.IDs = ids

	return svc.clients.RetrieveAllByIDs(ctx, pm)
}

func (svc service) listClientIDs(ctx context.Context, userID, permission string) ([]string, error) {
	tids, err := svc.auth.ListAllObjects(ctx, &mainflux.ListObjectsReq{
		SubjectType: userType,
		Subject:     userID,
		Permission:  permission,
		ObjectType:  thingType,
	})
	if err != nil {
		return nil, err
	}
	return tids.Policies, nil
}

func (svc service) filterAllowedThingIDs(ctx context.Context, userID, permission string, thingIDs []string) ([]string, error) {
	var ids []string
	tids, err := svc.auth.ListAllObjects(ctx, &mainflux.ListObjectsReq{
		SubjectType: userType,
		Subject:     userID,
		Permission:  permission,
		ObjectType:  thingType,
	})
	if err != nil {
		return nil, err
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

func (svc service) UpdateClient(ctx context.Context, token string, cli mfclients.Client) (mfclients.Client, error) {
	userID, err := svc.authorize(ctx, userType, tokenKind, token, editPermission, thingType, cli.ID)
	if err != nil {
		return mfclients.Client{}, err
	}

	client := mfclients.Client{
		ID:        cli.ID,
		Name:      cli.Name,
		Metadata:  cli.Metadata,
		UpdatedAt: time.Now(),
		UpdatedBy: userID,
	}
	return svc.clients.Update(ctx, client)
}

func (svc service) UpdateClientTags(ctx context.Context, token string, cli mfclients.Client) (mfclients.Client, error) {
	userID, err := svc.authorize(ctx, userType, tokenKind, token, editPermission, thingType, cli.ID)
	if err != nil {
		return mfclients.Client{}, err
	}

	client := mfclients.Client{
		ID:        cli.ID,
		Tags:      cli.Tags,
		UpdatedAt: time.Now(),
		UpdatedBy: userID,
	}
	return svc.clients.UpdateTags(ctx, client)
}

func (svc service) UpdateClientSecret(ctx context.Context, token, id, key string) (mfclients.Client, error) {
	userID, err := svc.authorize(ctx, userType, tokenKind, token, editPermission, thingType, id)
	if err != nil {
		return mfclients.Client{}, err
	}

	client := mfclients.Client{
		ID: id,
		Credentials: mfclients.Credentials{
			Secret: key,
		},
		UpdatedAt: time.Now(),
		UpdatedBy: userID,
		Status:    mfclients.EnabledStatus,
	}
	return svc.clients.UpdateSecret(ctx, client)
}

func (svc service) UpdateClientOwner(ctx context.Context, token string, cli mfclients.Client) (mfclients.Client, error) {
	userID, err := svc.authorize(ctx, userType, tokenKind, token, editPermission, thingType, cli.ID)
	if err != nil {
		return mfclients.Client{}, err
	}

	client := mfclients.Client{
		ID:        cli.ID,
		Owner:     cli.Owner,
		UpdatedAt: time.Now(),
		UpdatedBy: userID,
		Status:    mfclients.EnabledStatus,
	}
	return svc.clients.UpdateOwner(ctx, client)
}

func (svc service) EnableClient(ctx context.Context, token, id string) (mfclients.Client, error) {
	client := mfclients.Client{
		ID:        id,
		Status:    mfclients.EnabledStatus,
		UpdatedAt: time.Now(),
	}
	client, err := svc.changeClientStatus(ctx, token, client)
	if err != nil {
		return mfclients.Client{}, errors.Wrap(mfclients.ErrEnableClient, err)
	}

	return client, nil
}

func (svc service) DisableClient(ctx context.Context, token, id string) (mfclients.Client, error) {
	client := mfclients.Client{
		ID:        id,
		Status:    mfclients.DisabledStatus,
		UpdatedAt: time.Now(),
	}
	client, err := svc.changeClientStatus(ctx, token, client)
	if err != nil {
		return mfclients.Client{}, errors.Wrap(mfclients.ErrDisableClient, err)
	}

	if err := svc.clientCache.Remove(ctx, client.ID); err != nil {
		return client, err
	}

	return client, nil
}

func (svc service) Share(ctx context.Context, token, id, relation string, userids ...string) error {
	_, err := svc.authorize(ctx, userType, tokenKind, token, ownerPermission, thingType, id)
	if err != nil {
		return err
	}

	for _, userid := range userids {
		addPolicyReq := &mainflux.AddPolicyReq{
			SubjectType: userType,
			Subject:     userid,
			Relation:    relation,
			ObjectType:  thingType,
			Object:      id,
		}

		res, err := svc.auth.AddPolicy(ctx, addPolicyReq)
		if err != nil {
			return err
		}
		if !res.Authorized {
			return errors.ErrAuthorization
		}
	}
	return nil
}

func (svc service) Unshare(ctx context.Context, token, id, relation string, userids ...string) error {
	_, err := svc.authorize(ctx, userType, tokenKind, token, ownerPermission, thingType, id)
	if err != nil {
		return err
	}

	for _, userid := range userids {
		delPolicyReq := &mainflux.DeletePolicyReq{
			SubjectType: userType,
			Subject:     userid,
			Relation:    relation,
			ObjectType:  thingType,
			Object:      id,
		}

		res, err := svc.auth.DeletePolicy(ctx, delPolicyReq)
		if err != nil {
			return err
		}
		if !res.Deleted {
			return errors.ErrAuthorization
		}
	}
	return nil
}

func (svc service) changeClientStatus(ctx context.Context, token string, client mfclients.Client) (mfclients.Client, error) {
	userID, err := svc.authorize(ctx, userType, tokenKind, token, deletePermission, thingType, client.ID)
	if err != nil {
		return mfclients.Client{}, err
	}
	dbClient, err := svc.clients.RetrieveByID(ctx, client.ID)
	if err != nil {
		return mfclients.Client{}, err
	}
	if dbClient.Status == client.Status {
		return mfclients.Client{}, mfclients.ErrStatusAlreadyAssigned
	}

	client.UpdatedBy = userID
	return svc.clients.ChangeStatus(ctx, client)
}

func (svc service) ListClientsByGroup(ctx context.Context, token, groupID string, pm mfclients.Page) (mfclients.MembersPage, error) {
	if _, err := svc.authorize(ctx, userType, tokenKind, token, pm.Permission, groupType, groupID); err != nil {
		return mfclients.MembersPage{}, err
	}

	tids, err := svc.auth.ListAllObjects(ctx, &mainflux.ListObjectsReq{
		SubjectType: groupType,
		Subject:     groupID,
		Permission:  groupRelation,
		ObjectType:  thingType,
	})
	if err != nil {
		return mfclients.MembersPage{}, err
	}

	pm.IDs = tids.Policies

	cp, err := svc.clients.RetrieveAllByIDs(ctx, pm)
	if err != nil {
		return mfclients.MembersPage{}, err
	}

	return mfclients.MembersPage{
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
		return "", err
	}
	if err := svc.clientCache.Save(ctx, key, client.ID); err != nil {
		return "", err
	}

	return client.ID, nil
}

func (svc service) identify(ctx context.Context, token string) (string, error) {
	user, err := svc.auth.Identify(ctx, &mainflux.IdentityReq{Token: token})
	if err != nil {
		return "", err
	}

	return user.GetId(), nil
}

func (svc *service) authorize(ctx context.Context, subjType, subjKind, subj, perm, objType, obj string) (string, error) {
	req := &mainflux.AuthorizeReq{
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
		return "", errors.ErrAuthorization
	}

	return res.GetId(), nil
}
