// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package domains

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/absmach/magistrala/pkg/authn"
	svcerr "github.com/absmach/magistrala/pkg/errors/service"
	"github.com/absmach/magistrala/pkg/roles"
	"github.com/absmach/magistrala/pkg/svcutil"
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

const (
	OpUpdateDomain svcutil.Operation = iota
	OpRetrieveDomain
	OpEnableDomain
	OpDisableDomain
)

var expectedOperations = []svcutil.Operation{
	OpRetrieveDomain,
	OpUpdateDomain,
	OpEnableDomain,
	OpDisableDomain,
}

var operationNames = []string{
	"OpRetrieveDomain",
	"OpUpdateDomain",
	"OpEnableDomain",
	"OpDisableDomain",
}

func NewOperationPerm() svcutil.OperationPerm {
	return svcutil.NewOperationPerm(expectedOperations, operationNames)
}

// Metadata represents arbitrary JSON.
type Metadata map[string]interface{}

type DomainReq struct {
	Name     *string   `json:"name,omitempty"`
	Metadata *Metadata `json:"metadata,omitempty"`
	Tags     *[]string `json:"tags,omitempty"`
	Alias    *string   `json:"alias,omitempty"`
	Status   *Status   `json:"status,omitempty"`
}
type Domain struct {
	ID         string    `json:"id"`
	Name       string    `json:"name"`
	Metadata   Metadata  `json:"metadata,omitempty"`
	Tags       []string  `json:"tags,omitempty"`
	Alias      string    `json:"alias,omitempty"`
	Status     Status    `json:"status"`
	Permission string    `json:"permission,omitempty"`
	CreatedBy  string    `json:"created_by,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedBy  string    `json:"updated_by,omitempty"`
	UpdatedAt  time.Time `json:"updated_at,omitempty"`
}

type Page struct {
	Total      uint64   `json:"total"`
	Offset     uint64   `json:"offset"`
	Limit      uint64   `json:"limit"`
	Name       string   `json:"name,omitempty"`
	Order      string   `json:"-"`
	Dir        string   `json:"-"`
	Metadata   Metadata `json:"metadata,omitempty"`
	Tag        string   `json:"tag,omitempty"`
	Permission string   `json:"permission,omitempty"`
	Status     Status   `json:"status,omitempty"`
	ID         string   `json:"id,omitempty"`
	IDs        []string `json:"-"`
	Identity   string   `json:"identity,omitempty"`
	SubjectID  string   `json:"-"`
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

type Policy struct {
	SubjectType     string `json:"subject_type,omitempty"`
	SubjectID       string `json:"subject_id,omitempty"`
	SubjectRelation string `json:"subject_relation,omitempty"`
	Relation        string `json:"relation,omitempty"`
	ObjectType      string `json:"object_type,omitempty"`
	ObjectID        string `json:"object_id,omitempty"`
}

//go:generate mockery --name Service --output=./mocks --filename service.go --quiet --note "Copyright (c) Abstract Machines"
type Service interface {
	CreateDomain(ctx context.Context, sesssion authn.Session, d Domain) (Domain, error)
	RetrieveDomain(ctx context.Context, sesssion authn.Session, id string) (Domain, error)
	UpdateDomain(ctx context.Context, sesssion authn.Session, id string, d DomainReq) (Domain, error)
	EnableDomain(ctx context.Context, sesssion authn.Session, id string) (Domain, error)
	DisableDomain(ctx context.Context, sesssion authn.Session, id string) (Domain, error)
	FreezeDomain(ctx context.Context, sesssion authn.Session, id string) (Domain, error)
	ListDomains(ctx context.Context, sesssion authn.Session, page Page) (DomainsPage, error)
	DeleteUserFromDomains(ctx context.Context, id string) error
	roles.RoleManager
}

// Repository specifies Domain persistence API.
//
//go:generate mockery --name Repository --output=./mocks --filename repository.go  --quiet --note "Copyright (c) Abstract Machines"
type Repository interface {
	// Save creates db insert transaction for the given domain.
	Save(ctx context.Context, d Domain) (Domain, error)

	// RetrieveByID retrieves Domain by its unique ID.
	RetrieveByID(ctx context.Context, id string) (Domain, error)

	// RetrieveAllByIDs retrieves for given Domain IDs.
	RetrieveAllByIDs(ctx context.Context, pm Page) (DomainsPage, error)

	// Update updates the client name and metadata.
	Update(ctx context.Context, id string, userID string, d DomainReq) (Domain, error)

	// Delete
	Delete(ctx context.Context, id string) error

	// ListDomains list all the domains
	ListDomains(ctx context.Context, pm Page) (DomainsPage, error)

	roles.Repository
}

// Below codes should moved out of service, may be can be kept in `cmd/<svc>/main.go`

const (
	updatePermission          = "update_permission"
	enablePermission          = "enable_permission"
	disablePermission         = "disable_permission"
	readPermission            = "read_permission"
	membershipPermission      = "membership_permission"
	deletePermission          = "delete_permission"
	manageRolePermission      = "manage_role_permission"
	addRoleUsersPermission    = "add_role_users_permission"
	removeRoleUsersPermission = "remove_role_users_permission"
	viewRoleUsersPermission   = "view_role_users_permission"
)

const (
	ClientCreatePermission  = "client_create_permission"
	ChannelCreatePermission = "channel_create_permission"
	GroupCreatePermission   = "group_create_permission"
)

func NewOperationPermissionMap() map[svcutil.Operation]svcutil.Permission {
	opPerm := map[svcutil.Operation]svcutil.Permission{
		OpRetrieveDomain: readPermission,
		OpUpdateDomain:   updatePermission,
		OpEnableDomain:   enablePermission,
		OpDisableDomain:  disablePermission,
	}
	return opPerm
}

func NewRolesOperationPermissionMap() map[svcutil.Operation]svcutil.Permission {
	opPerm := map[svcutil.Operation]svcutil.Permission{
		roles.OpAddRole:                manageRolePermission,
		roles.OpRemoveRole:             manageRolePermission,
		roles.OpUpdateRoleName:         manageRolePermission,
		roles.OpRetrieveRole:           manageRolePermission,
		roles.OpRetrieveAllRoles:       manageRolePermission,
		roles.OpRoleAddActions:         manageRolePermission,
		roles.OpRoleListActions:        manageRolePermission,
		roles.OpRoleCheckActionsExists: manageRolePermission,
		roles.OpRoleRemoveActions:      manageRolePermission,
		roles.OpRoleRemoveAllActions:   manageRolePermission,
		roles.OpRoleAddMembers:         addRoleUsersPermission,
		roles.OpRoleListMembers:        viewRoleUsersPermission,
		roles.OpRoleCheckMembersExists: viewRoleUsersPermission,
		roles.OpRoleRemoveMembers:      removeRoleUsersPermission,
		roles.OpRoleRemoveAllMembers:   manageRolePermission,
	}
	return opPerm
}
