// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package atom

import "time"

type Attributes map[string]any

type Tenant struct {
	ID         string     `json:"id,omitempty"`
	Name       string     `json:"name"`
	Route      string     `json:"route,omitempty"`
	Tags       []string   `json:"tags,omitempty"`
	Status     string     `json:"status,omitempty"`
	Attributes Attributes `json:"attributes,omitempty"`
	CreatedBy  string     `json:"created_by,omitempty"`
	UpdatedBy  string     `json:"updated_by,omitempty"`
	CreatedAt  time.Time  `json:"created_at,omitempty"`
	UpdatedAt  time.Time  `json:"updated_at,omitempty"`
}

type Entity struct {
	ID         string     `json:"id,omitempty"`
	Kind       string     `json:"kind"`
	Name       string     `json:"name"`
	TenantID   string     `json:"tenant_id,omitempty"`
	Status     string     `json:"status,omitempty"`
	Attributes Attributes `json:"attributes,omitempty"`
	CreatedAt  time.Time  `json:"created_at,omitempty"`
	UpdatedAt  time.Time  `json:"updated_at,omitempty"`
}

type Group struct {
	ID          string     `json:"id,omitempty"`
	Name        string     `json:"name"`
	TenantID    string     `json:"tenant_id,omitempty"`
	Description string     `json:"description,omitempty"`
	ParentID    string     `json:"parent_id,omitempty"`
	Status      string     `json:"status,omitempty"`
	Attributes  Attributes `json:"attributes,omitempty"`
	CreatedAt   time.Time  `json:"created_at,omitempty"`
	UpdatedAt   time.Time  `json:"updated_at,omitempty"`
}

type Resource struct {
	ID         string     `json:"id,omitempty"`
	Kind       string     `json:"kind"`
	Name       string     `json:"name"`
	TenantID   string     `json:"tenant_id,omitempty"`
	OwnerID    string     `json:"owner_id,omitempty"`
	Attributes Attributes `json:"attributes,omitempty"`
	CreatedAt  time.Time  `json:"created_at,omitempty"`
	UpdatedAt  time.Time  `json:"updated_at,omitempty"`
}

type Query struct {
	IDs                []string
	Q                  string
	Kind               string
	TenantID           string
	Name               string
	Route              string
	Status             string
	AttributesContains Attributes
	Limit              uint64
	Offset             uint64
}

type AuthzRequest struct {
	SubjectID  string         `json:"subject_id"`
	Action     string         `json:"action"`
	ResourceID string         `json:"resource_id,omitempty"`
	ObjectKind string         `json:"object_kind,omitempty"`
	ObjectID   string         `json:"object_id,omitempty"`
	Context    map[string]any `json:"context,omitempty"`
}

type AuthzResponse struct {
	Allowed bool   `json:"allowed"`
	Reason  string `json:"reason,omitempty"`
}

type Capability struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}

type CapabilityList struct {
	Items []Capability `json:"items"`
	Total int64        `json:"total,omitempty"`
}

type CapabilityApplicability struct {
	ActionID    string `json:"action_id"`
	ActionName  string `json:"action_name"`
	Description string `json:"description,omitempty"`
	ObjectKind  string `json:"object_kind"`
	ObjectType  string `json:"object_type,omitempty"`
}

type CapabilityApplicabilitySpec struct {
	ActionName  string
	Description string
	ObjectKind  string
	ObjectType  string
}

type ActionAssignmentRule struct {
	ID         string `json:"id"`
	TenantID   string `json:"tenant_id,omitempty"`
	EntityKind string `json:"entity_kind"`
	ActionName string `json:"action_name"`
	ObjectKind string `json:"object_kind"`
	ObjectType string `json:"object_type,omitempty"`
	Decision   string `json:"decision"`
	IsAbsolute bool   `json:"is_absolute"`
	CreatedAt  string `json:"created_at,omitempty"`
}

type ActionAssignmentRuleList struct {
	Items []ActionAssignmentRule `json:"items"`
	Total int64                  `json:"total,omitempty"`
}

type ActionAssignmentRuleSpec struct {
	TenantID   string
	EntityKind string
	ActionName string
	ObjectKind string
	ObjectType string
	Decision   string
	IsAbsolute bool
}

