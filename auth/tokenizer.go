// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package auth

import "context"

// Tokenizer specifies API for encoding and decoding between string and Key.
type Tokenizer interface {
	// Issue converts API Key to its string representation.
	Issue(key Key) (token string, err error)

	// Parse extracts API Key data from string token.
	Parse(ctx context.Context, token string) (key Key, err error)

	// Revoke revokes the token.
	Revoke(ctx context.Context, token string) error
}

// TokenRepository specifies token persistence API.
//
//go:generate mockery --name TokenRepository --output=./mocks --filename token.go --quiet --note "Copyright (c) Abstract Machines"
type TokenRepository interface {
	// Save persists the token.
	Save(ctx context.Context, id string) (err error)

	// Contains checks if token with provided ID exists.
	Contains(ctx context.Context, id string) (ok bool)
}
