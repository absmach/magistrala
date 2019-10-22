// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package mocks

import (
	"github.com/mainflux/mainflux/users"
	"github.com/mainflux/mainflux/users/jwt"
)

// NewIdentityProvider creates "mirror" identity provider, i.e. generated
// token will hold value provided by the caller.
func NewIdentityProvider() users.IdentityProvider {
	return jwt.New("secret")
}
