// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package mainflux

// IDProvider specifies an API for generating unique identifiers.
type IDProvider interface {
	// ID generates the unique identifier.
	ID() (string, error)
}
