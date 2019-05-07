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
	ReadAll(string, uint64, uint64, map[string]string) (MessagesPage, error)
}

// MessagesPage contains page related metadata as well as list of messages that
// belong to this page.
type MessagesPage struct {
	Total    uint64
	Offset   uint64
	Limit    uint64
	Messages []mainflux.Message
}
