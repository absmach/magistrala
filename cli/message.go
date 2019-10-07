// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package cli

import "github.com/spf13/cobra"

const contentTypeSenml = "application/senml+json"

var cmdMessages = []cobra.Command{
	cobra.Command{
		Use:   "send",
		Short: "send <channel_id>[.<subtopic>...] <JSON_string> <thing_key>",
		Long:  `Sends message on the channel`,
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 3 {
				logUsage(cmd.Short)
				return
			}

			if err := sdk.SendMessage(args[0], args[1], args[2]); err != nil {
				logError(err)
				return
			}

			logOK()
		},
	},
	cobra.Command{
		Use:   "read",
		Short: "read <channel_id>[.<subtopic>...] <thing_key>",
		Long:  `Reads all channel messages`,
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 2 {
				logUsage(cmd.Short)
				return
			}

			m, err := sdk.ReadMessages(args[0], args[1])
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
		Use:   "messages",
		Short: "Send or read messages",
		Long:  `Send or read messages using the http-adapter and the configured database reader`,
		Run: func(cmd *cobra.Command, args []string) {
			logUsage("messages [send | read]")
		},
	}

	for i := range cmdMessages {
		cmd.AddCommand(&cmdMessages[i])
	}

	return &cmd
}
