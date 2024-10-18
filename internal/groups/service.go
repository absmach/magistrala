// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package groups

import (
	"context"
	"fmt"
	"time"

	"github.com/absmach/magistrala"
	"github.com/absmach/magistrala/pkg/apiutil"
	mgauthn "github.com/absmach/magistrala/pkg/authn"
	"github.com/absmach/magistrala/pkg/authz"
	mgauthz "github.com/absmach/magistrala/pkg/authz"
	mgclients "github.com/absmach/magistrala/pkg/clients"
	"github.com/absmach/magistrala/pkg/entityroles"
	"github.com/absmach/magistrala/pkg/errors"
	svcerr "github.com/absmach/magistrala/pkg/errors/service"
	"github.com/absmach/magistrala/pkg/groups"
	"github.com/absmach/magistrala/pkg/policies"
	"github.com/absmach/magistrala/pkg/roles"
	"github.com/absmach/magistrala/pkg/svcutil"
	"golang.org/x/sync/errgroup"
)

var (
	errMemberKind = errors.New("invalid member kind")
	errGroupIDs   = errors.New("invalid group ids")
)

type identity struct {
	ID       string
	DomainID string
	UserID   string
}

type service struct {
	groups     groups.Repository
	authz      mgauthz.Authorization
	policies   policies.Service
	idProvider magistrala.IDProvider
	opp        svcutil.OperationPerm
	entityroles.RolesSvc
}

// NewService returns a new Clients service implementation.
func NewService(repo groups.Repository, idp magistrala.IDProvider, policyService policies.Service, authz mgauthz.Authorization, groupsOpPerm map[svcutil.Operation]svcutil.Permission) (groups.Service, error) {
	opp := groups.NewOperationPerm()
	if err := opp.AddOperationPermissionMap(groupsOpPerm); err != nil {
		return nil, err
	}
	if err := opp.Validate(); err != nil {
		return nil, err
	}

	rolesSvc, err := entityroles.NewRolesSvc("group", repo, idp, policyService, groups.AvailableActions(), groups.BuiltInRoles())
	if err != nil {
		return service{}, err
	}
	return service{
		groups:     repo,
		idProvider: idp,
		policies:   policyService,
		authz:      authz,
		opp:        opp,
		RolesSvc:   rolesSvc,
	}, nil
}

func (svc service) CreateGroup(ctx context.Context, session mgauthn.Session, g groups.Group) (gr groups.Group, retErr error) {
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

	saved, err := svc.groups.Save(ctx, g)
	if err != nil {
		return groups.Group{}, errors.Wrap(svcerr.ErrCreateEntity, err)
	}

	defer func() {
		if retErr != nil {
			if errRollback := svc.groups.Delete(ctx, saved.ID); errRollback != nil {
				retErr = errors.Wrap(retErr, errors.Wrap(apiutil.ErrRollbackTx, errRollback))
			}
		}
	}()

	oprs := []roles.OptionalPolicy{}

	oprs = append(oprs, roles.OptionalPolicy{
		Namespace:   session.DomainID,
		SubjectType: policies.DomainType,
		Subject:     session.DomainID,
		Relation:    policies.DomainRelation,
		ObjectType:  policies.GroupType,
		Object:      saved.ID,
	})
	if saved.Parent != "" {
		oprs = append(oprs, roles.OptionalPolicy{
			Namespace:   session.DomainID,
			SubjectType: policies.GroupType,
			Subject:     saved.Parent,
			Relation:    policies.ParentGroupRelation,
			ObjectType:  policies.GroupType,
			Object:      saved.ID,
		})
	}
	newBuiltInRoleMembers := map[roles.BuiltInRoleName][]roles.Member{
		groups.BuiltInRoleAdmin:      {roles.Member(session.UserID)},
		groups.BuiltInRoleMembership: {},
	}
	if _, err := svc.AddNewEntityRoles(ctx, session.DomainUserID, session.DomainID, saved.ID, newBuiltInRoleMembers, oprs); err != nil {
		return groups.Group{}, errors.Wrap(svcerr.ErrAddPolicies, err)
	}

	return saved, nil
}

