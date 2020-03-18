// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package twins

import (
	"context"
	"time"
)

// State stores actual snapshot of entity's values
type State struct {
	TwinID     string
	ID         int64
	Definition int
	Created    time.Time
	Payload    map[string]interface{}
}

// StatesPage contains page related metadata as well as a list of twins that
// belong to this page.
type StatesPage struct {
	PageMetadata
	States []State
}

// StateRepository specifies a state persistence API.
type StateRepository interface {
	// Save persists the state
	Save(context.Context, State) error

	// Update updates the state
	Update(context.Context, State) error

	// Count returns the number of states related to state
	Count(context.Context, Twin) (int64, error)

	// RetrieveAll retrieves the subset of states related to twin specified by id
	RetrieveAll(ctx context.Context, offset uint64, limit uint64, id string) (StatesPage, error)

	// RetrieveLast retrieves the last saved state
	RetrieveLast(ctx context.Context, id string) (State, error)
}
