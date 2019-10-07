// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package logger

import (
	"fmt"
	"github.com/go-kit/kit/log"
	"io"
	"time"
)

// Logger specifies logging API.
type Logger interface {
	// Debug logs any object in JSON format on debug level.
	Debug(string)
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
	level     Level
}

// New returns wrapped go kit logger.
func New(out io.Writer, levelText string) (Logger, error) {
	var level Level
	err := level.UnmarshalText(levelText)
	if err != nil {
		return nil, fmt.Errorf(`{"level":"error","message":"%s: %s","ts":"%s"}`, err, levelText, time.RFC3339Nano)
	}
	l := log.NewJSONLogger(log.NewSyncWriter(out))
	l = log.With(l, "ts", log.DefaultTimestampUTC)
	return &logger{l, level}, err
}

func (l logger) Debug(msg string) {
	if Debug.isAllowed(l.level) {
		l.kitLogger.Log("level", Debug.String(), "message", msg)
	}
}

func (l logger) Info(msg string) {
	if Info.isAllowed(l.level) {
		l.kitLogger.Log("level", Info.String(), "message", msg)
	}
}

func (l logger) Warn(msg string) {
	if Warn.isAllowed(l.level) {
		l.kitLogger.Log("level", Warn.String(), "message", msg)
	}
}

func (l logger) Error(msg string) {
	if Error.isAllowed(l.level) {
		l.kitLogger.Log("level", Error.String(), "message", msg)
	}
}
