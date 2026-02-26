// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	apiutil "github.com/absmach/supermq/api/http/util"
	"github.com/absmach/supermq/pkg/errors"
	"github.com/absmach/supermq/pkg/permissions"
)

const (
	AnyIDs              = "*"
	RoleOperationPrefix = "role_"
)

const (
	OpCreate = "create"
	OpList   = "list"

	OpCreateClients  = "create_clients"
	OpListClients    = "list_clients"
	OpCreateChannels = "create_channels"
	OpListChannels   = "list_channels"
	OpCreateGroups   = "create_groups"
	OpListGroups     = "list_groups"

	OpShare   = "share"
	OpUnshare = "unshare"

	OpDashboardShare   = "dashboard_share"
	OpDashboardUnshare = "dashboard_unshare"

	OpPublish   = "publish"
	OpSubscribe = "subscribe"

	OpMessagePublish   = "message_publish"
	OpMessageSubscribe = "message_subscribe"
)

var errInvalidEntityOp = errors.NewRequestError("operation not valid for entity type")

type Operation = permissions.Operation

// Dashboard operations.
const (
	DashboardShareOp Operation = iota + 400
	DashboardUnshareOp
)

// Messages operations.
const (
	MessagePublishOp Operation = iota + 500
	MessageSubscribeOp
)

type EntityType uint32

const (
	GroupsType EntityType = iota
	ChannelsType
	ClientsType
	DashboardType
	MessagesType
	DomainsType
	UsersType
)

const (
	GroupsScopeStr   = "groups"
	ChannelsScopeStr = "channels"
	ClientsScopeStr  = "clients"
	DashboardsStr    = "dashboards"
	MessagesStr      = "messages"
	DomainsStr       = "domains"
	UsersStr         = "users"
)

func (et EntityType) String() string {
	switch et {
	case GroupsType:
		return GroupsScopeStr
	case ChannelsType:
		return ChannelsScopeStr
	case ClientsType:
		return ClientsScopeStr
	case DashboardType:
		return DashboardsStr
	case MessagesType:
		return MessagesStr
	case DomainsType:
		return DomainsStr
	case UsersType:
		return UsersStr
	default:
		return fmt.Sprintf("unknown domain entity type %d", et)
	}
}

func ParseEntityType(et string) (EntityType, error) {
	switch et {
	case GroupsScopeStr:
		return GroupsType, nil
	case ChannelsScopeStr:
		return ChannelsType, nil
	case ClientsScopeStr:
		return ClientsType, nil
	case DashboardsStr:
		return DashboardType, nil
	case MessagesStr:
		return MessagesType, nil
	case DomainsStr:
		return DomainsType, nil
	case UsersStr:
		return UsersType, nil
	default:
		return 0, fmt.Errorf("unknown domain entity type %s", et)
	}
}

func (et EntityType) MarshalJSON() ([]byte, error) {
	return json.Marshal(et.String())
}

func (et *EntityType) UnmarshalJSON(data []byte) error {
	str := strings.Trim(string(data), "\"")
	val, err := ParseEntityType(str)
	*et = val
	return err
}

func (et EntityType) MarshalText() ([]byte, error) {
	return []byte(et.String()), nil
}

func (et *EntityType) UnmarshalText(data []byte) (err error) {
	str := strings.Trim(string(data), "\"")
	*et, err = ParseEntityType(str)
	return err
}

func IsValidOperationForEntity(entityType EntityType, operation string) bool {
	switch entityType {
	case ClientsType, ChannelsType, GroupsType, DomainsType:
		return true
	case DashboardType:
		return operation == OpDashboardShare || operation == OpDashboardUnshare
	case MessagesType:
		return operation == OpMessagePublish || operation == OpMessageSubscribe
	default:
		return false
	}
}

// Example Scope as JSON
//
// [
//     {
//         "domain_id": "domain_1",
//         "entity_type": "groups",
//         "operation": "view",
//         "entity_id": "*"
//     },
//     {
//         "domain_id": "domain_1",
//         "entity_type": "channels",
//         "operation": "delete",
//         "entity_id": "channel1"
//     },
//     {
//         "domain_id": "domain_1",
//         "entity_type": "clients",
//         "operation": "update",
//         "entity_id": "*"
//     }
// ]

type Scope struct {
	ID         string     `json:"id"`
	PatID      string     `json:"pat_id"`
	DomainID   string     `json:"domain_id"`
	EntityType EntityType `json:"entity_type"`
	EntityID   string     `json:"entity_id"`
	Operation  string     `json:"operation"`
}

