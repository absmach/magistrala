// Copyright (c) Magistrala
// SPDX-License-Identifier: Apache-2.0

package groups

import (
	"context"
	"fmt"
	"time"

	"github.com/absmach/magistrala"
	"github.com/absmach/magistrala/internal/apiutil"
	mgclients "github.com/absmach/magistrala/pkg/clients"
	"github.com/absmach/magistrala/pkg/errors"
	"github.com/absmach/magistrala/pkg/groups"
)

var errParentUnAuthz = errors.New("failed to authorize parent group")

const (
	ownerRelation       = "owner"
	channelRelation     = "channel"
	groupRelation       = "group"
	parentGroupRelation = "parent_group"

	usersKind    = "users"
	groupsKind   = "groups"
	thingsKind   = "things"
	channelsKind = "channels"

	userType    = "user"
	groupType   = "group"
	thingType   = "thing"
	channelType = "channel"

	adminPermission      = "admin"
	ownerPermission      = "delete"
	deletePermission     = "delete"
	sharePermission      = "share"
	editPermission       = "edit"
	disconnectPermission = "disconnect"
	connectPermission    = "connect"
	viewPermission       = "view"
	memberPermission     = "member"

	tokenKind = "token"
)

type service struct {
	groups     groups.Repository
	auth       magistrala.AuthServiceClient
	idProvider magistrala.IDProvider
}

// NewService returns a new Clients service implementation.
func NewService(g groups.Repository, idp magistrala.IDProvider, auth magistrala.AuthServiceClient) groups.Service {
	return service{
		groups:     g,
		idProvider: idp,
		auth:       auth,
	}
}

func (svc service) CreateGroup(ctx context.Context, token string, g groups.Group) (groups.Group, error) {
	ownerID, err := svc.identify(ctx, token)
	if err != nil {
		return groups.Group{}, err
	}
	groupID, err := svc.idProvider.ID()
	if err != nil {
		return groups.Group{}, err
	}
	if g.Status != mgclients.EnabledStatus && g.Status != mgclients.DisabledStatus {
		return groups.Group{}, apiutil.ErrInvalidStatus
	}
	if g.Owner == "" {
		g.Owner = ownerID
	}

	g.ID = groupID
	g.CreatedAt = time.Now()

	if g.Parent != "" {
		_, err := svc.authorize(ctx, userType, token, editPermission, groupType, g.Parent)
		if err != nil {
			return groups.Group{}, errors.Wrap(errParentUnAuthz, err)
		}
	}

	g, err = svc.groups.Save(ctx, g)
	if err != nil {
		return groups.Group{}, err
	}

	policy := magistrala.AddPolicyReq{
		SubjectType: userType,
		Subject:     ownerID,
		Relation:    ownerRelation,
		ObjectType:  groupType,
		Object:      g.ID,
	}
	if _, err := svc.auth.AddPolicy(ctx, &policy); err != nil {
		return groups.Group{}, err
	}

	if g.Parent != "" {
		policy = magistrala.AddPolicyReq{
			SubjectType: groupType,
			Subject:     g.Parent,
			Relation:    parentGroupRelation,
			ObjectType:  groupType,
			Object:      g.ID,
		}
		if _, err := svc.auth.AddPolicy(ctx, &policy); err != nil {
			return groups.Group{}, err
		}
	}

	return g, nil
}

func (svc service) ViewGroup(ctx context.Context, token string, id string) (groups.Group, error) {
	_, err := svc.authorize(ctx, userType, token, viewPermission, groupType, id)
	if err != nil {
		return groups.Group{}, err
	}

	return svc.groups.RetrieveByID(ctx, id)
}

