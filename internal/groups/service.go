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
	"github.com/absmach/magistrala/pkg/policies"
	"golang.org/x/sync/errgroup"
)

var (
	errMemberKind = errors.New("invalid member kind")
	errGroupIDs   = errors.New("invalid group ids")
)

type service struct {
	groups     groups.Repository
	policies   policies.PolicyClient
	idProvider magistrala.IDProvider
}

// NewService returns a new Clients service implementation.
func NewService(g groups.Repository, idp magistrala.IDProvider, policyClient policies.PolicyClient) groups.Service {
	return service{
		groups:     g,
		idProvider: idp,
		policies:   policyClient,
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

	policyList, err := svc.addGroupPolicy(ctx, session.DomainUserID, session.DomainID, g.ID, g.Parent, kind)
	if err != nil {
		return groups.Group{}, err
	}

	defer func() {
		if err != nil {
			if errRollback := svc.policies.DeletePolicies(ctx, policyList); errRollback != nil {
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

func (svc service) ViewGroup(ctx context.Context, session auth.Session, id string) (groups.Group, error) {
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
	case policies.ThingsKind:
		cids, err := svc.policies.ListAllSubjects(ctx, policies.PolicyReq{
			SubjectType: policies.GroupType,
			Permission:  policies.GroupRelation,
			ObjectType:  policies.ThingType,
			Object:      memberID,
		})
		if err != nil {
			return groups.Page{}, err
		}
		ids, err = svc.filterAllowedGroupIDsOfUserID(ctx, session.DomainUserID, gm.Permission, cids.Policies)
		if err != nil {
			return groups.Page{}, err
		}
	case policies.GroupsKind:
		gids, err := svc.policies.ListAllObjects(ctx, policies.PolicyReq{
			SubjectType: policies.GroupType,
			Subject:     memberID,
			Permission:  policies.ParentGroupRelation,
			ObjectType:  policies.GroupType,
		})
		if err != nil {
			return groups.Page{}, err
		}
		ids, err = svc.filterAllowedGroupIDsOfUserID(ctx, session.DomainUserID, gm.Permission, gids.Policies)
		if err != nil {
			return groups.Page{}, err
		}
	case policies.ChannelsKind:
		gids, err := svc.policies.ListAllSubjects(ctx, policies.PolicyReq{
			SubjectType: policies.GroupType,
			Permission:  policies.ParentGroupRelation,
			ObjectType:  policies.GroupType,
			Object:      memberID,
		})
		if err != nil {
			return groups.Page{}, err
		}

		ids, err = svc.filterAllowedGroupIDsOfUserID(ctx, session.DomainUserID, gm.Permission, gids.Policies)
		if err != nil {
			return groups.Page{}, err
		}
	case policies.UsersKind:
		switch {
		case memberID != "" && session.UserID != memberID:
			gids, err := svc.policies.ListAllObjects(ctx, policies.PolicyReq{
				SubjectType: policies.UserType,
				Subject:     mgauth.EncodeDomainUserID(session.DomainID, memberID),
				Permission:  gm.Permission,
				ObjectType:  policies.GroupType,
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
	permissions, err := svc.policies.ListPermissions(ctx, policies.PolicyReq{
		SubjectType: policies.UserType,
		Subject:     userID,
		Object:      groupID,
		ObjectType:  policies.GroupType,
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
func (svc service) ListMembers(ctx context.Context, session auth.Session, groupID, permission, memberKind string) (groups.MembersPage, error) {
	switch memberKind {
	case policies.ThingsKind:
		tids, err := svc.policies.ListAllObjects(ctx, policies.PolicyReq{
			SubjectType: policies.GroupType,
			Subject:     groupID,
			Relation:    policies.GroupRelation,
			ObjectType:  policies.ThingType,
		})
		if err != nil {
			return groups.MembersPage{}, err
		}

		members := []groups.Member{}

		for _, id := range tids.Policies {
			members = append(members, groups.Member{
				ID:   id,
				Type: policies.ThingType,
			})
		}
		return groups.MembersPage{
			Total:   uint64(len(members)),
			Offset:  0,
			Limit:   uint64(len(members)),
			Members: members,
		}, nil
	case policies.UsersKind:
		uids, err := svc.policies.ListAllSubjects(ctx, policies.PolicyReq{
			SubjectType: policies.UserType,
			Permission:  permission,
			Object:      groupID,
			ObjectType:  policies.GroupType,
		})
		if err != nil {
			return groups.MembersPage{}, err
		}

		members := []groups.Member{}

		for _, id := range uids.Policies {
			members = append(members, groups.Member{
				ID:   id,
				Type: policies.UserType,
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
	policyList := []policies.PolicyReq{}
	switch memberKind {
	case policies.ThingsKind:
		for _, memberID := range memberIDs {
			policyList = append(policyList, policies.PolicyReq{
				Domain:      session.DomainID,
				SubjectType: policies.GroupType,
				SubjectKind: policies.ChannelsKind,
				Subject:     groupID,
				Relation:    relation,
				ObjectType:  policies.ThingType,
				Object:      memberID,
			})
		}
	case policies.ChannelsKind:
		for _, memberID := range memberIDs {
			policyList = append(policyList, policies.PolicyReq{
				Domain:      session.DomainID,
				SubjectType: policies.GroupType,
				Subject:     memberID,
				Relation:    relation,
				ObjectType:  policies.GroupType,
				Object:      groupID,
			})
		}
	case policies.GroupsKind:
		return svc.assignParentGroup(ctx, session.DomainID, groupID, memberIDs)

	case policies.UsersKind:
		for _, memberID := range memberIDs {
			policyList = append(policyList, policies.PolicyReq{
				Domain:      session.DomainID,
				SubjectType: policies.UserType,
				Subject:     mgauth.EncodeDomainUserID(session.DomainID, memberID),
				Relation:    relation,
				ObjectType:  policies.GroupType,
				Object:      groupID,
			})
		}
	default:
		return errMemberKind
	}

	if err := svc.policies.AddPolicies(ctx, policyList); err != nil {
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

	policyList := []policies.PolicyReq{}
	for _, group := range groupsPage.Groups {
		if group.Parent != "" {
			return errors.Wrap(svcerr.ErrConflict, fmt.Errorf("%s group already have parent", group.ID))
		}
		policyList = append(policyList, policies.PolicyReq{
			Domain:      domain,
			SubjectType: policies.GroupType,
			Subject:     parentGroupID,
			Relation:    policies.ParentGroupRelation,
			ObjectType:  policies.GroupType,
			Object:      group.ID,
		})
	}

	if err := svc.policies.AddPolicies(ctx, policyList); err != nil {
		return errors.Wrap(svcerr.ErrAddPolicies, err)
	}
	defer func() {
		if err != nil {
			if errRollback := svc.policies.DeletePolicies(ctx, policyList); errRollback != nil {
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

	policyList := []policies.PolicyReq{}
	for _, group := range groupsPage.Groups {
		if group.Parent != "" && group.Parent != parentGroupID {
			return errors.Wrap(svcerr.ErrConflict, fmt.Errorf("%s group doesn't have same parent", group.ID))
		}
		policyList = append(policyList, policies.PolicyReq{
			Domain:      domain,
			SubjectType: policies.GroupType,
			Subject:     parentGroupID,
			Relation:    policies.ParentGroupRelation,
			ObjectType:  policies.GroupType,
			Object:      group.ID,
		})
	}

	if err := svc.policies.DeletePolicies(ctx, policyList); err != nil {
		return errors.Wrap(svcerr.ErrDeletePolicies, err)
	}
	defer func() {
		if err != nil {
			if errRollback := svc.policies.AddPolicies(ctx, policyList); errRollback != nil {
				err = errors.Wrap(err, errors.Wrap(apiutil.ErrRollbackTx, errRollback))
			}
		}
	}()

	return svc.groups.UnassignParentGroup(ctx, parentGroupID, groupIDs...)
}

func (svc service) Unassign(ctx context.Context, session auth.Session, groupID, relation, memberKind string, memberIDs ...string) error {
	policyList := []policies.PolicyReq{}
	switch memberKind {
	case policies.ThingsKind:
		for _, memberID := range memberIDs {
			policyList = append(policyList, policies.PolicyReq{
				Domain:      session.DomainID,
				SubjectType: policies.GroupType,
				SubjectKind: policies.ChannelsKind,
				Subject:     groupID,
				Relation:    relation,
				ObjectType:  policies.ThingType,
				Object:      memberID,
			})
		}
	case policies.ChannelsKind:
		for _, memberID := range memberIDs {
			policyList = append(policyList, policies.PolicyReq{
				Domain:      session.DomainID,
				SubjectType: policies.GroupType,
				Subject:     memberID,
				Relation:    relation,
				ObjectType:  policies.GroupType,
				Object:      groupID,
			})
		}
	case policies.GroupsKind:
		return svc.unassignParentGroup(ctx, session.DomainID, groupID, memberIDs)
	case policies.UsersKind:
		for _, memberID := range memberIDs {
			policyList = append(policyList, policies.PolicyReq{
				Domain:      session.DomainID,
				SubjectType: policies.UserType,
				Subject:     mgauth.EncodeDomainUserID(session.DomainID, memberID),
				Relation:    relation,
				ObjectType:  policies.GroupType,
				Object:      groupID,
			})
		}
	default:
		return errMemberKind
	}

	if err := svc.policies.DeletePolicies(ctx, policyList); err != nil {
		return errors.Wrap(svcerr.ErrDeletePolicies, err)
	}
	return nil
}

func (svc service) DeleteGroup(ctx context.Context, session auth.Session, id string) error {
	req := policies.PolicyReq{
		SubjectType: policies.GroupType,
		Subject:     id,
	}
	if err := svc.policies.DeletePolicyFilter(ctx, req); err != nil {
		return errors.Wrap(svcerr.ErrDeletePolicies, err)
	}

	req = policies.PolicyReq{
		Object:     id,
		ObjectType: policies.GroupType,
	}

	if err := svc.policies.DeletePolicyFilter(ctx, req); err != nil {
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
	allowedIDs, err := svc.policies.ListAllObjects(ctx, policies.PolicyReq{
		SubjectType: policies.UserType,
		Subject:     userID,
		Permission:  permission,
		ObjectType:  policies.GroupType,
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

func (svc service) addGroupPolicy(ctx context.Context, userID, domainID, id, parentID, kind string) ([]policies.PolicyReq, error) {
	policyList := []policies.PolicyReq{}
	policyList = append(policyList, policies.PolicyReq{
		Domain:      domainID,
		SubjectType: policies.UserType,
		Subject:     userID,
		Relation:    policies.AdministratorRelation,
		ObjectKind:  kind,
		ObjectType:  policies.GroupType,
		Object:      id,
	})
	policyList = append(policyList, policies.PolicyReq{
		Domain:      domainID,
		SubjectType: policies.DomainType,
		Subject:     domainID,
		Relation:    policies.DomainRelation,
		ObjectType:  policies.GroupType,
		Object:      id,
	})
	if parentID != "" {
		policyList = append(policyList, policies.PolicyReq{
			Domain:      domainID,
			SubjectType: policies.GroupType,
			Subject:     parentID,
			Relation:    policies.ParentGroupRelation,
			ObjectKind:  kind,
			ObjectType:  policies.GroupType,
			Object:      id,
		})
	}
	if err := svc.policies.AddPolicies(ctx, policyList); err != nil {
		return policyList, errors.Wrap(svcerr.ErrAddPolicies, err)
	}

	return []policies.PolicyReq{}, nil
}
