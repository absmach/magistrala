// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package roles

import (
	"context"
	"fmt"
	"time"

	"github.com/absmach/supermq"
	"github.com/absmach/supermq/pkg/authn"
	"github.com/absmach/supermq/pkg/errors"
	svcerr "github.com/absmach/supermq/pkg/errors/service"
	"github.com/absmach/supermq/pkg/policies"
)

var (
	errRemoveOptionalDeletePolicies       = errors.New("failed to delete the additional requested policies")
	errRemoveOptionalFilterDeletePolicies = errors.New("failed to filter delete the additional requested policies")
	errRollbackRoles                      = errors.New("failed to rollback roles")
)

type roleProvisionerManger interface {
	RoleManager
	Provisioner
}

var _ roleProvisionerManger = (*ProvisionManageService)(nil)

type ProvisionManageService struct {
	entityType   string
	repo         Repository
	sidProvider  supermq.IDProvider
	policy       policies.Service
	actions      []Action
	builtInRoles map[BuiltInRoleName][]Action
}

func NewProvisionManageService(entityType string, repo Repository, policy policies.Service, sidProvider supermq.IDProvider, actions []Action, builtInRoles map[BuiltInRoleName][]Action) (ProvisionManageService, error) {
	rm := ProvisionManageService{
		entityType:   entityType,
		repo:         repo,
		sidProvider:  sidProvider,
		policy:       policy,
		actions:      actions,
		builtInRoles: builtInRoles,
	}
	return rm, nil
}

func toRolesActions(actions []string) []Action {
	roActions := []Action{}
	for _, action := range actions {
		roActions = append(roActions, Action(action))
	}
	return roActions
}

func roleActionsToString(roActions []Action) []string {
	actions := []string{}
	for _, roAction := range roActions {
		actions = append(actions, roAction.String())
	}
	return actions
}

func roleMembersToString(roMems []Member) []string {
	mems := []string{}
	for _, roMem := range roMems {
		mems = append(mems, roMem.String())
	}
	return mems
}

func (r ProvisionManageService) isActionAllowed(action Action) bool {
	for _, cap := range r.actions {
		if cap == action {
			return true
		}
	}
	return false
}

func (r ProvisionManageService) validateActions(actions []Action) error {
	for _, ac := range actions {
		action := Action(ac)
		if !r.isActionAllowed(action) {
			return errors.Wrap(svcerr.ErrMalformedEntity, fmt.Errorf("invalid action %s ", action))
		}
	}
	return nil
}

func (r ProvisionManageService) RemoveEntitiesRoles(ctx context.Context, domainID, userID string, entityIDs []string, optionalFilterDeletePolicies []policies.Policy, optionalDeletePolicies []policies.Policy) error {
	ears, emrs, err := r.repo.RetrieveEntitiesRolesActionsMembers(ctx, entityIDs)
	if err != nil {
		return err
	}

	deletePolicies := []policies.Policy{}
	for _, ear := range ears {
		deletePolicies = append(deletePolicies, policies.Policy{
			Subject:         ear.RoleID,
			SubjectRelation: policies.MemberRelation,
			SubjectType:     policies.RoleType,
			Relation:        ear.Action,
			ObjectType:      r.entityType,
			Object:          ear.EntityID,
		})
	}
	for _, emr := range emrs {
		deletePolicies = append(deletePolicies, policies.Policy{
			Subject:     policies.EncodeDomainUserID(domainID, emr.MemberID),
			SubjectType: policies.UserType,
			Relation:    policies.MemberRelation,
			ObjectType:  policies.RoleType,
			Object:      emr.RoleID,
		})
	}

	if err := r.policy.DeletePolicies(ctx, deletePolicies); err != nil {
		return errors.Wrap(svcerr.ErrDeletePolicies, err)
	}

	if len(optionalDeletePolicies) > 1 {
		if err := r.policy.DeletePolicies(ctx, optionalDeletePolicies); err != nil {
			return errors.Wrap(errRemoveOptionalDeletePolicies, err)
		}
	}

	for _, optionalFilterDeletePolicy := range optionalFilterDeletePolicies {
		if err := r.policy.DeletePolicyFilter(ctx, optionalFilterDeletePolicy); err != nil {
			return errors.Wrap(errRemoveOptionalFilterDeletePolicies, err)
		}
	}
	return nil
}

