// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package cli_test

import (
	"bytes"
	"testing"

	"github.com/absmach/magistrala/cli"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
)

// Test constants mirror cli package's unexported operation names (create,
// get, update, delete, enable, disable) — can't reference those directly
// from this external test package.
const (
	createCmd  = "create"
	getCmd     = "get"
	updateCmd  = "update"
	deleteCmd  = "delete"
	enableCmd  = "enable"
	disableCmd = "disable"
	allCmd     = "all"
)

func executeCommand(t *testing.T, root *cobra.Command, args ...string) string {
	t.Helper()
	buffer := new(bytes.Buffer)
	root.SetOut(buffer)
	root.SetErr(buffer)
	root.SetArgs(args)
	err := root.Execute()
	assert.NoError(t, err, "Error executing command")
	return buffer.String()
}

func setFlags(rootCmd *cobra.Command) *cobra.Command {
	rootCmd.PersistentFlags().BoolVarP(&cli.RawOutput, "raw", "r", cli.RawOutput, "Enables raw output mode for easier parsing of output")
	rootCmd.PersistentFlags().Uint64VarP(&cli.Limit, "limit", "l", 10, "Limit query parameter")
	rootCmd.PersistentFlags().Uint64VarP(&cli.Offset, "offset", "o", 0, "Offset query parameter")
	rootCmd.PersistentFlags().StringVarP(&cli.Name, "name", "n", "", "Name query parameter")

	return rootCmd
}
