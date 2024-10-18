package entityroles

import (
	"context"
	"fmt"
	"time"

	"github.com/absmach/magistrala"
	"github.com/absmach/magistrala/pkg/authn"
	"github.com/absmach/magistrala/pkg/errors"
	svcerr "github.com/absmach/magistrala/pkg/errors/service"
	"github.com/absmach/magistrala/pkg/policies"
	"github.com/absmach/magistrala/pkg/roles"
)

var errRollbackRoles = errors.New("failed to rollback roles")

var _ roles.Roles = (*RolesSvc)(nil)

type RolesSvc struct {
	entityType   string
	repo         roles.Repository
	idProvider   magistrala.IDProvider
	policy       policies.Service
	actions      []roles.Action
	builtInRoles map[roles.BuiltInRoleName][]roles.Action
}

func NewRolesSvc(entityType string, repo roles.Repository, idProvider magistrala.IDProvider, policyClient policies.Service, actions []roles.Action, builtInRoles map[roles.BuiltInRoleName][]roles.Action) (RolesSvc, error) {
	rolesSvc := RolesSvc{
		entityType:   entityType,
		repo:         repo,
		idProvider:   idProvider,
		policy:       policyClient,
		actions:      actions,
		builtInRoles: builtInRoles,
	}
	return rolesSvc, nil
}

func toRolesActions(actions []string) []roles.Action {
	roActions := []roles.Action{}
	for _, action := range actions {
		roActions = append(roActions, roles.Action(action))
	}
	return roActions
}

func roleActionsToString(roActions []roles.Action) []string {
	actions := []string{}
	for _, roAction := range roActions {
		actions = append(actions, roAction.String())
	}
	return actions
}

func roleMembersToString(roMems []roles.Member) []string {
	mems := []string{}
	for _, roMem := range roMems {
		mems = append(mems, roMem.String())
	}
	return mems
}

func (r RolesSvc) isActionAllowed(action roles.Action) bool {
	for _, cap := range r.actions {
		if cap == action {
			return true
		}
	}
	return false
}
func (r RolesSvc) validateActions(actions []roles.Action) error {
	for _, ac := range actions {
		action := roles.Action(ac)
		if !r.isActionAllowed(action) {
			return errors.Wrap(svcerr.ErrMalformedEntity, fmt.Errorf("invalid action %s ", action))
		}
	}
	return nil
}

func (r RolesSvc) AddRole(ctx context.Context, session authn.Session, entityID string, roleName string, optionalActions []string, optionalMembers []string) (fnRole roles.Role, fnErr error) {

	// ToDo: Research: Discuss: There an option to have id as entityID_roleName where in roleName all space are removed with _ and starts with letter and supports only alphanumeric, space and hyphen
	id, err := r.idProvider.ID()
	if err != nil {
		return roles.Role{}, errors.Wrap(svcerr.ErrCreateEntity, err)
	}

	if err := r.validateActions(toRolesActions(optionalActions)); err != nil {
		return roles.Role{}, errors.Wrap(svcerr.ErrMalformedEntity, err)
	}

	newRoleProvisions := []roles.RoleProvision{
		{
			Role: roles.Role{
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
			SubjectType:     "role",
			SubjectRelation: "member",
			Subject:         id,
			Relation:        cap,
			Object:          entityID,
			ObjectType:      r.entityType,
		})
	}

	for _, member := range optionalMembers {
		prs = append(prs, policies.Policy{
			SubjectType: "user",
			Subject:     policies.EncodeDomainUserID(session.DomainID, member),
			Relation:    "member",
			Object:      id,
			ObjectType:  "role",
		})
	}

	if len(prs) > 0 {
		if err := r.policy.AddPolicies(ctx, prs); err != nil {
			return roles.Role{}, errors.Wrap(svcerr.ErrCreateEntity, err)
		}

		defer func() {
			if fnErr != nil {
				if errRollBack := r.policy.DeletePolicies(ctx, prs); errRollBack != nil {
					fnErr = errors.Wrap(fnErr, errors.Wrap(errRollbackRoles, errRollBack))
				}
			}
		}()
	}

	newRoles, err := r.repo.AddRoles(context.Background(), newRoleProvisions)
	if err != nil {
		return roles.Role{}, errors.Wrap(svcerr.ErrCreateEntity, err)
	}

	if len(newRoles) == 0 {
		return roles.Role{}, svcerr.ErrCreateEntity
	}

	return newRoles[0], nil
}

