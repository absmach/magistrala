// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package domains

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/absmach/supermq/pkg/authn"
	svcerr "github.com/absmach/supermq/pkg/errors/service"
	"github.com/absmach/supermq/pkg/roles"
)

// Status represents Domain status.
type Status uint8

// Possible Domain status values.
const (
	// EnabledStatus represents enabled Domain.
	EnabledStatus Status = iota
	// DisabledStatus represents disabled Domain.
	DisabledStatus
	// FreezeStatus represents domain is in freezed state.
	FreezeStatus
	// DeletedStatus represents domain is in deleted state.
	DeletedStatus

	// AllStatus is used for querying purposes to list Domains irrespective
	// of their status - enabled, disabled, freezed, deleting. It is never stored in the
	// database as the actual domain status and should always be the larger than freeze status
	// value in this enumeration.
	AllStatus
)

// String representation of the possible status values.
const (
	Disabled = "disabled"
	Enabled  = "enabled"
	Freezed  = "freezed"
	Deleted  = "deleted"
	All      = "all"
	Unknown  = "unknown"
)

// String converts client/group status to string literal.
func (s Status) String() string {
	switch s {
	case DisabledStatus:
		return Disabled
	case EnabledStatus:
		return Enabled
	case AllStatus:
		return All
	case FreezeStatus:
		return Freezed
	case DeletedStatus:
		return Deleted
	default:
		return Unknown
	}
}

// ToStatus converts string value to a valid Domain status.
func ToStatus(status string) (Status, error) {
	switch status {
	case "", Enabled:
		return EnabledStatus, nil
	case Disabled:
		return DisabledStatus, nil
	case Freezed:
		return FreezeStatus, nil
	case Deleted:
		return DeletedStatus, nil
	case All:
		return AllStatus, nil
	}
	return Status(0), svcerr.ErrInvalidStatus
}

// Custom Marshaller for Domains status.
func (s Status) MarshalJSON() ([]byte, error) {
	return json.Marshal(s.String())
}

// Custom Unmarshaler for Domains status.
func (s *Status) UnmarshalJSON(data []byte) error {
	str := strings.Trim(string(data), "\"")
	val, err := ToStatus(str)
	*s = val
	return err
}

// Metadata represents arbitrary JSON.
type Metadata map[string]interface{}

type DomainReq struct {
	Name      *string    `json:"name,omitempty"`
	Metadata  *Metadata  `json:"metadata,omitempty"`
	Tags      *[]string  `json:"tags,omitempty"`
	Alias     *string    `json:"alias,omitempty"`
	Status    *Status    `json:"status,omitempty"`
	UpdatedBy *string    `json:"updated_by,omitempty"`
	UpdatedAt *time.Time `json:"updated_at,omitempty"`
}

