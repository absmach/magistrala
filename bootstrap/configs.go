// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package bootstrap

import "context"

// Config represents a bootstrap enrollment.
type Config struct {
	ID              string         `json:"id"`
	DomainID        string         `json:"domain_id,omitempty"`
	Name            string         `json:"name,omitempty"`
	ClientCert      string         `json:"client_cert,omitempty"`
	ClientKey       string         `json:"client_key,omitempty"`
	CACert          string         `json:"ca_cert,omitempty"`
	ExternalID      string         `json:"external_id"`
	ExternalKey     string         `json:"external_key"`
	Content         string         `json:"content,omitempty"`
	Status          Status         `json:"status"`
	ProfileID       string         `json:"profile_id,omitempty"`
	RenderContext   map[string]any `json:"render_context,omitempty"`
}

// Filter is used for the search filters.
type Filter struct {
	FullMatch    map[string]string
	PartialMatch map[string]string
}

// ConfigsPage contains page related metadata as well as list of Configs that
// belong to this page.
type ConfigsPage struct {
	Total   uint64   `json:"total"`
	Offset  uint64   `json:"offset"`
	Limit   uint64   `json:"limit"`
	Configs []Config `json:"configs"`
}

// ConfigRepository specifies a Config persistence API.
type ConfigRepository interface {
	// Save persists the Config. Successful operation is indicated by non-nil
	// error response.
	Save(ctx context.Context, cfg Config) (string, error)

	// RetrieveByID retrieves the Config having the provided identifier, that is owned
	// by the specified user.
	RetrieveByID(ctx context.Context, domainID, id string) (Config, error)

	// RetrieveAll retrieves a subset of Configs that are owned
	// by the specific user, with given filter parameters.
	RetrieveAll(ctx context.Context, domainID string, clientIDs []string, filter Filter, offset, limit uint64) ConfigsPage

	// RetrieveByExternalID returns Config for given external ID.
	RetrieveByExternalID(ctx context.Context, externalID string) (Config, error)

	// Update updates an existing Config. A non-nil error is returned
	// to indicate operation failure.
	Update(ctx context.Context, cfg Config) error

	// AssignProfile sets the profile reference for the given Config.
	AssignProfile(ctx context.Context, domainID, id, profileID string) error

	// UpdateCerts updates and returns an existing Config certificate and domainID.
	// A non-nil error is returned to indicate operation failure.
	UpdateCert(ctx context.Context, domainID, clientID, clientCert, clientKey, caCert string) (Config, error)

	// Remove removes the Config having the provided identifier, that is owned
	// by the specified user.
	Remove(ctx context.Context, domainID, id string) error

	// ChangeStatus changes the Status of the Config owned by the specific user.
	ChangeStatus(ctx context.Context, domainID, id string, status Status) error

	// RemoveClient removes Config of the Client with the given ID.
	// Used as a handler for client remove events.
	RemoveClient(ctx context.Context, id string) error
}
