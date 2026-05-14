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
	IDs      []string
	Q        string
	Kind     string
	TenantID string
	Name     string
	Route    string
	Status   string
	Limit    uint64
	Offset   uint64
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
	ID           string `json:"id"`
	Name         string `json:"name"`
	ResourceKind string `json:"resource_kind,omitempty"`
	Description  string `json:"description,omitempty"`
}

type CapabilityList struct {
	Items []Capability `json:"items"`
}

type PolicyBinding struct {
	ID          string         `json:"id"`
	TenantID    string         `json:"tenant_id,omitempty"`
	SubjectKind string         `json:"subject_kind"`
	SubjectID   string         `json:"subject_id"`
	GrantKind   string         `json:"grant_kind"`
	GrantID     string         `json:"grant_id"`
	ScopeKind   string         `json:"scope_kind"`
	ScopeRef    string         `json:"scope_ref,omitempty"`
	Effect      string         `json:"effect"`
	Conditions  map[string]any `json:"conditions,omitempty"`
	CreatedAt   time.Time      `json:"created_at,omitempty"`
}

type CreatePolicyBinding struct {
	TenantID    string         `json:"tenant_id,omitempty"`
	SubjectKind string         `json:"subject_kind"`
	SubjectID   string         `json:"subject_id"`
	GrantKind   string         `json:"grant_kind"`
	GrantID     string         `json:"grant_id"`
	ScopeKind   string         `json:"scope_kind"`
	ScopeRef    string         `json:"scope_ref,omitempty"`
	Effect      string         `json:"effect,omitempty"`
	Conditions  map[string]any `json:"conditions,omitempty"`
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
