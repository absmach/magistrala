// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package users

import "github.com/mainflux/mainflux/errors"

// IdentityProvider specifies an API for identity management via security
// tokens.
type IdentityProvider interface {
	// TemporaryKey generates the temporary access token.
	TemporaryKey(string) (string, errors.Error)

	// Identity extracts the entity identifier given its secret key.
	Identity(string) (string, errors.Error)
}