func (svc service) ListGroups(ctx context.Context, token string, memberKind, memberID string, gm groups.Page) (groups.Page, error) {
	var ids []string
	userID, err := svc.identify(ctx, token)
	if err != nil {
		return groups.Page{}, err
	}
	switch memberKind {
	case thingsKind:
		if _, err := svc.authorizeKind(ctx, userType, usersKind, userID, viewPermission, thingType, memberID); err != nil {
			return groups.Page{}, err
		}
		cids, err := svc.auth.ListAllSubjects(ctx, &magistrala.ListSubjectsReq{
			SubjectType: groupType,
			Permission:  groupRelation,
			ObjectType:  thingType,
			Object:      memberID,
		})
		if err != nil {
			return groups.Page{}, err
		}
		ids, err = svc.filterAllowedGroupIDsOfUserID(ctx, userID, gm.Permission, cids.Policies)
		if err != nil {
			return groups.Page{}, err
		}
	case groupsKind:
		if _, err := svc.authorizeKind(ctx, userType, usersKind, userID, gm.Permission, groupType, memberID); err != nil {
			return groups.Page{}, err
		}

		gids, err := svc.auth.ListAllObjects(ctx, &magistrala.ListObjectsReq{
			SubjectType: groupType,
			Subject:     memberID,
			Permission:  parentGroupRelation,
			ObjectType:  groupType,
		})
		if err != nil {
			return groups.Page{}, err
		}
		ids, err = svc.filterAllowedGroupIDsOfUserID(ctx, userID, gm.Permission, gids.Policies)
		if err != nil {
			return groups.Page{}, err
		}
	case channelsKind:
		if _, err := svc.authorizeKind(ctx, userType, usersKind, userID, viewPermission, groupType, memberID); err != nil {
			return groups.Page{}, err
		}
		gids, err := svc.auth.ListAllSubjects(ctx, &magistrala.ListSubjectsReq{
			SubjectType: groupType,
			Permission:  parentGroupRelation,
			ObjectType:  groupType,
			Object:      memberID,
		})
		if err != nil {
			return groups.Page{}, err
		}

		ids, err = svc.filterAllowedGroupIDsOfUserID(ctx, userID, gm.Permission, gids.Policies)
		if err != nil {
			return groups.Page{}, err
		}
	case usersKind:
		if memberID != "" && userID != memberID {
			if _, err := svc.authorizeKind(ctx, userType, usersKind, userID, ownerRelation, userType, memberID); err != nil {
				return groups.Page{}, err
			}
			gids, err := svc.auth.ListAllObjects(ctx, &magistrala.ListObjectsReq{
				SubjectType: userType,
				Subject:     memberID,
				Permission:  gm.Permission,
				ObjectType:  groupType,
			})
			if err != nil {
				return groups.Page{}, err
			}
			ids, err = svc.filterAllowedGroupIDsOfUserID(ctx, userID, gm.Permission, gids.Policies)
			if err != nil {
				return groups.Page{}, err
			}
		} else {
			ids, err = svc.listAllGroupsOfUserID(ctx, userID, gm.Permission)
			if err != nil {
				return groups.Page{}, err
			}
		}
	default:
		return groups.Page{}, fmt.Errorf("invalid member kind")
	}

	if len(ids) == 0 {
		return groups.Page{
			PageMeta: gm.PageMeta,
		}, nil
	}
	return svc.groups.RetrieveByIDs(ctx, gm, ids...)
}

func (svc service) ListMembers(ctx context.Context, token, groupID, permission, memberKind string) (groups.MembersPage, error) {
	_, err := svc.authorize(ctx, userType, token, viewPermission, groupType, groupID)
	if err != nil {
		return groups.MembersPage{}, err
	}
	switch memberKind {
	case thingsKind:
		tids, err := svc.auth.ListAllObjects(ctx, &magistrala.ListObjectsReq{
			SubjectType: groupType,
			Subject:     groupID,
			Relation:    groupRelation,
			ObjectType:  thingType,
		})
		if err != nil {
			return groups.MembersPage{}, err
		}

		members := []groups.Member{}

		for _, id := range tids.Policies {
			members = append(members, groups.Member{
				ID:   id,
				Type: thingType,
			})
		}
		return groups.MembersPage{
			Total:   uint64(len(members)),
			Offset:  0,
			Limit:   uint64(len(members)),
			Members: members,
		}, nil
	case usersKind:
		uids, err := svc.auth.ListAllSubjects(ctx, &magistrala.ListSubjectsReq{
			SubjectType: userType,
			Permission:  permission,
			Object:      groupID,
			ObjectType:  groupType,
		})
		if err != nil {
			return groups.MembersPage{}, err
		}

		members := []groups.Member{}

		for _, id := range uids.Policies {
			members = append(members, groups.Member{
				ID:   id,
				Type: userType,
			})
		}
		return groups.MembersPage{
			Total:   uint64(len(members)),
			Offset:  0,
			Limit:   uint64(len(members)),
			Members: members,
		}, nil
	default:
		return groups.MembersPage{}, fmt.Errorf("invalid member_kind")
	}
}

func (svc service) UpdateGroup(ctx context.Context, token string, g groups.Group) (groups.Group, error) {
	id, err := svc.authorize(ctx, userType, token, editPermission, groupType, g.ID)
	if err != nil {
		return groups.Group{}, err
	}

	g.UpdatedAt = time.Now()
	g.UpdatedBy = id

	return svc.groups.Update(ctx, g)
}

func (svc service) EnableGroup(ctx context.Context, token, id string) (groups.Group, error) {
	group := groups.Group{
		ID:        id,
		Status:    mgclients.EnabledStatus,
		UpdatedAt: time.Now(),
	}
	group, err := svc.changeGroupStatus(ctx, token, group)
	if err != nil {
		return groups.Group{}, err
	}
	return group, nil
}

func (svc service) DisableGroup(ctx context.Context, token, id string) (groups.Group, error) {
	group := groups.Group{
		ID:        id,
		Status:    mgclients.DisabledStatus,
		UpdatedAt: time.Now(),
	}
	group, err := svc.changeGroupStatus(ctx, token, group)
	if err != nil {
		return groups.Group{}, err
	}
	return group, nil
}

