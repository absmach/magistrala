// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package certs

import "context"

// ConfigsPage contains page related metadata as well as list
type Page struct {
	Total  uint64
	Offset uint64
	Limit  uint64
	Certs  []Cert
}

// Repository specifies a Config persistence API.
type Repository interface {
	// Save  saves cert for thing into database
	Save(ctx context.Context, cert Cert) (string, error)

	// RetrieveAll retrieve all issued certificates for given owner and thing id
	RetrieveAll(ctx context.Context, ownerID, thingID string, offset, limit uint64) (Page, error)

	// Remove certificate from DB for given thing
	Remove(ctx context.Context, thingID string) error

	// RetrieveByThing certificate by given thing
	RetrieveByThing(ctx context.Context, thingID string) (Cert, error)
}
