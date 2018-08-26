//
// Copyright (c) 2018
// Mainflux
//
// SPDX-License-Identifier: Apache-2.0
//

package logger

const (
	// Error level is used when logging errors.
	Error Level = iota + 1
	// Warn level is used when logging warnings.
	Warn
	// Info level is used when logging info data.
	Info
)

// Level represents severity level while logging.
type Level int

var levels = map[Level]string{
	Error: "error",
	Warn:  "warn",
	Info:  "info",
}

func (lvl Level) String() string {
	return levels[lvl]
}
