// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package logger

import "context"

var _ Logger = (*loggerMock)(nil)

type loggerMock struct{}

// NewMock returns wrapped go kit logger mock.
func NewMock() Logger {
	return &loggerMock{}
}

func (l loggerMock) Debug(ctx context.Context, msg string) {
}

func (l loggerMock) Info(ctx context.Context, msg string) {
}

func (l loggerMock) Warn(ctx context.Context, msg string) {
}

func (l loggerMock) Error(ctx context.Context, msg string) {
}

func (l loggerMock) Fatal(ctx context.Context, msg string) {
}
