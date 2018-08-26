//
// Copyright (c) 2018
// Mainflux
//
// SPDX-License-Identifier: Apache-2.0
//

//
// Copyright (c) 2018
// Mainflux
//
// SPDX-License-Identifier: Apache-2.0
//

package things

import (
	"strconv"
)

// IdentityProvider specifies an API for generating unique identifiers.
type IdentityProvider interface {
	// ID generates the unique identifier.
	ID() string
}

// FromString extracts an unsigned integer from given string value. Since
// unsigned integers are used for thing and channel identifiers, malformed
// extraction will result in ErrNotFound error.
func FromString(v string) (uint64, error) {
	base := 10
	bitSize := 64

	id, err := strconv.ParseUint(v, base, bitSize)
	if err != nil {
		return 0, ErrNotFound
	}

	return id, nil
}
