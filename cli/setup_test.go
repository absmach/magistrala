// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package cli_test

import (
	"bytes"
	"encoding/json"
	"os"
	"testing"

	"github.com/absmach/magistrala/cli"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
)

type outputLog uint8

const (
	usageLog outputLog = iota
	errLog
	entityLog
	okLog
)

func executeCommand(t *testing.T, root *cobra.Command, entity any, logMessage outputLog, args ...string) (out string) {
	buffer := new(bytes.Buffer)
	root.SetOut(buffer)
	root.SetErr(buffer)
	root.SetArgs(args)

	r, w, err := os.Pipe()
	assert.NoError(t, err, "Error creating pipe")
	r1, w1, err := os.Pipe()
	assert.NoError(t, err, "Error creating pipe")

	os.Stdout = w
	os.Stderr = w1

	_, err = root.ExecuteC()
	assert.NoError(t, err, "Error executing command")

	w.Close()
	w1.Close()

	var outputBuffer bytes.Buffer
	switch logMessage {
	case usageLog, okLog:
		_, err = outputBuffer.ReadFrom(r)
		assert.NoError(t, err, "Error reading from pipe")
		return outputBuffer.String()
	case errLog:
		var errBufffer bytes.Buffer
		_, err = errBufffer.ReadFrom(r1)
		assert.NoError(t, err, "Error reading from pipe")
		return errBufffer.String()
	case entityLog:
		_, err = outputBuffer.ReadFrom(r)
		assert.NoError(t, err, "Error reading from pipe")
		res := outputBuffer.Bytes()
		assert.Greater(t, len(res), 0, "Error reading from pipe")
		err = json.Unmarshal(res, entity)
		assert.NoError(t, err, "Error unmarshalling entity")
	default:
		return ""
	}

	return ""
}

func setFlags(rootCmd *cobra.Command) *cobra.Command {
	// Root Flags
	rootCmd.PersistentFlags().BoolVarP(
		&cli.RawOutput,
		"raw",
		"r",
		cli.RawOutput,
		"Enables raw output mode for easier parsing of output",
	)

	// Client and Channels Flags
	rootCmd.PersistentFlags().Uint64VarP(
		&cli.Limit,
		"limit",
		"l",
		10,
		"Limit query parameter",
	)

	rootCmd.PersistentFlags().Uint64VarP(
		&cli.Offset,
		"offset",
		"o",
		0,
		"Offset query parameter",
	)

	rootCmd.PersistentFlags().StringVarP(
		&cli.Name,
		"name",
		"n",
		"",
		"Name query parameter",
	)

	rootCmd.PersistentFlags().StringVarP(
		&cli.Identity,
		"identity",
		"I",
		"",
		"User identity query parameter",
	)

	rootCmd.PersistentFlags().StringVarP(
		&cli.Metadata,
		"metadata",
		"m",
		"",
		"Metadata query parameter",
	)

	rootCmd.PersistentFlags().StringVarP(
		&cli.Status,
		"status",
		"S",
		"",
		"User status query parameter",
	)

	rootCmd.PersistentFlags().StringVarP(
		&cli.State,
		"state",
		"z",
		"",
		"Bootstrap state query parameter",
	)

	rootCmd.PersistentFlags().StringVarP(
		&cli.Topic,
		"topic",
		"T",
		"",
		"Subscription topic query parameter",
	)

	rootCmd.PersistentFlags().StringVarP(
		&cli.Contact,
		"contact",
		"C",
		"",
		"Subscription contact query parameter",
	)

	return rootCmd
}
