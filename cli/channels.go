//
// Copyright (c) 2018
// Mainflux
//
// SPDX-License-Identifier: Apache-2.0
//

package cli

import (
	"encoding/json"

	mfxsdk "github.com/mainflux/mainflux/sdk/go"
	"github.com/spf13/cobra"
)

var cmdChannels = []cobra.Command{
	cobra.Command{
		Use:   "create",
		Short: "create <JSON_channel> <user_auth_token>",
		Long:  `Creates new channel and generates it's UUID`,
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 2 {
				logUsage(cmd.Short)
				return
			}

			var channel mfxsdk.Channel
			if err := json.Unmarshal([]byte(args[0]), &channel); err != nil {
				logError(err)
				return
			}

			id, err := sdk.CreateChannel(channel, args[1])
			if err != nil {
				logError(err)
				return
			}

			logCreated(id)
		},
	},
	cobra.Command{
		Use:   "get",
		Short: "get [all | <channel_id>] <user_auth_token>",
		Long:  `Gets list of all channels or gets channel by id`,
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 2 {
				logUsage(cmd.Short)
				return
			}

			if args[0] == "all" {
				l, err := sdk.Channels(args[1], uint64(Offset), uint64(Limit), Name)
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
	cobra.Command{
		Use:   "update",
		Short: "update <JSON_string> <user_auth_token>",
		Long:  `Updates channel record`,
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 2 {
				logUsage(cmd.Short)
				return
			}

			var channel mfxsdk.Channel
			if err := json.Unmarshal([]byte(args[0]), &channel); err != nil {
				logError(err)
				return
			}

			if err := sdk.UpdateChannel(channel, args[1]); err != nil {
				logError(err)
				return
			}

			logOK()
		},
	},
	cobra.Command{
		Use:   "delete",
		Short: "delete <channel_id> <user_auth_token>",
		Long:  `Delete channel by ID`,
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 2 {
				logUsage(cmd.Short)
				return
			}

			if err := sdk.DeleteChannel(args[0], args[1]); err != nil {
				logError(err)
				return
			}

			logOK()
		},
	},
	cobra.Command{
		Use:   "connections",
		Short: "connections <channel_id> <user_auth_token>",
		Long:  `List of Things connected to Channel`,
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 2 {
				logUsage(cmd.Short)
				return
			}

			cl, err := sdk.ThingsByChannel(args[1], args[0], uint64(Offset), uint64(Limit))
			if err != nil {
				logError(err)
				return
			}

			logJSON(cl)
		},
	},
}

// NewChannelsCmd returns channels command.
func NewChannelsCmd() *cobra.Command {
	cmd := cobra.Command{
		Use:   "channels",
		Short: "Channels management",
		Long:  `Channels management: create, get, update or delete Channel and get list of Things connected to Channel`,
		Run: func(cmd *cobra.Command, args []string) {
			logUsage("channels [create | get | update | delete | connections]")
		},
	}

	for i := range cmdChannels {
		cmd.AddCommand(&cmdChannels[i])
	}

	return &cmd
}
