// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package cli

import "github.com/spf13/cobra"

var cmdMessages = []cobra.Command{
	{
		Use:   "send <domain_id> <channel_id/subtopic> <JSON_string> <secret>",
		Short: "Send messages",
		Long:  `Sends message on the channel`,
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 4 {
				logUsageCmd(*cmd, cmd.Use)
				return
			}

			if err := sdk.SendMessage(cmd.Context(), args[0], args[1], args[2], args[3]); err != nil {
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
		Use:   "messages [send]",
		Short: "Send messages",
		Long:  `Send messages using the HTTP API`,
	}

	for i := range cmdMessages {
		cmd.AddCommand(&cmdMessages[i])
	}

	return &cmd
}