func (r ProvisionManageService) AddNewEntitiesRoles(ctx context.Context, domainID, userID string, entityIDs []string, optionalEntityPolicies []policies.Policy, newBuiltInRoleMembers map[BuiltInRoleName][]Member) (retRolesProvision []RoleProvision, retErr error) {
	var newRolesProvision []RoleProvision
	prs := []policies.Policy{}

	for _, entityID := range entityIDs {
		for defaultRole, defaultRoleMembers := range newBuiltInRoleMembers {
			actions, ok := r.builtInRoles[defaultRole]
			if !ok {
				return []RoleProvision{}, fmt.Errorf("default role %s not found in in-built roles", defaultRole)
			}

			sid, err := r.sidProvider.ID()
			if err != nil {
				return []RoleProvision{}, errors.Wrap(svcerr.ErrCreateEntity, err)
			}

			id := r.entityType + "_" + sid
			if err := r.validateActions(actions); err != nil {
				return []RoleProvision{}, errors.Wrap(svcerr.ErrMalformedEntity, err)
			}

			members := roleMembersToString(defaultRoleMembers)
			caps := roleActionsToString(actions)

			newRolesProvision = append(newRolesProvision, RoleProvision{
				Role: Role{
					ID:        id,
					Name:      defaultRole.String(),
					EntityID:  entityID,
					CreatedAt: time.Now(),
					CreatedBy: userID,
				},
				OptionalActions: caps,
				OptionalMembers: members,
			})

			for _, cap := range caps {
				prs = append(prs, policies.Policy{
					SubjectType:     policies.RoleType,
					SubjectRelation: policies.MemberRelation,
					Subject:         id,
					Relation:        cap,
					Object:          entityID,
					ObjectType:      r.entityType,
				})
			}

			for _, member := range members {
				prs = append(prs, policies.Policy{
					SubjectType: policies.UserType,
					Subject:     policies.EncodeDomainUserID(domainID, member),
					Relation:    policies.MemberRelation,
					Object:      id,
					ObjectType:  policies.RoleType,
				})
			}
		}
	}
	prs = append(prs, optionalEntityPolicies...)

	if len(prs) > 0 {
		if err := r.policy.AddPolicies(ctx, prs); err != nil {
			return []RoleProvision{}, errors.Wrap(svcerr.ErrCreateEntity, err)
		}
		defer func() {
			if retErr != nil {
				if errRollBack := r.policy.DeletePolicies(ctx, prs); errRollBack != nil {
					retErr = errors.Wrap(retErr, errors.Wrap(errRollbackRoles, errRollBack))
				}
			}
		}()
	}

	nprs, err := r.repo.AddRoles(ctx, newRolesProvision)
	if err != nil {
		return []RoleProvision{}, errors.Wrap(svcerr.ErrCreateEntity, err)
	}

	return nprs, nil
}

func (r ProvisionManageService) AddRole(ctx context.Context, session authn.Session, entityID string, roleName string, optionalActions []string, optionalMembers []string) (retRoleProvision RoleProvision, retErr error) {
	sid, err := r.sidProvider.ID()
	if err != nil {
		return RoleProvision{}, errors.Wrap(svcerr.ErrCreateEntity, err)
	}

	id := r.entityType + "_" + sid

	if err := r.validateActions(toRolesActions(optionalActions)); err != nil {
		return RoleProvision{}, errors.Wrap(svcerr.ErrMalformedEntity, err)
	}

	newRoleProvisions := []RoleProvision{
		{
			Role: Role{
				ID:        id,
				Name:      roleName,
				EntityID:  entityID,
				CreatedAt: time.Now(),
				CreatedBy: session.UserID,
			},
			OptionalActions: optionalActions,
			OptionalMembers: optionalMembers,
		},
	}
	prs := []policies.Policy{}

	for _, cap := range optionalActions {
		prs = append(prs, policies.Policy{
			SubjectType:     policies.RoleType,
			SubjectRelation: policies.MemberRelation,
			Subject:         id,
			Relation:        cap,
			Object:          entityID,
			ObjectType:      r.entityType,
		})
	}

	for _, member := range optionalMembers {
		prs = append(prs, policies.Policy{
			SubjectType: policies.UserType,
			Subject:     policies.EncodeDomainUserID(session.DomainID, member),
			Relation:    policies.MemberRelation,
			Object:      id,
			ObjectType:  policies.RoleType,
		})
	}

	if len(prs) > 0 {
		if err := r.policy.AddPolicies(ctx, prs); err != nil {
			return RoleProvision{}, errors.Wrap(svcerr.ErrCreateEntity, err)
		}

		defer func() {
			if retErr != nil {
				if errRollBack := r.policy.DeletePolicies(ctx, prs); errRollBack != nil {
					retErr = errors.Wrap(retErr, errors.Wrap(errRollbackRoles, errRollBack))
				}
			}
		}()
	}

	nrps, err := r.repo.AddRoles(ctx, newRoleProvisions)
	if err != nil {
		return RoleProvision{}, errors.Wrap(svcerr.ErrCreateEntity, err)
	}

	if len(nrps) == 0 {
		return RoleProvision{}, svcerr.ErrCreateEntity
	}

	return nrps[0], nil
}

