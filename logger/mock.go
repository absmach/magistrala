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
	return
}

func (l loggerMock) Info(msg string) {
	return
}

func (l loggerMock) Warn(msg string) {
	return
}

func (l loggerMock) Error(msg string) {
	return
}
