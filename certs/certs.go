// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package certs

import (
	"context"
)

// ConfigsPage contains page related metadata as well as list
type Page struct {
	Total  uint64
	Offset uint64
	Limit  int64
	Certs  []Cert
}

type Status int

const (
	AllCerts Status = iota
	ActiveCerts
	RevokedCerts
)

var StringToStatus = map[string]Status{
	"all":    AllCerts,
	"active": ActiveCerts,
	"revoke": RevokedCerts,
}

// Repository specifies a Config persistence API.
type Repository interface {
	// Save  saves cert for thing into database
	Save(ctx context.Context, cert Cert) error

	// Retrieve issued certificates for given owner ID with given any one of the following parameter
	// certificate id , certificate name, thing ID and certificate serial
	// If all the parameter given, all the condition are added in WHERE CLAUSE with AND condition
	// Example to retrieve only certificate with ID Retrieve(ctx, ownerID, certID, "", "", "", AllCerts, 0, 1)
	// Example to retrieve by Thing ID Retrieve(ctx, ownerID, "", thingID, "", "", AllCerts, 0, 10)
	// Example to retrieve only certificate with serial number Retrieve(ctx, ownerID, "", "", "", serial, AllCerts, 0, 1)
	Retrieve(ctx context.Context, ownerID, certID, thingID, serial, name string, status Status, offset uint64, limit int64) (Page, error)

	// RetrieveCount get count of certificate revoked if revokeCount parameter is true and also count of certificate not revoked if revokeCount parameter is false
	RetrieveCount(ctx context.Context, ownerID, certID, thingID, serial, name string, status Status) (uint64, error)

	// Update certificate from DB for a given certificate ID
	Update(ctx context.Context, ownerID string, cert Cert) error

	// Remove removes certificate from DB for a given certificate ID
	Remove(ctx context.Context, ownerID, certID string) error

	// RetrieveThingCerts retrieves all the certificate for the given thing ID , which doesn't required owner ID, used for thing removed event stream handler
	RetrieveThingCerts(ctx context.Context, thingID string) (Page, error)

	// RemoveThingCerts removes all the certificate for the given thing ID , which doesn't required owner ID, used for thing removed event stream handler
	RemoveThingCerts(ctx context.Context, thingID string) error
}
