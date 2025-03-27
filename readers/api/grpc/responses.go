// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package grpc

import (
	"github.com/absmach/supermq/readers"
)

type readMessagesRes struct {
	Total    uint64
	Messages []readers.Message
	readers.PageMetadata
}

type Message interface{}
