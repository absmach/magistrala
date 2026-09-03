// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package auth

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	apiutil "github.com/absmach/magistrala/api/http/util"
	"github.com/absmach/magistrala/pkg/errors"
	"github.com/absmach/magistrala/pkg/permissions"
)

const (
	AnyIDs              = "*"
	RoleOperationPrefix = "role_"
)

var errInvalidScope = errors.New("invalid scope")

// Role is the privilege level a PAT is issued at.
type Role uint32

const (
	UserRole Role = iota + 1
	AdminRole
)

func (r Role) String() string {
	switch r {
	case UserRole:
		return "user"
	case AdminRole:
		return "admin"
	default:
		return "unknown"
	}
}

func (r Role) Validate() bool {
	return UserRole <= r && r <= AdminRole
}

// EncodeWorkspaceUserID builds the composite identity a workspace-scoped user
// is authorized under. An empty workspace or user makes it empty rather than a
// half-formed identity that would match the wrong subject.
func EncodeWorkspaceUserID(workspaceID, userID string) string {
	if workspaceID == "" || userID == "" {
		return ""
	}
	return workspaceID + "_" + userID
}

const (
	OpCreate = "create"
	OpList   = "list"

	OpCreateDevices     = "create_devices"
	OpListDevices       = "list_devices"
	OpCreateDeviceTypes = "create_device_types"
	OpListDeviceTypes   = "list_device_types"
	OpCreateChannels    = "create_channels"
	OpListChannels      = "list_channels"
	OpCreateGroups      = "create_groups"
	OpListGroups        = "list_groups"

	OpShare   = "share"
	OpUnshare = "unshare"

	OpDashboardShare   = "dashboard_share"
	OpDashboardUnshare = "dashboard_unshare"

	OpPublish   = "publish"
	OpSubscribe = "subscribe"

	OpMessagePublish   = "message_publish"
	OpMessageSubscribe = "message_subscribe"
)

var (
	errInvalidEntityOp                 = errors.NewRequestError("operation not valid for entity type")
	errAlarmOpRequiresWildcardEntityID = errors.NewRequestError("alarm operations on rules entity type require wildcard entity ID")
)

// alarmOnlyOperations are RulesType operations authorized at the workspace level; only wildcard entity ID is valid.
var alarmOnlyOperations = map[string]struct{}{
	"alarm_assign":      {},
	"alarm_acknowledge": {},
	"alarm_resolve":     {},
}

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

// Pinned explicitly rather than left as iota: EntityType is never persisted
// or transmitted as a number (spec §8 C2), but pinning means this block can
// never become load-bearing by accident later. 2 (former ClientsType) is
// retired, not reused, so no future constant silently inherits old meaning.
const (
	GroupsType      EntityType = 0
	ChannelsType    EntityType = 1
	BootstrapType   EntityType = 3
	DashboardType   EntityType = 4
	MessagesType    EntityType = 5
	WorkspacesType  EntityType = 6
	UsersType       EntityType = 7
	RulesType       EntityType = 8
	ReportsType     EntityType = 9
	DevicesType     EntityType = 10
	DeviceTypesType EntityType = 11
)

const (
	GroupsScopeStr      = "groups"
	ChannelsScopeStr    = "channels"
	DevicesScopeStr     = "devices"
	DeviceTypesScopeStr = "device_types"
	BootstrapStr        = "bootstrap"
	DashboardsStr       = "dashboards"
	MessagesStr         = "messages"
	WorkspacesStr       = "workspaces"
	UsersStr            = "users"
	RulesScopeStr       = "rules"
	ReportsScopeStr     = "reports"
)

func (et EntityType) String() string {
	switch et {
	case GroupsType:
		return GroupsScopeStr
	case ChannelsType:
		return ChannelsScopeStr
	case DevicesType:
		return DevicesScopeStr
	case DeviceTypesType:
		return DeviceTypesScopeStr
	case BootstrapType:
		return BootstrapStr
	case DashboardType:
		return DashboardsStr
	case MessagesType:
		return MessagesStr
	case WorkspacesType:
		return WorkspacesStr
	case UsersType:
		return UsersStr
	case RulesType:
		return RulesScopeStr
	case ReportsType:
		return ReportsScopeStr
	default:
		return fmt.Sprintf("unknown workspace entity type %d", et)
	}
}

func ParseEntityType(et string) (EntityType, error) {
	switch et {
	case GroupsScopeStr:
		return GroupsType, nil
	case ChannelsScopeStr:
		return ChannelsType, nil
	case DevicesScopeStr:
		return DevicesType, nil
	case DeviceTypesScopeStr:
		return DeviceTypesType, nil
	case BootstrapStr:
		return BootstrapType, nil
	case DashboardsStr:
		return DashboardType, nil
	case MessagesStr:
		return MessagesType, nil
	case WorkspacesStr:
		return WorkspacesType, nil
	case UsersStr:
		return UsersType, nil
	case RulesScopeStr:
		return RulesType, nil
	case ReportsScopeStr:
		return ReportsType, nil
	default:
		return 0, fmt.Errorf("unknown workspace entity type %s", et)
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
	case DevicesType, DeviceTypesType, ChannelsType, GroupsType, BootstrapType, WorkspacesType, RulesType, ReportsType:
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
//         "workspace_id": "workspace_1",
//         "entity_type": "groups",
//         "operation": "view",
//         "entity_id": "*"
//     },
//     {
//         "workspace_id": "workspace_1",
//         "entity_type": "channels",
//         "operation": "delete",
//         "entity_id": "channel1"
//     },
//     {
//         "workspace_id": "workspace_1",
//         "entity_type": "devices",
//         "operation": "update",
//         "entity_id": "*"
//     }
// ]

type Scope struct {
	ID          string     `json:"id"`
	PatID       string     `json:"pat_id"`
	WorkspaceID string     `json:"workspace_id"`
	EntityType  EntityType `json:"entity_type"`
	EntityID    string     `json:"entity_id"`
	Operation   string     `json:"operation"`
}

func (s *Scope) UnmarshalJSON(data []byte) error {
	type Alias Scope
	aux := (*Alias)(s)

	if err := json.Unmarshal(data, aux); err != nil {
		return err
	}

	switch s.EntityType {
	case DevicesType:
		switch s.Operation {
		case OpCreate:
			s.Operation = OpCreateDevices
		case OpList:
			s.Operation = OpListDevices
		}
	case DeviceTypesType:
		switch s.Operation {
		case OpCreate:
			s.Operation = OpCreateDeviceTypes
		case OpList:
			s.Operation = OpListDeviceTypes
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

func (s *Scope) Authorized(entityType EntityType, workspaceID string, operation string, entityID string) bool {
	if s == nil {
		return false
	}

	if s.EntityType != entityType {
		return false
	}

	if s.WorkspaceID != "" && s.WorkspaceID != workspaceID {
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

	if s.WorkspaceID == "" {
		return apiutil.ErrMissingWorkspaceID
	}

	if !IsValidOperationForEntity(s.EntityType, s.Operation) {
		return errors.Wrap(apiutil.ErrInvalidQueryParams, errInvalidEntityOp)
	}

	if s.EntityType == RulesType {
		if _, ok := alarmOnlyOperations[s.Operation]; ok && s.EntityID != AnyIDs {
			return errors.Wrap(apiutil.ErrInvalidQueryParams, errAlarmOpRequiresWildcardEntityID)
		}
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
	Workspace  string
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
