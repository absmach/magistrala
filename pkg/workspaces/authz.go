// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package workspaces

import (
	"context"
)

type Status uint8

const (
	EnabledStatus Status = iota
	DisabledStatus
	FreezeStatus
	AllStatus
)

type Authorization interface {
	RetrieveStatus(ctx context.Context, id string) (Status, error)
}
