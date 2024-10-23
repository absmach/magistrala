// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package things

import (
	"context"

	"github.com/absmach/magistrala/pkg/postgres"
)

type ThingRepository struct {
	DB postgres.Database
}

// Repository is the interface that wraps the basic methods for
// a client repository.
//
//go:generate mockery --name Repository --output=./mocks --filename repository.go --quiet --note "Copyright (c) Abstract Machines"
type Repository interface {
	// RetrieveByID retrieves thing by its unique ID.
	RetrieveByID(ctx context.Context, id string) (Thing, error)

	// RetrieveAll retrieves all things.
	RetrieveAll(ctx context.Context, pm Page) (ThingsPage, error)

	// SearchThings retrieves things based on search criteria.
	SearchThings(ctx context.Context, pm Page) (ThingsPage, error)

	// RetrieveAllByIDs retrieves for given thing IDs .
	RetrieveAllByIDs(ctx context.Context, pm Page) (ThingsPage, error)

	// Update updates the thing name and metadata.
	Update(ctx context.Context, thing Thing) (Thing, error)

	// UpdateTags updates the thing tags.
	UpdateTags(ctx context.Context, thing Thing) (Thing, error)

	// UpdateIdentity updates identity for thing with given id.
	UpdateIdentity(ctx context.Context, thing Thing) (Thing, error)

	// UpdateSecret updates secret for thing with given identity.
	UpdateSecret(ctx context.Context, thing Thing) (Thing, error)

	// ChangeStatus changes thing status to enabled or disabled
	ChangeStatus(ctx context.Context, thing Thing) (Thing, error)

	// Delete deletes thing with given id
	Delete(ctx context.Context, id string) error

	// Save persists the thing account. A non-nil error is returned to indicate
	// operation failure.
	Save(ctx context.Context, thing ...Thing) ([]Thing, error)

	// RetrieveBySecret retrieves a thing based on the secret (key).
	RetrieveBySecret(ctx context.Context, key string) (Thing, error)
}
