// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package groups

import (
	"context"
	"time"

	"github.com/mainflux/mainflux"
	"github.com/mainflux/mainflux/internal/apiutil"
	mfclients "github.com/mainflux/mainflux/pkg/clients"
	"github.com/mainflux/mainflux/pkg/errors"
	"github.com/mainflux/mainflux/pkg/groups"
	tpolicies "github.com/mainflux/mainflux/things/policies"
	upolicies "github.com/mainflux/mainflux/users/policies"
)

const (
	thingsObjectKey = "things"

	updateRelationKey = "g_update"
	listRelationKey   = "g_list"
	deleteRelationKey = "g_delete"

	entityType = "group"
)

type service struct {
	uauth      upolicies.AuthServiceClient
	policies   tpolicies.Service
	groups     groups.Repository
	idProvider mainflux.IDProvider
}

// NewService returns a new Clients service implementation.
func NewService(uauth upolicies.AuthServiceClient, policies tpolicies.Service, g groups.Repository, idp mainflux.IDProvider) Service {
	return service{
		uauth:      uauth,
		policies:   policies,
		groups:     g,
		idProvider: idp,
	}
}

func (svc service) CreateGroups(ctx context.Context, token string, gs ...groups.Group) ([]groups.Group, error) {
	userID, err := svc.identify(ctx, token)
	if err != nil {
		return []groups.Group{}, err
	}

	var grps []groups.Group
	for _, g := range gs {
		if g.ID == "" {
			groupID, err := svc.idProvider.ID()
			if err != nil {
				return []groups.Group{}, err
			}
			g.ID = groupID
		}
		if g.Owner == "" {
			g.Owner = userID
		}

		if g.Status != mfclients.EnabledStatus && g.Status != mfclients.DisabledStatus {
			return []groups.Group{}, apiutil.ErrInvalidStatus
		}
		g.CreatedAt = time.Now()

		grp, err := svc.groups.Save(ctx, g)
		if err != nil {
			return []groups.Group{}, err
		}
		grps = append(grps, grp)
	}
	return grps, nil
}

func (svc service) ViewGroup(ctx context.Context, token string, id string) (groups.Group, error) {
	userID, err := svc.identify(ctx, token)
	if err != nil {
		return groups.Group{}, err
	}
	if err := svc.authorize(ctx, userID, id, listRelationKey); err != nil {
		return groups.Group{}, errors.Wrap(errors.ErrNotFound, err)
	}
	return svc.groups.RetrieveByID(ctx, id)
}

func (svc service) ListGroups(ctx context.Context, token string, gm groups.GroupsPage) (groups.GroupsPage, error) {
	userID, err := svc.identify(ctx, token)
	if err != nil {
		return groups.GroupsPage{}, err
	}

	// If the user is admin, fetch all channels from the database.
	if err := svc.authorize(ctx, userID, thingsObjectKey, listRelationKey); err == nil {
		return svc.groups.RetrieveAll(ctx, gm)
	}

	gm.Subject = userID
	gm.OwnerID = userID
	gm.Action = "g_list"
	return svc.groups.RetrieveAll(ctx, gm)
}

func (svc service) ListMemberships(ctx context.Context, token, clientID string, gm groups.GroupsPage) (groups.MembershipsPage, error) {
	userID, err := svc.identify(ctx, token)
	if err != nil {
		return groups.MembershipsPage{}, err
	}

	// If the user is admin, fetch all channels from the database.
	if err := svc.authorize(ctx, userID, thingsObjectKey, listRelationKey); err == nil {
		return svc.groups.Memberships(ctx, clientID, gm)
	}

	gm.OwnerID = userID
	gm.Action = listRelationKey
	return svc.groups.Memberships(ctx, clientID, gm)
}

func (svc service) UpdateGroup(ctx context.Context, token string, g groups.Group) (groups.Group, error) {
	userID, err := svc.identify(ctx, token)
	if err != nil {
		return groups.Group{}, err
	}

	if err := svc.authorize(ctx, userID, g.ID, updateRelationKey); err != nil {
		return groups.Group{}, errors.Wrap(errors.ErrNotFound, err)
	}

	g.Owner = userID
	g.UpdatedAt = time.Now()
	g.UpdatedBy = userID

	return svc.groups.Update(ctx, g)
}

func (svc service) EnableGroup(ctx context.Context, token, id string) (groups.Group, error) {
	group := groups.Group{
		ID:        id,
		Status:    mfclients.EnabledStatus,
		UpdatedAt: time.Now(),
	}
	group, err := svc.changeGroupStatus(ctx, token, group)
	if err != nil {
		return groups.Group{}, errors.Wrap(groups.ErrEnableGroup, err)
	}
	return group, nil
}

func (svc service) DisableGroup(ctx context.Context, token, id string) (groups.Group, error) {
	group := groups.Group{
		ID:        id,
		Status:    mfclients.DisabledStatus,
		UpdatedAt: time.Now(),
	}
	group, err := svc.changeGroupStatus(ctx, token, group)
	if err != nil {
		return groups.Group{}, errors.Wrap(groups.ErrDisableGroup, err)
	}
	return group, nil
}

func (svc service) changeGroupStatus(ctx context.Context, token string, group groups.Group) (groups.Group, error) {
	userID, err := svc.identify(ctx, token)
	if err != nil {
		return groups.Group{}, err
	}
	if err := svc.authorize(ctx, userID, group.ID, deleteRelationKey); err != nil {
		return groups.Group{}, errors.Wrap(errors.ErrNotFound, err)
	}
	dbGroup, err := svc.groups.RetrieveByID(ctx, group.ID)
	if err != nil {
		return groups.Group{}, err
	}

	if dbGroup.Status == group.Status {
		return groups.Group{}, mfclients.ErrStatusAlreadyAssigned
	}
	group.UpdatedBy = userID
	return svc.groups.ChangeStatus(ctx, group)
}

func (svc service) identify(ctx context.Context, token string) (string, error) {
	req := &upolicies.IdentifyReq{Token: token}
	res, err := svc.uauth.Identify(ctx, req)
	if err != nil {
		return "", errors.Wrap(errors.ErrAuthorization, err)
	}
	return res.GetId(), nil
}

func (svc service) authorize(ctx context.Context, subject, object, action string) error {
	// If the user is admin, skip authorization.
	if err := svc.checkAdmin(ctx, subject, thingsObjectKey, action); err == nil {
		return nil
	}

	areq := tpolicies.AccessRequest{Subject: subject, Object: object, Action: action, Entity: entityType}

	_, err := svc.policies.Authorize(ctx, areq)
	return err
}

func (svc service) checkAdmin(ctx context.Context, subject, object, action string) error {
	// for checking admin rights policy object, action and entity type are not important
	req := &upolicies.AuthorizeReq{
		Subject:    subject,
		Object:     object,
		Action:     action,
		EntityType: entityType,
	}
	res, err := svc.uauth.Authorize(ctx, req)
	if err != nil {
		return err
	}
	if !res.GetAuthorized() {
		return errors.ErrAuthorization
	}
	return nil
}
