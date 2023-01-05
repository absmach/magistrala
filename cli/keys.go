// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"time"

	"github.com/spf13/cobra"
)

var cmdAPIKeys = []cobra.Command{
	{
		Use:   "issue <duration> <user_auth_token>",
		Short: "Issue key",
		Long:  `Issues a new Key`,
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 2 {
				logUsage(cmd.Use)
				return
			}

			d, err := time.ParseDuration(args[0])
			if err != nil {
				logError(err)
				return
			}

			resp, err := sdk.Issue(d, args[1])
			if err != nil {
				logError(err)
				return
			}

			logJSON(resp)
		},
	},
	{
		Use:   "revoke <key_id> <user_auth_token>",
		Short: "Revoke key",
		Long:  `Removes API key from database`,
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 2 {
				logUsage(cmd.Use)
				return
			}

			if err := sdk.Revoke(args[0], args[1]); err != nil {
				logError(err)
				return
			}

			logOK()
		},
	},
	{
		Use:   "retrieve <key_id> <user_auth_token>",
		Short: "Retrieve key",
		Long:  `Retrieves API key with given id`,
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 2 {
				logUsage(cmd.Use)
				return
			}

			rk, err := sdk.RetrieveKey(args[0], args[1])
			if err != nil {
				logError(err)
				return
			}

			logJSON(rk)
		},
	},
}

// NewKeysCmd returns keys command.
func NewKeysCmd() *cobra.Command {
	cmd := cobra.Command{
		Use:   "keys [issue | revoke | retrieve]",
		Short: "Keys management",
		Long:  `Keys management: issue, revoke, or retrieve API key.`,
	}

	for i := range cmdAPIKeys {
		cmd.AddCommand(&cmdAPIKeys[i])
	}

	return &cmd
}
