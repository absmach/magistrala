// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package logger

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestUnmarshalText(t *testing.T) {
	cases := []struct {
		desc   string
		input  string
		output Level
		err    error
	}{
		{
			desc:   "select log level Not_A_Level",
			input:  "Not_A_Level",
			output: 0,
			err:    ErrInvalidLogLevel,
		},
		{
			desc:   "select log level Bad_Input",
			input:  "Bad_Input",
			output: 0,
			err:    ErrInvalidLogLevel,
		},

		{
			desc:   "select log level debug",
			input:  "debug",
			output: Debug,
			err:    nil,
		},
		{
			desc:   "select log level DEBUG",
			input:  "DEBUG",
			output: Debug,
			err:    nil,
		},
		{
			desc:   "select log level info",
			input:  "info",
			output: Info,
			err:    nil,
		},
		{
			desc:   "select log level INFO",
			input:  "INFO",
			output: Info,
			err:    nil,
		},
		{
			desc:   "select log level warn",
			input:  "warn",
			output: Warn,
			err:    nil,
		},
		{
			desc:   "select log level WARN",
			input:  "WARN",
			output: Warn,
			err:    nil,
		},
		{
			desc:   "select log level Error",
			input:  "Error",
			output: Error,
			err:    nil,
		},
		{
			desc:   "select log level ERROR",
			input:  "ERROR",
			output: Error,
			err:    nil,
		},
	}

	for _, tc := range cases {
		var logLevel Level
		err := logLevel.UnmarshalText(tc.input)
		assert.Equal(t, tc.output, logLevel, fmt.Sprintf("%s: expected %s got %d", tc.desc, tc.output, logLevel))
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected %s got %d", tc.desc, tc.err, err))
	}
}

func TestLevelIsAllowed(t *testing.T) {
	cases := []struct {
		desc           string
		requestedLevel Level
		allowedLevel   Level
		output         bool
	}{
		{
			desc:           "log debug when level debug",
			requestedLevel: Debug,
			allowedLevel:   Debug,
			output:         true,
		},
		{
			desc:           "log info when level debug",
			requestedLevel: Info,
			allowedLevel:   Debug,
			output:         true,
		},
		{
			desc:           "log warn when level debug",
			requestedLevel: Warn,
			allowedLevel:   Debug,
			output:         true,
		},
		{
			desc:           "log error when level debug",
			requestedLevel: Error,
			allowedLevel:   Debug,
			output:         true,
		},
		{
			desc:           "log warn when level info",
			requestedLevel: Warn,
			allowedLevel:   Info,
			output:         true,
		},
		{
			desc:           "log error when level warn",
			requestedLevel: Error,
			allowedLevel:   Warn,
			output:         true,
		},
		{
			desc:           "log error when level error",
			requestedLevel: Error,
			allowedLevel:   Error,
			output:         true,
		},

		{
			desc:           "log debug when level error",
			requestedLevel: Debug,
			allowedLevel:   Error,
			output:         false,
		},
		{
			desc:           "log info when level error",
			requestedLevel: Info,
			allowedLevel:   Error,
			output:         false,
		},
		{
			desc:           "log warn when level error",
			requestedLevel: Warn,
			allowedLevel:   Error,
			output:         false,
		},
		{
			desc:           "log debug when level warn",
			requestedLevel: Debug,
			allowedLevel:   Warn,
			output:         false,
		},
		{
			desc:           "log info when level warn",
			requestedLevel: Info,
			allowedLevel:   Warn,
			output:         false,
		},
		{
			desc:           "log debug when level info",
			requestedLevel: Debug,
			allowedLevel:   Info,
			output:         false,
		},
	}
	for _, tc := range cases {
		result := tc.requestedLevel.isAllowed(tc.allowedLevel)
		assert.Equal(t, tc.output, result, fmt.Sprintf("%s: expected %t got %t", tc.desc, tc.output, result))
	}
}