func (r RolesSvc) RemoveRole(ctx context.Context, session authn.Session, entityID, roleName string) error {
	ro, err := r.repo.RetrieveRoleByEntityIDAndName(ctx, entityID, roleName)
	if err != nil {
		return errors.Wrap(svcerr.ErrRemoveEntity, err)
	}
	req := policies.Policy{
		SubjectType: "role",
		Subject:     ro.ID,
	}
	// ToDo: Add Role as Object in DeletePolicyFilter
	if err := r.policy.DeletePolicyFilter(ctx, req); err != nil {
		return errors.Wrap(svcerr.ErrRemoveEntity, err)
	}

	if err := r.repo.RemoveRoles(ctx, []string{ro.ID}); err != nil {
		return errors.Wrap(svcerr.ErrRemoveEntity, err)
	}
	return nil
}

func (r RolesSvc) UpdateRoleName(ctx context.Context, session authn.Session, entityID, oldRoleName, newRoleName string) (roles.Role, error) {
	ro, err := r.repo.RetrieveRoleByEntityIDAndName(ctx, entityID, oldRoleName)
	if err != nil {
		return roles.Role{}, errors.Wrap(svcerr.ErrUpdateEntity, err)
	}
	ro, err = r.repo.UpdateRole(ctx, roles.Role{
		ID:        ro.ID,
		EntityID:  entityID,
		Name:      newRoleName,
		UpdatedBy: session.UserID,
		UpdatedAt: time.Now(),
	})
	if err != nil {
		return roles.Role{}, errors.Wrap(svcerr.ErrUpdateEntity, err)
	}
	return ro, nil
}

func (r RolesSvc) RetrieveRole(ctx context.Context, session authn.Session, entityID, roleName string) (roles.Role, error) {
	ro, err := r.repo.RetrieveRoleByEntityIDAndName(ctx, entityID, roleName)
	if err != nil {
		return roles.Role{}, errors.Wrap(svcerr.ErrViewEntity, err)
	}
	return ro, nil
}

func (r RolesSvc) RetrieveAllRoles(ctx context.Context, session authn.Session, entityID string, limit, offset uint64) (roles.RolePage, error) {
	ros, err := r.repo.RetrieveAllRoles(ctx, entityID, limit, offset)
	if err != nil {
		return roles.RolePage{}, errors.Wrap(svcerr.ErrViewEntity, err)
	}
	return ros, nil
}

func (r RolesSvc) ListAvailableActions(ctx context.Context, session authn.Session) ([]string, error) {
	acts := []string{}
	for _, a := range r.actions {
		acts = append(acts, string(a))
	}
	return acts, nil
}

