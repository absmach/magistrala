// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"encoding/json"

	mfxsdk "github.com/mainflux/mainflux/pkg/sdk/go"
	"github.com/spf13/cobra"
)

const all = "all"

var cmdChannels = []cobra.Command{
	{
		Use:   "create <JSON_channel> <user_auth_token>",
		Short: "Create channel",
		Long:  `Creates new channel and generates it's UUID`,
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 2 {
				logUsage(cmd.Use)
				return
			}

			var channel mfxsdk.Channel
			if err := json.Unmarshal([]byte(args[0]), &channel); err != nil {
				logError(err)
				return
			}

			channel, err := sdk.CreateChannel(channel, args[1])
			if err != nil {
				logError(err)
				return
			}

			logJSON(channel)
		},
	},
	{
		Use:   "get [all | <channel_id>] <user_auth_token>",
		Short: "Get channel",
		Long: `Get all channels or get channel by id. Channels can be filtered by name or metadata.
		all - lists all channels
		<channel_id> - shows thing with provided <channel_id>`,

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
				Name:     "",
				Offset:   Offset,
				Limit:    Limit,
				Metadata: metadata,
			}

			if args[0] == all {
				l, err := sdk.Channels(pageMetadata, args[1])
				if err != nil {
					logError(err)
					return
				}

				logJSON(l)
				return
			}
			c, err := sdk.Channel(args[0], args[1])
			if err != nil {
				logError(err)
				return
			}

			logJSON(c)
		},
	},
	{
		Use:   "update <channel_id> <JSON_string> <user_auth_token>",
		Short: "Update channel",
		Long:  `Updates channel record`,
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 3 {
				logUsage(cmd.Use)
				return
			}

			var channel mfxsdk.Channel
			if err := json.Unmarshal([]byte(args[1]), &channel); err != nil {
				logError(err)
				return
			}
			channel.ID = args[0]
			channel, err := sdk.UpdateChannel(channel, args[2])
			if err != nil {
				logError(err)
				return
			}

			logJSON(channel)
		},
	},
	{
		Use:   "connections <channel_id> <user_auth_token>",
		Short: "Connections list",
		Long:  `List of Things connected to a Channel`,
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 2 {
				logUsage(cmd.Use)
				return
			}
			pm := mfxsdk.PageMetadata{
				Offset: Offset,
				Limit:  Limit,
			}
			cl, err := sdk.ThingsByChannel(args[0], pm, args[1])
			if err != nil {
				logError(err)
				return
			}

			logJSON(cl)
		},
	},
	{
		Use:   "enable <channel_id> <user_auth_token>",
		Short: "Change channel status to enabled",
		Long:  `Change channel status to enabled`,
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 2 {
				logUsage(cmd.Use)
				return
			}

			channel, err := sdk.EnableChannel(args[0], args[1])
			if err != nil {
				logError(err)
				return
			}

			logJSON(channel)
		},
	},
	{
		Use:   "disable <channel_id> <user_auth_token>",
		Short: "Change channel status to disabled",
		Long:  `Change channel status to disabled`,
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 2 {
				logUsage(cmd.Use)
				return
			}

			channel, err := sdk.DisableChannel(args[0], args[1])
			if err != nil {
				logError(err)
				return
			}

			logJSON(channel)
		},
	},
}

// NewChannelsCmd returns channels command.
func NewChannelsCmd() *cobra.Command {
	cmd := cobra.Command{
		Use:   "channels [create | get | update | delete | connections | not-connected]",
		Short: "Channels management",
		Long:  `Channels management: create, get, update or delete Channel and get list of Things connected or not connected to a Channel`,
	}

	for i := range cmdChannels {
		cmd.AddCommand(&cmdChannels[i])
	}

	return &cmd
}
