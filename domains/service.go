// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package domains

import (
	"context"
	"time"

	"github.com/absmach/supermq"
	"github.com/absmach/supermq/pkg/authn"
	"github.com/absmach/supermq/pkg/errors"
	repoerr "github.com/absmach/supermq/pkg/errors/repository"
	svcerr "github.com/absmach/supermq/pkg/errors/service"
	"github.com/absmach/supermq/pkg/policies"
	"github.com/absmach/supermq/pkg/roles"
)

var (
	errCreateDomainPolicy = errors.New("failed to create domain policy")
	errRollbackRepo       = errors.New("failed to rollback repo")
)

type service struct {
	repo       Repository
	cache      Cache
	policy     policies.Service
	idProvider supermq.IDProvider
	roles.ProvisionManageService
}

var _ Service = (*service)(nil)

func New(repo Repository, cache Cache, policy policies.Service, idProvider supermq.IDProvider, sidProvider supermq.IDProvider, availableActions []roles.Action, builtInRoles map[roles.BuiltInRoleName][]roles.Action) (Service, error) {
	rpms, err := roles.NewProvisionManageService(policies.DomainType, repo, policy, sidProvider, availableActions, builtInRoles)
	if err != nil {
		return nil, err
	}

	return &service{
		repo:                   repo,
		cache:                  cache,
		policy:                 policy,
		idProvider:             idProvider,
		ProvisionManageService: rpms,
	}, nil
}

func (svc service) CreateDomain(ctx context.Context, session authn.Session, d Domain) (retDo Domain, retRps []roles.RoleProvision, retErr error) {
	d.CreatedBy = session.UserID

	if d.ID == "" {
		domainID, err := svc.idProvider.ID()
		if err != nil {
			return Domain{}, []roles.RoleProvision{}, errors.Wrap(svcerr.ErrCreateEntity, err)
		}
		d.ID = domainID
	}

	if d.Status != DisabledStatus && d.Status != EnabledStatus {
		return Domain{}, []roles.RoleProvision{}, svcerr.ErrInvalidStatus
	}

	d.CreatedAt = time.Now().UTC()

	// Domain is created in repo first, because Roles table have foreign key relation with Domain ID
	dom, err := svc.repo.SaveDomain(ctx, d)
	if err != nil {
		if errors.Contains(err, errors.ErrRouteNotAvailable) {
			return Domain{}, []roles.RoleProvision{}, errors.ErrRouteNotAvailable
		}
		return Domain{}, []roles.RoleProvision{}, errors.Wrap(svcerr.ErrCreateEntity, err)
	}
	defer func() {
		if retErr != nil {
			if errRollBack := svc.repo.DeleteDomain(ctx, d.ID); errRollBack != nil {
				retErr = errors.Wrap(retErr, errors.Wrap(errRollbackRepo, errRollBack))
			}
		}
	}()

	newBuiltInRoleMembers := map[roles.BuiltInRoleName][]roles.Member{
		BuiltInRoleAdmin: {roles.Member(session.UserID)},
	}

	optionalPolicies := []policies.Policy{
		{
			Subject:     policies.SuperMQObject,
			SubjectType: policies.PlatformType,
			Relation:    "organization",
			Object:      d.ID,
			ObjectType:  policies.DomainType,
		},
	}

	rps, err := svc.AddNewEntitiesRoles(ctx, d.ID, session.UserID, []string{d.ID}, optionalPolicies, newBuiltInRoleMembers)
	if err != nil {
		return Domain{}, []roles.RoleProvision{}, errors.Wrap(errCreateDomainPolicy, err)
	}

	return dom, rps, nil
}

func (svc service) RetrieveDomain(ctx context.Context, session authn.Session, id string, withRoles bool) (Domain, error) {
	var domain Domain
	var err error
	switch session.SuperAdmin {
	case true:
		domain, err = svc.repo.RetrieveDomainByID(ctx, id)
	default:
		switch withRoles {
		case true:
			domain, err = svc.repo.RetrieveDomainByIDWithRoles(ctx, id, session.UserID)
		default:
			domain, err = svc.repo.RetrieveDomainByID(ctx, id)
		}
	}
	if err != nil {
		return Domain{}, errors.Wrap(svcerr.ErrViewEntity, err)
	}
	return domain, nil
}