func (r RolesSvc) RoleAddActions(ctx context.Context, session authn.Session, entityID, roleName string, actions []string) (fnActs []string, fnErr error) {
	ro, err := r.repo.RetrieveRoleByEntityIDAndName(ctx, entityID, roleName)
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
			SubjectType:     "role",
			SubjectRelation: "member",
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
		if fnErr != nil {
			if errRollBack := r.policy.DeletePolicies(ctx, prs); errRollBack != nil {
				fnErr = errors.Wrap(fnErr, errors.Wrap(errRollbackRoles, errRollBack))
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

func (r RolesSvc) RoleListActions(ctx context.Context, session authn.Session, entityID, roleName string) ([]string, error) {
	ro, err := r.repo.RetrieveRoleByEntityIDAndName(ctx, entityID, roleName)
	if err != nil {
		return []string{}, errors.Wrap(svcerr.ErrViewEntity, err)
	}

	acts, err := r.repo.RoleListActions(ctx, ro.ID)
	if err != nil {
		return []string{}, errors.Wrap(svcerr.ErrViewEntity, err)
	}
	return acts, nil

}

func (r RolesSvc) RoleCheckActionsExists(ctx context.Context, session authn.Session, entityID, roleName string, actions []string) (bool, error) {
	ro, err := r.repo.RetrieveRoleByEntityIDAndName(ctx, entityID, roleName)
	if err != nil {
		return false, errors.Wrap(svcerr.ErrViewEntity, err)
	}

	result, err := r.repo.RoleCheckActionsExists(ctx, ro.ID, actions)
	if err != nil {
		return true, errors.Wrap(svcerr.ErrViewEntity, err)
	}
	return result, nil
}

func (r RolesSvc) RoleRemoveActions(ctx context.Context, session authn.Session, entityID, roleName string, actions []string) (err error) {
	ro, err := r.repo.RetrieveRoleByEntityIDAndName(ctx, entityID, roleName)
	if err != nil {
		return errors.Wrap(svcerr.ErrRemoveEntity, err)
	}

	if len(actions) == 0 {
		return svcerr.ErrMalformedEntity
	}

	prs := []policies.Policy{}
	for _, op := range actions {
		prs = append(prs, policies.Policy{
			SubjectType:     "role",
			SubjectRelation: "member",
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

func (r RolesSvc) RoleRemoveAllActions(ctx context.Context, session authn.Session, entityID, roleName string) error {
	ro, err := r.repo.RetrieveRoleByEntityIDAndName(ctx, entityID, roleName)
	if err != nil {
		return errors.Wrap(svcerr.ErrRemoveEntity, err)
	}

	prs := policies.Policy{
		SubjectType: "role",
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

func (r RolesSvc) RoleAddMembers(ctx context.Context, session authn.Session, entityID, roleName string, members []string) (fnMems []string, fnErr error) {
	ro, err := r.repo.RetrieveRoleByEntityIDAndName(ctx, entityID, roleName)
	if err != nil {
		return []string{}, errors.Wrap(svcerr.ErrUpdateEntity, err)
	}

	if len(members) == 0 {
		return []string{}, svcerr.ErrMalformedEntity
	}

	prs := []policies.Policy{}
	for _, mem := range members {
		prs = append(prs, policies.Policy{
			SubjectType: "user",
			Subject:     policies.EncodeDomainUserID(session.DomainID, mem),
			Relation:    "member",
			Object:      ro.ID,
			ObjectType:  "role",
		})
	}

	if err := r.policy.AddPolicies(ctx, prs); err != nil {
		return []string{}, errors.Wrap(svcerr.ErrAddPolicies, err)
	}

	defer func() {
		if fnErr != nil {
			if errRollBack := r.policy.DeletePolicies(ctx, prs); errRollBack != nil {
				fnErr = errors.Wrap(fnErr, errors.Wrap(errRollbackRoles, errRollBack))
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

func (r RolesSvc) RoleListMembers(ctx context.Context, session authn.Session, entityID, roleName string, limit, offset uint64) (roles.MembersPage, error) {
	ro, err := r.repo.RetrieveRoleByEntityIDAndName(ctx, entityID, roleName)
	if err != nil {
		return roles.MembersPage{}, errors.Wrap(svcerr.ErrViewEntity, err)
	}

	mp, err := r.repo.RoleListMembers(ctx, ro.ID, limit, offset)
	if err != nil {
		return roles.MembersPage{}, errors.Wrap(svcerr.ErrViewEntity, err)
	}
	return mp, nil
}

func (r RolesSvc) RoleCheckMembersExists(ctx context.Context, session authn.Session, entityID, roleName string, members []string) (bool, error) {
	ro, err := r.repo.RetrieveRoleByEntityIDAndName(ctx, entityID, roleName)
	if err != nil {
		return false, errors.Wrap(svcerr.ErrViewEntity, err)
	}

	result, err := r.repo.RoleCheckMembersExists(ctx, ro.ID, members)
	if err != nil {
		return true, errors.Wrap(svcerr.ErrViewEntity, err)
	}
	return result, nil
}

func (r RolesSvc) RoleRemoveMembers(ctx context.Context, session authn.Session, entityID, roleName string, members []string) (err error) {
	ro, err := r.repo.RetrieveRoleByEntityIDAndName(ctx, entityID, roleName)
	if err != nil {
		return errors.Wrap(svcerr.ErrRemoveEntity, err)
	}

	if len(members) == 0 {
		return svcerr.ErrMalformedEntity
	}

	prs := []policies.Policy{}
	for _, mem := range members {
		prs = append(prs, policies.Policy{
			SubjectType: "user",
			Subject:     policies.EncodeDomainUserID(session.DomainID, mem),
			Relation:    "member",
			Object:      ro.ID,
			ObjectType:  "role",
		})
	}

	if err := r.policy.DeletePolicies(ctx, prs); err != nil {
		return errors.Wrap(svcerr.ErrDeletePolicies, err)
	}

	ro.UpdatedAt = time.Now()
	// ro.UpdatedBy = userID
	if err := r.repo.RoleRemoveMembers(ctx, ro, members); err != nil {
		return errors.Wrap(svcerr.ErrRemoveEntity, err)
	}
	return nil
}

func (r RolesSvc) RoleRemoveAllMembers(ctx context.Context, session authn.Session, entityID, roleName string) (err error) {
	ro, err := r.repo.RetrieveRoleByEntityIDAndName(ctx, entityID, roleName)
	if err != nil {
		return errors.Wrap(svcerr.ErrRemoveEntity, err)
	}

	prs := policies.Policy{
		ObjectType:  "role",
		Object:      ro.ID,
		SubjectType: "user",
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

func (r RolesSvc) RemoveMembersFromAllRoles(ctx context.Context, session authn.Session, members []string) (err error) {
	return fmt.Errorf("not implemented")
}
func (r RolesSvc) RemoveMembersFromRoles(ctx context.Context, session authn.Session, members []string, roleNames []string) (err error) {
	return fmt.Errorf("not implemented")
}
func (r RolesSvc) RemoveActionsFromAllRoles(ctx context.Context, session authn.Session, actions []string) (err error) {
	return fmt.Errorf("not implemented")
}
func (r RolesSvc) RemoveActionsFromRoles(ctx context.Context, session authn.Session, actions []string, roleNames []string) (err error) {
	return fmt.Errorf("not implemented")
}

func (r RolesSvc) AddNewEntityRoles(ctx context.Context, userID, domainID, entityID string, newBuiltInRoleMembers map[roles.BuiltInRoleName][]roles.Member, optionalPolicies []roles.OptionalPolicy) (fnRolesProvision []roles.RoleProvision, fnErr error) {
	var newRolesProvision []roles.RoleProvision
	prs := []policies.Policy{}

	for defaultRole, defaultRoleMembers := range newBuiltInRoleMembers {
		actions, ok := r.builtInRoles[defaultRole]
		if !ok {
			return []roles.RoleProvision{}, fmt.Errorf("default role %s not found in in-built roles", defaultRole)
		}

		// There an option to have id as entityID_roleName where in roleName all space are removed with _ and starts with letter and supports only alphanumeric, space and hyphen
		id, err := r.idProvider.ID()
		if err != nil {
			return []roles.RoleProvision{}, errors.Wrap(svcerr.ErrCreateEntity, err)
		}

		if err := r.validateActions(actions); err != nil {
			return []roles.RoleProvision{}, errors.Wrap(svcerr.ErrMalformedEntity, err)
		}

		members := roleMembersToString(defaultRoleMembers)
		caps := roleActionsToString(actions)

		newRolesProvision = append(newRolesProvision, roles.RoleProvision{
			Role: roles.Role{
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
				SubjectType:     "role",
				SubjectRelation: "member",
				Subject:         id,
				Relation:        cap,
				Object:          entityID,
				ObjectType:      r.entityType,
			})
		}

		for _, member := range members {
			prs = append(prs, policies.Policy{
				SubjectType: "user",
				Subject:     policies.EncodeDomainUserID(domainID, member),
				Relation:    "member",
				Object:      id,
				ObjectType:  "role",
			})
		}

	}

	if len(optionalPolicies) > 0 {
		for _, policy := range optionalPolicies {
			prs = append(prs, policies.Policy{
				Domain:          policy.Namespace,
				SubjectType:     policy.SubjectType,
				SubjectRelation: policy.SubjectRelation,
				Subject:         policy.Subject,
				Relation:        policy.Relation,
				Permission:      policy.Permission,
				Object:          policy.Object,
				ObjectType:      policy.ObjectType,
			})
		}
	}

	if len(prs) > 0 {
		if err := r.policy.AddPolicies(ctx, prs); err != nil {
			return []roles.RoleProvision{}, errors.Wrap(svcerr.ErrCreateEntity, err)
		}
		defer func() {
			if fnErr != nil {
				if errRollBack := r.policy.DeletePolicies(ctx, prs); errRollBack != nil {
					fnErr = errors.Wrap(fnErr, errors.Wrap(errRollbackRoles, errRollBack))
				}
			}
		}()
	}

	if _, err := r.repo.AddRoles(ctx, newRolesProvision); err != nil {
		return []roles.RoleProvision{}, errors.Wrap(svcerr.ErrCreateEntity, err)
	}

	return newRolesProvision, nil
}

func (r RolesSvc) RemoveNewEntityRoles(ctx context.Context, userID, domainID, entityID string, optionalPolicies []roles.OptionalPolicy, newRolesProvision []roles.RoleProvision) error {
	prs := []policies.Policy{}

	roleIDs := []string{}
	for _, rp := range newRolesProvision {
		for _, cap := range rp.OptionalActions {
			prs = append(prs, policies.Policy{
				SubjectType:     "role",
				SubjectRelation: "member",
				Subject:         rp.ID,
				Relation:        cap,
				Object:          entityID,
				ObjectType:      r.entityType,
			})
		}

		for _, member := range rp.OptionalMembers {
			prs = append(prs, policies.Policy{
				SubjectType: "user",
				Subject:     policies.EncodeDomainUserID(domainID, member),
				Relation:    "member",
				Object:      rp.ID,
				ObjectType:  "role",
			})
		}
		roleIDs = append(roleIDs, rp.ID)
	}

	if len(optionalPolicies) > 0 {
		for _, policy := range optionalPolicies {
			prs = append(prs, policies.Policy{
				Domain:          policy.Namespace,
				SubjectType:     policy.SubjectType,
				SubjectRelation: policy.SubjectRelation,
				Subject:         policy.Subject,
				Relation:        policy.Relation,
				Permission:      policy.Permission,
				Object:          policy.Object,
				ObjectType:      policy.ObjectType,
			})
		}
	}

	if len(prs) > 0 {
		if err := r.policy.DeletePolicies(ctx, prs); err != nil {
			return errors.Wrap(svcerr.ErrDeletePolicies, err)
		}
	}

	if err := r.repo.RemoveRoles(ctx, roleIDs); err != nil {
		return errors.Wrap(svcerr.ErrRemoveEntity, err)
	}

	return nil
}
