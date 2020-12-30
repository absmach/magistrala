// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package readers

import "errors"

// ErrNotFound indicates that requested entity doesn't exist.
var ErrNotFound = errors.New("entity not found")

// MessageRepository specifies message reader API.
type MessageRepository interface {
	// ReadAll skips given number of messages for given channel and returns next
	// limited number of messages.
	ReadAll(chanID string, offset, limit uint64, query map[string]string) (MessagesPage, error)
}

// Message represents any message format.
type Message interface{}

// MessagesPage contains page related metadata as well as list of messages that
// belong to this page.
type MessagesPage struct {
	Total    uint64
	Offset   uint64
	Limit    uint64
	Messages []Message
}
