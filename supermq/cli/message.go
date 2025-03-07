// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package cli

import "github.com/spf13/cobra"

var cmdMessages = []cobra.Command{
	{
		Use:   "send <channel_id.subtopic> <JSON_string> <client_secret>",
		Short: "Send messages",
		Long:  `Sends message on the channel`,
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 3 {
				logUsageCmd(*cmd, cmd.Use)
				return
			}

			if err := sdk.SendMessage(args[0], args[1], args[2]); err != nil {
				logErrorCmd(*cmd, err)
				return
			}

			logOKCmd(*cmd)
		},
	},
}

// NewMessagesCmd returns messages command.
func NewMessagesCmd() *cobra.Command {
	cmd := cobra.Command{
		Use:   "messages [send | read]",
		Short: "Send messages",
		Long:  `Send messages using the http-adapter`,
	}

	for i := range cmdMessages {
		cmd.AddCommand(&cmdMessages[i])
	}

	return &cmd
}
