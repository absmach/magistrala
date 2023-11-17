// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package twins

import (
	"context"
	"time"
)

// Metadata stores arbitrary twin data.
type Metadata map[string]interface{}

// Attribute stores individual attribute data.
type Attribute struct {
	Name         string `json:"name"`
	Channel      string `json:"channel"`
	Subtopic     string `json:"subtopic"`
	PersistState bool   `json:"persist_state"`
}

// Definition stores entity's attributes.
type Definition struct {
	ID         int         `json:"id"`
	Created    time.Time   `json:"created"`
	Attributes []Attribute `json:"attributes"`
	Delta      int64       `json:"delta"`
}

// Twin is a Magistrala data system representation. Each twin is owned
// by a single user, and is assigned with the unique identifier.
type Twin struct {
	Owner       string
	ID          string
	Name        string
	Created     time.Time
	Updated     time.Time
	Revision    int
	Definitions []Definition
	Metadata    Metadata
}

// PageMetadata contains page metadata that helps navigation.
type PageMetadata struct {
	Total  uint64
	Offset uint64
	Limit  uint64
}

// Page contains page related metadata as well as a list of twins that
// belong to this page.
type Page struct {
	PageMetadata
	Twins []Twin
}

// TwinRepository specifies a twin persistence API.
type TwinRepository interface {
	// Save persists the twin
	Save(ctx context.Context, twin Twin) (string, error)

	// Update performs an update to the existing twin. A non-nil error is
	// returned to indicate operation failure.
	Update(ctx context.Context, twin Twin) error

	// RetrieveByID retrieves the twin having the provided identifier.
	RetrieveByID(ctx context.Context, twinID string) (Twin, error)

	// RetrieveByAttribute retrieves twin ids whose definition contains
	// the attribute with given channel and subtopic
	RetrieveByAttribute(ctx context.Context, channel, subtopic string) ([]string, error)

	// RetrieveAll retrieves the subset of twins owned by the specified user.
	RetrieveAll(ctx context.Context, owner string, offset, limit uint64, name string, metadata Metadata) (Page, error)

	// Remove removes the twin having the provided identifier.
	Remove(ctx context.Context, twinID string) error
}

// TwinCache contains twin caching interface.
type TwinCache interface {
	// Save stores twin ID as element of channel-subtopic keyed set and vice versa.
	Save(ctx context.Context, twin Twin) error

	// SaveIDs stores twin IDs as elements of channel-subtopic keyed set and vice versa.
	SaveIDs(ctx context.Context, channel, subtopic string, twinIDs []string) error

	// Update updates update twin id and channel-subtopic attributes mapping
	Update(ctx context.Context, twin Twin) error

	// ID returns twin IDs for given attribute.
	IDs(ctx context.Context, channel, subtopic string) ([]string, error)

	// Removes twin from cache based on twin id.
	Remove(ctx context.Context, twinID string) error
}
