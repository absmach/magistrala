// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package bootstrap

import (
	"context"
	"time"
)

// ContentFormat enumerates the supported output formats for rendered profile templates.
type ContentFormat string

const (
	ContentFormatGoTemplate ContentFormat = "go-template"
	ContentFormatRaw        ContentFormat = "raw"
	ContentFormatJSON       ContentFormat = "json"
	ContentFormatYAML       ContentFormat = "yaml"
	ContentFormatTOML       ContentFormat = "toml"
)

// ContentType identifies how a device should parse rendered profile content.
type ContentType string

const (
	ContentTypeJSON      ContentType = "application/json"
	ContentTypeYAML      ContentType = "application/yaml"
	ContentTypeTOML      ContentType = "application/toml"
	ContentTypeTextPlain ContentType = "text/plain"
)

func defaultContentType(format ContentFormat) ContentType {
	switch format {
	case ContentFormatJSON:
		return ContentTypeJSON
	case ContentFormatYAML:
		return ContentTypeYAML
	case ContentFormatTOML:
		return ContentTypeTOML
	default:
		return ContentTypeTextPlain
	}
}

func validContentType(contentType ContentType) bool {
	switch contentType {
	case ContentTypeJSON, ContentTypeYAML, ContentTypeTOML, ContentTypeTextPlain:
		return true
	default:
		return false
	}
}

// Profile is a user-managed device configuration template.
type Profile struct {
	ID              string         `json:"id"`
	WorkspaceID     string         `json:"workspace_id,omitempty"`
	Name            string         `json:"name"`
	Description     string         `json:"description,omitempty"`
	ContentFormat   ContentFormat  `json:"content_format"`
	ContentType     ContentType    `json:"content_type"`
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

	// RetrieveByID returns the Profile with the given ID inside the given workspace.
	RetrieveByID(ctx context.Context, workspaceID, id string) (Profile, error)

	// RetrieveAll returns a page of Profiles belonging to the given workspace, optionally filtered by name.
	RetrieveAll(ctx context.Context, workspaceID string, offset, limit uint64, name string) (ProfilesPage, error)

	// Update updates editable fields of the given Profile and returns the updated Profile.
	Update(ctx context.Context, p Profile) (Profile, error)

	// Delete removes the Profile with the given ID from the given workspace.
	Delete(ctx context.Context, workspaceID, id string) error
}
