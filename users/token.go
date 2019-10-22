// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package users

// Tokenizer specifies API for password reset token manipulation
type Tokenizer interface {

	// Generate generate new random token. Offset can be used to
	// manipulate token validity in time useful for testing.
	Generate(email string, offset int) (string, error)

	// Verify verifies token validity
	Verify(tok string) (string, error)
}
