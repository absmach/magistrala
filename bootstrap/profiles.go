// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package bootstrap

import (
	"context"
	"time"
)

// TemplateFormat enumerates supported content template formats.
type TemplateFormat string

const (
	TemplateFormatGoTemplate TemplateFormat = "go-template"
	TemplateFormatRaw        TemplateFormat = "raw"
	TemplateFormatJSON       TemplateFormat = "json"
	TemplateFormatYAML       TemplateFormat = "yaml"
	TemplateFormatTOML       TemplateFormat = "toml"
)

// Profile is a user-managed device configuration template.
type Profile struct {
	ID              string         `json:"id"`
	DomainID        string         `json:"domain_id,omitempty"`
	Name            string         `json:"name"`
	Description     string         `json:"description,omitempty"`
	TemplateFormat  TemplateFormat `json:"template_format"`
	ContentTemplate string         `json:"content_template,omitempty"`
	Defaults        map[string]any `json:"defaults,omitempty"`
	BindingSlots    []BindingSlot  `json:"binding_slots,omitempty"`
	Version         int            `json:"version,omitempty"`
	CreatedAt       time.Time      `json:"created_at,omitempty"`
	UpdatedAt       time.Time      `json:"updated_at,omitempty"`
}

// BindingSlot declares a named resource placeholder that a profile template can use.
type BindingSlot struct {
	Name     string   `json:"name"`
	Type     string   `json:"type"`
	Required bool     `json:"required"`
	Fields   []string `json:"fields,omitempty"`
}

// ProfilesPage contains pagination metadata and a slice of Profiles.
type ProfilesPage struct {
	Total    uint64    `json:"total"`
	Offset   uint64    `json:"offset"`
	Limit    uint64    `json:"limit"`
	Profiles []Profile `json:"profiles"`
}

// ProfileRepository specifies the persistence API for Profiles.
type ProfileRepository interface {
	// Save persists a new Profile and returns it with server-assigned fields set.
	Save(ctx context.Context, p Profile) (Profile, error)

	// RetrieveByID returns the Profile with the given ID inside the given domain.
	RetrieveByID(ctx context.Context, domainID, id string) (Profile, error)

	// RetrieveAll returns a page of Profiles belonging to the given domain.
	RetrieveAll(ctx context.Context, domainID string, offset, limit uint64) (ProfilesPage, error)

	// Update updates editable fields of the given Profile.
	Update(ctx context.Context, p Profile) error

	// Delete removes the Profile with the given ID from the given domain.
	Delete(ctx context.Context, domainID, id string) error
}
