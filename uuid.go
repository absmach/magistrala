// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package mainflux

// UUIDProvider specifies an API for generating unique identifiers.
type UUIDProvider interface {
	// ID generates the unique identifier.
	ID() (string, error)
}
