// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package opcua

import "context"

// RouteMapRepository store route-map between the OPC-UA Server and Magistrala.
type RouteMapRepository interface {
	// Save stores/routes pair OPC-UA Server & Magistrala.
	Save(context.Context, string, string) error

	// Get returns the stored Magistrala route-map for a given OPC-UA pair.
	Get(context.Context, string) (string, error)

	// Remove route-map from cache.
	Remove(context.Context, string) error
}
