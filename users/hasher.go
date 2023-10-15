// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package users

// Hasher specifies an API for generating hashes of an arbitrary textual
// content.
type Hasher interface {
	// Hash generates the hashed string from plain-text.
	Hash(string) (string, error)

	// Compare compares plain-text version to the hashed one. An error should
	// indicate failed comparison.
	Compare(string, string) error
}
