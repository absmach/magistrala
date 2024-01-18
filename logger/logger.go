// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package logger

import (
	"fmt"
	"io"
	"log/slog"
	"time"
)

// New returns wrapped slog logger.
func New(w io.Writer, levelText string) (*slog.Logger, error) {
	var level slog.Level
	if err := level.UnmarshalText([]byte(levelText)); err != nil {
		return &slog.Logger{}, fmt.Errorf(`{"level":"error","message":"%s: %s","ts":"%s"}`, err, levelText, time.RFC3339Nano)
	}

	logHandler := slog.NewJSONHandler(w, &slog.HandlerOptions{
		Level: level,
	})

	return slog.New(logHandler), nil
}
