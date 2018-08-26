//
// Copyright (c) 2018
// Mainflux
//
// SPDX-License-Identifier: Apache-2.0
//

package logger

import (
	"io"

	"github.com/go-kit/kit/log"
)

// Logger specifies logging API.
type Logger interface {
	// Info logs any object in JSON format on info level.
	Info(string)
	// Warn logs any object in JSON format on warning level.
	Warn(string)
	// Error logs any object in JSON format on error level.
	Error(string)
}

var _ Logger = (*logger)(nil)

type logger struct {
	kitLogger log.Logger
}

// New returns wrapped go kit logger.
func New(out io.Writer) Logger {
	l := log.NewJSONLogger(log.NewSyncWriter(out))
	l = log.With(l, "ts", log.DefaultTimestampUTC)
	return &logger{l}
}

func (l logger) Info(msg string) {
	l.kitLogger.Log("level", Info.String(), "message", msg)
}

func (l logger) Warn(msg string) {
	l.kitLogger.Log("level", Warn.String(), "message", msg)
}

func (l logger) Error(msg string) {
	l.kitLogger.Log("level", Error.String(), "message", msg)
}
