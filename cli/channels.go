// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"encoding/json"

	mgxsdk "github.com/absmach/magistrala/pkg/sdk/go"
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
				logUsageCmd(*cmd, cmd.Use)
				return
			}

			var channel mgxsdk.Channel
			if err := json.Unmarshal([]byte(args[0]), &channel); err != nil {
				logErrorCmd(*cmd, err)
				return
			}

			channel, err := sdk.CreateChannel(channel, args[1])
			if err != nil {
				logErrorCmd(*cmd, err)
				return
			}

			logJSONCmd(*cmd, channel)
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
				logUsageCmd(*cmd, cmd.Use)
				return
			}
			metadata, err := convertMetadata(Metadata)
			if err != nil {
				logErrorCmd(*cmd, err)
				return
			}
			pageMetadata := mgxsdk.PageMetadata{
				Name:     "",
				Offset:   Offset,
				Limit:    Limit,
				Metadata: metadata,
			}

			if args[0] == all {
				l, err := sdk.Channels(pageMetadata, args[1])
				if err != nil {
					logErrorCmd(*cmd, err)
					return
				}

				logJSONCmd(*cmd, l)
				return
			}
			c, err := sdk.Channel(args[0], args[1])
			if err != nil {
				logErrorCmd(*cmd, err)
				return
			}

			logJSONCmd(*cmd, c)
		},
	},
	{
		Use:   "delete <channel_id> <user_auth_token>",
		Short: "Delete channel",
		Long: "Delete channel by id.\n" +
			"Usage:\n" +
			"\tmagistrala-cli channels delete <channel_id> $USERTOKEN - delete the given channel ID\n",
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 2 {
				logUsageCmd(*cmd, cmd.Use)
				return
			}
			if err := sdk.DeleteChannel(args[0], args[1]); err != nil {
				logErrorCmd(*cmd, err)
				return
			}
			logOKCmd(*cmd)
		},
	},
	{
		Use:   "update <channel_id> <JSON_string> <user_auth_token>",
		Short: "Update channel",
		Long:  `Updates channel record`,
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 3 {
				logUsageCmd(*cmd, cmd.Use)
				return
			}

			var channel mgxsdk.Channel
			if err := json.Unmarshal([]byte(args[1]), &channel); err != nil {
				logErrorCmd(*cmd, err)
				return
			}
			channel.ID = args[0]
			channel, err := sdk.UpdateChannel(channel, args[2])
			if err != nil {
				logErrorCmd(*cmd, err)
				return
			}

			logJSONCmd(*cmd, channel)
		},
	},
	{
		Use:   "connections <channel_id> <user_auth_token>",
		Short: "Connections list",
		Long:  `List of Things connected to a Channel`,
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 2 {
				logUsageCmd(*cmd, cmd.Use)
				return
			}
			pm := mgxsdk.PageMetadata{
				Offset:  Offset,
				Limit:   Limit,
				Channel: args[0],
			}
			cl, err := sdk.ThingsByChannel(pm, args[1])
			if err != nil {
				logErrorCmd(*cmd, err)
				return
			}

			logJSONCmd(*cmd, cl)
		},
	},
	{
		Use:   "enable <channel_id> <user_auth_token>",
		Short: "Change channel status to enabled",
		Long:  `Change channel status to enabled`,
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 2 {
				logUsageCmd(*cmd, cmd.Use)
				return
			}

			channel, err := sdk.EnableChannel(args[0], args[1])
			if err != nil {
				logErrorCmd(*cmd, err)
				return
			}

			logJSONCmd(*cmd, channel)
		},
	},
	{
		Use:   "disable <channel_id> <user_auth_token>",
		Short: "Change channel status to disabled",
		Long:  `Change channel status to disabled`,
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 2 {
				logUsageCmd(*cmd, cmd.Use)
				return
			}

			channel, err := sdk.DisableChannel(args[0], args[1])
			if err != nil {
				logErrorCmd(*cmd, err)
				return
			}

			logJSONCmd(*cmd, channel)
		},
	},
	{
		Use:   "users <channel_id> <user_auth_token>",
		Short: "List users",
		Long: "List users of a channel\n" +
			"Usage:\n" +
			"\tmagistrala-cli channels users <channel_id> $USERTOKEN\n",
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 2 {
				logUsageCmd(*cmd, cmd.Use)
				return
			}
			pm := mgxsdk.PageMetadata{
				Offset:  Offset,
				Limit:   Limit,
				Channel: args[0],
			}
			ul, err := sdk.ListChannelUsers(pm, args[1])
			if err != nil {
				logErrorCmd(*cmd, err)
				return
			}

			logJSONCmd(*cmd, ul)
		},
	},
	{
		Use:   "groups <channel_id> <user_auth_token>",
		Short: "List groups",
		Long: "List groups of a channel\n" +
			"Usage:\n" +
			"\tmagistrala-cli channels groups <channel_id> $USERTOKEN\n",
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 2 {
				logUsageCmd(*cmd, cmd.Use)
				return
			}
			pm := mgxsdk.PageMetadata{
				Offset:  Offset,
				Limit:   Limit,
				Channel: args[0],
			}
			ul, err := sdk.ListChannelUserGroups(pm, args[1])
			if err != nil {
				logErrorCmd(*cmd, err)
				return
			}

			logJSONCmd(*cmd, ul)
		},
	},
}

