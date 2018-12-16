//
// Copyright (c) 2018
// Mainflux
//
// SPDX-License-Identifier: Apache-2.0
//

package cli

import (
	mfxsdk "github.com/mainflux/mainflux/sdk/go"
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

			user := mfxsdk.User{
				Email:    args[0],
				Password: args[1],
			}
			if err := sdk.CreateUser(user); err != nil {
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

			user := mfxsdk.User{
				Email:    args[0],
				Password: args[1],
			}
			token, err := sdk.CreateToken(user)
			if err != nil {
				logError(err)
				return
			}

			logCreated(token)
		},
	},
}

// NewUsersCmd returns users command.
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
