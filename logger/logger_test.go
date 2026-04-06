// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package logger_test

import (
	"log/slog"
	"testing"

	mglog "github.com/absmach/magistrala/logger"
	"github.com/stretchr/testify/assert"
)

type mockWriter struct {
	value []byte
}

func (writer *mockWriter) Write(p []byte) (int, error) {
	writer.value = p
	return len(p), nil
}

func TestLoggerInitialization(t *testing.T) {
	cases := []struct {
		desc  string
		level string
	}{
		{
			desc:  "debug level",
			level: slog.LevelDebug.String(),
		},
		{
			desc:  "info level",
			level: slog.LevelInfo.String(),
		},
		{
			desc:  "warn level",
			level: slog.LevelWarn.String(),
		},
		{
			desc:  "error level",
			level: slog.LevelError.String(),
		},
		{
			desc:  "invalid level",
			level: "invalid",
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			writer := &mockWriter{}
			logger, err := mglog.New(writer, tc.level)
			if tc.level == "invalid" {
				assert.NotNil(t, err, "expected error during logger initialization")
				assert.NotNil(t, logger, "logger should not be nil when an error occurs")
			} else {
				assert.Nil(t, err, "unexpected error during logger initialization")
				assert.NotNil(t, logger, "logger should not be nil")
			}
		})
	}
}
