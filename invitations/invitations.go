// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package invitations

import (
	"context"
	"encoding/json"
	"time"

	"github.com/absmach/magistrala/pkg/apiutil"
	"github.com/absmach/magistrala/pkg/policies"
)

// Invitation is an invitation to join a domain.
type Invitation struct {
	InvitedBy   string    `json:"invited_by"`
	UserID      string    `json:"user_id"`
	DomainID    string    `json:"domain_id"`
	Token       string    `json:"token,omitempty"`
	Relation    string    `json:"relation,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at,omitempty"`
	ConfirmedAt time.Time `json:"confirmed_at,omitempty"`
	RejectedAt  time.Time `json:"rejected_at,omitempty"`
	Resend      bool      `json:"resend,omitempty"`
}

// Page is a page of invitations.
type Page struct {
	Offset            uint64 `json:"offset" db:"offset"`
	Limit             uint64 `json:"limit" db:"limit"`
	InvitedBy         string `json:"invited_by,omitempty" db:"invited_by,omitempty"`
	UserID            string `json:"user_id,omitempty" db:"user_id,omitempty"`
	DomainID          string `json:"domain_id,omitempty" db:"domain_id,omitempty"`
	Relation          string `json:"relation,omitempty" db:"relation,omitempty"`
	InvitedByOrUserID string `db:"invited_by_or_user_id,omitempty"`
	State             State  `json:"state,omitempty"`
}

// InvitationPage is a page of invitations.
type InvitationPage struct {
	Total       uint64       `json:"total"`
	Offset      uint64       `json:"offset"`
	Limit       uint64       `json:"limit"`
	Invitations []Invitation `json:"invitations"`
}

func (page InvitationPage) MarshalJSON() ([]byte, error) {
	type Alias InvitationPage
	a := struct {
		Alias
	}{
		Alias: Alias(page),
	}

	if a.Invitations == nil {
		a.Invitations = make([]Invitation, 0)
	}

	return json.Marshal(a)
}

// Service is an interface that defines methods for managing invitations.
//
//go:generate mockery --name Service --output=./mocks --filename service.go --quiet --note "Copyright (c) Abstract Machines"
type Service interface {
	// SendInvitation sends an invitation to the given user.
	// Only domain administrators and platform administrators can send invitations.
	SendInvitation(ctx context.Context, token string, invitation Invitation) (err error)

	// ViewInvitation returns an invitation.
	// People who can view invitations are:
	// - the invited user: they can view their own invitations
	// - the user who sent the invitation
	// - domain administrators
	// - platform administrators
	ViewInvitation(ctx context.Context, token, userID, domainID string) (invitation Invitation, err error)

	// ListInvitations returns a list of invitations.
	// People who can list invitations are:
	// - platform administrators can list all invitations
	// - domain administrators can list invitations for their domain
	// By default, it will list invitations the current user has sent or received.
	ListInvitations(ctx context.Context, token string, page Page) (invitations InvitationPage, err error)

	// AcceptInvitation accepts an invitation by adding the user to the domain.
	AcceptInvitation(ctx context.Context, token, domainID string) (err error)

	// DeleteInvitation deletes an invitation.
	// People who can delete invitations are:
	// - the invited user: they can delete their own invitations
	// - the user who sent the invitation
	// - domain administrators
	// - platform administrators
	DeleteInvitation(ctx context.Context, token, userID, domainID string) (err error)

	// RejectInvitation rejects an invitation.
	// People who can reject invitations are:
	// - the invited user: they can reject their own invitations
	RejectInvitation(ctx context.Context, token, domainID string) (err error)
}

//go:generate mockery --name Repository --output=./mocks --filename repository.go --quiet --note "Copyright (c) Abstract Machines"
type Repository interface {
	// Create creates an invitation.
	Create(ctx context.Context, invitation Invitation) (err error)

	// Retrieve returns an invitation.
	Retrieve(ctx context.Context, userID, domainID string) (Invitation, error)

	// RetrieveAll returns a list of invitations based on the given page.
	RetrieveAll(ctx context.Context, page Page) (invitations InvitationPage, err error)

	// UpdateToken updates an invitation by setting the token.
	UpdateToken(ctx context.Context, invitation Invitation) (err error)

	// UpdateConfirmation updates an invitation by setting the confirmation time.
	UpdateConfirmation(ctx context.Context, invitation Invitation) (err error)

	// UpdateRejection updates an invitation by setting the rejection time.
	UpdateRejection(ctx context.Context, invitation Invitation) (err error)

	// Delete deletes an invitation.
	Delete(ctx context.Context, userID, domainID string) (err error)
}

// CheckRelation checks if the given relation is valid.
// It returns an error if the relation is empty or invalid.
func CheckRelation(relation string) error {
	if relation == "" {
		return apiutil.ErrMissingRelation
	}
	if relation != policies.AdministratorRelation &&
		relation != policies.EditorRelation &&
		relation != policies.ContributorRelation &&
		relation != policies.MemberRelation &&
		relation != policies.GuestRelation &&
		relation != policies.DomainRelation &&
		relation != policies.ParentGroupRelation &&
		relation != policies.RoleGroupRelation &&
		relation != policies.GroupRelation &&
		relation != policies.PlatformRelation {
		return apiutil.ErrInvalidRelation
	}

	return nil
}
