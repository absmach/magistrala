// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package groups

import (
	"context"
	"fmt"
	"time"

	"github.com/absmach/magistrala"
	mgauth "github.com/absmach/magistrala/auth"
	"github.com/absmach/magistrala/pkg/apiutil"
	"github.com/absmach/magistrala/pkg/auth"
	mgclients "github.com/absmach/magistrala/pkg/clients"
	"github.com/absmach/magistrala/pkg/errors"
	svcerr "github.com/absmach/magistrala/pkg/errors/service"
	"github.com/absmach/magistrala/pkg/groups"
	"github.com/absmach/magistrala/pkg/policy"
	"golang.org/x/sync/errgroup"
)

var (
	errParentUnAuthz = errors.New("failed to authorize parent group")
	errMemberKind    = errors.New("invalid member kind")
	errGroupIDs      = errors.New("invalid group ids")
)

type service struct {
	groups     groups.Repository
	policy     policy.PolicyClient
	idProvider magistrala.IDProvider
}

// NewService returns a new Clients service implementation.
func NewService(g groups.Repository, idp magistrala.IDProvider, policyClient policy.PolicyClient) groups.Service {
	return service{
		groups:     g,
		idProvider: idp,
		policy:     policyClient,
	}
}

func (svc service) CreateGroup(ctx context.Context, session auth.Session, kind string, g groups.Group) (gr groups.Group, err error) {
	groupID, err := svc.idProvider.ID()
	if err != nil {
		return groups.Group{}, err
	}
	if g.Status != mgclients.EnabledStatus && g.Status != mgclients.DisabledStatus {
		return groups.Group{}, svcerr.ErrInvalidStatus
	}

	g.ID = groupID
	g.CreatedAt = time.Now()
	g.Domain = session.DomainID

	policies, err := svc.addGroupPolicy(ctx, session.DomainUserID, session.DomainID, g.ID, g.Parent, kind)
	if err != nil {
		return groups.Group{}, err
	}

	defer func() {
		if err != nil {
			if errRollback := svc.policy.DeletePolicies(ctx, policies); errRollback != nil {
				err = errors.Wrap(errors.Wrap(errors.ErrRollbackTx, errRollback), err)
			}
		}
	}()

	saved, err := svc.groups.Save(ctx, g)
	if err != nil {
		return groups.Group{}, errors.Wrap(svcerr.ErrCreateEntity, err)
	}

	return saved, nil
}

func (svc service) ViewGroup(ctx context.Context, id string) (groups.Group, error) {
	group, err := svc.groups.RetrieveByID(ctx, id)
	if err != nil {
		return groups.Group{}, errors.Wrap(svcerr.ErrViewEntity, err)
	}

	return group, nil
}

func (svc service) ViewGroupPerms(ctx context.Context, session auth.Session, id string) ([]string, error) {
	return svc.listUserGroupPermission(ctx, session.DomainUserID, id)
}

func (svc service) ListGroups(ctx context.Context, session auth.Session, memberKind, memberID string, gm groups.Page) (groups.Page, error) {
	var ids []string
	var err error

	switch memberKind {
	case policy.ThingsKind:
		cids, err := svc.policy.ListAllSubjects(ctx, policy.PolicyReq{
			SubjectType: policy.GroupType,
			Permission:  policy.GroupRelation,
			ObjectType:  policy.ThingType,
			Object:      memberID,
		})
		if err != nil {
			return groups.Page{}, err
		}
		ids, err = svc.filterAllowedGroupIDsOfUserID(ctx, session.DomainUserID, gm.Permission, cids.Policies)
		if err != nil {
			return groups.Page{}, err
		}
	case policy.GroupsKind:
		gids, err := svc.policy.ListAllObjects(ctx, policy.PolicyReq{
			SubjectType: policy.GroupType,
			Subject:     memberID,
			Permission:  policy.ParentGroupRelation,
			ObjectType:  policy.GroupType,
		})
		if err != nil {
			return groups.Page{}, err
		}
		ids, err = svc.filterAllowedGroupIDsOfUserID(ctx, session.DomainUserID, gm.Permission, gids.Policies)
		if err != nil {
			return groups.Page{}, err
		}
	case policy.ChannelsKind:
		gids, err := svc.policy.ListAllSubjects(ctx, policy.PolicyReq{
			SubjectType: policy.GroupType,
			Permission:  policy.ParentGroupRelation,
			ObjectType:  policy.GroupType,
			Object:      memberID,
		})
		if err != nil {
			return groups.Page{}, err
		}

		ids, err = svc.filterAllowedGroupIDsOfUserID(ctx, session.DomainUserID, gm.Permission, gids.Policies)
		if err != nil {
			return groups.Page{}, err
		}
	case policy.UsersKind:
		switch {
		case memberID != "" && session.UserID != memberID:
			gids, err := svc.policy.ListAllObjects(ctx, policy.PolicyReq{
				SubjectType: policy.UserType,
				Subject:     mgauth.EncodeDomainUserID(session.DomainID, memberID),
				Permission:  gm.Permission,
				ObjectType:  policy.GroupType,
			})
			if err != nil {
				return groups.Page{}, err
			}
			ids, err = svc.filterAllowedGroupIDsOfUserID(ctx, session.DomainUserID, gm.Permission, gids.Policies)
			if err != nil {
				return groups.Page{}, err
			}
		default:
			switch session.SuperAdmin {
			case true:
				gm.PageMeta.DomainID = session.DomainID
			default:
				ids, err = svc.listAllGroupsOfUserID(ctx, session.DomainUserID, gm.Permission)
				if err != nil {
					return groups.Page{}, err
				}
			}
		}
	default:
		return groups.Page{}, errMemberKind
	}
	gp, err := svc.groups.RetrieveByIDs(ctx, gm, ids...)
	if err != nil {
		return groups.Page{}, errors.Wrap(svcerr.ErrViewEntity, err)
	}

	if gm.ListPerms && len(gp.Groups) > 0 {
		g, ctx := errgroup.WithContext(ctx)

		for i := range gp.Groups {
			// Copying loop variable "i" to avoid "loop variable captured by func literal"
			iter := i
			g.Go(func() error {
				return svc.retrievePermissions(ctx, session.DomainUserID, &gp.Groups[iter])
			})
		}

		if err := g.Wait(); err != nil {
			return groups.Page{}, err
		}
	}
	return gp, nil
}