var channelAssignCmds = []cobra.Command{
	{
		Use:   "users <relation> <user_ids> <channel_id> <user_auth_token>",
		Short: "Assign users",
		Long: "Assign users to a channel\n" +
			"Usage:\n" +
			"\tmagistrala-cli channels assign users <relation> '[\"<user_id_1>\", \"<user_id_2>\"]' <channel_id> $USERTOKEN\n",
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 4 {
				logUsageCmd(*cmd, cmd.Use)
				return
			}
			var userIDs []string
			if err := json.Unmarshal([]byte(args[1]), &userIDs); err != nil {
				logErrorCmd(*cmd, err)
				return
			}
			if err := sdk.AddUserToChannel(args[2], mgxsdk.UsersRelationRequest{Relation: args[0], UserIDs: userIDs}, args[3]); err != nil {
				logErrorCmd(*cmd, err)
				return
			}
			logOKCmd(*cmd)
		},
	},
	{
		Use:   "groups  <group_ids> <channel_id> <user_auth_token>",
		Short: "Assign groups",
		Long: "Assign groups to a channel\n" +
			"Usage:\n" +
			"\tmagistrala-cli channels assign groups  '[\"<group_id_1>\", \"<group_id_2>\"]' <channel_id> $USERTOKEN\n",

		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 3 {
				logUsageCmd(*cmd, cmd.Use)
				return
			}
			var groupIDs []string
			if err := json.Unmarshal([]byte(args[0]), &groupIDs); err != nil {
				logErrorCmd(*cmd, err)
				return
			}
			if err := sdk.AddUserGroupToChannel(args[1], mgxsdk.UserGroupsRequest{UserGroupIDs: groupIDs}, args[2]); err != nil {
				logErrorCmd(*cmd, err)
				return
			}
			logOKCmd(*cmd)
		},
	},
}

var channelUnassignCmds = []cobra.Command{
	{
		Use:   "groups  <group_ids> <channel_id> <user_auth_token>",
		Short: "Unassign groups",
		Long: "Unassign groups from a channel\n" +
			"Usage:\n" +
			"\tmagistrala-cli channels unassign groups '[\"<group_id_1>\", \"<group_id_2>\"]'  <channel_id> $USERTOKEN\n",
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 3 {
				logUsageCmd(*cmd, cmd.Use)
				return
			}
			var groupIDs []string
			if err := json.Unmarshal([]byte(args[0]), &groupIDs); err != nil {
				logErrorCmd(*cmd, err)
				return
			}
			if err := sdk.RemoveUserGroupFromChannel(args[1], mgxsdk.UserGroupsRequest{UserGroupIDs: groupIDs}, args[2]); err != nil {
				logErrorCmd(*cmd, err)
				return
			}
			logOKCmd(*cmd)
		},
	},

	{
		Use:   "users <relation> <user_ids> <channel_id> <user_auth_token>",
		Short: "Unassign users",
		Long: "Unassign users from a channel\n" +
			"Usage:\n" +
			"\tmagistrala-cli channels unassign users <relation> '[\"<user_id_1>\", \"<user_id_2>\"]'  <channel_id> $USERTOKEN\n",
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 4 {
				logUsageCmd(*cmd, cmd.Use)
				return
			}
			var userIDs []string
			if err := json.Unmarshal([]byte(args[1]), &userIDs); err != nil {
				logErrorCmd(*cmd, err)
				return
			}
			if err := sdk.RemoveUserFromChannel(args[2], mgxsdk.UsersRelationRequest{Relation: args[0], UserIDs: userIDs}, args[3]); err != nil {
				logErrorCmd(*cmd, err)
				return
			}
			logOKCmd(*cmd)
		},
	},
}

func NewChannelAssignCmds() *cobra.Command {
	cmd := cobra.Command{
		Use:   "assign [users | groups]",
		Short: "Assign users or groups to a channel",
		Long:  "Assign users or groups to a channel",
	}
	for i := range channelAssignCmds {
		cmd.AddCommand(&channelAssignCmds[i])
	}
	return &cmd
}

func NewChannelUnassignCmds() *cobra.Command {
	cmd := cobra.Command{
		Use:   "unassign [users | groups]",
		Short: "Unassign users or groups from a channel",
		Long:  "Unassign users or groups from a channel",
	}
	for i := range channelUnassignCmds {
		cmd.AddCommand(&channelUnassignCmds[i])
	}
	return &cmd
}

// NewChannelsCmd returns channels command.
func NewChannelsCmd() *cobra.Command {
	cmd := cobra.Command{
		Use:   "channels [create | get | update | delete | connections | not-connected | assign | unassign | users | groups]",
		Short: "Channels management",
		Long:  `Channels management: create, get, update or delete Channel and get list of Things connected or not connected to a Channel`,
	}

	for i := range cmdChannels {
		cmd.AddCommand(&cmdChannels[i])
	}

	cmd.AddCommand(NewChannelAssignCmds())
	cmd.AddCommand(NewChannelUnassignCmds())
	return &cmd
}
