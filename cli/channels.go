//
// Copyright (c) 2018
// Mainflux
//
// SPDX-License-Identifier: Apache-2.0
//

package cli

import (
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
			id, err := sdk.CreateChannel(args[0], args[1])
			if err != nil {
				logError(err)
				return
			}
			dump(id)
		},
	},
	cobra.Command{
		Use:   "get",
		Short: "get all/<channel_id> <user_auth_token>",
		Long:  `Gets list of all channels or gets channel by id`,
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 2 {
				logUsage(cmd.Short)
				return
			}
			if args[0] == "all" {
				l, err := sdk.Channels(args[1])
				if err != nil {
					logError(err)
					return
				}
				dump(l)
				return
			}
			c, err := sdk.Channel(args[0], args[1])
			if err != nil {
				logError(err)
				return
			}
			dump(c)
		},
	},
	cobra.Command{
		Use:   "update",
		Short: "update <channel_id> <JSON_string> <user_auth_token>",
		Long:  `Updates channel record`,
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 3 {
				logUsage(cmd.Short)
				return
			}
			if err := sdk.UpdateChannel(args[0], args[1], args[2]); err != nil {
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
}

func NewChannelsCmd() *cobra.Command {
	cmd := cobra.Command{
		Use:   "channels",
		Short: "Manipulation with channels",
		Long:  `Manipulation with channels: create, delete or update channels`,
		Run: func(cmd *cobra.Command, args []string) {
			logUsage(cmd.Short)
		},
	}

	for i := range cmdChannels {
		cmd.AddCommand(&cmdChannels[i])
	}

	return &cmd
}
