// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package logger

var _ Logger = (*loggerMock)(nil)

type loggerMock struct{}

// NewMock returns wrapped go kit logger mock.
func NewMock() Logger {
	return &loggerMock{}
}

func (l loggerMock) Debug(msg string) {
}

func (l loggerMock) Info(msg string) {
}

func (l loggerMock) Warn(msg string) {
}

func (l loggerMock) Error(msg string) {
}