type Domain struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Metadata  Metadata  `json:"metadata,omitempty"`
	Tags      []string  `json:"tags,omitempty"`
	Alias     string    `json:"alias,omitempty"`
	Status    Status    `json:"status"`
	RoleID    string    `json:"role_id,omitempty"`
	RoleName  string    `json:"role_name,omitempty"`
	Actions   []string  `json:"actions,omitempty"`
	CreatedBy string    `json:"created_by,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedBy string    `json:"updated_by,omitempty"`
	UpdatedAt time.Time `json:"updated_at,omitempty"`
}

type Page struct {
	Total    uint64   `json:"total"`
	Offset   uint64   `json:"offset"`
	Limit    uint64   `json:"limit"`
	Name     string   `json:"name,omitempty"`
	Order    string   `json:"-"`
	Dir      string   `json:"-"`
	Metadata Metadata `json:"metadata,omitempty"`
	Tag      string   `json:"tag,omitempty"`
	RoleName string   `json:"role_name,omitempty"`
	RoleID   string   `json:"role_id,omitempty"`
	Actions  []string `json:"actions,omitempty"`
	Status   Status   `json:"status,omitempty"`
	ID       string   `json:"id,omitempty"`
	IDs      []string `json:"-"`
	Identity string   `json:"identity,omitempty"`
	UserID   string   `json:"user_id,omitempty"`
}

type DomainsPage struct {
	Total   uint64   `json:"total"`
	Offset  uint64   `json:"offset"`
	Limit   uint64   `json:"limit"`
	Domains []Domain `json:"domains"`
}

func (page DomainsPage) MarshalJSON() ([]byte, error) {
	type Alias DomainsPage
	a := struct {
		Alias
	}{
		Alias: Alias(page),
	}

	if a.Domains == nil {
		a.Domains = make([]Domain, 0)
	}

	return json.Marshal(a)
}

//go:generate mockery --name Service --output=./mocks --filename service.go --quiet --note "Copyright (c) Abstract Machines"
type Service interface {
	// CreateDomain creates a new domain.
	CreateDomain(ctx context.Context, sesssion authn.Session, d Domain) (Domain, []roles.RoleProvision, error)

	// RetrieveDomain retrieves a domain specified by the provided ID.
	RetrieveDomain(ctx context.Context, sesssion authn.Session, id string) (Domain, error)

	// UpdateDomain updates the domain specified by the provided ID.
	UpdateDomain(ctx context.Context, sesssion authn.Session, id string, d DomainReq) (Domain, error)

	// EnableDomain enables the domain specified by the provided ID.
	EnableDomain(ctx context.Context, sesssion authn.Session, id string) (Domain, error)

	// DisableDomain disables the domain specified by the provided ID.
	// Only platform administrators and domain admins can disable domains.
	DisableDomain(ctx context.Context, sesssion authn.Session, id string) (Domain, error)

	// FreezeDomain freezes the domain specified by the provided ID.
	// Only platform administrators can freeze domains.
	FreezeDomain(ctx context.Context, sesssion authn.Session, id string) (Domain, error)

	// ListDomains returns a list of domains.
	ListDomains(ctx context.Context, sesssion authn.Session, page Page) (DomainsPage, error)

	// SendInvitation sends an invitation to the given user.
	// Only domain administrators and platform administrators can send invitations.
	SendInvitation(ctx context.Context, session authn.Session, invitation Invitation) (err error)

	// ViewInvitation returns an invitation.
	// People who can view invitations are:
	// - the invited user: they can view their own invitations
	// - the user who sent the invitation
	// - domain administrators
	// - platform administrators
	ViewInvitation(ctx context.Context, session authn.Session, inviteeUserID, domainID string) (invitation Invitation, err error)

	// ListInvitations returns a list of invitations.
	// People who can list invitations are:
	// - platform administrators can list all invitations
	// - domain administrators can list invitations for their domain
	// By default, it will list invitations the current user has sent or received.
	ListInvitations(ctx context.Context, session authn.Session, page InvitationPageMeta) (invitations InvitationPage, err error)

	// AcceptInvitation accepts an invitation by adding the user to the domain.
	AcceptInvitation(ctx context.Context, session authn.Session, domainID string) (err error)

	// DeleteInvitation deletes an invitation.
	// People who can delete invitations are:
	// - the invited user: they can delete their own invitations
	// - the user who sent the invitation
	// - domain administrators
	// - platform administrators
	DeleteInvitation(ctx context.Context, session authn.Session, inviteeUserID, domainID string) (err error)

	// RejectInvitation rejects an invitation.
	// People who can reject invitations are:
	// - the invited user: they can reject their own invitations
	RejectInvitation(ctx context.Context, session authn.Session, domainID string) (err error)

	roles.RoleManager
}

// Repository specifies Domain persistence API.
//
//go:generate mockery --name Repository --output=./mocks --filename repository.go  --quiet --note "Copyright (c) Abstract Machines"
type Repository interface {
	// SaveDomain creates db insert transaction for the given domain.
	SaveDomain(ctx context.Context, d Domain) (Domain, error)

	// RetrieveDomainByID retrieves a domain by its unique ID.
	RetrieveDomainByID(ctx context.Context, id string) (Domain, error)

	// RetrieveDomainByUserAndID retrieves a domain by its unique ID and user ID.
	RetrieveDomainByUserAndID(ctx context.Context, userID, id string) (Domain, error)

	// RetrieveAllDomainsByIDs retrieves for given Domain IDs.
	RetrieveAllDomainsByIDs(ctx context.Context, pm Page) (DomainsPage, error)

	// UpdateDomain updates the domain name and metadata.
	UpdateDomain(ctx context.Context, id string, d DomainReq) (Domain, error)

	// DeleteDomain deletes the domain.
	DeleteDomain(ctx context.Context, id string) error

	// ListDomains list all the domains
	ListDomains(ctx context.Context, pm Page) (DomainsPage, error)

	// CreateInvitation creates an invitation.
	SaveInvitation(ctx context.Context, invitation Invitation) (err error)

	// RetrieveInvitation retrieves an invitation.
	RetrieveInvitation(ctx context.Context, userID, domainID string) (Invitation, error)

	// RetrieveAllInvitations retrieves all invitations.
	RetrieveAllInvitations(ctx context.Context, page InvitationPageMeta) (invitations InvitationPage, err error)

	// UpdateConfirmation updates an invitation by setting the confirmation time.
	UpdateConfirmation(ctx context.Context, invitation Invitation) (err error)

	// UpdateRejection updates an invitation by setting the rejection time.
	UpdateRejection(ctx context.Context, invitation Invitation) (err error)

	// Delete deletes an invitation.
	DeleteInvitation(ctx context.Context, userID, domainID string) (err error)

	roles.Repository
}

// Cache contains domains caching interface.
//
//go:generate mockery --name Cache --output=./mocks --filename cache.go --quiet --note "Copyright (c) Abstract Machines"
type Cache interface {
	// Save stores pair domain status and  domain id.
	Save(ctx context.Context, domainID string, status Status) error

	// Status returns domain status for given domain ID.
	Status(ctx context.Context, domainID string) (Status, error)

	// Removes domain from cache.
	Remove(ctx context.Context, domainID string) error
}
