// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

// Define OperationType.
type OperationType uint32

const (
	CreateOp OperationType = iota
	ReadOp
	ListOp
	UpdateOp
	DeleteOp
)

func (ot OperationType) String() string {
	switch ot {
	case CreateOp:
		return "create"
	case ReadOp:
		return "read"
	case ListOp:
		return "list"
	case UpdateOp:
		return "update"
	case DeleteOp:
		return "delete"
	default:
		return fmt.Sprintf("unknown operation type %d", ot)
	}
}

func (ot OperationType) MarshalJSON() ([]byte, error) {
	return []byte(ot.String()), nil
}
func (ot OperationType) MarshalText() (text []byte, err error) {
	return []byte(ot.String()), nil
}

// Define EntityType.
type EntityType uint32

const (
	DomainsScope EntityType = iota
	GroupsScope
	ChannelsScope
	ThingsScope
)

func (et EntityType) String() string {
	switch et {
	case DomainsScope:
		return "domains"
	case GroupsScope:
		return "groups"
	case ChannelsScope:
		return "channels"
	case ThingsScope:
		return "things"
	default:
		return fmt.Sprintf("unknown entity type %d", et)
	}
}

func (et EntityType) MarshalJSON() ([]byte, error) {
	return []byte(et.String()), nil
}
func (et EntityType) MarshalText() (text []byte, err error) {
	return []byte(et.String()), nil
}

// ScopeValue interface for Any entity ids or for sets of entity ids.
type ScopeValue interface {
	Contains(id string) bool
}

// AnyIDs implements ScopeValue for any entity id value.
type AnyIDs struct{}

func (s AnyIDs) Contains(id string) bool { return true }

// SelectedIDs implements ScopeValue for sets of entity ids.
type SelectedIDs map[string]struct{}

func (s SelectedIDs) Contains(id string) bool { _, ok := s[id]; return ok }

// OperationRegistry contains map of OperationType with value of AnyIDs or SelectedIDs.
type OperationRegistry[T ScopeValue] struct {
	Operations map[OperationType]T `json:"operations,omitempty"`
}

// `EntityRegistry` contains map of Entity types with all its related operations registry.
// Example Visualization of `EntityRegistry`.
//
//	{
//		"entities": {
//			"domains": {
//				"operations": {
//				"create": {}
//				}
//			},
//			"groups": {
//				"operations": {
//				"read": {
//					"group1": {},
//					"group2": {}
//				}
//				}
//			}
//		}
//	}
type EntityRegistry struct {
	Entities map[EntityType]OperationRegistry[ScopeValue] `json:"entities,omitempty"`
}

// Add adds entry in Registry.
func (er *EntityRegistry) Add(entityType EntityType, operation OperationType, entityIDs ...string) {
	var value ScopeValue

	switch {
	case len(entityIDs) == 0, len(entityIDs) == 1 && entityIDs[0] == "*":
		value = AnyIDs{}

	default:
		var sids SelectedIDs
		for _, entityID := range entityIDs {
			if sids == nil {
				sids = make(SelectedIDs)
			}
			sids[entityID] = struct{}{}
		}
		value = sids
	}

	if er.Entities == nil {
		er.Entities = make(map[EntityType]OperationRegistry[ScopeValue])
	}
	if _, exists := er.Entities[entityType]; !exists {
		er.Entities[entityType] = OperationRegistry[ScopeValue]{
			Operations: make(map[OperationType]ScopeValue),
		}
	}
	er.Entities[entityType].Operations[operation] = value
}

func (er *EntityRegistry) Delete(entityType EntityType, operation OperationType, entityIDs ...string) error {
	if er.Entities == nil {
		return nil
	}

	opReg, exists := er.Entities[entityType]
	if !exists {
		return nil
	}

	opEntityIDs, exists := opReg.Operations[operation]
	if !exists {
		return nil
	}

	if len(entityIDs) == 0 {
		delete(opReg.Operations, operation)
		return nil
	}

	switch eIDs := any(opEntityIDs).(type) {
	case AnyIDs:
		delete(opReg.Operations, operation)
	case SelectedIDs:
		for _, entityID := range entityIDs {
			if !eIDs.Contains(entityID) {
				return fmt.Errorf("invalid entity ID in list")
			}
		}
		for _, entityID := range entityIDs {
			delete(eIDs, entityID)
			if len(eIDs) == 0 {
				delete(opReg.Operations, operation)
			}
		}
	}
	return nil
}

// Check entry in Registry.
func (er *EntityRegistry) Check(entityType EntityType, operation OperationType, id string) bool {
	if er.Entities == nil {
		return false
	}
	operations, exists := er.Entities[entityType]
	if !exists {
		return false
	}
	scopeValue, exists := operations.Operations[operation]
	if !exists {
		return false
	}
	return scopeValue.Contains(id)
}

func (er *EntityRegistry) String() string {
	str, err := json.MarshalIndent(er, "", "  ")
	if err != nil {
		return fmt.Sprintf("failed to convert scope/entity_registry to string: json marshal error :%s", err.Error())
	}
	return string(str)
}

// PAT represents Personal Access Token.
// Example Visualization of PAT.
//
//	{
//		"id": "new id",
//		"user": "user 1",
//		"scopes": {
//		  "entities": {
//			"domains": {
//			  "operations": {
//				"create": {}
//			  }
//			},
//			"groups": {
//			  "operations": {
//				"read": {
//				  "group1": {},
//				  "group2": {}
//				}
//			  }
//			}
//		  }
//		},
//		"issued_at": "2024-06-17T14:52:22.670691615+05:30",
//		"expires_at": "2024-06-20T14:52:22.670691708+05:30"
//	  }
type PAT struct {
	ID        string         `json:"id,omitempty"`
	User      string         `json:"user,omitempty"`
	Scopes    EntityRegistry `json:"scopes,omitempty"`
	IssuedAt  time.Time      `json:"issued_at,omitempty"`
	ExpiresAt time.Time      `json:"expires_at,omitempty"`
}

func (pat PAT) String() string {
	str, err := json.MarshalIndent(pat, "", "  ")
	if err != nil {
		return fmt.Sprintf("failed to convert PAT to string: json marshal error :%s", err.Error())
	}
	return string(str)
}

// Expired verifies if the key is expired.
func (pat PAT) Expired() bool {
	return pat.ExpiresAt.UTC().Before(time.Now().UTC())
}

// KeyRepository specifies Key persistence API.
//
//go:generate mockery --name KeyRepository --output=./mocks --filename keys.go --quiet --note "Copyright (c) Abstract Machines"
type PATRepository interface {
	// Save persists the Key. A non-nil error is returned to indicate
	// operation failure
	Save(ctx context.Context, key Key) (id string, err error)

	// Retrieve retrieves Key by its unique identifier.
	Retrieve(ctx context.Context, issuer string, id string) (key Key, err error)

	// Remove removes Key with provided ID.
	Remove(ctx context.Context, issuer string, id string) error
}
