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
)

const AnyIDs = "*"

type Operation uint32

const (
	CreateOp Operation = iota
	ReadOp
	ListOp
	UpdateOp
	DeleteOp
	ShareOp
	UnshareOp
	PublishOp
	SubscribeOp
)

const (
	createOpStr    = "create"
	readOpStr      = "read"
	listOpStr      = "list"
	updateOpStr    = "update"
	deleteOpStr    = "delete"
	shareOpStr     = "share"
	UnshareOpStr   = "unshare"
	PublishOpStr   = "publish"
	SubscribeOpStr = "subscribe"
)

func (op Operation) String() string {
	switch op {
	case CreateOp:
		return createOpStr
	case ReadOp:
		return readOpStr
	case ListOp:
		return listOpStr
	case UpdateOp:
		return updateOpStr
	case DeleteOp:
		return deleteOpStr
	case ShareOp:
		return shareOpStr
	case UnshareOp:
		return UnshareOpStr
	case PublishOp:
		return PublishOpStr
	case SubscribeOp:
		return SubscribeOpStr
	default:
		return fmt.Sprintf("unknown operation type %d", op)
	}
}

func (op Operation) ValidString() (string, error) {
	str := op.String()
	if str == fmt.Sprintf("unknown operation type %d", op) {
		return "", errors.New(str)
	}
	return str, nil
}

func ParseOperation(op string) (Operation, error) {
	switch op {
	case createOpStr:
		return CreateOp, nil
	case readOpStr:
		return ReadOp, nil
	case listOpStr:
		return ListOp, nil
	case updateOpStr:
		return UpdateOp, nil
	case deleteOpStr:
		return DeleteOp, nil
	case shareOpStr:
		return ShareOp, nil
	case UnshareOpStr:
		return UnshareOp, nil
	case PublishOpStr:
		return PublishOp, nil
	case SubscribeOpStr:
		return SubscribeOp, nil
	default:
		return 0, fmt.Errorf("unknown operation type %s", op)
	}
}

func (op Operation) MarshalJSON() ([]byte, error) {
	return json.Marshal(op.String())
}

func (op *Operation) UnmarshalJSON(data []byte) error {
	str := strings.Trim(string(data), "\"")
	val, err := ParseOperation(str)
	*op = val
	return err
}

func (op Operation) MarshalText() (text []byte, err error) {
	return []byte(op.String()), nil
}

func (op *Operation) UnmarshalText(data []byte) (err error) {
	str := strings.Trim(string(data), "\"")
	*op, err = ParseOperation(str)
	return err
}

type EntityType uint32

const (
	GroupsType EntityType = iota
	ChannelsType
	ClientsType
	DomainsType
	UsersType
	DashboardType
	MessagesType
)

const (
	GroupsScopeStr   = "groups"
	ChannelsScopeStr = "channels"
	ClientsScopeStr  = "clients"
	DomainsStr       = "domains"
	UsersStr         = "users"
	DashboardsStr    = "dashboards"
	MessagesStr      = "messages"
)

func (et EntityType) String() string {
	switch et {
	case GroupsType:
		return GroupsScopeStr
	case ChannelsType:
		return ChannelsScopeStr
	case ClientsType:
		return ClientsScopeStr
	case DomainsType:
		return DomainsStr
	case UsersType:
		return UsersStr
	case DashboardType:
		return DashboardsStr
	case MessagesType:
		return MessagesStr
	default:
		return fmt.Sprintf("unknown domain entity type %d", et)
	}
}

func (et EntityType) ValidString() (string, error) {
	str := et.String()
	if str == fmt.Sprintf("unknown operation type %d", et) {
		return "", errors.New(str)
	}
	return str, nil
}

