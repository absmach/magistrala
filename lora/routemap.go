// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package lora

import "context"

// RouteMapRepository store route map between Lora App Server and Mainflux
type RouteMapRepository interface {
	// Save stores/routes pair lora application topic & mainflux channel.
	Save(context.Context, string, string) error

	// Channel returns mainflux channel for given lora application.
	Get(context.Context, string) (string, error)

	// Removes mapping from cache.
	Remove(context.Context, string) error
}
