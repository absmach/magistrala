// Copyright (c) Magistrala
// SPDX-License-Identifier: Apache-2.0

package groups

import (
	"context"
	"fmt"
	"time"

	"github.com/absmach/magistrala"
	"github.com/absmach/magistrala/auth"
	"github.com/absmach/magistrala/internal/apiutil"
	mgclients "github.com/absmach/magistrala/pkg/clients"
	"github.com/absmach/magistrala/pkg/errors"
	"github.com/absmach/magistrala/pkg/groups"
)

var (
	errParentUnAuthz  = errors.New("failed to authorize parent group")
	errMemberKind     = errors.New("invalid member kind")
	errAddPolicies    = errors.New("failed to add policies")
	errDeletePolicies = errors.New("failed to delete policies")
	errRetrieveGroups = errors.New("failed to retrieve groups")
	errGroupIDs       = errors.New("invalid group ids")
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

func (svc service) CreateGroup(ctx context.Context, token, kind string, g groups.Group) (gr groups.Group, err error) {
	res, err := svc.identify(ctx, token)
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

	g.ID = groupID
	g.CreatedAt = time.Now()

	if g.Parent != "" {
		_, err := svc.authorize(ctx, auth.UserType, token, auth.EditPermission, auth.GroupType, g.Parent)
		if err != nil {
			return groups.Group{}, errors.Wrap(errParentUnAuthz, err)
		}
	}

	g, err = svc.groups.Save(ctx, g)
	if err != nil {
		return groups.Group{}, err
	}
	// IMPROVEMENT NOTE: Add defer function , if return err is not nil, then delete group

	policies := magistrala.AddPoliciesReq{}
	policies.AddPoliciesReq = append(policies.AddPoliciesReq, &magistrala.AddPolicyReq{
		Domain:      res.GetDomainId(),
		SubjectType: auth.UserType,
		Subject:     res.GetId(),
		Relation:    auth.AdministratorRelation,
		ObjectKind:  kind,
		ObjectType:  auth.GroupType,
		Object:      g.ID,
	})
	policies.AddPoliciesReq = append(policies.AddPoliciesReq, &magistrala.AddPolicyReq{
		Domain:      res.GetDomainId(),
		SubjectType: auth.DomainType,
		Subject:     res.GetDomainId(),
		Relation:    auth.DomainRelation,
		ObjectType:  auth.GroupType,
		Object:      g.ID,
	})
	if g.Parent != "" {
		policies.AddPoliciesReq = append(policies.AddPoliciesReq, &magistrala.AddPolicyReq{
			Domain:      res.GetDomainId(),
			SubjectType: auth.GroupType,
			Subject:     g.Parent,
			Relation:    auth.ParentGroupRelation,
			ObjectKind:  kind,
			ObjectType:  auth.GroupType,
			Object:      g.ID,
		})
	}
	if _, err := svc.auth.AddPolicies(ctx, &policies); err != nil {
		return groups.Group{}, err
	}

	return g, nil
}

func (svc service) ViewGroup(ctx context.Context, token, id string) (groups.Group, error) {
	_, err := svc.authorize(ctx, auth.UserType, token, auth.ViewPermission, auth.GroupType, id)
	if err != nil {
		return groups.Group{}, err
	}

	return svc.groups.RetrieveByID(ctx, id)
}

func (svc service) ListGroups(ctx context.Context, token, memberKind, memberID string, gm groups.Page) (groups.Page, error) {
	var ids []string
	res, err := svc.identify(ctx, token)
	if err != nil {
		return groups.Page{}, err
	}
	switch memberKind {
	case auth.ThingsKind:
		if _, err := svc.authorizeKind(ctx, auth.UserType, auth.UsersKind, res.GetId(), auth.ViewPermission, auth.ThingType, memberID); err != nil {
			return groups.Page{}, err
		}
		cids, err := svc.auth.ListAllSubjects(ctx, &magistrala.ListSubjectsReq{
			SubjectType: auth.GroupType,
			Permission:  auth.GroupRelation,
			ObjectType:  auth.ThingType,
			Object:      memberID,
		})
		if err != nil {
			return groups.Page{}, err
		}
		ids, err = svc.filterAllowedGroupIDsOfUserID(ctx, res.GetId(), gm.Permission, cids.Policies)
		if err != nil {
			return groups.Page{}, err
		}
	case auth.GroupsKind:
		if _, err := svc.authorizeKind(ctx, auth.UserType, auth.UsersKind, res.GetId(), gm.Permission, auth.GroupType, memberID); err != nil {
			return groups.Page{}, err
		}

		gids, err := svc.auth.ListAllObjects(ctx, &magistrala.ListObjectsReq{
			SubjectType: auth.GroupType,
			Subject:     memberID,
			Permission:  auth.ParentGroupRelation,
			ObjectType:  auth.GroupType,
		})
		if err != nil {
			return groups.Page{}, err
		}
		ids, err = svc.filterAllowedGroupIDsOfUserID(ctx, res.GetId(), gm.Permission, gids.Policies)
		if err != nil {
			return groups.Page{}, err
		}
	case auth.ChannelsKind:
		if _, err := svc.authorizeKind(ctx, auth.UserType, auth.UsersKind, res.GetId(), auth.ViewPermission, auth.GroupType, memberID); err != nil {
			return groups.Page{}, err
		}
		gids, err := svc.auth.ListAllSubjects(ctx, &magistrala.ListSubjectsReq{
			SubjectType: auth.GroupType,
			Permission:  auth.ParentGroupRelation,
			ObjectType:  auth.GroupType,
			Object:      memberID,
		})
		if err != nil {
			return groups.Page{}, err
		}

		ids, err = svc.filterAllowedGroupIDsOfUserID(ctx, res.GetId(), gm.Permission, gids.Policies)
		if err != nil {
			return groups.Page{}, err
		}
	case auth.UsersKind:
		if memberID != "" && res.GetUserId() != memberID {
			if _, err := svc.authorizeKind(ctx, auth.UserType, auth.UsersKind, res.GetId(), auth.AdminPermission, auth.DomainType, res.GetDomainId()); err != nil {
				return groups.Page{}, err
			}
			gids, err := svc.auth.ListAllObjects(ctx, &magistrala.ListObjectsReq{
				SubjectType: auth.UserType,
				Subject:     auth.EncodeDomainUserID(res.GetDomainId(), memberID),
				Permission:  gm.Permission,
				ObjectType:  auth.GroupType,
			})
			if err != nil {
				return groups.Page{}, err
			}
			ids, err = svc.filterAllowedGroupIDsOfUserID(ctx, res.GetId(), gm.Permission, gids.Policies)
			if err != nil {
				return groups.Page{}, err
			}
		} else {
			ids, err = svc.listAllGroupsOfUserID(ctx, res.GetId(), gm.Permission)
			if err != nil {
				return groups.Page{}, err
			}
		}
	default:
		return groups.Page{}, errMemberKind
	}

	if len(ids) == 0 {
		return groups.Page{
			PageMeta: gm.PageMeta,
		}, nil
	}
	return svc.groups.RetrieveByIDs(ctx, gm, ids...)
}

// IMPROVEMENT NOTE: remove this function and all its related auxillary function, ListMembers are moved to respective service
func (svc service) ListMembers(ctx context.Context, token, groupID, permission, memberKind string) (groups.MembersPage, error) {
	_, err := svc.authorize(ctx, auth.UserType, token, auth.ViewPermission, auth.GroupType, groupID)
	if err != nil {
		return groups.MembersPage{}, err
	}
	switch memberKind {
	case auth.ThingsKind:
		tids, err := svc.auth.ListAllObjects(ctx, &magistrala.ListObjectsReq{
			SubjectType: auth.GroupType,
			Subject:     groupID,
			Relation:    auth.GroupRelation,
			ObjectType:  auth.ThingType,
		})
		if err != nil {
			return groups.MembersPage{}, err
		}

		members := []groups.Member{}

		for _, id := range tids.Policies {
			members = append(members, groups.Member{
				ID:   id,
				Type: auth.ThingType,
			})
		}
		return groups.MembersPage{
			Total:   uint64(len(members)),
			Offset:  0,
			Limit:   uint64(len(members)),
			Members: members,
		}, nil
	case auth.UsersKind:
		uids, err := svc.auth.ListAllSubjects(ctx, &magistrala.ListSubjectsReq{
			SubjectType: auth.UserType,
			Permission:  permission,
			Object:      groupID,
			ObjectType:  auth.GroupType,
		})
		if err != nil {
			return groups.MembersPage{}, err
		}

		members := []groups.Member{}

		for _, id := range uids.Policies {
			members = append(members, groups.Member{
				ID:   id,
				Type: auth.UserType,
			})
		}
		return groups.MembersPage{
			Total:   uint64(len(members)),
			Offset:  0,
			Limit:   uint64(len(members)),
			Members: members,
		}, nil
	default:
		return groups.MembersPage{}, errMemberKind
	}
}

func (svc service) UpdateGroup(ctx context.Context, token string, g groups.Group) (groups.Group, error) {
	id, err := svc.authorize(ctx, auth.UserType, token, auth.EditPermission, auth.GroupType, g.ID)
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
	res, err := svc.identify(ctx, token)
	if err != nil {
		return err
	}
	if _, err := svc.authorizeKind(ctx, auth.UserType, auth.UsersKind, res.GetId(), auth.EditPermission, auth.GroupType, groupID); err != nil {
		return err
	}

	policies := magistrala.AddPoliciesReq{}
	switch memberKind {
	case auth.ThingsKind:
		for _, memberID := range memberIDs {
			policies.AddPoliciesReq = append(policies.AddPoliciesReq, &magistrala.AddPolicyReq{
				Domain:      res.GetDomainId(),
				SubjectType: auth.GroupType,
				SubjectKind: auth.ChannelsKind,
				Subject:     groupID,
				Relation:    relation,
				ObjectType:  auth.ThingType,
				Object:      memberID,
			})
		}
	case auth.ChannelsKind:
		for _, memberID := range memberIDs {
			policies.AddPoliciesReq = append(policies.AddPoliciesReq, &magistrala.AddPolicyReq{
				Domain:      res.GetDomainId(),
				SubjectType: auth.GroupType,
				Subject:     memberID,
				Relation:    relation,
				ObjectType:  auth.GroupType,
				Object:      groupID,
			})
		}
	case auth.GroupsKind:
		return svc.assignParentGroup(ctx, res.GetDomainId(), groupID, memberIDs)

	case auth.UsersKind:
		for _, memberID := range memberIDs {
			policies.AddPoliciesReq = append(policies.AddPoliciesReq, &magistrala.AddPolicyReq{
				Domain:      res.GetDomainId(),
				SubjectType: auth.UserType,
				Subject:     auth.EncodeDomainUserID(res.GetDomainId(), memberID),
				Relation:    relation,
				ObjectType:  auth.GroupType,
				Object:      groupID,
			})
		}
	default:
		return errMemberKind
	}

	if _, err := svc.auth.AddPolicies(ctx, &policies); err != nil {
		return errors.Wrap(errAddPolicies, err)
	}

	return nil
}

func (svc service) assignParentGroup(ctx context.Context, domain, parentGroupID string, groupIDs []string) (err error) {
	groups, err := svc.groups.RetrieveByIDs(ctx, groups.Page{PageMeta: groups.PageMeta{Limit: 1<<63 - 1}}, groupIDs...)
	if err != nil {
		return errors.Wrap(errRetrieveGroups, err)
	}
	if len(groups.Groups) == 0 {
		return errGroupIDs
	}
	var addPolicies magistrala.AddPoliciesReq
	var deletePolicies magistrala.DeletePoliciesReq
	for _, group := range groups.Groups {
		if group.Parent != "" {
			return fmt.Errorf("%s group already have parent", group.ID)
		}
		addPolicies.AddPoliciesReq = append(addPolicies.AddPoliciesReq, &magistrala.AddPolicyReq{
			Domain:      domain,
			SubjectType: auth.GroupType,
			Subject:     parentGroupID,
			Relation:    auth.ParentGroupRelation,
			ObjectType:  auth.GroupType,
			Object:      group.ID,
		})
		deletePolicies.DeletePoliciesReq = append(deletePolicies.DeletePoliciesReq, &magistrala.DeletePolicyReq{
			Domain:      domain,
			SubjectType: auth.GroupType,
			Subject:     parentGroupID,
			Relation:    auth.ParentGroupRelation,
			ObjectType:  auth.GroupType,
			Object:      group.ID,
		})
	}

	if _, err := svc.auth.AddPolicies(ctx, &addPolicies); err != nil {
		return err
	}
	defer func() {
		if err != nil {
			if _, errRollback := svc.auth.DeletePolicies(ctx, &deletePolicies); errRollback != nil {
				err = errors.Wrap(err, errors.Wrap(apiutil.ErrRollbackTx, errRollback))
			}
		}
	}()

	return svc.groups.AssignParentGroup(ctx, parentGroupID, groupIDs...)
}

func (svc service) unassignParentGroup(ctx context.Context, domain, parentGroupID string, groupIDs []string) error {
	groups, err := svc.groups.RetrieveByIDs(ctx, groups.Page{PageMeta: groups.PageMeta{Limit: 1<<63 - 1}}, groupIDs...)
	if err != nil {
		return errors.Wrap(errRetrieveGroups, err)
	}
	if len(groups.Groups) == 0 {
		return errGroupIDs
	}
	var addPolicies magistrala.AddPoliciesReq
	var deletePolicies magistrala.DeletePoliciesReq
	for _, group := range groups.Groups {
		if group.Parent != "" && group.Parent != parentGroupID {
			return fmt.Errorf("%s group doesn't have same parent", group.ID)
		}
		addPolicies.AddPoliciesReq = append(addPolicies.AddPoliciesReq, &magistrala.AddPolicyReq{
			Domain:      domain,
			SubjectType: auth.GroupType,
			Subject:     parentGroupID,
			Relation:    auth.ParentGroupRelation,
			ObjectType:  auth.GroupType,
			Object:      group.ID,
		})
		deletePolicies.DeletePoliciesReq = append(deletePolicies.DeletePoliciesReq, &magistrala.DeletePolicyReq{
			Domain:      domain,
			SubjectType: auth.GroupType,
			Subject:     parentGroupID,
			Relation:    auth.ParentGroupRelation,
			ObjectType:  auth.GroupType,
			Object:      group.ID,
		})
	}

	if _, err := svc.auth.DeletePolicies(ctx, &deletePolicies); err != nil {
		return err
	}
	defer func() {
		if err != nil {
			if _, errRollback := svc.auth.AddPolicies(ctx, &addPolicies); errRollback != nil {
				err = errors.Wrap(err, errors.Wrap(apiutil.ErrRollbackTx, errRollback))
			}
		}
	}()

	return svc.groups.UnassignParentGroup(ctx, parentGroupID, groupIDs...)
}

func (svc service) Unassign(ctx context.Context, token, groupID, relation, memberKind string, memberIDs ...string) error {
	res, err := svc.identify(ctx, token)
	if err != nil {
		return err
	}
	if _, err := svc.authorizeKind(ctx, auth.UserType, auth.UsersKind, res.GetId(), auth.EditPermission, auth.GroupType, groupID); err != nil {
		return err
	}

	policies := magistrala.DeletePoliciesReq{}

	switch memberKind {
	case auth.ThingsKind:
		for _, memberID := range memberIDs {
			policies.DeletePoliciesReq = append(policies.DeletePoliciesReq, &magistrala.DeletePolicyReq{
				Domain:      res.GetDomainId(),
				SubjectType: auth.GroupType,
				SubjectKind: auth.ChannelsKind,
				Subject:     groupID,
				Relation:    relation,
				ObjectType:  auth.ThingType,
				Object:      memberID,
			})
		}
	case auth.ChannelsKind:
		for _, memberID := range memberIDs {
			policies.DeletePoliciesReq = append(policies.DeletePoliciesReq, &magistrala.DeletePolicyReq{
				Domain:      res.GetDomainId(),
				SubjectType: auth.GroupType,
				Subject:     memberID,
				Relation:    relation,
				ObjectType:  auth.GroupType,
				Object:      groupID,
			})
		}
	case auth.GroupsKind:
		return svc.unassignParentGroup(ctx, res.GetDomainId(), groupID, memberIDs)
	case auth.UsersKind:
		for _, memberID := range memberIDs {
			policies.DeletePoliciesReq = append(policies.DeletePoliciesReq, &magistrala.DeletePolicyReq{
				Domain:      res.GetDomainId(),
				SubjectType: auth.UserType,
				Subject:     auth.EncodeDomainUserID(res.GetDomainId(), memberID),
				Relation:    relation,
				ObjectType:  auth.GroupType,
				Object:      groupID,
			})
		}
	default:
		return errMemberKind
	}

	if _, err := svc.auth.DeletePolicies(ctx, &policies); err != nil {
		return errors.Wrap(errDeletePolicies, err)
	}
	return nil
}

func (svc service) filterAllowedGroupIDsOfUserID(ctx context.Context, userID, permission string, groupIDs []string) ([]string, error) {
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

func (svc service) listAllGroupsOfUserID(ctx context.Context, userID, permission string) ([]string, error) {
	allowedIDs, err := svc.auth.ListAllObjects(ctx, &magistrala.ListObjectsReq{
		SubjectType: auth.UserType,
		Subject:     userID,
		Permission:  permission,
		ObjectType:  auth.GroupType,
	})
	if err != nil {
		return []string{}, err
	}
	return allowedIDs.Policies, nil
}

func (svc service) changeGroupStatus(ctx context.Context, token string, group groups.Group) (groups.Group, error) {
	id, err := svc.authorize(ctx, auth.UserType, token, auth.EditPermission, auth.GroupType, group.ID)
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

func (svc service) identify(ctx context.Context, token string) (*magistrala.IdentityRes, error) {
	res, err := svc.auth.Identify(ctx, &magistrala.IdentityReq{Token: token})
	if err != nil {
		return nil, err
	}
	if res.GetId() == "" || res.GetDomainId() == "" {
		return nil, errors.ErrDomainAuthorization
	}
	return res, nil
}

func (svc service) authorize(ctx context.Context, subjectType, subject, permission, objectType, object string) (string, error) {
	req := &magistrala.AuthorizeReq{
		SubjectType: subjectType,
		SubjectKind: auth.TokenKind,
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