func (r ProvisionManageService) RemoveRole(ctx context.Context, session authn.Session, entityID, roleID string) error {
	ro, err := r.repo.RetrieveEntityRole(ctx, entityID, roleID)
	if err != nil {
		return errors.Wrap(svcerr.ErrRemoveEntity, err)
	}
	req := policies.Policy{
		SubjectType: policies.RoleType,
		Subject:     ro.ID,
	}
	if err := r.policy.DeletePolicyFilter(ctx, req); err != nil {
		return errors.Wrap(svcerr.ErrRemoveEntity, err)
	}

	if err := r.repo.RemoveRoles(ctx, []string{ro.ID}); err != nil {
		return errors.Wrap(svcerr.ErrRemoveEntity, err)
	}
	return nil
}

func (r ProvisionManageService) UpdateRoleName(ctx context.Context, session authn.Session, entityID, roleID, newRoleName string) (Role, error) {
	ro, err := r.repo.RetrieveEntityRole(ctx, entityID, roleID)
	if err != nil {
		return Role{}, errors.Wrap(svcerr.ErrUpdateEntity, err)
	}
	ro, err = r.repo.UpdateRole(ctx, Role{
		ID:        ro.ID,
		EntityID:  entityID,
		Name:      newRoleName,
		UpdatedBy: session.UserID,
		UpdatedAt: time.Now(),
	})
	if err != nil {
		return Role{}, errors.Wrap(svcerr.ErrUpdateEntity, err)
	}
	return ro, nil
}

func (r ProvisionManageService) RetrieveRole(ctx context.Context, session authn.Session, entityID, roleID string) (Role, error) {
	ro, err := r.repo.RetrieveEntityRole(ctx, entityID, roleID)
	if err != nil {
		return Role{}, errors.Wrap(svcerr.ErrViewEntity, err)
	}
	return ro, nil
}

func (r ProvisionManageService) RetrieveAllRoles(ctx context.Context, session authn.Session, entityID string, limit, offset uint64) (RolePage, error) {
	ros, err := r.repo.RetrieveAllRoles(ctx, entityID, limit, offset)
	if err != nil {
		return RolePage{}, errors.Wrap(svcerr.ErrViewEntity, err)
	}
	return ros, nil
}

func (r ProvisionManageService) ListAvailableActions(ctx context.Context, session authn.Session) ([]string, error) {
	acts := []string{}
	for _, a := range r.actions {
		acts = append(acts, string(a))
	}
	return acts, nil
}

func (r ProvisionManageService) RoleAddActions(ctx context.Context, session authn.Session, entityID, roleID string, actions []string) (retActs []string, retErr error) {
	ro, err := r.repo.RetrieveEntityRole(ctx, entityID, roleID)
	if err != nil {
		return []string{}, errors.Wrap(svcerr.ErrUpdateEntity, err)
	}

	if len(actions) == 0 {
		return []string{}, svcerr.ErrMalformedEntity
	}

	if err := r.validateActions(toRolesActions(actions)); err != nil {
		return []string{}, errors.Wrap(svcerr.ErrMalformedEntity, err)
	}

	prs := []policies.Policy{}
	for _, cap := range actions {
		prs = append(prs, policies.Policy{
			SubjectType:     policies.RoleType,
			SubjectRelation: policies.MemberRelation,
			Subject:         ro.ID,
			Relation:        cap,
			Object:          entityID,
			ObjectType:      r.entityType,
		})
	}

	if err := r.policy.AddPolicies(ctx, prs); err != nil {
		return []string{}, errors.Wrap(svcerr.ErrAddPolicies, err)
	}

	defer func() {
		if retErr != nil {
			if errRollBack := r.policy.DeletePolicies(ctx, prs); errRollBack != nil {
				retErr = errors.Wrap(retErr, errors.Wrap(errRollbackRoles, errRollBack))
			}
		}
	}()

	ro.UpdatedAt = time.Now()
	ro.UpdatedBy = session.UserID

	resActs, err := r.repo.RoleAddActions(ctx, ro, actions)
	if err != nil {
		return []string{}, errors.Wrap(svcerr.ErrCreateEntity, err)
	}
	return resActs, nil
}

