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

// Define DomainEntityType.
type DomainEntityType uint32

const (
	DomainManagementScope DomainEntityType = iota
	DomainGroupsScope
	DomainChannelsScope
	DomainThingsScope
	DomainNullScope
)

func (det DomainEntityType) String() string {
	switch det {
	case DomainManagementScope:
		return "domain_management"
	case DomainGroupsScope:
		return "groups"
	case DomainChannelsScope:
		return "channels"
	case DomainThingsScope:
		return "things"
	default:
		return fmt.Sprintf("unknown domain entity type %d", det)
	}
}

func (det DomainEntityType) MarshalJSON() ([]byte, error) {
	return []byte(det.String()), nil
}
func (det DomainEntityType) MarshalText() (text []byte, err error) {
	return []byte(det.String()), nil
}

// Define DomainEntityType.
type PlatformEntityType uint32

const (
	PlatformUsersScope PlatformEntityType = iota
	PlatformDomainsScope
)

func (pet PlatformEntityType) String() string {
	switch pet {
	case PlatformUsersScope:
		return "users"
	case PlatformDomainsScope:
		return "domains"
	default:
		return fmt.Sprintf("unknown platform entity type %d", pet)
	}
}

func (pet PlatformEntityType) MarshalJSON() ([]byte, error) {
	return []byte(pet.String()), nil
}
func (pet PlatformEntityType) MarshalText() (text []byte, err error) {
	return []byte(pet.String()), nil
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

// OperationScope contains map of OperationType with value of AnyIDs or SelectedIDs.
type OperationScope struct {
	Operations map[OperationType]ScopeValue `json:"operations,omitempty"`
}

func (os *OperationScope) Add(operation OperationType, entityIDs ...string) error {
	var value ScopeValue

	if os == nil || os.Operations == nil {
		os.Operations = make(map[OperationType]ScopeValue)
	}

	if len(entityIDs) == 0 {
		return fmt.Errorf("entity ID is missing")
	}
	switch {
	case len(entityIDs) == 1 && entityIDs[0] == "*":
		value = AnyIDs{}
	default:
		var sids SelectedIDs
		for _, entityID := range entityIDs {
			if entityID == "*" {
				return fmt.Errorf("list contains wildcard")
			}
			if sids == nil {
				sids = make(SelectedIDs)
			}
			sids[entityID] = struct{}{}
		}
		value = sids
	}
	os.Operations[operation] = value
	return nil
}

func (os *OperationScope) Delete(operation OperationType, entityIDs ...string) error {
	if os == nil || os.Operations == nil {
		return nil
	}

	opEntityIDs, exists := os.Operations[operation]
	if !exists {
		return nil
	}

	if len(entityIDs) == 0 {
		return fmt.Errorf("failed to delete operation %s: entity ID is missing", operation.String())
	}

	switch eIDs := opEntityIDs.(type) {
	case AnyIDs:
		if !(len(entityIDs) == 1 && entityIDs[0] == "*") {
			return fmt.Errorf("failed to delete operation %s: invalid list", operation.String())
		}
		delete(os.Operations, operation)
		return nil
	case SelectedIDs:
		for _, entityID := range entityIDs {
			if !eIDs.Contains(entityID) {
				return fmt.Errorf("failed to delete operation %s: invalid entity ID in list", operation.String())
			}
		}
		for _, entityID := range entityIDs {
			delete(eIDs, entityID)
			if len(eIDs) == 0 {
				delete(os.Operations, operation)
			}
		}
		return nil
	default:
		return fmt.Errorf("failed to delete operation: invalid entity id type %d", operation)
	}
}

func (os *OperationScope) Check(operation OperationType, entityIDs ...string) bool {
	if os == nil || os.Operations == nil {
		return false
	}

	if scopeValue, ok := os.Operations[operation]; ok {
		if len(entityIDs) == 0 {
			_, ok := scopeValue.(AnyIDs)
			return ok
		}
		for _, entityID := range entityIDs {
			if !scopeValue.Contains(entityID) {
				return false
			}
		}
		return true
	}

	return false
}

type DomainScope struct {
	DomainManagement OperationScope                      `json:"domain_management,omitempty"`
	Entities         map[DomainEntityType]OperationScope `json:"entities,omitempty"`
}

// Add entry in Domain scope.
func (ds *DomainScope) Add(domainEntityType DomainEntityType, operation OperationType, entityIDs ...string) error {
	if ds == nil {
		return fmt.Errorf("failed to add domain %s scope: domain_scope is nil and not initialized", domainEntityType)
	}

	if domainEntityType < DomainManagementScope || domainEntityType > DomainThingsScope {
		return fmt.Errorf("failed to add domain %d scope: invalid domain entity type", domainEntityType)
	}
	if domainEntityType == DomainManagementScope {
		if err := ds.DomainManagement.Add(operation, entityIDs...); err != nil {
			return fmt.Errorf("failed to delete domain management scope: %w", err)
		}
	}

	if ds.Entities == nil {
		ds.Entities = make(map[DomainEntityType]OperationScope)
	}

	opReg, ok := ds.Entities[domainEntityType]
	if !ok {
		opReg = OperationScope{}
	}

	if err := opReg.Add(operation, entityIDs...); err != nil {
		return fmt.Errorf("failed to add domain %s scope: %w ", domainEntityType.String(), err)
	}
	ds.Entities[domainEntityType] = opReg
	return nil
}

// Delete entry in Domain scope.
func (ds *DomainScope) Delete(domainEntityType DomainEntityType, operation OperationType, entityIDs ...string) error {
	if ds == nil {
		return nil
	}

	if domainEntityType < DomainManagementScope || domainEntityType > DomainThingsScope {
		return fmt.Errorf("failed to delete domain %d scope: invalid domain entity type", domainEntityType)
	}
	if ds.Entities == nil {
		return nil
	}

	if domainEntityType == DomainManagementScope {
		if err := ds.DomainManagement.Delete(operation, entityIDs...); err != nil {
			return fmt.Errorf("failed to delete domain management scope: %w", err)
		}
	}

	os, exists := ds.Entities[domainEntityType]
	if !exists {
		return nil
	}

	if err := os.Delete(operation, entityIDs...); err != nil {
		return fmt.Errorf("failed to delete domain %s scope: %w", domainEntityType.String(), err)
	}

	if len(os.Operations) == 0 {
		delete(ds.Entities, domainEntityType)
	}
	return nil
}

// Check entry in Domain scope.
func (ds *DomainScope) Check(domainEntityType DomainEntityType, operation OperationType, ids ...string) bool {
	if ds.Entities == nil {
		return false
	}
	if domainEntityType < DomainManagementScope || domainEntityType > DomainThingsScope {
		return false
	}
	if domainEntityType == DomainManagementScope {
		return ds.DomainManagement.Check(operation, ids...)
	}
	os, exists := ds.Entities[domainEntityType]
	if !exists {
		return false
	}

	return os.Check(operation, ids...)
}

func (ds *DomainScope) String() string {
	str, err := json.MarshalIndent(ds, "", "  ")
	if err != nil {
		return fmt.Sprintf("failed to convert domain_scope to string: json marshal error :%s", err.Error())
	}
	return string(str)
}

// Example Scope as JSON
//
//	{
//	    "platform": {
//	        "users": {
//	            "create": {},
//	            "read": {},
//	            "list": {},
//	            "update": {},
//	            "delete": {}
//	        }
//	    },
//	    "domains": {
//	        "domain_1": {
//	            "entities": {
//	                "groups": {
//	                    "create": {}, // this for all groups in domain
//	                },
//	                "channels": {
//	                    // for particular channel in domain
//	                    "delete": {
//	                        "channel1": {},
//	                        "channel2":{}
//	                    }
//	                },
//	                "things": {
//	                    "update": {} // this for all things in domain
//	                }
//	            }
//	        }
//	    }
//	}
type Scope struct {
	Users   OperationScope         `json:"users,omitempty"`
	Domains map[string]DomainScope `json:"domains,omitempty"`
}

// Add entry in Domain scope.
func (s *Scope) Add(platformEntityType PlatformEntityType, optionalDomainID string, optionalDomainEntityType DomainEntityType, operation OperationType, entityIDs ...string) error {
	if s == nil {
		return fmt.Errorf("failed to add platform %s scope: scope is nil and not initialized", platformEntityType.String())
	}
	switch platformEntityType {
	case PlatformUsersScope:
		if err := s.Users.Add(operation, entityIDs...); err != nil {
			return fmt.Errorf("failed to add platform %s scope: %w", platformEntityType.String(), err)
		}
	case PlatformDomainsScope:
		if optionalDomainID == "" {
			return fmt.Errorf("failed to add platform %s scope: invalid domain id", platformEntityType.String())
		}
		if s.Domains == nil || len(s.Domains) == 0 {
			s.Domains = make(map[string]DomainScope)
		}

		ds, ok := s.Domains[optionalDomainID]
		if !ok {
			ds = DomainScope{}
		}
		if err := ds.Add(optionalDomainEntityType, operation, entityIDs...); err != nil {
			return fmt.Errorf("failed to add platform %s id %s  scope : %w", platformEntityType.String(), optionalDomainID, err)
		}
		s.Domains[optionalDomainID] = ds
	default:
		return fmt.Errorf("failed to add platform %d scope: invalid platform entity type ", platformEntityType)
	}
	return nil
}

// Delete entry in Domain scope.
func (s *Scope) Delete(platformEntityType PlatformEntityType, optionalDomainID string, optionalDomainEntityType DomainEntityType, operation OperationType, entityIDs ...string) error {
	if s == nil {
		return nil
	}
	switch platformEntityType {
	case PlatformUsersScope:
		if err := s.Users.Delete(operation, entityIDs...); err != nil {
			return fmt.Errorf("failed to delete platform %s scope: %w", platformEntityType.String(), err)
		}
	case PlatformDomainsScope:
		if optionalDomainID == "" {
			return fmt.Errorf("failed to delete platform %s scope: invalid domain id", platformEntityType.String())
		}
		ds, ok := s.Domains[optionalDomainID]
		if !ok {
			return nil
		}
		if err := ds.Delete(optionalDomainEntityType, operation, entityIDs...); err != nil {
			return fmt.Errorf("failed to delete platform %s id %s  scope : %w", platformEntityType.String(), optionalDomainID, err)
		}
	default:
		return fmt.Errorf("failed to add platform %d scope: invalid platform entity type ", platformEntityType)
	}
	return nil
}

// Check entry in Domain scope.
func (s *Scope) Check(platformEntityType PlatformEntityType, optionalDomainID string, optionalDomainEntityType DomainEntityType, operation OperationType, entityIDs ...string) bool {
	if s == nil {
		return false
	}
	switch platformEntityType {
	case PlatformUsersScope:
		return s.Users.Check(operation, entityIDs...)
	case PlatformDomainsScope:
		ds, ok := s.Domains[optionalDomainID]
		if !ok {
			return false
		}
		return ds.Check(optionalDomainEntityType, operation, entityIDs...)
	default:
		return false
	}
}

func (s *Scope) String() string {
	str, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return fmt.Sprintf("failed to convert scope to string: json marshal error :%s", err.Error())
	}
	return string(str)
}

