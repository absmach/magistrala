// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package auth

// Tokenizer specifies API for encoding and decoding between string and Key.
type Tokenizer interface {
	// Issue converts API Key to its string representation.
	Issue(key Key) (token string, err error)

	// Parse extracts API Key data from string token.
	Parse(token string) (key Key, err error)
}
