// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package mocks

import (
	"bytes"
	"log/slog"
)

// NewMock returns wrapped slog logger mock.
func NewMock() *slog.Logger {
	buf := &bytes.Buffer{}

	return slog.New(slog.NewJSONHandler(buf, nil))
}