func (svc service) Assign(ctx context.Context, token, groupID, relation, memberKind string, memberIDs ...string) error {
	_, err := svc.authorize(ctx, userType, token, editPermission, groupType, groupID)
	if err != nil {
		return err
	}

	prs := []*magistrala.AddPolicyReq{}
	switch memberKind {
	case thingsKind:
		for _, memberID := range memberIDs {
			prs = append(prs, &magistrala.AddPolicyReq{
				SubjectType: groupType,
				Subject:     groupID,
				Relation:    relation,
				ObjectType:  thingType,
				Object:      memberID,
			})
		}
	case groupsKind:
		for _, memberID := range memberIDs {
			prs = append(prs, &magistrala.AddPolicyReq{
				SubjectType: groupType,
				Subject:     memberID,
				Relation:    relation,
				ObjectType:  groupType,
				Object:      groupID,
			})
		}
	case usersKind:
		for _, memberID := range memberIDs {
			prs = append(prs, &magistrala.AddPolicyReq{
				SubjectType: userType,
				Subject:     memberID,
				Relation:    relation,
				ObjectType:  groupType,
				Object:      groupID,
			})
		}
	default:
		return fmt.Errorf("invalid member kind")
	}

	for _, pr := range prs {
		if _, err := svc.auth.AddPolicy(ctx, pr); err != nil {
			return fmt.Errorf("failed to add policies : %w", err)
		}
	}
	return nil
}

func (svc service) Unassign(ctx context.Context, token, groupID, relation, memberKind string, memberIDs ...string) error {
	_, err := svc.authorize(ctx, userType, token, editPermission, groupType, groupID)
	if err != nil {
		return err
	}

	prs := []*magistrala.DeletePolicyReq{}

	switch memberKind {
	case thingsKind:
		for _, memberID := range memberIDs {
			prs = append(prs, &magistrala.DeletePolicyReq{
				SubjectType: groupType,
				Subject:     groupID,
				Relation:    relation,
				ObjectType:  thingType,
				Object:      memberID,
			})
		}
	case groupsKind:
		for _, memberID := range memberIDs {
			prs = append(prs, &magistrala.DeletePolicyReq{
				SubjectType: groupType,
				Subject:     memberID,
				Relation:    relation,
				ObjectType:  groupType,
				Object:      groupID,
			})
		}
	case usersKind:
		for _, memberID := range memberIDs {
			prs = append(prs, &magistrala.DeletePolicyReq{
				SubjectType: userType,
				Subject:     memberID,
				Relation:    relation,
				ObjectType:  groupType,
				Object:      groupID,
			})
		}
	default:
		return fmt.Errorf("invalid member kind")
	}

	for _, pr := range prs {
		if _, err := svc.auth.DeletePolicy(ctx, pr); err != nil {
			return fmt.Errorf("failed to delete policies : %w", err)
		}
	}

	return nil
}

func (svc service) filterAllowedGroupIDsOfUserID(ctx context.Context, userID string, permission string, groupIDs []string) ([]string, error) {
	var ids []string
	allowedIDs, err := svc.listAllGroupsOfUserID(ctx, userID, permission)
	if err != nil {
		return []string{}, err
	}

	for _, gid := range groupIDs {
		for _, id := range allowedIDs {
			if id == gid {
				ids = append(ids, id)
			}
		}
	}
	return ids, nil
}

func (svc service) listAllGroupsOfUserID(ctx context.Context, userID string, permission string) ([]string, error) {
	allowedIDs, err := svc.auth.ListAllObjects(ctx, &magistrala.ListObjectsReq{
		SubjectType: userType,
		Subject:     userID,
		Permission:  permission,
		ObjectType:  groupType,
	})
	if err != nil {
		return []string{}, err
	}
	return allowedIDs.Policies, nil
}

func (svc service) changeGroupStatus(ctx context.Context, token string, group groups.Group) (groups.Group, error) {
	id, err := svc.authorize(ctx, userType, token, editPermission, groupType, group.ID)
	if err != nil {
		return groups.Group{}, err
	}
	dbGroup, err := svc.groups.RetrieveByID(ctx, group.ID)
	if err != nil {
		return groups.Group{}, err
	}
	if dbGroup.Status == group.Status {
		return groups.Group{}, mgclients.ErrStatusAlreadyAssigned
	}

	group.UpdatedBy = id
	return svc.groups.ChangeStatus(ctx, group)
}

func (svc service) identify(ctx context.Context, token string) (string, error) {
	user, err := svc.auth.Identify(ctx, &magistrala.IdentityReq{Token: token})
	if err != nil {
		return "", err
	}
	return user.GetId(), nil
}

func (svc service) authorize(ctx context.Context, subjectType, subject, permission, objectType, object string) (string, error) {
	req := &magistrala.AuthorizeReq{
		SubjectType: subjectType,
		SubjectKind: tokenKind,
		Subject:     subject,
		Permission:  permission,
		Object:      object,
		ObjectType:  objectType,
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

func (svc service) authorizeKind(ctx context.Context, subjectType, subjectKind, subject, permission, objectType, object string) (string, error) {
	req := &magistrala.AuthorizeReq{
		SubjectType: subjectType,
		SubjectKind: subjectKind,
		Subject:     subject,
		Permission:  permission,
		Object:      object,
		ObjectType:  objectType,
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
