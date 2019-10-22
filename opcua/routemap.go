// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package opcua

// RouteMapRepository store route-map between the OPC-UA Server and Mainflux
type RouteMapRepository interface {
	// Save stores/routes pair OPC-UA Server & Mainflux.
	Save(string, string) error

	// Get returns the stored Mainflux route-map for a given OPC-UA pair.
	Get(string) (string, error)

	// Remove Remove route-map from cache.
	Remove(string) error
}
