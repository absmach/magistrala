// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"os"

	"github.com/absmach/magistrala/cli"
)

func main() {
	// Cobra has already reported the error, so only the exit code is left.
	if err := cli.NewRootCmd().Execute(); err != nil {
		os.Exit(1)
	}
}