type PermissionBlock struct {
	ID         string         `json:"id"`
	TenantID   string         `json:"tenant_id,omitempty"`
	ScopeMode  string         `json:"scope_mode"`
	ObjectKind string         `json:"object_kind,omitempty"`
	ObjectType string         `json:"object_type,omitempty"`
	ObjectID   string         `json:"object_id,omitempty"`
	GroupID    string         `json:"group_id,omitempty"`
	Effect     string         `json:"effect"`
	Conditions map[string]any `json:"conditions,omitempty"`
	Actions    []Capability   `json:"actions,omitempty"`
}

type CreatePermissionBlock struct {
	TenantID   string         `json:"tenant_id,omitempty"`
	ScopeMode  string         `json:"scope_mode"`
	ObjectKind string         `json:"object_kind,omitempty"`
	ObjectType string         `json:"object_type,omitempty"`
	ObjectID   string         `json:"object_id,omitempty"`
	GroupID    string         `json:"group_id,omitempty"`
	Effect     string         `json:"effect,omitempty"`
	Conditions map[string]any `json:"conditions,omitempty"`
	ActionIDs  []string       `json:"action_ids"`
}

type DirectPolicy struct {
	ID                string          `json:"id"`
	TenantID          string          `json:"tenant_id,omitempty"`
	SubjectKind       string          `json:"subject_kind"`
	SubjectID         string          `json:"subject_id"`
	PermissionBlockID string          `json:"permission_block_id"`
	PermissionBlock   PermissionBlock `json:"permission_block,omitempty"`
	CreatedAt         time.Time       `json:"created_at,omitempty"`
}

type CreateDirectPolicy struct {
	TenantID          string `json:"tenant_id,omitempty"`
	SubjectKind       string `json:"subject_kind"`
	SubjectID         string `json:"subject_id"`
	PermissionBlockID string `json:"permission_block_id"`
}

type DirectPolicyQuery struct {
	TenantID    string
	SubjectKind string
	SubjectID   string
	Limit       uint64
	Offset      uint64
}

type DirectPolicyList struct {
	Items []DirectPolicy `json:"items"`
	Total uint64         `json:"total"`
}

type AuthorizedObjectIDsQuery struct {
	SubjectID  string
	Action     string
	ObjectKind string
	ObjectType string
	TenantID   string
	Q          string
	Limit      uint64
	Offset     uint64
}

type AuthorizedObjectIDs struct {
	IDs   []string `json:"ids"`
	Total uint64   `json:"total"`
}

type TokenClaims struct {
	SubjectID string `json:"sub"`
	SessionID string `json:"sid,omitempty"`
	TenantID  string `json:"tid,omitempty"`
	ExpiresAt int64  `json:"exp,omitempty"`
	IssuedAt  int64  `json:"iat,omitempty"`
}

type IntrospectionResponse struct {
	Active    bool   `json:"active"`
	EntityID  string `json:"entity_id"`
	TenantID  string `json:"tenant_id,omitempty"`
	SessionID string `json:"session_id,omitempty"`
}

type LoginRequest struct {
	Identifier string `json:"identifier"`
	Secret     string `json:"secret"`
	Kind       string `json:"kind,omitempty"`
}

type LoginResponse struct {
	Token     string    `json:"token"`
	EntityID  string    `json:"entity_id"`
	SessionID string    `json:"session_id"`
	ExpiresAt time.Time `json:"expires_at"`
}

type APIKeyResponse struct {
	CredentialID string     `json:"credentialId"`
	Key          string     `json:"key"`
	ExpiresAt    *time.Time `json:"expiresAt,omitempty"`
}

type ResourceList struct {
	Items []Resource `json:"items"`
	Total uint64     `json:"total"`
}

type TenantList struct {
	Items []Tenant `json:"items"`
	Total uint64   `json:"total"`
}

type EntityList struct {
	Items []Entity `json:"items"`
	Total uint64   `json:"total"`
}

type GroupList struct {
	Items []Group `json:"items"`
	Total uint64  `json:"total"`
}

type ObjectFields struct {
	ID          string
	Kind        string
	Name        string
	TenantID    string
	OwnerID     string
	Status      string
	Route       string
	ParentID    string
	Tags        []string
	Metadata    map[string]any
	Private     map[string]any
	CreatedBy   string
	UpdatedBy   string
	CreatedAt   time.Time
	UpdatedAt   time.Time
	Description string
}
