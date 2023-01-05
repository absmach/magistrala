// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"encoding/json"

	mfxsdk "github.com/mainflux/mainflux/pkg/sdk/go"
	"github.com/spf13/cobra"
)

var cmdUsers = []cobra.Command{
	{
		Use:   "create <username> <password> <user_auth_token>",
		Short: "Create user",
		Long:  `Creates new user`,
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) < 2 || len(args) > 3 {
				logUsage(cmd.Use)
				return
			}
			if len(args) == 2 {
				args = append(args, "")
			}

			user := mfxsdk.User{
				Email:    args[0],
				Password: args[1],
			}
			id, err := sdk.CreateUser(user, args[2])
			if err != nil {
				logError(err)
				return
			}

			logCreated(id)
		},
	},
	{
		Use:   "get [all | <user_id> ] <user_auth_token>",
		Short: "Get users",
		Long: `Get all users or get user by id. Users can be filtered by name or metadata
		all - lists all users
		<user_id> - shows user with provided <user_id>`,
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 2 {
				logUsage(cmd.Use)
				return
			}
			metadata, err := convertMetadata(Metadata)
			if err != nil {
				logError(err)
				return
			}
			pageMetadata := mfxsdk.PageMetadata{
				Email:    "",
				Offset:   uint64(Offset),
				Limit:    uint64(Limit),
				Metadata: metadata,
				Status:   Status,
			}
			if args[0] == "all" {
				l, err := sdk.Users(pageMetadata, args[1])
				if err != nil {
					logError(err)
					return
				}
				logJSON(l)
				return
			}
			u, err := sdk.User(args[0], args[1])
			if err != nil {
				logError(err)
				return
			}

			logJSON(u)
		},
	},
	{
		Use:   "token <username> <password>",
		Short: "Get token",
		Long:  `Generate new token`,
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 2 {
				logUsage(cmd.Use)
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
	{
		Use:   "update <JSON_string> <user_auth_token>",
		Short: "Update user",
		Long:  `Update user metadata`,
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 2 {
				logUsage(cmd.Use)
				return
			}

			var user mfxsdk.User
			if err := json.Unmarshal([]byte(args[0]), &user.Metadata); err != nil {
				logError(err)
				return
			}

			if err := sdk.UpdateUser(user, args[1]); err != nil {
				logError(err)
				return
			}

			logOK()
		},
	},
	{
		Use:   "password <old_password> <password> <user_auth_token>",
		Short: "Update password",
		Long:  `Update user password`,
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 3 {
				logUsage(cmd.Use)
				return
			}

			if err := sdk.UpdatePassword(args[0], args[1], args[2]); err != nil {
				logError(err)
				return
			}

			logOK()
		},
	},
	{
		Use:   "enable <user_id> <user_auth_token>",
		Short: "Change user status to enabled",
		Long:  `Change user status to enabled`,
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 2 {
				logUsage(cmd.Use)
				return
			}

			if err := sdk.EnableUser(args[0], args[1]); err != nil {
				logError(err)
				return
			}

			logOK()
		},
	},
	{
		Use:   "disable <user_id> <user_auth_token>",
		Short: "Change user status to disabled",
		Long:  `Change user status to disabled`,
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 2 {
				logUsage(cmd.Use)
				return
			}

			if err := sdk.DisableUser(args[0], args[1]); err != nil {
				logError(err)
				return
			}

			logOK()
		},
	},
}

// NewUsersCmd returns users command.
func NewUsersCmd() *cobra.Command {
	cmd := cobra.Command{
		Use:   "users [create | get | update | token | password | enable | disable]",
		Short: "Users management",
		Long:  `Users management: create accounts and tokens"`,
	}

	for i := range cmdUsers {
		cmd.AddCommand(&cmdUsers[i])
	}

	return &cmd
}
