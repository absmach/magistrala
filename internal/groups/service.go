// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package groups

import (
	"context"
	"fmt"
	"time"

	"github.com/absmach/magistrala"
	"github.com/absmach/magistrala/auth"
	grpcclient "github.com/absmach/magistrala/auth/api/grpc"
	"github.com/absmach/magistrala/pkg/apiutil"
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
	auth       grpcclient.AuthServiceClient
	policy     policy.PolicyClient
	idProvider magistrala.IDProvider
}

// NewService returns a new Clients service implementation.
func NewService(g groups.Repository, idp magistrala.IDProvider, authClient grpcclient.AuthServiceClient, policyClient policy.PolicyClient) groups.Service {
	return service{
		groups:     g,
		idProvider: idp,
		auth:       authClient,
		policy:     policyClient,
	}
}

func (svc service) CreateGroup(ctx context.Context, token, kind string, g groups.Group) (gr groups.Group, err error) {
	res, err := svc.identify(ctx, token)
	if err != nil {
		return groups.Group{}, err
	}
	// If domain is disabled , then this authorization will fail for all non-admin domain users
	if _, err := svc.authorizeKind(ctx, "", auth.UserType, auth.UsersKind, res.GetId(), auth.CreatePermission, auth.DomainType, res.GetDomainId()); err != nil {
		return groups.Group{}, err
	}
	groupID, err := svc.idProvider.ID()
	if err != nil {
		return groups.Group{}, err
	}
	if g.Status != mgclients.EnabledStatus && g.Status != mgclients.DisabledStatus {
		return groups.Group{}, svcerr.ErrInvalidStatus
	}

	g.ID = groupID
	g.CreatedAt = time.Now()
	g.Domain = res.GetDomainId()
	if g.Parent != "" {
		_, err := svc.authorizeToken(ctx, auth.UserType, token, auth.EditPermission, auth.GroupType, g.Parent)
		if err != nil {
			return groups.Group{}, errors.Wrap(errParentUnAuthz, err)
		}
	}

	policies, err := svc.addGroupPolicy(ctx, res.GetId(), res.GetDomainId(), g.ID, g.Parent, kind)
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

func (svc service) ViewGroup(ctx context.Context, token, id string) (groups.Group, error) {
	_, err := svc.authorizeToken(ctx, auth.UserType, token, auth.ViewPermission, auth.GroupType, id)
	if err != nil {
		return groups.Group{}, err
	}

	group, err := svc.groups.RetrieveByID(ctx, id)
	if err != nil {
		return groups.Group{}, errors.Wrap(svcerr.ErrViewEntity, err)
	}

	return group, nil
}

func (svc service) ViewGroupPerms(ctx context.Context, token, id string) ([]string, error) {
	res, err := svc.identify(ctx, token)
	if err != nil {
		return nil, err
	}

	return svc.listUserGroupPermission(ctx, res.GetId(), id)
}

func (svc service) ListGroups(ctx context.Context, token, memberKind, memberID string, gm groups.Page) (groups.Page, error) {
	var ids []string
	res, err := svc.identify(ctx, token)
	if err != nil {
		return groups.Page{}, err
	}
	switch memberKind {
	case auth.ThingsKind:
		if _, err := svc.authorizeKind(ctx, res.GetDomainId(), auth.UserType, auth.UsersKind, res.GetId(), auth.ViewPermission, auth.ThingType, memberID); err != nil {
			return groups.Page{}, err
		}
		cids, err := svc.policy.ListAllSubjects(ctx, policy.PolicyReq{
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
		if _, err := svc.authorizeKind(ctx, res.GetDomainId(), auth.UserType, auth.UsersKind, res.GetId(), gm.Permission, auth.GroupType, memberID); err != nil {
			return groups.Page{}, err
		}

		gids, err := svc.policy.ListAllObjects(ctx, policy.PolicyReq{
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
		if _, err := svc.authorizeKind(ctx, res.GetDomainId(), auth.UserType, auth.UsersKind, res.GetId(), auth.ViewPermission, auth.GroupType, memberID); err != nil {
			return groups.Page{}, err
		}
		gids, err := svc.policy.ListAllSubjects(ctx, policy.PolicyReq{
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
		switch {
		case memberID != "" && res.GetUserId() != memberID:
			if _, err := svc.authorizeKind(ctx, res.GetDomainId(), auth.UserType, auth.UsersKind, res.GetId(), auth.AdminPermission, auth.DomainType, res.GetDomainId()); err != nil {
				return groups.Page{}, err
			}
			gids, err := svc.policy.ListAllObjects(ctx, policy.PolicyReq{
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
		default:
			switch svc.checkSuperAdmin(ctx, res.GetUserId()) {
			case nil:
				gm.PageMeta.DomainID = res.GetDomainId()
			default:
				// If domain is disabled , then this authorization will fail for all non-admin domain users
				if _, err := svc.authorizeKind(ctx, "", auth.UserType, auth.UsersKind, res.GetId(), auth.MembershipPermission, auth.DomainType, res.GetDomainId()); err != nil {
					return groups.Page{}, err
				}
				ids, err = svc.listAllGroupsOfUserID(ctx, res.GetId(), gm.Permission)
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
				return svc.retrievePermissions(ctx, res.GetId(), &gp.Groups[iter])
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
		SubjectType: auth.UserType,
		Subject:     userID,
		Object:      groupID,
		ObjectType:  auth.GroupType,
	}, []string{})
	if err != nil {
		return []string{}, err
	}
	if len(permissions) == 0 {
		return []string{}, svcerr.ErrAuthorization
	}
	return permissions, nil
}

func (svc service) checkSuperAdmin(ctx context.Context, userID string) error {
	res, err := svc.auth.Authorize(ctx, &magistrala.AuthorizeReq{
		SubjectType: auth.UserType,
		Subject:     userID,
		Permission:  auth.AdminPermission,
		ObjectType:  auth.PlatformType,
		Object:      auth.MagistralaObject,
	})
	if err != nil {
		return errors.Wrap(svcerr.ErrAuthorization, err)
	}
	if !res.Authorized {
		return svcerr.ErrAuthorization
	}
	return nil
}

// IMPROVEMENT NOTE: remove this function and all its related auxiliary function, ListMembers are moved to respective service.
func (svc service) ListMembers(ctx context.Context, token, groupID, permission, memberKind string) (groups.MembersPage, error) {
	_, err := svc.authorizeToken(ctx, auth.UserType, token, auth.ViewPermission, auth.GroupType, groupID)
	if err != nil {
		return groups.MembersPage{}, err
	}
	switch memberKind {
	case auth.ThingsKind:
		tids, err := svc.policy.ListAllObjects(ctx, policy.PolicyReq{
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
		uids, err := svc.policy.ListAllSubjects(ctx, policy.PolicyReq{
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
	id, err := svc.authorizeToken(ctx, auth.UserType, token, auth.EditPermission, auth.GroupType, g.ID)
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
	if _, err := svc.authorizeKind(ctx, res.GetDomainId(), auth.UserType, auth.UsersKind, res.GetId(), auth.EditPermission, auth.GroupType, groupID); err != nil {
		return err
	}

	policies := []policy.PolicyReq{}
	switch memberKind {
	case auth.ThingsKind:
		for _, memberID := range memberIDs {
			policies = append(policies, policy.PolicyReq{
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
			policies = append(policies, policy.PolicyReq{
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
			policies = append(policies, policy.PolicyReq{
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
			SubjectType: auth.GroupType,
			Subject:     parentGroupID,
			Relation:    auth.ParentGroupRelation,
			ObjectType:  auth.GroupType,
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
			SubjectType: auth.GroupType,
			Subject:     parentGroupID,
			Relation:    auth.ParentGroupRelation,
			ObjectType:  auth.GroupType,
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

func (svc service) Unassign(ctx context.Context, token, groupID, relation, memberKind string, memberIDs ...string) error {
	res, err := svc.identify(ctx, token)
	if err != nil {
		return err
	}
	if _, err := svc.authorizeKind(ctx, res.GetDomainId(), auth.UserType, auth.UsersKind, res.GetId(), auth.EditPermission, auth.GroupType, groupID); err != nil {
		return err
	}

	policies := []policy.PolicyReq{}

	switch memberKind {
	case auth.ThingsKind:
		for _, memberID := range memberIDs {
			policies = append(policies, policy.PolicyReq{
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
			policies = append(policies, policy.PolicyReq{
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
			policies = append(policies, policy.PolicyReq{
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

	if err := svc.policy.DeletePolicies(ctx, policies); err != nil {
		return errors.Wrap(svcerr.ErrDeletePolicies, err)
	}
	return nil
}

func (svc service) DeleteGroup(ctx context.Context, token, id string) error {
	res, err := svc.identify(ctx, token)
	if err != nil {
		return err
	}
	if _, err := svc.authorizeKind(ctx, res.GetDomainId(), auth.UserType, auth.UsersKind, res.GetId(), auth.DeletePermission, auth.GroupType, id); err != nil {
		return err
	}

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
	id, err := svc.authorizeToken(ctx, auth.UserType, token, auth.EditPermission, auth.GroupType, group.ID)
	if err != nil {
		return groups.Group{}, err
	}
	dbGroup, err := svc.groups.RetrieveByID(ctx, group.ID)
	if err != nil {
		return groups.Group{}, errors.Wrap(svcerr.ErrViewEntity, err)
	}
	if dbGroup.Status == group.Status {
		return groups.Group{}, errors.ErrStatusAlreadyAssigned
	}

	group.UpdatedBy = id
	return svc.groups.ChangeStatus(ctx, group)
}

func (svc service) identify(ctx context.Context, token string) (*magistrala.IdentityRes, error) {
	res, err := svc.auth.Identify(ctx, &magistrala.IdentityReq{Token: token})
	if err != nil {
		return nil, errors.Wrap(svcerr.ErrAuthentication, err)
	}
	if res.GetId() == "" || res.GetDomainId() == "" {
		return nil, svcerr.ErrDomainAuthorization
	}
	return res, nil
}

func (svc service) authorizeToken(ctx context.Context, subjectType, subject, permission, objectType, object string) (string, error) {
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
		return "", errors.Wrap(svcerr.ErrAuthorization, err)
	}
	if !res.GetAuthorized() {
		return "", svcerr.ErrAuthorization
	}
	return res.GetId(), nil
}

func (svc service) authorizeKind(ctx context.Context, domainID, subjectType, subjectKind, subject, permission, objectType, object string) (string, error) {
	req := &magistrala.AuthorizeReq{
		Domain:      domainID,
		SubjectType: subjectType,
		SubjectKind: subjectKind,
		Subject:     subject,
		Permission:  permission,
		Object:      object,
		ObjectType:  objectType,
	}
	res, err := svc.auth.Authorize(ctx, req)
	if err != nil {
		return "", errors.Wrap(svcerr.ErrAuthorization, err)
	}
	if !res.GetAuthorized() {
		return "", svcerr.ErrAuthorization
	}
	return res.GetId(), nil
}

func (svc service) addGroupPolicy(ctx context.Context, userID, domainID, id, parentID, kind string) ([]policy.PolicyReq, error) {
	policies := []policy.PolicyReq{}
	policies = append(policies, policy.PolicyReq{
		Domain:      domainID,
		SubjectType: auth.UserType,
		Subject:     userID,
		Relation:    auth.AdministratorRelation,
		ObjectKind:  kind,
		ObjectType:  auth.GroupType,
		Object:      id,
	})
	policies = append(policies, policy.PolicyReq{
		Domain:      domainID,
		SubjectType: auth.DomainType,
		Subject:     domainID,
		Relation:    auth.DomainRelation,
		ObjectType:  auth.GroupType,
		Object:      id,
	})
	if parentID != "" {
		policies = append(policies, policy.PolicyReq{
			Domain:      domainID,
			SubjectType: auth.GroupType,
			Subject:     parentID,
			Relation:    auth.ParentGroupRelation,
			ObjectKind:  kind,
			ObjectType:  auth.GroupType,
			Object:      id,
		})
	}
	if err := svc.policy.AddPolicies(ctx, policies); err != nil {
		return policies, errors.Wrap(svcerr.ErrAddPolicies, err)
	}

	return []policy.PolicyReq{}, nil
}