func (svc service) UpdateDomain(ctx context.Context, session authn.Session, id string, d DomainReq) (Domain, error) {
	updatedAt := time.Now().UTC()
	d.UpdatedAt = &updatedAt
	d.UpdatedBy = &session.UserID
	dom, err := svc.repo.UpdateDomain(ctx, id, d)
	if err != nil {
		return Domain{}, errors.Wrap(svcerr.ErrUpdateEntity, err)
	}
	return dom, nil
}

func (svc service) EnableDomain(ctx context.Context, session authn.Session, id string) (Domain, error) {
	status := EnabledStatus
	updatedAt := time.Now().UTC()
	dom, err := svc.repo.UpdateDomain(ctx, id, DomainReq{Status: &status, UpdatedBy: &session.UserID, UpdatedAt: &updatedAt})
	if err != nil {
		return Domain{}, errors.Wrap(svcerr.ErrUpdateEntity, err)
	}
	if err := svc.cache.RemoveStatus(ctx, id); err != nil {
		return dom, errors.Wrap(svcerr.ErrRemoveEntity, err)
	}

	return dom, nil
}

func (svc service) DisableDomain(ctx context.Context, session authn.Session, id string) (Domain, error) {
	status := DisabledStatus
	updatedAt := time.Now().UTC()
	dom, err := svc.repo.UpdateDomain(ctx, id, DomainReq{Status: &status, UpdatedBy: &session.UserID, UpdatedAt: &updatedAt})
	if err != nil {
		return Domain{}, errors.Wrap(svcerr.ErrUpdateEntity, err)
	}
	if err := svc.cache.RemoveStatus(ctx, id); err != nil {
		return dom, errors.Wrap(svcerr.ErrRemoveEntity, err)
	}

	return dom, nil
}

// Only SuperAdmin can freeze the domain.
func (svc service) FreezeDomain(ctx context.Context, session authn.Session, id string) (Domain, error) {
	status := FreezeStatus
	updatedAt := time.Now().UTC()
	dom, err := svc.repo.UpdateDomain(ctx, id, DomainReq{Status: &status, UpdatedBy: &session.UserID, UpdatedAt: &updatedAt})
	if err != nil {
		return Domain{}, errors.Wrap(svcerr.ErrUpdateEntity, err)
	}
	if err := svc.cache.RemoveStatus(ctx, id); err != nil {
		return dom, errors.Wrap(svcerr.ErrRemoveEntity, err)
	}

	return dom, nil
}

func (svc service) ListDomains(ctx context.Context, session authn.Session, p Page) (DomainsPage, error) {
	p.UserID = session.UserID
	if session.SuperAdmin {
		p.UserID = ""
	}

	dp, err := svc.repo.ListDomains(ctx, p)
	if err != nil {
		return DomainsPage{}, errors.Wrap(svcerr.ErrViewEntity, err)
	}
	return dp, nil
}

func (svc service) DeleteDomain(ctx context.Context, session authn.Session, id string) error {
	updatedAt := time.Now().UTC()
	updatedBy := session.UserID
	updatedStatus := DeletedStatus

	_, err := svc.repo.UpdateDomain(ctx, id, DomainReq{
		Status:    &updatedStatus,
		UpdatedAt: &updatedAt,
		UpdatedBy: &updatedBy,
	})
	if err != nil {
		return errors.Wrap(svcerr.ErrRemoveEntity, err)
	}

	return nil
}

func (svc *service) SendInvitation(ctx context.Context, session authn.Session, invitation Invitation) (Invitation, error) {
	role, err := svc.repo.RetrieveRole(ctx, invitation.RoleID)
	if err != nil {
		return Invitation{}, errors.Wrap(svcerr.ErrInvalidRole, err)
	}
	invitation.RoleName = role.Name

	// Retrieve domain to get domain name
	domain, err := svc.repo.RetrieveDomainByID(ctx, invitation.DomainID)
	if err != nil {
		return Invitation{}, errors.Wrap(svcerr.ErrViewEntity, err)
	}
	invitation.DomainName = domain.Name

	invitation.InvitedBy = session.UserID
	invitation.CreatedAt = time.Now().UTC()

	if invitation.Resend {
		if err := svc.resendInvitation(ctx, invitation); err != nil {
			return Invitation{}, err
		}
		return invitation, nil
	}

	if err := svc.repo.SaveInvitation(ctx, invitation); err != nil {
		return Invitation{}, errors.Wrap(svcerr.ErrCreateEntity, err)
	}
	return invitation, nil
}