func (s *Scope) UnmarshalJSON(data []byte) error {
	type Alias Scope
	aux := (*Alias)(s)

	if err := json.Unmarshal(data, aux); err != nil {
		return err
	}

	switch s.EntityType {
	case ClientsType:
		switch s.Operation {
		case OpCreate:
			s.Operation = OpCreateClients
		case OpList:
			s.Operation = OpListClients
		}
	case ChannelsType:
		switch s.Operation {
		case OpCreate:
			s.Operation = OpCreateChannels
		case OpList:
			s.Operation = OpListChannels
		}
	case GroupsType:
		switch s.Operation {
		case OpCreate:
			s.Operation = OpCreateGroups
		case OpList:
			s.Operation = OpListGroups
		}
	case DashboardType:
		switch s.Operation {
		case OpShare:
			s.Operation = OpDashboardShare
		case OpUnshare:
			s.Operation = OpDashboardUnshare
		}
	case MessagesType:
		switch s.Operation {
		case OpPublish:
			s.Operation = OpMessagePublish
		case OpSubscribe:
			s.Operation = OpMessageSubscribe
		}
	}

	return nil
}

func (s *Scope) Authorized(entityType EntityType, domainID string, operation string, entityID string) bool {
	if s == nil {
		return false
	}

	if s.EntityType != entityType {
		return false
	}

	if s.DomainID != "" && s.DomainID != domainID {
		return false
	}

	if s.Operation != operation {
		return false
	}

	if s.EntityID == "*" {
		return true
	}

	if s.EntityID == entityID {
		return true
	}

	return false
}

func (s *Scope) Validate() error {
	if s == nil {
		return errInvalidScope
	}
	if s.EntityID == "" {
		return apiutil.ErrMissingEntityID
	}

	if s.DomainID == "" {
		return apiutil.ErrMissingDomainID
	}

	if !IsValidOperationForEntity(s.EntityType, s.Operation) {
		return errors.Wrap(apiutil.ErrInvalidQueryParams, errInvalidEntityOp)
	}

	return nil
}

// PATAuthz represents the PAT authorization request fields.
type PATAuthz struct {
	PatID      string
	UserID     string
	EntityType EntityType
	EntityID   string
	Operation  string
	Domain     string
}

// PAT represents Personal Access Token.
type PAT struct {
	ID          string    `json:"id,omitempty"`
	User        string    `json:"user_id,omitempty"`
	Name        string    `json:"name,omitempty"`
	Description string    `json:"description,omitempty"`
	Secret      string    `json:"secret,omitempty"`
	Role        Role      `json:"role,omitempty"`
	IssuedAt    time.Time `json:"issued_at,omitempty"`
	ExpiresAt   time.Time `json:"expires_at,omitempty"`
	UpdatedAt   time.Time `json:"updated_at,omitempty"`
	LastUsedAt  time.Time `json:"last_used_at,omitempty"`
	Revoked     bool      `json:"revoked,omitempty"`
	RevokedAt   time.Time `json:"revoked_at,omitempty"`
	Status      Status    `json:"status,omitempty"`
}

type PATSPageMeta struct {
	Offset uint64 `json:"offset"`
	Limit  uint64 `json:"limit"`
	Name   string `json:"name"`
	ID     string `json:"id"`
	Status Status `json:"status"`
}
type PATSPage struct {
	Total  uint64 `json:"total"`
	Offset uint64 `json:"offset"`
	Limit  uint64 `json:"limit"`
	PATS   []PAT  `json:"pats"`
}

type ScopesPageMeta struct {
	Offset uint64 `json:"offset"`
	Limit  uint64 `json:"limit"`
	PatID  string `json:"pat_id"`
	ID     string `json:"id"`
}

type ScopesPage struct {
	Total  uint64  `json:"total"`
	Offset uint64  `json:"offset"`
	Limit  uint64  `json:"limit"`
	Scopes []Scope `json:"scopes"`
}

func (pat PAT) MarshalBinary() ([]byte, error) {
	return json.Marshal(pat)
}

func (pat *PAT) UnmarshalBinary(data []byte) error {
	return json.Unmarshal(data, pat)
}

// Validate checks if the PAT has valid fields.
func (pat *PAT) Validate() error {
	if pat == nil {
		return errors.New("PAT cannot be nil")
	}
	if pat.Name == "" {
		return errors.New("PAT name cannot be empty")
	}
	if pat.User == "" {
		return errors.New("PAT user cannot be empty")
	}
	return nil
}

