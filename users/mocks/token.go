// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package mocks

import (
	"github.com/mainflux/mainflux/users"
	"github.com/mainflux/mainflux/users/token"
)

// NewTokenizer provides tokenizer for the test
func NewTokenizer() users.Tokenizer {
	return token.New([]byte("secret"), 1)
}
