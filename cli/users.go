//
// Copyright (c) 2018
// Mainflux
//
// SPDX-License-Identifier: Apache-2.0
//

package cli

import (
	"github.com/spf13/cobra"
)

var cmdUsers = []cobra.Command{
	cobra.Command{
		Use:   "create",
		Short: "create <username> <password>",
		Long:  `Creates new user`,
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 2 {
				logUsage(cmd.Short)
				return
			}
			if err := sdk.CreateUser(args[0], args[1]); err != nil {
				logError(err)
				return
			}
			logOK()
		},
	},
	cobra.Command{
		Use:   "token",
		Short: "token <username> <password>",
		Long:  `Creates new token`,
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 2 {
				logUsage(cmd.Short)
				return
			}
			token, err := sdk.CreateToken(args[0], args[1])
			if err != nil {
				logError(err)
				return
			}
			dump(token)
		},
	},
}

func NewUsersCmd() *cobra.Command {
	cmd := cobra.Command{
		Use:   "users",
		Short: "users create/token <email> <password>",
		Long:  `Manages users in the system (create account or token)`,
		Run: func(cmd *cobra.Command, args []string) {
			logUsage(cmd.Short)
		},
	}

	for i := range cmdUsers {
		cmd.AddCommand(&cmdUsers[i])
	}

	return &cmd
}
