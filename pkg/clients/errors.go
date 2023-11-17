// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package clients

import "errors"

var (
	// ErrInvalidStatus indicates invalid status.
	ErrInvalidStatus = errors.New("invalid client status")

	// ErrEnableClient indicates error in enabling client.
	ErrEnableClient = errors.New("failed to enable client")

	// ErrDisableClient indicates error in disabling client.
	ErrDisableClient = errors.New("failed to disable client")
)
