// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package logger_test

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"testing"

	log "github.com/mainflux/mainflux/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Env vars needed for testing Fatal in subprocess.
const (
	testMsg     = "TEST_MSG"
	testFlag    = "TEST_FLAG"
	testFlagVal = "assert_test"
)

var _ io.Writer = (*mockWriter)(nil)
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
	Fatal   string `json:"fatal,omitempty"` // needed for Fatal messages
}

func TestDebug(t *testing.T) {
	cases := []struct {
		desc   string
		input  string
		level  string
		output logMsg
	}{
		{
			desc:   "debug log ordinary string",
			input:  "input_string",
			level:  log.Debug.String(),
			output: logMsg{log.Debug.String(), "input_string", ""},
		},
		{
			desc:   "debug log empty string",
			input:  "",
			level:  log.Debug.String(),
			output: logMsg{log.Debug.String(), "", ""},
		},
		{
			desc:   "debug ordinary string lvl not allowed",
			input:  "input_string",
			level:  log.Info.String(),
			output: logMsg{"", "", ""},
		},
		{
			desc:   "debug empty string lvl not allowed",
			input:  "",
			level:  log.Info.String(),
			output: logMsg{"", "", ""},
		},
	}

	for _, tc := range cases {
		writer := mockWriter{}
		logger, err = log.New(&writer, tc.level)
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
		logger.Debug(tc.input)
		output, err = writer.Read()
		assert.Equal(t, tc.output, output, fmt.Sprintf("%s: expected %s got %s", tc.desc, tc.output, output))
	}
}

func TestInfo(t *testing.T) {
	cases := []struct {
		desc   string
		input  string
		level  string
		output logMsg
	}{
		{
			desc:   "info log ordinary string",
			input:  "input_string",
			level:  log.Info.String(),
			output: logMsg{log.Info.String(), "input_string", ""},
		},
		{
			desc:   "info log empty string",
			input:  "",
			level:  log.Info.String(),
			output: logMsg{log.Info.String(), "", ""},
		},
		{
			desc:   "info ordinary string lvl not allowed",
			input:  "input_string",
			level:  log.Warn.String(),
			output: logMsg{"", "", ""},
		},
		{
			desc:   "info empty string lvl not allowed",
			input:  "",
			level:  log.Warn.String(),
			output: logMsg{"", "", ""},
		},
	}

	for _, tc := range cases {
		writer := mockWriter{}
		logger, err = log.New(&writer, tc.level)
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
		logger.Info(tc.input)
		output, err = writer.Read()
		assert.Equal(t, tc.output, output, fmt.Sprintf("%s: expected %s got %s", tc.desc, tc.output, output))
	}
}

func TestWarn(t *testing.T) {
	cases := []struct {
		desc   string
		input  string
		level  string
		output logMsg
	}{
		{
			desc:   "warn log ordinary string",
			input:  "input_string",
			level:  log.Warn.String(),
			output: logMsg{log.Warn.String(), "input_string", ""},
		},
		{
			desc:   "warn log empty string",
			input:  "",
			level:  log.Warn.String(),
			output: logMsg{log.Warn.String(), "", ""},
		},
		{
			desc:   "warn ordinary string lvl not allowed",
			input:  "input_string",
			level:  log.Error.String(),
			output: logMsg{"", "", ""},
		},
		{
			desc:   "warn empty string lvl not allowed",
			input:  "",
			level:  log.Error.String(),
			output: logMsg{"", "", ""},
		},
	}

	for _, tc := range cases {
		writer := mockWriter{}
		logger, err = log.New(&writer, tc.level)
		require.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
		logger.Warn(tc.input)
		output, err = writer.Read()
		assert.Equal(t, tc.output, output, fmt.Sprintf("%s: expected %s got %s", tc.desc, tc.output, output))
	}
}

func TestError(t *testing.T) {
	cases := []struct {
		desc   string
		input  string
		output logMsg
	}{
		{
			desc:   "error log ordinary string",
			input:  "input_string",
			output: logMsg{log.Error.String(), "input_string", ""},
		},
		{
			desc:   "error log empty string",
			input:  "",
			output: logMsg{log.Error.String(), "", ""},
		},
	}

	writer := mockWriter{}
	logger, err := log.New(&writer, log.Error.String())
	require.Nil(t, err)
	for _, tc := range cases {
		logger.Error(tc.input)
		output, err := writer.Read()
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
		assert.Equal(t, tc.output, output, fmt.Sprintf("%s: expected %s got %s", tc.desc, tc.output, output))
	}
}

func TestFatal(t *testing.T) {
	// This is the actually Fatal call we test that will
	// be executed in the subprocess spawned by the test.
	if os.Getenv(testFlag) == testFlagVal {
		logger, err := log.New(os.Stderr, log.Error.String())
		require.Nil(t, err)
		msg := os.Getenv(testMsg)
		logger.Fatal(msg)
		return
	}

	cases := []struct {
		desc   string
		input  string
		output logMsg
	}{
		{
			desc:   "error log ordinary string",
			input:  "input_string",
			output: logMsg{"", "", "input_string"},
		},
		{
			desc:   "error log empty string",
			input:  "",
			output: logMsg{"", "", ""},
		},
	}
	writer := mockWriter{}
	for _, tc := range cases {
		// This command will run this same test as a separate subprocess.
		// It needs to be executed as a subprocess because we need to test os.Exit(1) call.
		cmd := exec.Command(os.Args[0], "-test.run=TestFatal")
		// This flag is used to prevent an infinite loop of spawning this test and never
		// actually running the necessary Fatal call.
		cmd.Env = append(os.Environ(), fmt.Sprintf("%s=%s", testFlag, testFlagVal))
		cmd.Stderr = &writer
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", testMsg, tc.input))
		err := cmd.Run()
		if e, ok := err.(*exec.ExitError); ok && !e.Success() {
			res, err := writer.Read()
			require.Nil(t, err, "required successful buffer read")
			assert.Equal(t, 1, e.ExitCode(), fmt.Sprintf("%s: expected exit code %d, got %d", tc.desc, 1, e.ExitCode()))
			assert.Equal(t, tc.output, res, fmt.Sprintf("%s: expected output %s got %s", tc.desc, tc.output, res))
			continue
		}
		t.Fatal("subprocess ran successfully, want non-zero exit status")
	}
}
