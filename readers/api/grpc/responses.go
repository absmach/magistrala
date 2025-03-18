// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package grpc

import (

	"github.com/absmach/supermq/readers"
)

type readMessagesRes struct {
	Total uint64 `json:"total"`
	Messages []readers.Message `json:"messages"`
	readers.PageMetadata
}

type Message interface{}
