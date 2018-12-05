//
// Copyright (c) 2018
// Mainflux
//
// SPDX-License-Identifier: Apache-2.0
//

package readers

import (
	"errors"

	"github.com/mainflux/mainflux"
)

// ErrNotFound indicates that requested entity doesn't exist.
var ErrNotFound = errors.New("entity not found")

// MessageRepository specifies message reader API.
type MessageRepository interface {
	// ReadAll skips given number of messages for given channel and returns next
	// limited number of messages.
	ReadAll(string, uint64, uint64) []mainflux.Message
}
