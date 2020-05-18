// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package twins

import (
	"context"
	"time"
)

// Metadata stores arbitrary twin data
type Metadata map[string]interface{}

// Attribute stores individual attribute data
type Attribute struct {
	Name         string `json:"name"`
	Channel      string `json:"channel"`
	Subtopic     string `json:"subtopic"`
	PersistState bool   `json:"persist_state"`
}

// Definition stores entity's attributes
type Definition struct {
	ID         int         `json:"id"`
	Created    time.Time   `json:"created"`
	Attributes []Attribute `json:"attributes"`
	Delta      int64       `json:"delta"`
}

// Twin is a Mainflux data system representation. Each twin is owned
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
	Save(context.Context, Twin) (string, error)

	// Update performs an update to the existing twin. A non-nil error is
	// returned to indicate operation failure.
	Update(context.Context, Twin) error

	// RetrieveByID retrieves the twin having the provided identifier.
	RetrieveByID(ctx context.Context, id string) (Twin, error)

	// RetrieveByAttribute retrieves twin ids whose definition contains
	// the attribute with given channel and subtopic
	RetrieveByAttribute(ctx context.Context, channel, subtopic string) ([]string, error)

	// RetrieveAll retrieves the subset of twins owned by the specified user.
	RetrieveAll(context.Context, string, uint64, uint64, string, Metadata) (Page, error)

	// Remove removes the twin having the provided identifier.
	Remove(ctx context.Context, id string) error
}
