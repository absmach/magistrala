// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package logger

import "log/slog"

type RunInfo struct {
	Level   slog.Level
	Details []slog.Attr
	Message string
}
