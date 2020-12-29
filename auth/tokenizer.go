// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package auth

// Tokenizer specifies API for encoding and decoding between string and Key.
type Tokenizer interface {
	// Issue converts API Key to its string representation.
	Issue(Key) (string, error)

	// Parse extracts API Key data from string token.
	Parse(string) (Key, error)
}
