// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package logger

import (
	"errors"
	"strings"
)

const (
	// Error level is used when logging errors.
	Error Level = iota + 1
	// Warn level is used when logging warnings.
	Warn
	// Info level is used when logging info data.
	Info
	// Debug level is used when logging debugging info.
	Debug
)

var ErrInvalidLogLevel = errors.New("unrecognized log level")

// Level represents severity level while logging.
type Level int

var levels = map[Level]string{
	Error: "error",
	Warn:  "warn",
	Info:  "info",
	Debug: "debug",
}

func (lvl Level) String() string {
	return levels[lvl]
}

func (lvl Level) isAllowed(logLevel Level) bool {
	return lvl <= logLevel
}

func (lvl *Level) UnmarshalText(text string) error {
	switch string(strings.ToLower(text)) {
	case "debug":
		*lvl = Debug
	case "info":
		*lvl = Info
	case "warn":
		*lvl = Warn
	case "error":
		*lvl = Error
	default:
		return ErrInvalidLogLevel
	}
	return nil
}