func (svc service) ViewGroup(ctx context.Context, session mgauthn.Session, id string) (groups.Group, error) {
	group, err := svc.groups.RetrieveByID(ctx, id)
	if err != nil {
		return groups.Group{}, errors.Wrap(svcerr.ErrViewEntity, err)
	}

	return group, nil
}

func (svc service) ViewGroupPerms(ctx context.Context, session mgauthn.Session, id string) ([]string, error) {
	return svc.listUserGroupPermission(ctx, session.DomainUserID, id)
}

func (svc service) ListGroups(ctx context.Context, session mgauthn.Session, gm groups.PageMeta) (groups.Page, error) {
	var ids []string
	var err error

	switch session.SuperAdmin {
	case true:
		gm.DomainID = session.DomainID
	default:
		ids, err = svc.listAllGroupsOfUserID(ctx, session.DomainUserID, gm.Permission)
		if err != nil {
			return groups.Page{}, err
		}
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
	permissions, err := svc.policies.ListPermissions(ctx, policies.Policy{
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

func (svc service) UpdateGroup(ctx context.Context, session mgauthn.Session, g groups.Group) (groups.Group, error) {
	g.UpdatedAt = time.Now()
	g.UpdatedBy = session.UserID

	return svc.groups.Update(ctx, g)
}

func (svc service) EnableGroup(ctx context.Context, session mgauthn.Session, id string) (groups.Group, error) {

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

func (svc service) DisableGroup(ctx context.Context, session mgauthn.Session, id string) (groups.Group, error) {
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

func (svc service) RetrieveGroupHierarchy(ctx context.Context, session mgauthn.Session, id string, hm groups.HierarchyPageMeta) (groups.HierarchyPage, error) {
	hp, err := svc.groups.RetrieveHierarchy(ctx, id, hm)
	if err != nil {
		return groups.HierarchyPage{}, errors.Wrap(svcerr.ErrViewEntity, err)
	}
	hids := svc.getGroupIDs(hp.Groups)
	ids, err := svc.filterAllowedGroupIDsOfUserID(ctx, session.DomainUserID, "read_permission", hids)
	if err != nil {
		return groups.HierarchyPage{}, errors.Wrap(svcerr.ErrViewEntity, err)
	}
	hp.Groups = svc.allowedGroups(hp.Groups, ids)
	return hp, nil
}

func (svc service) allowedGroups(gps []groups.Group, ids []string) []groups.Group {
	aIDs := make(map[string]struct{}, len(ids))

	for _, id := range ids {
		aIDs[id] = struct{}{}
	}

	aGroups := []groups.Group{}
	for _, g := range gps {
		ag := g
		if _, ok := aIDs[g.ID]; !ok {
			ag = groups.Group{ID: "xxxx-xxxx-xxxx-xxxx", Level: g.Level}
		}
		aGroups = append(aGroups, ag)
	}
	return aGroups
}
func (svc service) getGroupIDs(gps []groups.Group) []string {
	hids := []string{}
	for _, g := range gps {
		hids = append(hids, g.ID)
		if len(g.Children) > 0 {
			children := make([]groups.Group, len(g.Children))
			for i, child := range g.Children {
				children[i] = *child
			}
			cids := svc.getGroupIDs(children)
			hids = append(hids, cids...)
		}
	}
	return hids
}
func (svc service) AddParentGroup(ctx context.Context, session mgauthn.Session, id, parentID string) (retErr error) {

	group, err := svc.groups.RetrieveByID(ctx, id)
	if err != nil {
		return errors.Wrap(svcerr.ErrViewEntity, err)
	}

	var pols []policies.Policy
	if group.Parent != "" {
		return errors.Wrap(svcerr.ErrConflict, fmt.Errorf("%s group already have parent", group.ID))
	}
	pols = append(pols, policies.Policy{
		Domain:      session.DomainID,
		SubjectType: policies.GroupType,
		Subject:     parentID,
		Relation:    policies.ParentGroupRelation,
		ObjectType:  policies.GroupType,
		Object:      group.ID,
	})

	if err := svc.policies.AddPolicies(ctx, pols); err != nil {
		return errors.Wrap(svcerr.ErrAddPolicies, err)
	}
	defer func() {
		if retErr != nil {
			if errRollback := svc.policies.DeletePolicies(ctx, pols); errRollback != nil {
				retErr = errors.Wrap(retErr, errors.Wrap(apiutil.ErrRollbackTx, errRollback))
			}
		}
	}()

	if err := svc.groups.AssignParentGroup(ctx, parentID, group.ID); err != nil {
		return err
	}
	return nil
}

func (svc service) RemoveParentGroup(ctx context.Context, session mgauthn.Session, id string) (retErr error) {

	group, err := svc.groups.RetrieveByID(ctx, id)
	if err != nil {
		return errors.Wrap(svcerr.ErrViewEntity, err)
	}

	if group.Parent != "" {
		if err := svc.authorize(ctx, groups.OpRemoveChildrenGroups, mgauthz.PolicyReq{
			Domain:      session.DomainID,
			SubjectType: policies.UserType,
			Subject:     session.DomainUserID,
			Object:      group.Parent,
			ObjectType:  policies.GroupType,
		}); err != nil {
			return err
		}

		var pols []policies.Policy

		pols = append(pols, policies.Policy{
			Domain:      session.DomainID,
			SubjectType: policies.GroupType,
			Subject:     group.Parent,
			Relation:    policies.ParentGroupRelation,
			ObjectType:  policies.GroupType,
			Object:      group.ID,
		})

		if err := svc.policies.DeletePolicies(ctx, pols); err != nil {
			return errors.Wrap(svcerr.ErrDeletePolicies, err)
		}
		defer func() {
			if retErr != nil {
				if errRollback := svc.policies.AddPolicies(ctx, pols); errRollback != nil {
					retErr = errors.Wrap(retErr, errors.Wrap(apiutil.ErrRollbackTx, errRollback))
				}
			}
		}()

		return svc.groups.UnassignParentGroup(ctx, group.Parent, group.ID)
	}

	return nil
}

func (svc service) ViewParentGroup(ctx context.Context, session mgauthn.Session, id string) (groups.Group, error) {
	g, err := svc.groups.RetrieveByID(ctx, id)
	if err != nil {
		return groups.Group{}, err
	}

	if g.Parent == "" {
		return groups.Group{}, nil
	}
	if err := svc.authorize(ctx, groups.OpViewGroup, mgauthz.PolicyReq{
		Domain:      session.DomainID,
		SubjectType: policies.UserType,
		Subject:     session.DomainUserID,
		Object:      g.Parent,
		ObjectType:  policies.GroupType,
	}); err != nil {
		return groups.Group{}, err
	}

	pg, err := svc.groups.RetrieveByID(ctx, g.Parent)
	if err != nil {
		return groups.Group{}, err
	}
	return pg, nil

}

func (svc service) AddChildrenGroups(ctx context.Context, session mgauthn.Session, parentGroupID string, childrenGroupIDs []string) (retErr error) {
	childrenGroupsPage, err := svc.groups.RetrieveByIDs(ctx, groups.PageMeta{Limit: 1<<63 - 1}, childrenGroupIDs...)
	if err != nil {
		return errors.Wrap(svcerr.ErrViewEntity, err)
	}
	if len(childrenGroupsPage.Groups) == 0 {
		return errGroupIDs
	}

	for _, childGroup := range childrenGroupsPage.Groups {
		if childGroup.Parent != "" {
			return errors.Wrap(svcerr.ErrConflict, fmt.Errorf("%s group already have parent", childGroup.ID))
		}
		if err := svc.authorize(ctx, groups.OpAddParentGroup, mgauthz.PolicyReq{
			Domain:      session.DomainID,
			SubjectType: policies.UserType,
			Subject:     session.DomainUserID,
			Object:      childGroup.ID,
			ObjectType:  policies.GroupType,
		}); err != nil {
			return err
		}
	}

	var pols []policies.Policy
	for _, childGroup := range childrenGroupsPage.Groups {
		pols = append(pols, policies.Policy{
			Domain:      session.DomainID,
			SubjectType: policies.GroupType,
			Subject:     parentGroupID,
			Relation:    policies.ParentGroupRelation,
			ObjectType:  policies.GroupType,
			Object:      childGroup.ID,
		})
	}

	if err := svc.policies.AddPolicies(ctx, pols); err != nil {
		return errors.Wrap(svcerr.ErrAddPolicies, err)
	}
	defer func() {
		if retErr != nil {
			if errRollback := svc.policies.DeletePolicies(ctx, pols); errRollback != nil {
				retErr = errors.Wrap(retErr, errors.Wrap(apiutil.ErrRollbackTx, errRollback))
			}
		}
	}()

	return svc.groups.AssignParentGroup(ctx, parentGroupID, childrenGroupIDs...)
}

func (svc service) RemoveChildrenGroups(ctx context.Context, session mgauthn.Session, parentGroupID string, childrenGroupIDs []string) (retErr error) {
	childrenGroupsPage, err := svc.groups.RetrieveByIDs(ctx, groups.PageMeta{Limit: 1<<63 - 1}, childrenGroupIDs...)
	if err != nil {
		return errors.Wrap(svcerr.ErrViewEntity, err)
	}
	if len(childrenGroupsPage.Groups) == 0 {
		return errGroupIDs
	}

	var pols []policies.Policy

	for _, group := range childrenGroupsPage.Groups {
		if group.Parent != "" && group.Parent != parentGroupID {
			return errors.Wrap(svcerr.ErrConflict, fmt.Errorf("%s group doesn't have same parent", group.ID))
		}
		pols = append(pols, policies.Policy{
			Domain:      session.DomainID,
			SubjectType: policies.GroupType,
			Subject:     parentGroupID,
			Relation:    policies.ParentGroupRelation,
			ObjectType:  policies.GroupType,
			Object:      group.ID,
		})
	}

	if err := svc.policies.DeletePolicies(ctx, pols); err != nil {
		return errors.Wrap(svcerr.ErrDeletePolicies, err)
	}
	defer func() {
		if retErr != nil {
			if errRollback := svc.policies.AddPolicies(ctx, pols); errRollback != nil {
				retErr = errors.Wrap(retErr, errors.Wrap(apiutil.ErrRollbackTx, errRollback))
			}
		}
	}()

	return svc.groups.UnassignParentGroup(ctx, parentGroupID, childrenGroupIDs...)
}

func (svc service) RemoveAllChildrenGroups(ctx context.Context, session mgauthn.Session, id string) error {
	pol := policies.Policy{
		Domain:      session.DomainID,
		SubjectType: policies.GroupType,
		Subject:     id,
		Relation:    policies.ParentGroupRelation,
		ObjectType:  policies.GroupType,
	}

	if err := svc.policies.DeletePolicyFilter(ctx, pol); err != nil {
		return errors.Wrap(svcerr.ErrDeletePolicies, err)
	}

	return svc.groups.UnassignAllChildrenGroup(ctx, id)
}

func (svc service) ListChildrenGroups(ctx context.Context, session mgauthn.Session, id string, pm groups.PageMeta) (groups.Page, error) {
	cids, err := svc.policies.ListAllObjects(ctx, policies.Policy{
		SubjectType: policies.GroupType,
		Subject:     id,
		Permission:  policies.ParentGroupRelation,
		ObjectType:  policies.GroupType,
	})
	if err != nil {
		return groups.Page{}, err
	}

	ids, err := svc.filterAllowedGroupIDsOfUserID(ctx, session.DomainUserID, pm.Permission, cids.Policies)
	if err != nil {
		return groups.Page{}, errors.Wrap(svcerr.ErrViewEntity, err)
	}
	gp, err := svc.groups.RetrieveByIDs(ctx, pm, ids...)
	if err != nil {
		return groups.Page{}, errors.Wrap(svcerr.ErrViewEntity, err)
	}
	return gp, nil
}

// func (svc service) AddChannels(ctx context.Context, session mgauthn.Session, id string, channelIDs []string) error {
// 	userInfo, err := svc.identify(ctx, token)
// 	if err != nil {
// 		return err
// 	}

// 	if err := svc.authorize(ctx, groups.OpAddChannels, mgauthz.PolicyReq{
// 		Domain:      session.DomainID,
// 		SubjectType: policies.UserType,
// 		Subject:     session.DomainUserID,
// 		Object:      id,
// 		ObjectType:  policies.GroupType,
// 	}); err != nil {
// 		return err
// 	}

// 	policies := magistrala.AddPoliciesReq{}

// 	for _, channelID := range channelIDs {
// 		policies.AddPoliciesReq = append(policies.AddPoliciesReq, &magistrala.AddPolicyReq{
// 			Domain:      session.DomainID,
// 			SubjectType: policies.GroupType,
// 			SubjectKind: policies.ChannelsKind,
// 			Subject:     id,
// 			Relation:    policies.ParentGroupRelation,
// 			ObjectType:  policies.ThingType,
// 			Object:      channelID,
// 		})
// 	}

// 	if _, err := svc.policies.AddPolicies(ctx, &policies); err != nil {
// 		return errors.Wrap(svcerr.ErrAddPolicies, err)
// 	}

// 	return nil
// }

// func (svc service) RemoveChannels(ctx context.Context, session mgauthn.Session, id string, channelIDs []string) error {
// 	userInfo, err := svc.identify(ctx, token)
// 	if err != nil {
// 		return err
// 	}

// 	if err := svc.authorize(ctx, groups.OpAddChannels, mgauthz.PolicyReq{
// 		Domain:      session.DomainID,
// 		SubjectType: policies.UserType,
// 		Subject:     session.DomainUserID,
// 		Object:      id,
// 		ObjectType:  policies.GroupType,
// 	}); err != nil {
// 		return err
// 	}
// 	policies := magistrala.DeletePoliciesReq{}

// 	for _, channelID := range channelIDs {
// 		policies.DeletePoliciesReq = append(policies.DeletePoliciesReq, &magistrala.DeletePolicyReq{
// 			Domain:      session.DomainID,
// 			SubjectType: policies.GroupType,
// 			Subject:     id,
// 			Relation:    policies.ParentGroupRelation,
// 			ObjectType:  policies.ChannelType,
// 			Object:      channelID,
// 		})
// 	}
// 	if _, err := svc.policies.DeletePolicies(ctx, &policies); err != nil {
// 		return errors.Wrap(svcerr.ErrDeletePolicies, err)
// 	}

// 	return nil
// }

// func (svc service) ListChannels(ctx context.Context, session mgauthn.Session, id, gm groups.Page) (groups.Page, error) {
// 	return groups.Page{}, nil
// }

// func (svc service) AddThings(ctx context.Context, session mgauthn.Session, id string, thingIDs []string) error {
// 	userInfo, err := svc.identify(ctx, token)
// 	if err != nil {
// 		return err
// 	}

// 	if err := svc.authorize(ctx, groups.OpAddChannels, mgauthz.PolicyReq{
// 		Domain:      session.DomainID,
// 		SubjectType: policies.UserType,
// 		Subject:     session.DomainUserID,
// 		Object:      id,
// 		ObjectType:  policies.GroupType,
// 	}); err != nil {
// 		return err
// 	}
// 	policies := magistrala.AddPoliciesReq{}

// 	for _, thingID := range thingIDs {
// 		policies.AddPoliciesReq = append(policies.AddPoliciesReq, &magistrala.AddPolicyReq{
// 			Domain:      session.DomainID,
// 			SubjectType: policies.GroupType,
// 			SubjectKind: policies.ChannelsKind,
// 			Subject:     id,
// 			Relation:    policies.ParentGroupRelation,
// 			ObjectType:  policies.ThingType,
// 			Object:      thingID,
// 		})
// 	}

// 	if _, err := svc.policies.AddPolicies(ctx, &policies); err != nil {
// 		return errors.Wrap(svcerr.ErrAddPolicies, err)
// 	}

// 	return nil
// }

// func (svc service) RemoveThings(ctx context.Context, session mgauthn.Session, id string, thingIDs []string) error {
// 	userInfo, err := svc.identify(ctx, token)
// 	if err != nil {
// 		return err
// 	}

// 	if err := svc.authorize(ctx, groups.OpRemoveAllChannels, mgauthz.PolicyReq{
// 		Domain:      session.DomainID,
// 		SubjectType: policies.UserType,
// 		Subject:     session.DomainUserID,
// 		Object:      id,
// 		ObjectType:  policies.GroupType,
// 	}); err != nil {
// 		return err
// 	}
// 	policies := magistrala.DeletePoliciesReq{}

// 	for _, thingID := range thingIDs {
// 		policies.DeletePoliciesReq = append(policies.DeletePoliciesReq, &magistrala.DeletePolicyReq{
// 			Domain:      session.DomainID,
// 			SubjectType: policies.GroupType,
// 			Subject:     id,
// 			Relation:    policies.ParentGroupRelation,
// 			ObjectType:  policies.ThingType,
// 			Object:      thingID,
// 		})
// 	}
// 	if _, err := svc.policies.DeletePolicies(ctx, &policies); err != nil {
// 		return errors.Wrap(svcerr.ErrDeletePolicies, err)
// 	}

// 	return nil
// }

// func (svc service) RemoveAllThings(ctx context.Context, session mgauthn.Session, id string) error {
// 	userInfo, err := svc.identify(ctx, token)
// 	if err != nil {
// 		return err
// 	}

// 	if err := svc.authorize(ctx, groups.OpRemoveAllThings, mgauthz.PolicyReq{
// 		Domain:      session.DomainID,
// 		SubjectType: policies.UserType,
// 		Subject:     session.DomainUserID,
// 		Object:      id,
// 		ObjectType:  policies.GroupType,
// 	}); err != nil {
// 		return err
// 	}

// 	policy := magistrala.DeletePolicyFilterReq{
// 		Domain:      session.DomainID,
// 		SubjectType: policies.GroupType,
// 		Subject:     id,
// 		Relation:    policies.ParentGroupRelation,
// 		ObjectType:  policies.ThingType,
// 	}

// 	if _, err := svc.policies.DeletePolicyFilter(ctx, &policy); err != nil {
// 		return errors.Wrap(svcerr.ErrDeletePolicies, err)
// 	}
// 	return nil
// }

// func (svc service) ListThings(ctx context.Context, session mgauthn.Session, id, gm groups.Page) (groups.Page, error) {
// 	return groups.Page{}, nil
// }

func (svc service) DeleteGroup(ctx context.Context, session mgauthn.Session, id string) error {
	if err := svc.policies.DeletePolicyFilter(ctx, policies.Policy{
		SubjectType: policies.GroupType,
		Subject:     id,
	}); err != nil {
		return errors.Wrap(svcerr.ErrDeletePolicies, err)
	}

	if err := svc.policies.DeletePolicyFilter(ctx, policies.Policy{
		ObjectType: policies.GroupType,
		Object:     id,
	}); err != nil {
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
	allowedIDs, err := svc.policies.ListAllObjects(ctx, policies.Policy{
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

func (svc service) changeGroupStatus(ctx context.Context, session mgauthn.Session, group groups.Group) (groups.Group, error) {
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
func (svc service) authorize(ctx context.Context, op svcutil.Operation, pr authz.PolicyReq) error {
	perm, err := svc.opp.GetPermission(op)
	if err != nil {
		return err
	}
	pr.Permission = perm.String()
	if err := svc.authz.Authorize(ctx, pr); err != nil {
		return err
	}
	return nil
}
