//
// Copyright (c) 2018
// Mainflux
//
// SPDX-License-Identifier: Apache-2.0
//

package users

// IdentityProvider specifies an API for identity management via security
// tokens.
type IdentityProvider interface {
	// TemporaryKey generates the temporary access token.
	TemporaryKey(string) (string, error)

	// Identity extracts the entity identifier given its secret key.
	Identity(string) (string, error)
}