func (svc *service) resendInvitation(ctx context.Context, invitation Invitation) error {
	inv, err := svc.repo.RetrieveInvitation(ctx, invitation.InviteeUserID, invitation.DomainID)
	if err != nil {
		return errors.Wrap(svcerr.ErrViewEntity, err)
	}
	if !inv.ConfirmedAt.IsZero() {
		return svcerr.ErrInvitationAlreadyAccepted
	}
	if !inv.RejectedAt.IsZero() {
		invitation.RejectedAt = time.Time{}
		invitation.UpdatedAt = time.Now().UTC()
		if err := svc.repo.UpdateRejection(ctx, invitation); err != nil {
			return errors.Wrap(svcerr.ErrUpdateEntity, err)
		}
	}

	return nil
}

func (svc *service) ListInvitations(ctx context.Context, session authn.Session, page InvitationPageMeta) (invitations InvitationPage, err error) {
	page.InviteeUserID = session.UserID
	ip, err := svc.repo.RetrieveAllInvitations(ctx, page)
	if err != nil {
		return InvitationPage{}, errors.Wrap(svcerr.ErrViewEntity, err)
	}
	return ip, nil
}

func (svc *service) ListDomainInvitations(ctx context.Context, session authn.Session, page InvitationPageMeta) (invitations InvitationPage, err error) {
	page.DomainID = session.DomainID
	ip, err := svc.repo.RetrieveAllInvitations(ctx, page)
	if err != nil {
		return InvitationPage{}, errors.Wrap(svcerr.ErrViewEntity, err)
	}
	return ip, nil
}

func (svc *service) AcceptInvitation(ctx context.Context, session authn.Session, domainID string) (invitation Invitation, err error) {
	inv, err := svc.repo.RetrieveInvitation(ctx, session.UserID, domainID)
	if err != nil {
		return Invitation{}, errors.Wrap(svcerr.ErrViewEntity, err)
	}

	if inv.InviteeUserID != session.UserID {
		return Invitation{}, svcerr.ErrAuthorization
	}

	if !inv.ConfirmedAt.IsZero() {
		return Invitation{}, svcerr.ErrInvitationAlreadyAccepted
	}

	if !inv.RejectedAt.IsZero() {
		return Invitation{}, svcerr.ErrInvitationAlreadyRejected
	}

	inv, err = svc.populateDetails(ctx, inv, domainID)
	if err != nil {
		return Invitation{}, err
	}

	session.DomainID = domainID

	if _, err := svc.RoleAddMembers(ctx, session, domainID, inv.RoleID, []string{session.UserID}); err != nil {
		return Invitation{}, errors.Wrap(svcerr.ErrUpdateEntity, err)
	}

	inv.ConfirmedAt = time.Now().UTC()
	inv.UpdatedAt = inv.ConfirmedAt

	if err := svc.repo.UpdateConfirmation(ctx, inv); err != nil {
		return Invitation{}, errors.Wrap(svcerr.ErrUpdateEntity, err)
	}

	return inv, nil
}

func (svc *service) RejectInvitation(ctx context.Context, session authn.Session, domainID string) (Invitation, error) {
	inv, err := svc.repo.RetrieveInvitation(ctx, session.UserID, domainID)
	if err != nil {
		return Invitation{}, errors.Wrap(svcerr.ErrViewEntity, err)
	}

	if inv.InviteeUserID != session.UserID {
		return Invitation{}, svcerr.ErrAuthorization
	}

	if !inv.ConfirmedAt.IsZero() {
		return Invitation{}, svcerr.ErrInvitationAlreadyAccepted
	}

	if !inv.RejectedAt.IsZero() {
		return Invitation{}, svcerr.ErrInvitationAlreadyRejected
	}

	inv, err = svc.populateDetails(ctx, inv, domainID)
	if err != nil {
		return Invitation{}, err
	}

	inv.RejectedAt = time.Now().UTC()
	inv.UpdatedAt = inv.RejectedAt

	if err := svc.repo.UpdateRejection(ctx, inv); err != nil {
		return Invitation{}, errors.Wrap(svcerr.ErrUpdateEntity, err)
	}

	return inv, nil
}