func (r ProvisionManageService) RoleListActions(ctx context.Context, session authn.Session, entityID, roleID string) ([]string, error) {
	ro, err := r.repo.RetrieveEntityRole(ctx, entityID, roleID)
	if err != nil {
		return []string{}, errors.Wrap(svcerr.ErrViewEntity, err)
	}

	acts, err := r.repo.RoleListActions(ctx, ro.ID)
	if err != nil {
		return []string{}, errors.Wrap(svcerr.ErrViewEntity, err)
	}
	return acts, nil
}

func (r ProvisionManageService) RoleCheckActionsExists(ctx context.Context, session authn.Session, entityID, roleID string, actions []string) (bool, error) {
	ro, err := r.repo.RetrieveEntityRole(ctx, entityID, roleID)
	if err != nil {
		return false, errors.Wrap(svcerr.ErrViewEntity, err)
	}

	result, err := r.repo.RoleCheckActionsExists(ctx, ro.ID, actions)
	if err != nil {
		return true, errors.Wrap(svcerr.ErrViewEntity, err)
	}
	return result, nil
}

func (r ProvisionManageService) RoleRemoveActions(ctx context.Context, session authn.Session, entityID, roleID string, actions []string) (err error) {
	ro, err := r.repo.RetrieveEntityRole(ctx, entityID, roleID)
	if err != nil {
		return errors.Wrap(svcerr.ErrRemoveEntity, err)
	}

	if len(actions) == 0 {
		return svcerr.ErrMalformedEntity
	}

	prs := []policies.Policy{}
	for _, op := range actions {
		prs = append(prs, policies.Policy{
			SubjectType:     policies.RoleType,
			SubjectRelation: policies.MemberRelation,
			Subject:         ro.ID,
			Relation:        op,
			Object:          entityID,
			ObjectType:      r.entityType,
		})
	}

	if err := r.policy.DeletePolicies(ctx, prs); err != nil {
		return errors.Wrap(svcerr.ErrDeletePolicies, err)
	}
	ro.UpdatedAt = time.Now()
	ro.UpdatedBy = session.UserID
	if err := r.repo.RoleRemoveActions(ctx, ro, actions); err != nil {
		return errors.Wrap(svcerr.ErrRemoveEntity, err)
	}
	return nil
}

func (r ProvisionManageService) RoleRemoveAllActions(ctx context.Context, session authn.Session, entityID, roleID string) error {
	ro, err := r.repo.RetrieveEntityRole(ctx, entityID, roleID)
	if err != nil {
		return errors.Wrap(svcerr.ErrRemoveEntity, err)
	}

	prs := policies.Policy{
		SubjectType: policies.RoleType,
		Subject:     ro.ID,
	}

	if err := r.policy.DeletePolicyFilter(ctx, prs); err != nil {
		return errors.Wrap(svcerr.ErrDeletePolicies, err)
	}

	ro.UpdatedAt = time.Now()
	ro.UpdatedBy = session.UserID

	if err := r.repo.RoleRemoveAllActions(ctx, ro); err != nil {
		return errors.Wrap(svcerr.ErrRemoveEntity, err)
	}
	return nil
}

func (r ProvisionManageService) RoleAddMembers(ctx context.Context, session authn.Session, entityID, roleID string, members []string) (retMems []string, retErr error) {
	ro, err := r.repo.RetrieveEntityRole(ctx, entityID, roleID)
	if err != nil {
		return []string{}, errors.Wrap(svcerr.ErrUpdateEntity, err)
	}

	if len(members) == 0 {
		return []string{}, svcerr.ErrMalformedEntity
	}

	prs := []policies.Policy{}
	for _, mem := range members {
		prs = append(prs, policies.Policy{
			SubjectType: policies.UserType,
			Subject:     policies.EncodeDomainUserID(session.DomainID, mem),
			Relation:    policies.MemberRelation,
			Object:      ro.ID,
			ObjectType:  policies.RoleType,
		})
	}

	if err := r.policy.AddPolicies(ctx, prs); err != nil {
		return []string{}, errors.Wrap(svcerr.ErrAddPolicies, err)
	}

	defer func() {
		if retErr != nil {
			if errRollBack := r.policy.DeletePolicies(ctx, prs); errRollBack != nil {
				retErr = errors.Wrap(retErr, errors.Wrap(errRollbackRoles, errRollBack))
			}
		}
	}()

	ro.UpdatedAt = time.Now()
	ro.UpdatedBy = session.UserID

	mems, err := r.repo.RoleAddMembers(ctx, ro, members)
	if err != nil {
		return []string{}, errors.Wrap(svcerr.ErrCreateEntity, err)
	}
	return mems, nil
}

