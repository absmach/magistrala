// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package bootstrap

import (
	"context"
	"time"
)

// BindingRequest carries a user's intent to bind a named profile slot to
// a concrete resource.
type BindingRequest struct {
	Slot       string `json:"slot"`
	Type       string `json:"type"`        // "client" | "channel" | "cert"
	ResourceID string `json:"resource_id"` // ID of the resource in its owning service
}

// BindingSnapshot is a Bootstrap-owned point-in-time copy of the resource
// fields needed for template rendering. It is populated at binding time so
// that the render path never calls external services.
type BindingSnapshot struct {
	ConfigID       string         `json:"config_id"`
	Slot           string         `json:"slot"`
	Type           string         `json:"type"`
	ResourceID     string         `json:"resource_id"`
	Snapshot       map[string]any `json:"snapshot,omitempty"`
	SecretSnapshot map[string]any `json:"secret_snapshot,omitempty"` // encrypted at rest
	UpdatedAt      time.Time      `json:"updated_at,omitempty"`
}

// BindingStore is the persistence interface for BindingSnapshots.
type BindingStore interface {
	// Save upserts all given snapshots for the config.
	Save(ctx context.Context, configID string, bindings []BindingSnapshot) error

	// Retrieve returns all snapshots for the given config.
	Retrieve(ctx context.Context, configID string) ([]BindingSnapshot, error)

	// Delete removes the snapshot for a specific slot of a config.
	Delete(ctx context.Context, configID, slot string) error
}

// ResolveRequest carries everything the BindingResolver needs to snapshot a
// set of resource bindings.
type ResolveRequest struct {
	Enrollment Config
	Token      string
	Requested  []BindingRequest
}

// BindingResolver validates that requested resources exist in their owning
// services, verifies type and slot compatibility, and returns snapshots ready
// for storage. It is called at binding time only; the render path must not
// call it.
type BindingResolver interface {
	Resolve(ctx context.Context, req ResolveRequest) ([]BindingSnapshot, error)
}

// RenderContext is the typed value injected into Go templates during rendering.
type RenderContext struct {
	Device   DeviceContext
	Vars     map[string]any
	Bindings map[string]BindingContext
}

// DeviceContext holds enrollment identity fields available inside templates.
type DeviceContext struct {
	ID         string
	ExternalID string
	DomainID   string
}

// BindingContext holds the resolved resource data available inside templates
// for a specific slot.
type BindingContext struct {
	Type     string
	ID       string
	Snapshot map[string]any
	Secret   map[string]any
}