func (svc *service) DeleteInvitation(ctx context.Context, session authn.Session, inviteeUserID, domainID string) error {
	if session.UserID == inviteeUserID {
		if err := svc.repo.DeleteUsersInvitations(ctx, domainID, inviteeUserID); err != nil {
			return errors.Wrap(svcerr.ErrRemoveEntity, err)
		}
		return nil
	}

	inv, err := svc.repo.RetrieveInvitation(ctx, inviteeUserID, domainID)
	if err != nil {
		return errors.Wrap(svcerr.ErrRemoveEntity, err)
	}

	if !inv.ConfirmedAt.IsZero() {
		return errors.Wrap(svcerr.ErrRemoveEntity, svcerr.ErrInvitationAlreadyAccepted)
	}

	if !inv.RejectedAt.IsZero() {
		return errors.Wrap(svcerr.ErrRemoveEntity, svcerr.ErrInvitationAlreadyRejected)
	}

	if err := svc.repo.DeleteUsersInvitations(ctx, domainID, inviteeUserID); err != nil {
		return errors.Wrap(svcerr.ErrRemoveEntity, err)
	}

	return nil
}

// Add domain and role names for an invitation if they are not already set.
func (svc *service) populateDetails(ctx context.Context, inv Invitation, domainID string) (Invitation, error) {
	// Populate domain name if not already set
	if inv.DomainName == "" {
		domain, err := svc.repo.RetrieveDomainByID(ctx, domainID)
		if err != nil {
			return Invitation{}, errors.Wrap(svcerr.ErrViewEntity, err)
		}
		inv.DomainName = domain.Name
	}

	// Populate role name if not already set
	if inv.RoleName == "" {
		role, err := svc.repo.RetrieveRole(ctx, inv.RoleID)
		if err != nil {
			return Invitation{}, errors.Wrap(svcerr.ErrViewEntity, err)
		}
		inv.RoleName = role.Name
	}

	return inv, nil
}

// Add addition removal of user from invitations.
func (svc *service) RemoveEntityMembers(ctx context.Context, session authn.Session, entityID string, members []string) error {
	if err := svc.repo.DeleteUsersInvitations(ctx, entityID, members...); err != nil && err != repoerr.ErrNotFound {
		return err
	}

	return svc.ProvisionManageService.RemoveEntityMembers(ctx, session, entityID, members)
}

func (svc *service) RoleRemoveMembers(ctx context.Context, session authn.Session, entityID, roleID string, members []string) error {
	ro, err := svc.repo.RetrieveEntityRole(ctx, entityID, roleID)
	if err != nil {
		return errors.Wrap(svcerr.ErrViewEntity, err)
	}

	if _, err := svc.ProvisionManageService.BuiltInRoleActions(roles.BuiltInRoleName(ro.Name)); err == nil {
		membersPage, err := svc.repo.RoleListMembers(ctx, ro.ID, 0, 0)
		if err != nil {
			return errors.Wrap(svcerr.ErrViewEntity, err)
		}
		if membersPage.Total <= uint64(len(members)) {
			return svcerr.ErrRetainOneMember
		}
	}

	if err := svc.repo.DeleteUsersInvitations(ctx, entityID, members...); err != nil && err != repoerr.ErrNotFound {
		return err
	}

	return svc.ProvisionManageService.RoleRemoveMembers(ctx, session, entityID, roleID, members)
}

func (svc *service) RoleRemoveAllMembers(ctx context.Context, session authn.Session, entityID, roleID string) error {
	ro, err := svc.repo.RetrieveEntityRole(ctx, entityID, roleID)
	if err != nil {
		return errors.Wrap(svcerr.ErrViewEntity, err)
	}

	if _, err := svc.ProvisionManageService.BuiltInRoleActions(roles.BuiltInRoleName(ro.Name)); err == nil {
		return svcerr.ErrRetainOneMember
	}

	return svc.ProvisionManageService.RoleRemoveAllMembers(ctx, session, entityID, roleID)
}
