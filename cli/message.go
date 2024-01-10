// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	mgxsdk "github.com/absmach/magistrala/pkg/sdk/go"
	"github.com/spf13/cobra"
)

var cmdMessages = []cobra.Command{
	{
		Use:   "send <channel_id.subtopic> <JSON_string> <thing_key>",
		Short: "Send messages",
		Long:  `Sends message on the channel`,
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 3 {
				logUsage(cmd.Use)
				return
			}

			if err := sdk.SendMessage(args[0], args[1], args[2]); err != nil {
				logError(err)
				return
			}

			logOK()
		},
	},
	{
		Use:   "read <channel_id.subtopic> <user_token>",
		Short: "Read messages",
		Long: "Reads all channel messages\n" +
			"Usage:\n" +
			"\tmagistrala-cli messages read <channel_id.subtopic> <user_token> --offset <offset> --limit <limit> - lists all messages with provided offset and limit\n",
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 2 {
				logUsage(cmd.Use)
				return
			}
			pageMetadata := mgxsdk.PageMetadata{
				Offset: Offset,
				Limit:  Limit,
			}

			m, err := sdk.ReadMessages(pageMetadata, args[0], args[1])
			if err != nil {
				logError(err)
				return
			}

			logJSON(m)
		},
	},
}

// NewMessagesCmd returns messages command.
func NewMessagesCmd() *cobra.Command {
	cmd := cobra.Command{
		Use:   "messages [send | read]",
		Short: "Send or read messages",
		Long:  `Send or read messages using the http-adapter and the configured database reader`,
	}

	for i := range cmdMessages {
		cmd.AddCommand(&cmdMessages[i])
	}

	return &cmd
}
