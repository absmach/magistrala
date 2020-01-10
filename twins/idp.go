// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package twins

// IdentityProvider specifies an API for generating unique identifiers.
type IdentityProvider interface {
	// ID generates the unique identifier.
	ID() (string, error)

	// IsValid checks whether string is a valid uuid4.
	IsValid(u4 string) error
}
