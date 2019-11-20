// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package users

import "github.com/mainflux/mainflux/errors"

// Hasher specifies an API for generating hashes of an arbitrary textual
// content.
type Hasher interface {
	// Hash generates the hashed string from plain-text.
	Hash(string) (string, errors.Error)

	// Compare compares plain-text version to the hashed one. An error should
	// indicate failed comparison.
	Compare(string, string) errors.Error
}