// PAT represents Personal Access Token.
type PAT struct {
	ID         string    `json:"id,omitempty"`
	User       string    `json:"user,omitempty"`
	Name       string    `json:"name,omitempty"`
	Scope      Scope     `json:"scope,omitempty"`
	IssuedAt   time.Time `json:"issued_at,omitempty"`
	ExpiresAt  time.Time `json:"expires_at,omitempty"`
	UpdatedAt  time.Time `json:"updated_at,omitempty"`
	LastUsedAt time.Time `json:"last_used_at,omitempty"`
	Revoked    bool      `json:"revoked,omitempty"`
	RevokedAt  time.Time `json:"revoked_at,omitempty"`
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

type PATService interface {
	Create(ctx context.Context, token string, scope Scope) (PAT, error)
	Retrieve(ctx context.Context, token string, patID string) (PAT, error)
	List(ctx context.Context, token string) (PAT, error)
	Revoke(ctx context.Context, token string, paToken string) error
	Delete(ctx context.Context, token string, paToken string) error
}

// KeyRepository specifies Key persistence API.
//
//go:generate mockery --name KeyRepository --output=./mocks --filename keys.go --quiet --note "Copyright (c) Abstract Machines"
type PATRepository interface {
	// Save persists the Key. A non-nil error is returned to indicate
	// operation failure
	Save(ctx context.Context, pat PAT) (id string, err error)

	// Retrieve retrieves Key by its unique identifier.
	Retrieve(ctx context.Context, id string) (pat PAT, err error)

	// Remove removes Key with provided ID.
	Remove(ctx context.Context, id string) error
}
