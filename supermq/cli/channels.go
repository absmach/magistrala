// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"encoding/json"

	smqsdk "github.com/absmach/supermq/pkg/sdk"
	"github.com/spf13/cobra"
)

const all = "all"

var cmdChannels = []cobra.Command{
	{
		Use:   "create <JSON_channel> <domain_id> <user_auth_token>",
		Short: "Create channel",
		Long:  `Creates new channel and generates it's UUID`,
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 3 {
				logUsageCmd(*cmd, cmd.Use)
				return
			}

			var channel smqsdk.Channel
			if err := json.Unmarshal([]byte(args[0]), &channel); err != nil {
				logErrorCmd(*cmd, err)
				return
			}

			channel, err := sdk.CreateChannel(channel, args[1], args[2])
			if err != nil {
				logErrorCmd(*cmd, err)
				return
			}

			logJSONCmd(*cmd, channel)
		},
	},
	{
		Use:   "get [all | <channel_id>] <domain_id> <user_auth_token>",
		Short: "Get channel",
		Long: `Get all channels or get channel by id. Channels can be filtered by name or metadata.
		all - lists all channels
		<channel_id> - shows client with provided <channel_id>`,

		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 3 {
				logUsageCmd(*cmd, cmd.Use)
				return
			}
			metadata, err := convertMetadata(Metadata)
			if err != nil {
				logErrorCmd(*cmd, err)
				return
			}
			pageMetadata := smqsdk.PageMetadata{
				Name:     "",
				Offset:   Offset,
				Limit:    Limit,
				Metadata: metadata,
			}

			if args[0] == all {
				l, err := sdk.Channels(pageMetadata, args[1], args[2])
				if err != nil {
					logErrorCmd(*cmd, err)
					return
				}

				logJSONCmd(*cmd, l)
				return
			}
			c, err := sdk.Channel(args[0], args[1], args[2])
			if err != nil {
				logErrorCmd(*cmd, err)
				return
			}

			logJSONCmd(*cmd, c)
		},
	},
	{
		Use:   "delete <channel_id> <domain_id> <user_auth_token>",
		Short: "Delete channel",
		Long: "Delete channel by id.\n" +
			"Usage:\n" +
			"\tsupermq-cli channels delete <channel_id> $DOMAINID $USERTOKEN - delete the given channel ID\n",
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 3 {
				logUsageCmd(*cmd, cmd.Use)
				return
			}
			if err := sdk.DeleteChannel(args[0], args[1], args[2]); err != nil {
				logErrorCmd(*cmd, err)
				return
			}
			logOKCmd(*cmd)
		},
	},
	{
		Use:   "update <channel_id> <JSON_string> <domain_id> <user_auth_token>",
		Short: "Update channel",
		Long:  `Updates channel record`,
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 4 {
				logUsageCmd(*cmd, cmd.Use)
				return
			}

			var channel smqsdk.Channel
			if err := json.Unmarshal([]byte(args[1]), &channel); err != nil {
				logErrorCmd(*cmd, err)
				return
			}
			channel.ID = args[0]
			channel, err := sdk.UpdateChannel(channel, args[2], args[3])
			if err != nil {
				logErrorCmd(*cmd, err)
				return
			}

			logJSONCmd(*cmd, channel)
		},
	},
	{
		Use:   "enable <channel_id> <domain_id> <user_auth_token>",
		Short: "Change channel status to enabled",
		Long:  `Change channel status to enabled`,
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 3 {
				logUsageCmd(*cmd, cmd.Use)
				return
			}

			channel, err := sdk.EnableChannel(args[0], args[1], args[2])
			if err != nil {
				logErrorCmd(*cmd, err)
				return
			}

			logJSONCmd(*cmd, channel)
		},
	},
	{
		Use:   "disable <channel_id> <domain_id> <user_auth_token>",
		Short: "Change channel status to disabled",
		Long:  `Change channel status to disabled`,
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 3 {
				logUsageCmd(*cmd, cmd.Use)
				return
			}

			channel, err := sdk.DisableChannel(args[0], args[1], args[2])
			if err != nil {
				logErrorCmd(*cmd, err)
				return
			}

			logJSONCmd(*cmd, channel)
		},
	},
	{
		Use:   "users <channel_id> <domain_id> <user_auth_token>",
		Short: "List users",
		Long: "List users of a channel\n" +
			"Usage:\n" +
			"\tsupermq-cli channels users <channel_id> $DOMAINID $USERTOKEN\n",
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 3 {
				logUsageCmd(*cmd, cmd.Use)
				return
			}
			pm := smqsdk.PageMetadata{
				Offset: Offset,
				Limit:  Limit,
			}
			ul, err := sdk.ListChannelMembers(args[0], args[1], pm, args[2])
			if err != nil {
				logErrorCmd(*cmd, err)
				return
			}

			logJSONCmd(*cmd, ul)
		},
	},
}

// NewChannelsCmd returns channels command.
func NewChannelsCmd() *cobra.Command {
	cmd := cobra.Command{
		Use:   "channels [create | get | update | delete | connections | not-connected | assign | unassign | users | groups]",
		Short: "Channels management",
		Long:  `Channels management: create, get, update or delete Channel and get list of Clients connected or not connected to a Channel`,
	}

	for i := range cmdChannels {
		cmd.AddCommand(&cmdChannels[i])
	}

	return &cmd
}
