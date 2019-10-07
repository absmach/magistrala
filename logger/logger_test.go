// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

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
var writer mockWriter
var logger log.Logger
var err error
var output logMsg

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

func TestDebug(t *testing.T) {
	cases := map[string]struct {
		input    string
		logLevel string
		output   logMsg
	}{
		"debug log ordinary string":             {"input_string", log.Debug.String(), logMsg{log.Debug.String(), "input_string"}},
		"debug log empty string":                {"", log.Debug.String(), logMsg{log.Debug.String(), ""}},
		"debug ordinary string lvl not allowed": {"input_string", log.Info.String(), logMsg{"", ""}},
		"debug empty string lvl not allowed":    {"", log.Info.String(), logMsg{"", ""}},
	}

	for desc, tc := range cases {
		writer = mockWriter{}
		logger, err = log.New(&writer, tc.logLevel)
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", desc, err))
		logger.Debug(tc.input)
		output, err = writer.Read()
		assert.Equal(t, tc.output, output, fmt.Sprintf("%s: expected %s got %s", desc, tc.output, output))
	}
}

func TestInfo(t *testing.T) {
	cases := map[string]struct {
		input    string
		logLevel string
		output   logMsg
	}{
		"info log ordinary string":             {"input_string", log.Info.String(), logMsg{log.Info.String(), "input_string"}},
		"info log empty string":                {"", log.Info.String(), logMsg{log.Info.String(), ""}},
		"info ordinary string lvl not allowed": {"input_string", log.Warn.String(), logMsg{"", ""}},
		"info empty string lvl not allowed":    {"", log.Warn.String(), logMsg{"", ""}},
	}

	for desc, tc := range cases {
		writer = mockWriter{}
		logger, err = log.New(&writer, tc.logLevel)
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", desc, err))
		logger.Info(tc.input)
		output, err = writer.Read()
		assert.Equal(t, tc.output, output, fmt.Sprintf("%s: expected %s got %s", desc, tc.output, output))
	}
}

func TestWarn(t *testing.T) {
	cases := map[string]struct {
		input    string
		logLevel string
		output   logMsg
	}{
		"warn log ordinary string":             {"input_string", log.Warn.String(), logMsg{log.Warn.String(), "input_string"}},
		"warn log empty string":                {"", log.Warn.String(), logMsg{log.Warn.String(), ""}},
		"warn ordinary string lvl not allowed": {"input_string", log.Error.String(), logMsg{"", ""}},
		"warn empty string lvl not allowed":    {"", log.Error.String(), logMsg{"", ""}},
	}

	for desc, tc := range cases {
		writer = mockWriter{}
		logger, err = log.New(&writer, tc.logLevel)
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", desc, err))
		logger.Warn(tc.input)
		output, err = writer.Read()
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
	logger, _ := log.New(&writer, log.Error.String())

	for desc, tc := range cases {
		logger.Error(tc.input)
		output, err := writer.Read()
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", desc, err))
		assert.Equal(t, tc.output, output, fmt.Sprintf("%s: expected %s got %s", desc, tc.output, output))
	}
}