func ParseEntityType(et string) (EntityType, error) {
	switch et {
	case GroupsScopeStr:
		return GroupsType, nil
	case ChannelsScopeStr:
		return ChannelsType, nil
	case ClientsScopeStr:
		return ClientsType, nil
	case DomainsStr:
		return DomainsType, nil
	case UsersStr:
		return UsersType, nil
	case DashboardsStr:
		return DashboardType, nil
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

// Example Scope as JSON
//
// [
//     {
//         "optional_domain_id": "domain_1",
//         "entity_type": "groups",
//         "operation": "create",
//         "entity_id": "*"
//     },
//     {
//         "optional_domain_id": "domain_1",
//         "entity_type": "channels",
//         "operation": "delete",
//         "entity_id": "channel1"
//     },
//     {
//         "optional_domain_id": "domain_1",
//         "entity_type": "things",
//         "operation": "update",
//         "entity_id": "*"
//     }
// ]

type Scope struct {
	ID               string     `json:"id,omitempty"`
	PatID            string     `json:"pat_id,omitempty"`
	OptionalDomainID string     `json:"optional_domain_id,omitempty"`
	EntityType       EntityType `json:"entity_type,omitempty"`
	EntityID         string     `json:"entity_id,omitempty"`
	Operation        Operation  `json:"operation,omitempty"`
}

func (s *Scope) Authorized(entityType EntityType, optionalDomainID string, operation Operation, entityID string) bool {
	if s == nil {
		return false
	}

	if s.EntityType != entityType {
		return false
	}

	if optionalDomainID != "" && s.OptionalDomainID != optionalDomainID {
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

	switch s.EntityType {
	case ChannelsType, GroupsType, ClientsType:
		if s.OptionalDomainID == "" {
			return apiutil.ErrMissingDomainID
		}
	}

	return nil
}

// PAT represents Personal Access Token.
type PAT struct {
	ID          string    `json:"id,omitempty"`
	User        string    `json:"user_id,omitempty"`
	Name        string    `json:"name,omitempty"`
	Description string    `json:"description,omitempty"`
	Secret      string    `json:"secret,omitempty"`
	IssuedAt    time.Time `json:"issued_at,omitempty"`
	ExpiresAt   time.Time `json:"expires_at,omitempty"`
	UpdatedAt   time.Time `json:"updated_at,omitempty"`
	LastUsedAt  time.Time `json:"last_used_at,omitempty"`
	Revoked     bool      `json:"revoked,omitempty"`
	RevokedAt   time.Time `json:"revoked_at,omitempty"`
}

type PATSPageMeta struct {
	Offset uint64 `json:"offset"`
	Limit  uint64 `json:"limit"`
}
type PATSPage struct {
	Total  uint64 `json:"total"`
	Offset uint64 `json:"offset"`
	Limit  uint64 `json:"limit"`
	PATS   []PAT  `json:"pats,omitempty"`
}

type ScopesPageMeta struct {
	Offset uint64 `json:"offset"`
	Limit  uint64 `json:"limit"`
	PatID  string `json:"pat_id"`
	ID     string `json:"id"`
}

type ScopesPage struct {
	Total  uint64  `json:"total"`
	Offset uint64  `json:"offset,omitempty"`
	Limit  uint64  `json:"limit,omitempy"`
	Scopes []Scope `json:"scopes,omitempty"`
}

func (pat PAT) MarshalBinary() ([]byte, error) {
	return json.Marshal(pat)
}

func (pat *PAT) UnmarshalBinary(data []byte) error {
	return json.Unmarshal(data, pat)
}

func (pat *PAT) String() string {
	str, err := json.MarshalIndent(pat, "", "  ")
	if err != nil {
		return fmt.Sprintf("failed to convert PAT to string: json marshal error :%s", err.Error())
	}
	return string(str)
}

// PATS specifies function which are required for Personal access Token implementation.
//go:generate mockery --name PATS --output=./mocks --filename pats.go --quiet --note "Copyright (c) Abstract Machines"

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
	AuthorizePAT(ctx context.Context, userID, patID string, entityType EntityType, optionalDomainID string, operation Operation, entityID string) error
}

// PATSRepository specifies PATS persistence API.
//
//go:generate mockery --name PATSRepository --output=./mocks --filename patsrepo.go --quiet --note "Copyright (c) Abstract Machines"
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

	CheckScope(ctx context.Context, userID, patID string, entityType EntityType, optionalDomainID string, operation Operation, entityID string) error

	RemoveAllScope(ctx context.Context, patID string) error
}

//go:generate mockery --name Cache --output=./mocks --filename cache.go --quiet --note "Copyright (c) Abstract Machines"
type Cache interface {
	Save(ctx context.Context, userID string, scopes []Scope) error

	CheckScope(ctx context.Context, userID, patID, optionalDomainID string, entityType EntityType, operation Operation, entityID string) bool

	Remove(ctx context.Context, userID string, scopesID []string) error

	RemoveUserAllScope(ctx context.Context, userID string) error

	RemoveAllScope(ctx context.Context, userID, patID string) error
}
