// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"encoding/json"

	mgclients "github.com/absmach/magistrala/pkg/clients"
	mgxsdk "github.com/absmach/magistrala/pkg/sdk/go"
	"github.com/spf13/cobra"
)

var cmdUsers = []cobra.Command{
	{
		Use:   "create <name> <username> <password> <user_auth_token>",
		Short: "Create user",
		Long: "Create user with provided name, username and password. Token is optional\n" +
			"For example:\n" +
			"\tmagistrala-cli users create user user@example.com 12345678 $USER_AUTH_TOKEN\n",
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) < 3 || len(args) > 4 {
				logUsage(cmd.Use)
				return
			}
			if len(args) == 3 {
				args = append(args, "")
			}

			user := mgxsdk.User{
				Name: args[0],
				Credentials: mgxsdk.Credentials{
					Identity: args[1],
					Secret:   args[2],
				},
				Status: mgclients.EnabledStatus.String(),
			}
			user, err := sdk.CreateUser(user, args[3])
			if err != nil {
				logError(err)
				return
			}

			logJSON(user)
		},
	},
	{
		Use:   "get [all | <user_id> ] <user_auth_token>",
		Short: "Get users",
		Long: "Get all users or get user by id. Users can be filtered by name or metadata or status\n" +
			"Usage:\n" +
			"\tmagistrala-cli users get all <user_auth_token> - lists all users\n" +
			"\tmagistrala-cli users get all <user_auth_token> --offset <offset> --limit <limit> - lists all users with provided offset and limit\n" +
			"\tmagistrala-cli users get <user_id> <user_auth_token> - shows user with provided <user_id>\n",
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
			pageMetadata := mgxsdk.PageMetadata{
				Identity: Identity,
				Offset:   Offset,
				Limit:    Limit,
				Metadata: metadata,
				Status:   Status,
			}
			if args[0] == all {
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
		Use:   "token <username> <password> [<domainID>]",
		Short: "Get token",
		Long: "Generate new token from username and password\n" +
			"For example:\n" +
			"\tmagistrala-cli users token user@example.com 12345678\n",
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 3 && len(args) != 2 {
				logUsage(cmd.Use)
				return
			}

			lg := mgxsdk.Login{
				Identity: args[0],
				Secret:   args[1],
			}
			if len(args) == 3 {
				lg.DomainID = args[2]
			}

			token, err := sdk.CreateToken(lg)
			if err != nil {
				logError(err)
				return
			}

			logJSON(token)
		},
	},
	{
		Use:   "refreshtoken <token> [<domainID>]",
		Short: "Get token",
		Long: "Generate new token from refresh token\n" +
			"For example:\n" +
			"\tmagistrala-cli users refreshtoken <refresh_token>\n",
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 2 && len(args) != 1 {
				logUsage(cmd.Use)
				return
			}

			lg := mgxsdk.Login{}
			if len(args) == 2 {
				lg.DomainID = args[1]
			}
			token, err := sdk.RefreshToken(lg, args[0])
			if err != nil {
				logError(err)
				return
			}

			logJSON(token)
		},
	},
	{
		Use:   "update [<user_id> <JSON_string> | tags <user_id> <tags> | identity <user_id> <identity> ] <user_auth_token>",
		Short: "Update user",
		Long: "Updates either user name and metadata or user tags or user identity\n" +
			"Usage:\n" +
			"\tmagistrala-cli users update <user_id> '{\"name\":\"new name\", \"metadata\":{\"key\": \"value\"}}' $USERTOKEN - updates user name and metadata\n" +
			"\tmagistrala-cli users update tags <user_id> '[\"tag1\", \"tag2\"]' $USERTOKEN - updates user tags\n" +
			"\tmagistrala-cli users update identity <user_id> newidentity@example.com $USERTOKEN - updates user identity\n",
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 4 && len(args) != 3 {
				logUsage(cmd.Use)
				return
			}

			var user mgxsdk.User
			if args[0] == "tags" {
				if err := json.Unmarshal([]byte(args[2]), &user.Tags); err != nil {
					logError(err)
					return
				}
				user.ID = args[1]
				user, err := sdk.UpdateUserTags(user, args[3])
				if err != nil {
					logError(err)
					return
				}

				logJSON(user)
				return
			}

			if args[0] == "identity" {
				user.ID = args[1]
				user.Credentials.Identity = args[2]
				user, err := sdk.UpdateUserIdentity(user, args[3])
				if err != nil {
					logError(err)
					return
				}

				logJSON(user)
				return

			}

			if args[0] == "role" {
				user.ID = args[1]
				user.Role = args[2]
				user, err := sdk.UpdateUserRole(user, args[3])
				if err != nil {
					logError(err)
					return
				}

				logJSON(user)
				return

			}

			if err := json.Unmarshal([]byte(args[1]), &user); err != nil {
				logError(err)
				return
			}
			user.ID = args[0]
			user, err := sdk.UpdateUser(user, args[2])
			if err != nil {
				logError(err)
				return
			}

			logJSON(user)
		},
	},
	{
		Use:   "profile <user_auth_token>",
		Short: "Get user profile",
		Long: "Get user profile\n" +
			"Usage:\n" +
			"\tmagistrala-cli users profile $USERTOKEN\n",
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 1 {
				logUsage(cmd.Use)
				return
			}

			user, err := sdk.UserProfile(args[0])
			if err != nil {
				logError(err)
				return
			}

			logJSON(user)
		},
	},
	{
		Use:   "resetpasswordrequest <email>",
		Short: "Send reset password request",
		Long: "Send reset password request\n" +
			"Usage:\n" +
			"\tmagistrala-cli users resetpasswordrequest example@mail.com\n",
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 1 {
				logUsage(cmd.Use)
				return
			}

			if err := sdk.ResetPasswordRequest(args[0]); err != nil {
				logError(err)
				return
			}

			logOK()
		},
	},
	{
		Use:   "resetpassword <password> <confpass> <password_request_token>",
		Short: "Reset password",
		Long: "Reset password\n" +
			"Usage:\n" +
			"\tmagistrala-cli users resetpassword 12345678 12345678 $REQUESTTOKEN\n",
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 3 {
				logUsage(cmd.Use)
				return
			}

			if err := sdk.ResetPassword(args[0], args[1], args[2]); err != nil {
				logError(err)
				return
			}

			logOK()
		},
	},
	{
		Use:   "password <old_password> <password> <user_auth_token>",
		Short: "Update password",
		Long: "Update password\n" +
			"Usage:\n" +
			"\tmagistrala-cli users password old_password new_password $USERTOKEN\n",
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 3 {
				logUsage(cmd.Use)
				return
			}

			user, err := sdk.UpdatePassword(args[0], args[1], args[2])
			if err != nil {
				logError(err)
				return
			}

			logJSON(user)
		},
	},
	{
		Use:   "enable <user_id> <user_auth_token>",
		Short: "Change user status to enabled",
		Long: "Change user status to enabled\n" +
			"Usage:\n" +
			"\tmagistrala-cli users enable <user_id> <user_auth_token>\n",
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 2 {
				logUsage(cmd.Use)
				return
			}

			user, err := sdk.EnableUser(args[0], args[1])
			if err != nil {
				logError(err)
				return
			}

			logJSON(user)
		},
	},
	{
		Use:   "disable <user_id> <user_auth_token>",
		Short: "Change user status to disabled",
		Long: "Change user status to disabled\n" +
			"Usage:\n" +
			"\tmagistrala-cli users disable <user_id> <user_auth_token>\n",
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 2 {
				logUsage(cmd.Use)
				return
			}

			user, err := sdk.DisableUser(args[0], args[1])
			if err != nil {
				logError(err)
				return
			}

			logJSON(user)
		},
	},

	{
		Use:   "channels <user_id> <user_auth_token>",
		Short: "List channels",
		Long: "List channels of user\n" +
			"Usage:\n" +
			"\tmagistrala-cli users channels <user_id> <user_auth_token>\n",
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 2 {
				logUsage(cmd.Use)
				return
			}

			pm := mgxsdk.PageMetadata{
				Offset: Offset,
				Limit:  Limit,
				User:   args[0],
			}

			cp, err := sdk.ListUserChannels(pm, args[1])
			if err != nil {
				logError(err)
				return
			}

			logJSON(cp)
		},
	},

	{
		Use:   "things <user_id> <user_auth_token>",
		Short: "List things",
		Long: "List things of user\n" +
			"Usage:\n" +
			"\tmagistrala-cli users things <user_id> <user_auth_token>\n",
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 2 {
				logUsage(cmd.Use)
				return
			}

			pm := mgxsdk.PageMetadata{
				Offset: Offset,
				Limit:  Limit,
				User:   args[0],
			}

			tp, err := sdk.ListUserThings(pm, args[1])
			if err != nil {
				logError(err)
				return
			}

			logJSON(tp)
		},
	},

	{
		Use:   "domains <user_id> <user_auth_token>",
		Short: "List domains",
		Long: "List user's domains\n" +
			"Usage:\n" +
			"\tmagistrala-cli users domains <user_id> <user_auth_token>\n",
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 2 {
				logUsage(cmd.Use)
				return
			}

			pm := mgxsdk.PageMetadata{
				Offset: Offset,
				Limit:  Limit,
				User:   args[0],
			}

			dp, err := sdk.ListUserDomains(pm, args[1])
			if err != nil {
				logError(err)
				return
			}

			logJSON(dp)
		},
	},

	{
		Use:   "groups <user_id> <user_auth_token>",
		Short: "List groups",
		Long: "List groups of user\n" +
			"Usage:\n" +
			"\tmagistrala-cli users groups <user_id> <user_auth_token>\n",
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 2 {
				logUsage(cmd.Use)
				return
			}

			pm := mgxsdk.PageMetadata{
				Offset: Offset,
				Limit:  Limit,
				User:   args[0],
			}

			users, err := sdk.ListUserGroups(pm, args[1])
			if err != nil {
				logError(err)
				return
			}

			logJSON(users)
		},
	},
}

// NewUsersCmd returns users command.
func NewUsersCmd() *cobra.Command {
	cmd := cobra.Command{
		Use:   "users [create | get | update | token | password | enable | disable | channels | things | groups]",
		Short: "Users management",
		Long:  `Users management: create accounts and tokens"`,
	}

	for i := range cmdUsers {
		cmd.AddCommand(&cmdUsers[i])
	}

	return &cmd
}
