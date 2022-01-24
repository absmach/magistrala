// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package cli

import "github.com/spf13/cobra"

// NewHealthCmd returns health check command.
func NewHealthCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "health",
		Short: "Health Check",
		Long:  `Mainflux Things service Health Check`,
		Run: func(cmd *cobra.Command, args []string) {
			v, err := sdk.Health()
			if err != nil {
				logError(err)
				return
			}

			logJSON(v)
		},
	}
}