func (r ProvisionManageService) RoleListMembers(ctx context.Context, session authn.Session, entityID, roleID string, limit, offset uint64) (MembersPage, error) {
	ro, err := r.repo.RetrieveEntityRole(ctx, entityID, roleID)
	if err != nil {
		return MembersPage{}, errors.Wrap(svcerr.ErrViewEntity, err)
	}

	mp, err := r.repo.RoleListMembers(ctx, ro.ID, limit, offset)
	if err != nil {
		return MembersPage{}, errors.Wrap(svcerr.ErrViewEntity, err)
	}
	return mp, nil
}

func (r ProvisionManageService) RoleCheckMembersExists(ctx context.Context, session authn.Session, entityID, roleID string, members []string) (bool, error) {
	ro, err := r.repo.RetrieveEntityRole(ctx, entityID, roleID)
	if err != nil {
		return false, errors.Wrap(svcerr.ErrViewEntity, err)
	}

	result, err := r.repo.RoleCheckMembersExists(ctx, ro.ID, members)
	if err != nil {
		return true, errors.Wrap(svcerr.ErrViewEntity, err)
	}
	return result, nil
}

func (r ProvisionManageService) RoleRemoveMembers(ctx context.Context, session authn.Session, entityID, roleID string, members []string) (err error) {
	ro, err := r.repo.RetrieveEntityRole(ctx, entityID, roleID)
	if err != nil {
		return errors.Wrap(svcerr.ErrRemoveEntity, err)
	}

	if len(members) == 0 {
		return svcerr.ErrMalformedEntity
	}

	prs := []policies.Policy{}
	for _, mem := range members {
		prs = append(prs, policies.Policy{
			SubjectType: policies.UserType,
			Subject:     policies.EncodeDomainUserID(session.DomainID, mem),
			Relation:    policies.MemberRelation,
			Object:      ro.ID,
			ObjectType:  policies.RoleType,
		})
	}

	if err := r.policy.DeletePolicies(ctx, prs); err != nil {
		return errors.Wrap(svcerr.ErrDeletePolicies, err)
	}

	ro.UpdatedAt = time.Now()
	ro.UpdatedBy = session.UserID
	if err := r.repo.RoleRemoveMembers(ctx, ro, members); err != nil {
		return errors.Wrap(svcerr.ErrRemoveEntity, err)
	}
	return nil
}

func (r ProvisionManageService) RoleRemoveAllMembers(ctx context.Context, session authn.Session, entityID, roleID string) (err error) {
	ro, err := r.repo.RetrieveEntityRole(ctx, entityID, roleID)
	if err != nil {
		return errors.Wrap(svcerr.ErrRemoveEntity, err)
	}

	prs := policies.Policy{
		ObjectType:  policies.RoleType,
		Object:      ro.ID,
		SubjectType: policies.UserType,
	}

	if err := r.policy.DeletePolicyFilter(ctx, prs); err != nil {
		return errors.Wrap(svcerr.ErrDeletePolicies, err)
	}

	ro.UpdatedAt = time.Now()
	ro.UpdatedBy = session.UserID

	if err := r.repo.RoleRemoveAllMembers(ctx, ro); err != nil {
		return errors.Wrap(svcerr.ErrRemoveEntity, err)
	}
	return nil
}

func (r ProvisionManageService) ListEntityMembers(ctx context.Context, session authn.Session, entityID string, pageQuery MembersRolePageQuery) (MembersRolePage, error) {
	mp, err := r.repo.ListEntityMembers(ctx, entityID, pageQuery)
	if err != nil {
		return MembersRolePage{}, err
	}
	return mp, nil
}

func (r ProvisionManageService) RemoveEntityMembers(ctx context.Context, session authn.Session, entityID string, members []string) error {
	if err := r.repo.RemoveEntityMembers(ctx, entityID, members); err != nil {
		return err
	}
	return nil
}

func (r ProvisionManageService) RemoveMemberFromAllRoles(ctx context.Context, session authn.Session, member string) (err error) {
	if err := r.repo.RemoveMemberFromAllRoles(ctx, member); err != nil {
		return errors.Wrap(svcerr.ErrRemoveEntity, err)
	}

	prs := policies.Policy{
		ObjectType:   policies.RoleType,
		ObjectPrefix: r.entityType + "_",
		SubjectType:  policies.UserType,
	}

	if err := r.policy.DeletePolicyFilter(ctx, prs); err != nil {
		return errors.Wrap(svcerr.ErrDeletePolicies, err)
	}

	return fmt.Errorf("not implemented")
}
