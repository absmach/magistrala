// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	mgxsdk "github.com/absmach/magistrala/pkg/sdk/go"
	"github.com/spf13/cobra"
)

var cmdEvents = cobra.Command{
	Use:   "get <id> <entity_type> <user_auth_token>",
	Short: "Get events",
	Long: "Get events\n" +
		"Usage:\n" +
		"\tmagistrala-cli events get <id> <entity_type> <user_auth_token> - lists all events\n" +
		"\tmagistrala-cli events get <id> <entity_type> <user_auth_token> --offset <offset> --limit <limit> - lists all events with provided offset and limit\n",
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) != 3 {
			logUsage(cmd.Use)
			return
		}

		pageMetadata := mgxsdk.PageMetadata{
			Offset: Offset,
			Limit:  Limit,
		}

		events, err := sdk.Events(pageMetadata, args[0], args[1], args[2])
		if err != nil {
			logError(err)
			return
		}

		logJSON(events)
	},
}

// NewEventsCmd returns invitations command.
func NewEventsCmd() *cobra.Command {
	cmd := cobra.Command{
		Use:   "events get",
		Short: "events logs",
		Long:  `events to read event history`,
	}

	cmd.AddCommand(&cmdEvents)

	return &cmd
}