// PATS specifies function which are required for Personal access Token implementation.
type PATS interface {
	// Create function creates new PAT for given valid inputs.
	CreatePAT(ctx context.Context, token, name, description string, duration time.Duration) (PAT, error)

	// UpdateName function updates the name for the given PAT ID.
	UpdatePATName(ctx context.Context, token, patID, name string) (PAT, error)

	// UpdateDescription function updates the description for the given PAT ID.
	UpdatePATDescription(ctx context.Context, token, patID, description string) (PAT, error)

	// Retrieve function retrieves the PAT for given ID.
	RetrievePAT(ctx context.Context, userID string, patID string) (PAT, error)

	// RemoveAllPAT function removes all PATs of user.
	RemoveAllPAT(ctx context.Context, token string) error

	// ListPATS function lists all the PATs for the user.
	ListPATS(ctx context.Context, token string, pm PATSPageMeta) (PATSPage, error)

	// Delete function deletes the PAT for given ID.
	DeletePAT(ctx context.Context, token, patID string) error

	// ResetSecret function reset the secret and creates new secret for the given ID.
	ResetPATSecret(ctx context.Context, token, patID string, duration time.Duration) (PAT, error)

	// RevokeSecret function revokes the secret for the given ID.
	RevokePATSecret(ctx context.Context, token, patID string) error

	// AddScope function adds a new scope.
	AddScope(ctx context.Context, token, patID string, scopes []Scope) error

	// RemoveScope function removes a scope.
	RemoveScope(ctx context.Context, token string, patID string, scopeIDs ...string) error

	// RemovePATAllScope function removes all scope.
	RemovePATAllScope(ctx context.Context, token, patID string) error

	// List function lists all the Scopes for the patID.
	ListScopes(ctx context.Context, token string, pm ScopesPageMeta) (ScopesPage, error)

	// IdentifyPAT function will valid the secret.
	IdentifyPAT(ctx context.Context, paToken string) (PAT, error)

	// AuthorizePAT function will valid the secret and check the given scope exists.
	AuthorizePAT(ctx context.Context, userID, patID string, entityType EntityType, domainID string, operation string, entityID string) error
}

// PATSRepository specifies PATS persistence API.
type PATSRepository interface {
	// Save persists the PAT
	Save(ctx context.Context, pat PAT) (err error)

	// Retrieve retrieves users PAT by its unique identifier.
	Retrieve(ctx context.Context, userID, patID string) (pat PAT, err error)

	// RetrieveScope retrieves PAT scopes by its unique identifier.
	RetrieveScope(ctx context.Context, pm ScopesPageMeta) (scopes ScopesPage, err error)

	// RetrieveSecretAndRevokeStatus retrieves secret and revoke status of PAT by its unique identifier.
	RetrieveSecretAndRevokeStatus(ctx context.Context, userID, patID string) (string, bool, bool, error)

	// UpdateName updates the name of a PAT.
	UpdateName(ctx context.Context, userID, patID, name string) (PAT, error)

	// UpdateDescription updates the description of a PAT.
	UpdateDescription(ctx context.Context, userID, patID, description string) (PAT, error)

	// UpdateTokenHash updates the token hash of a PAT.
	UpdateTokenHash(ctx context.Context, userID, patID, tokenHash string, expiryAt time.Time) (PAT, error)

	// RetrieveAll retrieves all PATs belongs to userID.
	RetrieveAll(ctx context.Context, userID string, pm PATSPageMeta) (pats PATSPage, err error)

	// Revoke PAT with provided ID.
	Revoke(ctx context.Context, userID, patID string) error

	// Reactivate PAT with provided ID.
	Reactivate(ctx context.Context, userID, patID string) error

	// Remove removes Key with provided ID.
	Remove(ctx context.Context, userID, patID string) error

	// RemoveAllPAT removes all PAT for a given user.
	RemoveAllPAT(ctx context.Context, userID string) error

	AddScope(ctx context.Context, userID string, scopes []Scope) error

	RemoveScope(ctx context.Context, userID string, scopesIDs ...string) error

	CheckScope(ctx context.Context, userID, patID string, entityType EntityType, domainID string, operation string, entityID string) error

	RemoveAllScope(ctx context.Context, patID string) error
}

type Cache interface {
	Save(ctx context.Context, userID string, scopes []Scope) error

	CheckScope(ctx context.Context, userID, patID, optionalDomainID string, entityType EntityType, operation string, entityID string) bool

	Remove(ctx context.Context, userID string, scopesID []string) error

	RemoveUserAllScope(ctx context.Context, userID string) error

	RemoveAllScope(ctx context.Context, userID, patID string) error
}
