//
// Copyright (c) 2018
// Mainflux
//
// SPDX-License-Identifier: Apache-2.0
//

package logger_test

import (
	"encoding/json"
	"fmt"
	"io"
	"testing"

	log "github.com/mainflux/mainflux/logger"
	"github.com/stretchr/testify/assert"
)

var _ io.Writer = (*mockWriter)(nil)

type mockWriter struct {
	value []byte
}

func (writer *mockWriter) Write(p []byte) (int, error) {
	writer.value = p
	return len(p), nil
}

func (writer *mockWriter) Read() (logMsg, error) {
	var output logMsg
	err := json.Unmarshal(writer.value, &output)
	return output, err
}

type logMsg struct {
	Level   string `json:"level"`
	Message string `json:"message"`
}

func TestInfo(t *testing.T) {
	cases := map[string]struct {
		input  string
		output logMsg
	}{
		"info log ordinary string": {"input_string", logMsg{log.Info.String(), "input_string"}},
		"info log empty string":    {"", logMsg{log.Info.String(), ""}},
	}

	writer := mockWriter{}
	logger := log.New(&writer)

	for desc, tc := range cases {
		logger.Info(tc.input)
		output, err := writer.Read()
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", desc, err))
		assert.Equal(t, tc.output, output, fmt.Sprintf("%s: expected %s got %s", desc, tc.output, output))
	}
}

func TestWarn(t *testing.T) {
	cases := map[string]struct {
		input  string
		output logMsg
	}{
		"warn log ordinary string": {"input_string", logMsg{log.Warn.String(), "input_string"}},
		"warn log empty string":    {"", logMsg{log.Warn.String(), ""}},
	}

	writer := mockWriter{}
	logger := log.New(&writer)

	for desc, tc := range cases {
		logger.Warn(tc.input)
		output, err := writer.Read()
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", desc, err))
		assert.Equal(t, tc.output, output, fmt.Sprintf("%s: expected %s got %s", desc, tc.output, output))
	}
}

func TestError(t *testing.T) {
	cases := map[string]struct {
		input  string
		output logMsg
	}{
		"error log ordinary string": {"input_string", logMsg{log.Error.String(), "input_string"}},
		"error log empty string":    {"", logMsg{log.Error.String(), ""}},
	}

	writer := mockWriter{}
	logger := log.New(&writer)

	for desc, tc := range cases {
		logger.Error(tc.input)
		output, err := writer.Read()
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", desc, err))
		assert.Equal(t, tc.output, output, fmt.Sprintf("%s: expected %s got %s", desc, tc.output, output))
	}
}
