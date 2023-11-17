// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package magistrala

// IDProvider specifies an API for generating unique identifiers.
type IDProvider interface {
	// ID generates the unique identifier.
	ID() (string, error)
}
