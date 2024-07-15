// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/absmach/magistrala/pkg/errors"
)

var errAddEntityToAnyIDs = errors.New("could not add entity id to any ID scope value")

// Define OperationType.
type OperationType uint32

const (
	CreateOp OperationType = iota
	ReadOp
	ListOp
	UpdateOp
	DeleteOp
)

const (
	createOpStr = "create"
	readOpStr   = "read"
	listOpStr   = "list"
	updateOpStr = "update"
	deleteOpStr = "delete"
)

func (ot OperationType) String() string {
	switch ot {
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
	default:
		return fmt.Sprintf("unknown operation type %d", ot)
	}
}

func (ot OperationType) ValidString() (string, error) {
	str := ot.String()
	if str == fmt.Sprintf("unknown operation type %d", ot) {
		return "", errors.New(str)
	}
	return str, nil
}

func ParseOperationType(ot string) (OperationType, error) {
	switch ot {
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
	default:
		return 0, fmt.Errorf("unknown operation type %s", ot)
	}
}

func (ot OperationType) MarshalJSON() ([]byte, error) {
	return []byte(ot.String()), nil
}

func (ot OperationType) MarshalText() (text []byte, err error) {
	return []byte(ot.String()), nil
}

func (ot *OperationType) UnmarshalText(data []byte) (err error) {
	*ot, err = ParseOperationType(string(data))
	return err
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

const (
	domainManagementScopeStr = "domain_management"
	domainGroupsScopeStr     = "groups"
	domainChannelsScopeStr   = "channels"
	domainThingsScopeStr     = "things"
)

func (det DomainEntityType) String() string {
	switch det {
	case DomainManagementScope:
		return domainManagementScopeStr
	case DomainGroupsScope:
		return domainGroupsScopeStr
	case DomainChannelsScope:
		return domainChannelsScopeStr
	case DomainThingsScope:
		return domainThingsScopeStr
	default:
		return fmt.Sprintf("unknown domain entity type %d", det)
	}
}

func (det DomainEntityType) ValidString() (string, error) {
	str := det.String()
	if str == fmt.Sprintf("unknown operation type %d", det) {
		return "", errors.New(str)
	}
	return str, nil
}

func ParseDomainEntityType(det string) (DomainEntityType, error) {
	switch det {
	case domainManagementScopeStr:
		return DomainManagementScope, nil
	case domainGroupsScopeStr:
		return DomainGroupsScope, nil
	case domainChannelsScopeStr:
		return DomainChannelsScope, nil
	case domainThingsScopeStr:
		return DomainThingsScope, nil
	default:
		return 0, fmt.Errorf("unknown domain entity type %s", det)
	}
}

func (det DomainEntityType) MarshalJSON() ([]byte, error) {
	return []byte(det.String()), nil
}

func (det DomainEntityType) MarshalText() ([]byte, error) {
	return []byte(det.String()), nil
}

func (det *DomainEntityType) UnmarshalText(data []byte) (err error) {
	*det, err = ParseDomainEntityType(string(data))
	return err
}

// Define DomainEntityType.
type PlatformEntityType uint32

const (
	PlatformUsersScope PlatformEntityType = iota
	PlatformDomainsScope
)

const (
	platformUsersScopeStr   = "users"
	platformDomainsScopeStr = "domains"
)

func (pet PlatformEntityType) String() string {
	switch pet {
	case PlatformUsersScope:
		return platformUsersScopeStr
	case PlatformDomainsScope:
		return platformDomainsScopeStr
	default:
		return fmt.Sprintf("unknown platform entity type %d", pet)
	}
}

func (pet PlatformEntityType) ValidString() (string, error) {
	str := pet.String()
	if str == fmt.Sprintf("unknown platform entity type %d", pet) {
		return "", errors.New(str)
	}
	return str, nil
}

func ParsePlatformEntityType(pet string) (PlatformEntityType, error) {
	switch pet {
	case platformUsersScopeStr:
		return PlatformUsersScope, nil
	case platformDomainsScopeStr:
		return PlatformDomainsScope, nil
	default:
		return 0, fmt.Errorf("unknown platform entity type %s", pet)
	}
}

func (pet PlatformEntityType) MarshalJSON() ([]byte, error) {
	return []byte(pet.String()), nil
}

func (pet PlatformEntityType) MarshalText() (text []byte, err error) {
	return []byte(pet.String()), nil
}

func (pet *PlatformEntityType) UnmarshalText(data []byte) (err error) {
	*pet, err = ParsePlatformEntityType(string(data))
	return err
}

// ScopeValue interface for Any entity ids or for sets of entity ids.
type ScopeValue interface {
	Contains(id string) bool
	Values() []string
	AddValues(ids ...string) error
	RemoveValues(ids ...string) error
}

// AnyIDs implements ScopeValue for any entity id value.
type AnyIDs struct{}

func (s AnyIDs) Contains(id string) bool           { return true }
func (s AnyIDs) Values() []string                  { return []string{"*"} }
func (s *AnyIDs) AddValues(ids ...string) error    { return errAddEntityToAnyIDs }
func (s *AnyIDs) RemoveValues(ids ...string) error { return errAddEntityToAnyIDs }

// SelectedIDs implements ScopeValue for sets of entity ids.
type SelectedIDs map[string]struct{}

func (s SelectedIDs) Contains(id string) bool { _, ok := s[id]; return ok }
func (s SelectedIDs) Values() []string {
	values := []string{}
	for value := range s {
		values = append(values, value)
	}
	return values
}

func (s *SelectedIDs) AddValues(ids ...string) error {
	if *s == nil {
		*s = make(SelectedIDs)
	}
	for _, id := range ids {
		(*s)[id] = struct{}{}
	}
	return nil
}

func (s *SelectedIDs) RemoveValues(ids ...string) error {
	if *s == nil {
		return nil
	}
	for _, id := range ids {
		delete(*s, id)
	}
	return nil
}

// OperationScope contains map of OperationType with value of AnyIDs or SelectedIDs.
type OperationScope map[OperationType]ScopeValue

func (os *OperationScope) UnmarshalJSON(data []byte) error {
	type tempOperationScope map[OperationType]json.RawMessage

	var tempScope tempOperationScope
	if err := json.Unmarshal(data, &tempScope); err != nil {
		return err
	}
	// Initialize the Operations map
	*os = OperationScope{}

	for opType, rawMessage := range tempScope {
		var stringValue string
		var stringArrayValue []string

		// Try to unmarshal as string
		if err := json.Unmarshal(rawMessage, &stringValue); err == nil {
			if err := os.Add(opType, stringValue); err != nil {
				return err
			}
			continue
		}

		// Try to unmarshal as []string
		if err := json.Unmarshal(rawMessage, &stringArrayValue); err == nil {
			if err := os.Add(opType, stringArrayValue...); err != nil {
				return err
			}
			continue
		}

		// If neither unmarshalling succeeded, return an error
		return fmt.Errorf("invalid ScopeValue for OperationType %v", opType)
	}

	return nil
}

func (os OperationScope) MarshalJSON() ([]byte, error) {
	tempOperationScope := make(map[OperationType]interface{})
	for oType, scope := range os {
		value := scope.Values()
		if len(value) == 1 && value[0] == "*" {
			tempOperationScope[oType] = "*"
			continue
		}
		tempOperationScope[oType] = value
	}

	b, err := json.Marshal(tempOperationScope)
	if err != nil {
		return nil, err
	}
	return b, nil
}

func (os *OperationScope) Add(operation OperationType, entityIDs ...string) error {
	var value ScopeValue

	if os == nil {
		os = &OperationScope{}
	}

	if len(entityIDs) == 0 {
		return fmt.Errorf("entity ID is missing")
	}
	switch {
	case len(entityIDs) == 1 && entityIDs[0] == "*":
		value = &AnyIDs{}
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
		value = &sids
	}
	(*os)[operation] = value
	return nil
}

func (os *OperationScope) Delete(operation OperationType, entityIDs ...string) error {
	if os == nil {
		return nil
	}

	opEntityIDs, exists := (*os)[operation]
	if !exists {
		return nil
	}

	if len(entityIDs) == 0 {
		return fmt.Errorf("failed to delete operation %s: entity ID is missing", operation.String())
	}

	switch eIDs := opEntityIDs.(type) {
	case *AnyIDs:
		if !(len(entityIDs) == 1 && entityIDs[0] == "*") {
			return fmt.Errorf("failed to delete operation %s: invalid list", operation.String())
		}
		delete((*os), operation)
		return nil
	case *SelectedIDs:
		for _, entityID := range entityIDs {
			if !eIDs.Contains(entityID) {
				return fmt.Errorf("failed to delete operation %s: invalid entity ID in list", operation.String())
			}
		}
		for _, entityID := range entityIDs {
			delete(*eIDs, entityID)
			if len(*eIDs) == 0 {
				delete((*os), operation)
			}
		}
		return nil
	default:
		return fmt.Errorf("failed to delete operation: invalid entity id type %d", operation)
	}
}

func (os *OperationScope) Check(operation OperationType, entityIDs ...string) bool {
	if os == nil {
		return false
	}

	if scopeValue, ok := (*os)[operation]; ok {
		if len(entityIDs) == 0 {
			_, ok := scopeValue.(*AnyIDs)
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

	if len(os) == 0 {
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
	str, err := json.Marshal(s) // , "", "  ")
	if err != nil {
		return fmt.Sprintf("failed to convert scope to string: json marshal error :%s", err.Error())
	}
	return string(str)
}

// PAT represents Personal Access Token.
type PAT struct {
	ID          string    `json:"id,omitempty"`
	User        string    `json:"user,omitempty"`
	Name        string    `json:"name,omitempty"`
	Description string    `json:"description,omitempty"`
	Secret      string    `json:"secret,omitempty"`
	Scope       Scope     `json:"scope,omitempty"`
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
	PATS   []PAT  `json:"pats"`
}

func (pat *PAT) String() string {
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

// PATS specifies function which are required for Personal access Token implementation.
//go:generate mockery --name PATS --output=./mocks --filename pats.go --quiet --note "Copyright (c) Abstract Machines"

type PATS interface {
	// Create function creates new PAT for given valid inputs.
	CreatePAT(ctx context.Context, token, name, description string, duration time.Duration, scope Scope) (PAT, error)

	// UpdateName function updates the name for the given PAT ID.
	UpdatePATName(ctx context.Context, token, patID, name string) (PAT, error)

	// UpdateDescription function updates the description for the given PAT ID.
	UpdatePATDescription(ctx context.Context, token, patID, description string) (PAT, error)

	// Retrieve function retrieves the PAT for given ID.
	RetrievePAT(ctx context.Context, token, patID string) (PAT, error)

	// List function lists all the PATs for the user.
	ListPATS(ctx context.Context, token string, pm PATSPageMeta) (PATSPage, error)

	// Delete function deletes the PAT for given ID.
	DeletePAT(ctx context.Context, token, patID string) error

	// ResetSecret function reset the secret and creates new secret for the given ID.
	ResetPATSecret(ctx context.Context, token, patID string, duration time.Duration) (PAT, error)

	// RevokeSecret function revokes the secret for the given ID.
	RevokePATSecret(ctx context.Context, token, patID string) error

	// AddScope function adds a new scope entry.
	AddPATScopeEntry(ctx context.Context, token, patID string, platformEntityType PlatformEntityType, optionalDomainID string, optionalDomainEntityType DomainEntityType, operation OperationType, entityIDs ...string) (Scope, error)

	// RemoveScope function removes a scope entry.
	RemovePATScopeEntry(ctx context.Context, token, patID string, platformEntityType PlatformEntityType, optionalDomainID string, optionalDomainEntityType DomainEntityType, operation OperationType, entityIDs ...string) (Scope, error)

	// ClearAllScope function removes all scope entry.
	ClearPATAllScopeEntry(ctx context.Context, token, patID string) error

	// IdentifyPAT function will valid the secret.
	IdentifyPAT(ctx context.Context, paToken string) (PAT, error)

	// AuthorizePAT function will valid the secret and check the given scope exists.
	AuthorizePAT(ctx context.Context, paToken string, platformEntityType PlatformEntityType, optionalDomainID string, optionalDomainEntityType DomainEntityType, operation OperationType, entityIDs ...string) error

	// CheckPAT function will check the given scope exists.
	CheckPAT(ctx context.Context, userID, patID string, platformEntityType PlatformEntityType, optionalDomainID string, optionalDomainEntityType DomainEntityType, operation OperationType, entityIDs ...string) error
}

// PATSRepository specifies PATS persistence API.
//
//go:generate mockery --name PATSRepository --output=./mocks --filename patsrepo.go --quiet --note "Copyright (c) Abstract Machines"
type PATSRepository interface {
	// Save persists the PAT
	Save(ctx context.Context, pat PAT) (err error)

	// Retrieve retrieves users PAT by its unique identifier.
	Retrieve(ctx context.Context, userID, patID string) (pat PAT, err error)

	// RetrieveSecretAndRevokeStatus retrieves secret and revoke status of PAT by its unique identifier.
	RetrieveSecretAndRevokeStatus(ctx context.Context, userID, patID string) (string, bool, error)

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

	AddScopeEntry(ctx context.Context, userID, patID string, platformEntityType PlatformEntityType, optionalDomainID string, optionalDomainEntityType DomainEntityType, operation OperationType, entityIDs ...string) (Scope, error)

	RemoveScopeEntry(ctx context.Context, userID, patID string, platformEntityType PlatformEntityType, optionalDomainID string, optionalDomainEntityType DomainEntityType, operation OperationType, entityIDs ...string) (Scope, error)

	CheckScopeEntry(ctx context.Context, userID, patID string, platformEntityType PlatformEntityType, optionalDomainID string, optionalDomainEntityType DomainEntityType, operation OperationType, entityIDs ...string) error

	RemoveAllScopeEntry(ctx context.Context, userID, patID string) error
}