// Experimental functions used for async calling of svc.listUserThingPermission. This might be helpful during listing of large number of entities.
func (svc service) retrievePermissions(ctx context.Context, userID string, group *groups.Group) error {
	permissions, err := svc.listUserGroupPermission(ctx, userID, group.ID)
	if err != nil {
		return err
	}
	group.Permissions = permissions
	return nil
}

func (svc service) listUserGroupPermission(ctx context.Context, userID, groupID string) ([]string, error) {
	permissions, err := svc.policy.ListPermissions(ctx, policy.PolicyReq{
		SubjectType: policy.UserType,
		Subject:     userID,
		Object:      groupID,
		ObjectType:  policy.GroupType,
	}, []string{})
	if err != nil {
		return []string{}, err
	}
	if len(permissions) == 0 {
		return []string{}, svcerr.ErrAuthorization
	}
	return permissions, nil
}

// IMPROVEMENT NOTE: remove this function and all its related auxiliary function, ListMembers are moved to respective service.
func (svc service) ListMembers(ctx context.Context, groupID, permission, memberKind string) (groups.MembersPage, error) {
	switch memberKind {
	case policy.ThingsKind:
		tids, err := svc.policy.ListAllObjects(ctx, policy.PolicyReq{
			SubjectType: policy.GroupType,
			Subject:     groupID,
			Relation:    policy.GroupRelation,
			ObjectType:  policy.ThingType,
		})
		if err != nil {
			return groups.MembersPage{}, err
		}

		members := []groups.Member{}

		for _, id := range tids.Policies {
			members = append(members, groups.Member{
				ID:   id,
				Type: policy.ThingType,
			})
		}
		return groups.MembersPage{
			Total:   uint64(len(members)),
			Offset:  0,
			Limit:   uint64(len(members)),
			Members: members,
		}, nil
	case policy.UsersKind:
		uids, err := svc.policy.ListAllSubjects(ctx, policy.PolicyReq{
			SubjectType: policy.UserType,
			Permission:  permission,
			Object:      groupID,
			ObjectType:  policy.GroupType,
		})
		if err != nil {
			return groups.MembersPage{}, err
		}

		members := []groups.Member{}

		for _, id := range uids.Policies {
			members = append(members, groups.Member{
				ID:   id,
				Type: policy.UserType,
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

func (svc service) UpdateGroup(ctx context.Context, session auth.Session, g groups.Group) (groups.Group, error) {
	g.UpdatedAt = time.Now()
	g.UpdatedBy = session.UserID

	return svc.groups.Update(ctx, g)
}

func (svc service) EnableGroup(ctx context.Context, session auth.Session, id string) (groups.Group, error) {
	group := groups.Group{
		ID:        id,
		Status:    mgclients.EnabledStatus,
		UpdatedAt: time.Now(),
	}
	group, err := svc.changeGroupStatus(ctx, session, group)
	if err != nil {
		return groups.Group{}, err
	}
	return group, nil
}

func (svc service) DisableGroup(ctx context.Context, session auth.Session, id string) (groups.Group, error) {
	group := groups.Group{
		ID:        id,
		Status:    mgclients.DisabledStatus,
		UpdatedAt: time.Now(),
	}
	group, err := svc.changeGroupStatus(ctx, session, group)
	if err != nil {
		return groups.Group{}, err
	}
	return group, nil
}

func (svc service) Assign(ctx context.Context, session auth.Session, groupID, relation, memberKind string, memberIDs ...string) error {
	policies := []policy.PolicyReq{}
	switch memberKind {
	case policy.ThingsKind:
		for _, memberID := range memberIDs {
			policies = append(policies, policy.PolicyReq{
				Domain:      session.DomainID,
				SubjectType: policy.GroupType,
				SubjectKind: policy.ChannelsKind,
				Subject:     groupID,
				Relation:    relation,
				ObjectType:  policy.ThingType,
				Object:      memberID,
			})
		}
	case policy.ChannelsKind:
		for _, memberID := range memberIDs {
			policies = append(policies, policy.PolicyReq{
				Domain:      session.DomainID,
				SubjectType: policy.GroupType,
				Subject:     memberID,
				Relation:    relation,
				ObjectType:  policy.GroupType,
				Object:      groupID,
			})
		}
	case policy.GroupsKind:
		return svc.assignParentGroup(ctx, session.DomainID, groupID, memberIDs)

	case policy.UsersKind:
		for _, memberID := range memberIDs {
			policies = append(policies, policy.PolicyReq{
				Domain:      session.DomainID,
				SubjectType: policy.UserType,
				Subject:     mgauth.EncodeDomainUserID(session.DomainID, memberID),
				Relation:    relation,
				ObjectType:  policy.GroupType,
				Object:      groupID,
			})
		}
	default:
		return errMemberKind
	}

	if err := svc.policy.AddPolicies(ctx, policies); err != nil {
		return errors.Wrap(svcerr.ErrAddPolicies, err)
	}

	return nil
}

func (svc service) assignParentGroup(ctx context.Context, domain, parentGroupID string, groupIDs []string) (err error) {
	groupsPage, err := svc.groups.RetrieveByIDs(ctx, groups.Page{PageMeta: groups.PageMeta{Limit: 1<<63 - 1}}, groupIDs...)
	if err != nil {
		return errors.Wrap(svcerr.ErrViewEntity, err)
	}
	if len(groupsPage.Groups) == 0 {
		return errGroupIDs
	}

	policies := []policy.PolicyReq{}
	for _, group := range groupsPage.Groups {
		if group.Parent != "" {
			return errors.Wrap(svcerr.ErrConflict, fmt.Errorf("%s group already have parent", group.ID))
		}
		policies = append(policies, policy.PolicyReq{
			Domain:      domain,
			SubjectType: policy.GroupType,
			Subject:     parentGroupID,
			Relation:    policy.ParentGroupRelation,
			ObjectType:  policy.GroupType,
			Object:      group.ID,
		})
	}

	if err := svc.policy.AddPolicies(ctx, policies); err != nil {
		return errors.Wrap(svcerr.ErrAddPolicies, err)
	}
	defer func() {
		if err != nil {
			if errRollback := svc.policy.DeletePolicies(ctx, policies); errRollback != nil {
				err = errors.Wrap(err, errors.Wrap(apiutil.ErrRollbackTx, errRollback))
			}
		}
	}()

	return svc.groups.AssignParentGroup(ctx, parentGroupID, groupIDs...)
}

func (svc service) unassignParentGroup(ctx context.Context, domain, parentGroupID string, groupIDs []string) (err error) {
	groupsPage, err := svc.groups.RetrieveByIDs(ctx, groups.Page{PageMeta: groups.PageMeta{Limit: 1<<63 - 1}}, groupIDs...)
	if err != nil {
		return errors.Wrap(svcerr.ErrViewEntity, err)
	}
	if len(groupsPage.Groups) == 0 {
		return errGroupIDs
	}

	policies := []policy.PolicyReq{}
	for _, group := range groupsPage.Groups {
		if group.Parent != "" && group.Parent != parentGroupID {
			return errors.Wrap(svcerr.ErrConflict, fmt.Errorf("%s group doesn't have same parent", group.ID))
		}
		policies = append(policies, policy.PolicyReq{
			Domain:      domain,
			SubjectType: policy.GroupType,
			Subject:     parentGroupID,
			Relation:    policy.ParentGroupRelation,
			ObjectType:  policy.GroupType,
			Object:      group.ID,
		})
	}

	if err := svc.policy.DeletePolicies(ctx, policies); err != nil {
		return errors.Wrap(svcerr.ErrDeletePolicies, err)
	}
	defer func() {
		if err != nil {
			if errRollback := svc.policy.AddPolicies(ctx, policies); errRollback != nil {
				err = errors.Wrap(err, errors.Wrap(apiutil.ErrRollbackTx, errRollback))
			}
		}
	}()

	return svc.groups.UnassignParentGroup(ctx, parentGroupID, groupIDs...)
}

func (svc service) Unassign(ctx context.Context, session auth.Session, groupID, relation, memberKind string, memberIDs ...string) error {
	policies := []policy.PolicyReq{}
	switch memberKind {
	case policy.ThingsKind:
		for _, memberID := range memberIDs {
			policies = append(policies, policy.PolicyReq{
				Domain:      session.DomainID,
				SubjectType: policy.GroupType,
				SubjectKind: policy.ChannelsKind,
				Subject:     groupID,
				Relation:    relation,
				ObjectType:  policy.ThingType,
				Object:      memberID,
			})
		}
	case policy.ChannelsKind:
		for _, memberID := range memberIDs {
			policies = append(policies, policy.PolicyReq{
				Domain:      session.DomainID,
				SubjectType: policy.GroupType,
				Subject:     memberID,
				Relation:    relation,
				ObjectType:  policy.GroupType,
				Object:      groupID,
			})
		}
	case policy.GroupsKind:
		return svc.unassignParentGroup(ctx, session.DomainID, groupID, memberIDs)
	case policy.UsersKind:
		for _, memberID := range memberIDs {
			policies = append(policies, policy.PolicyReq{
				Domain:      session.DomainID,
				SubjectType: policy.UserType,
				Subject:     mgauth.EncodeDomainUserID(session.DomainID, memberID),
				Relation:    relation,
				ObjectType:  policy.GroupType,
				Object:      groupID,
			})
		}
	default:
		return errMemberKind
	}

	if err := svc.policy.DeletePolicies(ctx, policies); err != nil {
		return errors.Wrap(svcerr.ErrDeletePolicies, err)
	}
	return nil
}

func (svc service) DeleteGroup(ctx context.Context, id string) error {
	req := policy.PolicyReq{
		SubjectType: policy.GroupType,
		Subject:     id,
	}
	if err := svc.policy.DeletePolicyFilter(ctx, req); err != nil {
		return errors.Wrap(svcerr.ErrDeletePolicies, err)
	}

	req = policy.PolicyReq{
		Object:     id,
		ObjectType: policy.GroupType,
	}

	if err := svc.policy.DeletePolicyFilter(ctx, req); err != nil {
		return errors.Wrap(svcerr.ErrDeletePolicies, err)
	}

	if err := svc.groups.Delete(ctx, id); err != nil {
		return err
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
	allowedIDs, err := svc.policy.ListAllObjects(ctx, policy.PolicyReq{
		SubjectType: policy.UserType,
		Subject:     userID,
		Permission:  permission,
		ObjectType:  policy.GroupType,
	})
	if err != nil {
		return []string{}, err
	}
	return allowedIDs.Policies, nil
}

func (svc service) changeGroupStatus(ctx context.Context, session auth.Session, group groups.Group) (groups.Group, error) {
	dbGroup, err := svc.groups.RetrieveByID(ctx, group.ID)
	if err != nil {
		return groups.Group{}, errors.Wrap(svcerr.ErrViewEntity, err)
	}
	if dbGroup.Status == group.Status {
		return groups.Group{}, errors.ErrStatusAlreadyAssigned
	}

	group.UpdatedBy = session.UserID
	return svc.groups.ChangeStatus(ctx, group)
}

func (svc service) addGroupPolicy(ctx context.Context, userID, domainID, id, parentID, kind string) ([]policy.PolicyReq, error) {
	policies := []policy.PolicyReq{}
	policies = append(policies, policy.PolicyReq{
		Domain:      domainID,
		SubjectType: policy.UserType,
		Subject:     userID,
		Relation:    policy.AdministratorRelation,
		ObjectKind:  kind,
		ObjectType:  policy.GroupType,
		Object:      id,
	})
	policies = append(policies, policy.PolicyReq{
		Domain:      domainID,
		SubjectType: policy.DomainType,
		Subject:     domainID,
		Relation:    policy.DomainRelation,
		ObjectType:  policy.GroupType,
		Object:      id,
	})
	if parentID != "" {
		policies = append(policies, policy.PolicyReq{
			Domain:      domainID,
			SubjectType: policy.GroupType,
			Subject:     parentID,
			Relation:    policy.ParentGroupRelation,
			ObjectKind:  kind,
			ObjectType:  policy.GroupType,
			Object:      id,
		})
	}
	if err := svc.policy.AddPolicies(ctx, policies); err != nil {
		return policies, errors.Wrap(svcerr.ErrAddPolicies, err)
	}

	return []policy.PolicyReq{}, nil
}
