// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

// NewHealthCmd returns health check command.
func NewHealthCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "health <service>",
		Short: "Health Check",
		Long: "Magistrala service Health Check\n" +
			"Supported services: fluxmq\n" +
			"usage:\n" +
			"\tmagistrala-cli health <service>",
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 1 {
				logUsageCmd(*cmd, cmd.Use)
				return
			}
			if args[0] != "fluxmq" {
				logErrorCmd(*cmd, fmt.Errorf("unsupported health service %q; supported services: fluxmq", args[0]))
				return
			}
			v, err := sdk.Health(args[0])
			if err != nil {
				logErrorCmd(*cmd, err)
				return
			}

			logJSONCmd(*cmd, v)
		},
	}
}
