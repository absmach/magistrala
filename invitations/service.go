// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package invitations

import (
	"context"
	"time"

	"github.com/absmach/magistrala"
	"github.com/absmach/magistrala/auth"
	mgauthn "github.com/absmach/magistrala/pkg/authn"
	mgauthz "github.com/absmach/magistrala/pkg/authz"
	"github.com/absmach/magistrala/pkg/errors"
	svcerr "github.com/absmach/magistrala/pkg/errors/service"
	"github.com/absmach/magistrala/pkg/policies"
	mgsdk "github.com/absmach/magistrala/pkg/sdk/go"
)

type service struct {
	authn mgauthn.Authentication
	authz mgauthz.Authorization
	token magistrala.TokenServiceClient
	repo  Repository
	sdk   mgsdk.SDK
}

// ErrMemberExist indicates that the user is already a member of the domain.
var ErrMemberExist = errors.New("user is already a member of the domain")

func NewService(authn mgauthn.Authentication, authz mgauthz.Authorization, token magistrala.TokenServiceClient, repo Repository, sdk mgsdk.SDK) Service {
	return &service{
		authn: authn,
		authz: authz,
		token: token,
		repo:  repo,
		sdk:   sdk,
	}
}

func (svc *service) SendInvitation(ctx context.Context, token string, invitation Invitation) error {
	if err := CheckRelation(invitation.Relation); err != nil {
		return err
	}

	session, err := svc.authn.Authenticate(ctx, token)
	if err != nil {
		return err
	}
	invitation.InvitedBy = session.UserID

	domainUserId := auth.EncodeDomainUserID(invitation.DomainID, invitation.UserID)
	if err := svc.authorize(ctx, domainUserId, policies.MembershipPermission, policies.DomainType, invitation.DomainID); err == nil {
		// return error if the user is already a member of the domain
		return errors.Wrap(svcerr.ErrConflict, ErrMemberExist)
	}

	if err := svc.checkAdmin(ctx, session.DomainUserID, invitation.DomainID); err != nil {
		return err
	}

	joinToken, err := svc.token.Issue(ctx, &magistrala.IssueReq{UserId: session.UserID, DomainId: &invitation.DomainID, Type: uint32(auth.InvitationKey)})
	if err != nil {
		return err
	}
	invitation.Token = joinToken.GetAccessToken()

	if invitation.Resend {
		invitation.UpdatedAt = time.Now()

		return svc.repo.UpdateToken(ctx, invitation)
	}

	invitation.CreatedAt = time.Now()

	return svc.repo.Create(ctx, invitation)
}

func (svc *service) ViewInvitation(ctx context.Context, token, userID, domainID string) (invitation Invitation, err error) {
	session, err := svc.authn.Authenticate(ctx, token)
	if err != nil {
		return Invitation{}, err
	}
	inv, err := svc.repo.Retrieve(ctx, userID, domainID)
	if err != nil {
		return Invitation{}, err
	}
	inv.Token = ""

	if session.UserID == userID {
		return inv, nil
	}

	if inv.InvitedBy == session.UserID {
		return inv, nil
	}

	if err := svc.checkAdmin(ctx, session.DomainUserID, domainID); err != nil {
		return Invitation{}, err
	}

	return inv, nil
}

func (svc *service) ListInvitations(ctx context.Context, token string, page Page) (invitations InvitationPage, err error) {
	session, err := svc.authn.Authenticate(ctx, token)
	if err != nil {
		return InvitationPage{}, err
	}

	if err := svc.authorize(ctx, session.DomainUserID, policies.AdminPermission, policies.PlatformType, policies.MagistralaObject); err == nil {
		return svc.repo.RetrieveAll(ctx, page)
	}

	if page.DomainID != "" {
		if err := svc.checkAdmin(ctx, session.DomainUserID, page.DomainID); err != nil {
			return InvitationPage{}, err
		}

		return svc.repo.RetrieveAll(ctx, page)
	}

	page.InvitedByOrUserID = session.UserID

	return svc.repo.RetrieveAll(ctx, page)
}

func (svc *service) AcceptInvitation(ctx context.Context, token, domainID string) error {
	session, err := svc.authn.Authenticate(ctx, token)
	if err != nil {
		return err
	}

	inv, err := svc.repo.Retrieve(ctx, session.UserID, domainID)
	if err != nil {
		return err
	}

	if inv.UserID != session.UserID {
		return svcerr.ErrAuthorization
	}

	if !inv.ConfirmedAt.IsZero() {
		return svcerr.ErrInvitationAlreadyAccepted
	}

	if !inv.RejectedAt.IsZero() {
		return svcerr.ErrInvitationAlreadyRejected
	}

	req := mgsdk.UsersRelationRequest{
		Relation: inv.Relation,
		UserIDs:  []string{session.UserID},
	}
	if sdkerr := svc.sdk.AddUserToDomain(inv.DomainID, req, inv.Token); sdkerr != nil {
		return sdkerr
	}

	inv.ConfirmedAt = time.Now()
	inv.UpdatedAt = inv.ConfirmedAt
	return svc.repo.UpdateConfirmation(ctx, inv)
}

func (svc *service) RejectInvitation(ctx context.Context, token, domainID string) error {
	session, err := svc.authn.Authenticate(ctx, token)
	if err != nil {
		return err
	}

	inv, err := svc.repo.Retrieve(ctx, session.UserID, domainID)
	if err != nil {
		return err
	}

	if inv.UserID != session.UserID {
		return svcerr.ErrAuthorization
	}

	if !inv.ConfirmedAt.IsZero() {
		return svcerr.ErrInvitationAlreadyAccepted
	}

	if !inv.RejectedAt.IsZero() {
		return svcerr.ErrInvitationAlreadyRejected
	}

	inv.RejectedAt = time.Now()
	inv.UpdatedAt = inv.RejectedAt
	return svc.repo.UpdateRejection(ctx, inv)
}

func (svc *service) DeleteInvitation(ctx context.Context, token, userID, domainID string) error {
	session, err := svc.authn.Authenticate(ctx, token)
	if err != nil {
		return err
	}
	if session.UserID == userID {
		return svc.repo.Delete(ctx, userID, domainID)
	}

	inv, err := svc.repo.Retrieve(ctx, userID, domainID)
	if err != nil {
		return err
	}

	if inv.InvitedBy == session.UserID {
		return svc.repo.Delete(ctx, userID, domainID)
	}

	if err := svc.checkAdmin(ctx, session.DomainUserID, domainID); err != nil {
		return err
	}

	return svc.repo.Delete(ctx, userID, domainID)
}

func (svc *service) authorize(ctx context.Context, subj, perm, objType, obj string) error {
	req := mgauthz.PolicyReq{
		SubjectType: policies.UserType,
		SubjectKind: policies.UsersKind,
		Subject:     subj,
		Permission:  perm,
		ObjectType:  objType,
		Object:      obj,
	}
	if err := svc.authz.Authorize(ctx, req); err != nil {
		return err
	}

	return nil
}

// checkAdmin checks if the given user is a domain or platform administrator.
func (svc *service) checkAdmin(ctx context.Context, userID, domainID string) error {
	if err := svc.authorize(ctx, userID, policies.AdminPermission, policies.DomainType, domainID); err == nil {
		return nil
	}

	if err := svc.authorize(ctx, userID, policies.AdminPermission, policies.PlatformType, policies.MagistralaObject); err == nil {
		return nil
	}

	return svcerr.ErrAuthorization
}
