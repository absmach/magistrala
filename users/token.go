// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package users

import "github.com/mainflux/mainflux/errors"

// Tokenizer specifies API for password reset token manipulation
type Tokenizer interface {

	// Generate generate new random token. Offset can be used to
	// manipulate token validity in time useful for testing.
	Generate(email string, offset int) (string, errors.Error)

	// Verify verifies token validity
	Verify(tok string) (string, errors.Error)
}
