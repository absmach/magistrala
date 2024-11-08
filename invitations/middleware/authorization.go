// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package middleware

import (
	"context"

	"github.com/absmach/magistrala/auth"
	"github.com/absmach/magistrala/invitations"
	"github.com/absmach/magistrala/pkg/authn"
	"github.com/absmach/magistrala/pkg/authz"
	"github.com/absmach/magistrala/pkg/errors"
	svcerr "github.com/absmach/magistrala/pkg/errors/service"
	"github.com/absmach/magistrala/pkg/policies"
)

// ErrMemberExist indicates that the user is already a member of the domain.
var ErrMemberExist = errors.New("user is already a member of the domain")

var _ invitations.Service = (*tracing)(nil)

type authorizationMiddleware struct {
	authz authz.Authorization
	svc   invitations.Service
}

func AuthorizationMiddleware(authz authz.Authorization, svc invitations.Service) invitations.Service {
	return &authorizationMiddleware{authz, svc}
}

func (am *authorizationMiddleware) SendInvitation(ctx context.Context, session authn.Session, invitation invitations.Invitation) (err error) {
	if err := am.checkAdmin(ctx, session.UserID, session.DomainID); err != nil {
		return err
	}
	session.DomainUserID = auth.EncodeDomainUserID(session.DomainID, session.UserID)
	domainUserId := auth.EncodeDomainUserID(invitation.DomainID, invitation.UserID)
	if err := am.authorize(ctx, domainUserId, policies.MembershipPermission, policies.DomainType, invitation.DomainID); err == nil {
		// return error if the user is already a member of the domain
		return errors.Wrap(svcerr.ErrConflict, ErrMemberExist)
	}

	if err := am.checkAdmin(ctx, session.DomainUserID, invitation.DomainID); err != nil {
		return err
	}

	return am.svc.SendInvitation(ctx, session, invitation)
}

func (am *authorizationMiddleware) ViewInvitation(ctx context.Context, session authn.Session, userID, domain string) (invitation invitations.Invitation, err error) {
	session.DomainUserID = auth.EncodeDomainUserID(session.DomainID, session.UserID)
	if session.UserID != userID {
		if err := am.checkAdmin(ctx, session.DomainUserID, domain); err != nil {
			return invitations.Invitation{}, err
		}
	}

	return am.svc.ViewInvitation(ctx, session, userID, domain)
}

func (am *authorizationMiddleware) ListInvitations(ctx context.Context, session authn.Session, page invitations.Page) (invs invitations.InvitationPage, err error) {
	session.DomainUserID = auth.EncodeDomainUserID(session.DomainID, session.UserID)
	if err := am.authorize(ctx, session.DomainUserID, policies.AdminPermission, policies.PlatformType, policies.MagistralaObject); err == nil {
		session.SuperAdmin = true
	}

	if !session.SuperAdmin {
		switch {
		case page.DomainID != "":
			if err := am.authorize(ctx, session.DomainUserID, policies.AdminPermission, policies.DomainType, page.DomainID); err != nil {
				return invitations.InvitationPage{}, err
			}
		default:
			page.InvitedByOrUserID = session.UserID
		}
	}

	return am.svc.ListInvitations(ctx, session, page)
}

func (am *authorizationMiddleware) AcceptInvitation(ctx context.Context, session authn.Session, domainID string) (err error) {
	return am.svc.AcceptInvitation(ctx, session, domainID)
}

func (am *authorizationMiddleware) RejectInvitation(ctx context.Context, session authn.Session, domainID string) (err error) {
	return am.svc.RejectInvitation(ctx, session, domainID)
}

func (am *authorizationMiddleware) DeleteInvitation(ctx context.Context, session authn.Session, userID, domainID string) (err error) {
	session.DomainUserID = auth.EncodeDomainUserID(session.DomainID, session.UserID)
	if err := am.checkAdmin(ctx, session.DomainUserID, domainID); err != nil {
		return err
	}

	return am.svc.DeleteInvitation(ctx, session, userID, domainID)
}

// checkAdmin checks if the given user is a domain or platform administrator.
func (am *authorizationMiddleware) checkAdmin(ctx context.Context, userID, domainID string) error {
	if err := am.authorize(ctx, userID, policies.AdminPermission, policies.DomainType, domainID); err == nil {
		return nil
	}

	if err := am.authorize(ctx, userID, policies.AdminPermission, policies.PlatformType, policies.MagistralaObject); err == nil {
		return nil
	}

	return svcerr.ErrAuthorization
}

func (am *authorizationMiddleware) authorize(ctx context.Context, subj, perm, objType, obj string) error {
	req := authz.PolicyReq{
		SubjectType: policies.UserType,
		SubjectKind: policies.UsersKind,
		Subject:     subj,
		Permission:  perm,
		ObjectType:  objType,
		Object:      obj,
	}
	if err := am.authz.Authorize(ctx, req); err != nil {
		return err
	}

	return nil
}
