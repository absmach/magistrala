// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"encoding/json"

	mfclients "github.com/mainflux/mainflux/pkg/clients"
	mfxsdk "github.com/mainflux/mainflux/pkg/sdk/go"
	"github.com/spf13/cobra"
)

var cmdGroups = []cobra.Command{
	{
		Use:   "create <JSON_group> <user_auth_token>",
		Short: "Create group",
		Long: "Creates new group\n" +
			"Usage:\n" +
			"\tmainflux-cli groups create '{\"name\":\"new group\", \"description\":\"new group description\", \"metadata\":{\"key\": \"value\"}}' $USERTOKEN\n",
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 2 {
				logUsage(cmd.Use)
				return
			}
			var group mfxsdk.Group
			if err := json.Unmarshal([]byte(args[0]), &group); err != nil {
				logError(err)
				return
			}
			group.Status = mfclients.EnabledStatus.String()
			group, err := sdk.CreateGroup(group, args[1])
			if err != nil {
				logError(err)
				return
			}
			logJSON(group)
		},
	},
	{
		Use:   "update <JSON_group> <user_auth_token>",
		Short: "Update group",
		Long: "Updates group\n" +
			"Usage:\n" +
			"\tmainflux-cli groups update '{\"id\":\"<group_id>\", \"name\":\"new group\", \"description\":\"new group description\", \"metadata\":{\"key\": \"value\"}}' $USERTOKEN\n",
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 2 {
				logUsage(cmd.Use)
				return
			}

			var group mfxsdk.Group
			if err := json.Unmarshal([]byte(args[0]), &group); err != nil {
				logError(err)
				return
			}

			group, err := sdk.UpdateGroup(group, args[1])
			if err != nil {
				logError(err)
				return
			}

			logJSON(group)
		},
	},
	{
		Use:   "get [all | children <group_id> | parents <group_id> | members <group_id> | <group_id>] <user_auth_token>",
		Short: "Get group",
		Long: "Get all users groups, group children or group by id.\n" +
			"Usage:\n" +
			"\tmainflux-cli groups get all $USERTOKEN - lists all groups\n" +
			"\tmainflux-cli groups get children <group_id> $USERTOKEN - lists all children groups of <group_id>\n" +
			"\tmainflux-cli groups get parents <group_id> $USERTOKEN - lists all parent groups of <group_id>\n" +
			"\tmainflux-cli groups get <group_id> $USERTOKEN - shows group with provided group ID\n",
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) < 2 {
				logUsage(cmd.Use)
				return
			}
			if args[0] == all {
				if len(args) > 2 {
					logUsage(cmd.Use)
					return
				}
				pm := mfxsdk.PageMetadata{
					Offset: Offset,
					Limit:  Limit,
				}
				l, err := sdk.Groups(pm, args[1])
				if err != nil {
					logError(err)
					return
				}
				logJSON(l)
				return
			}
			if args[0] == "children" {
				if len(args) > 3 {
					logUsage(cmd.Use)
					return
				}
				pm := mfxsdk.PageMetadata{
					Offset: Offset,
					Limit:  Limit,
				}
				l, err := sdk.Children(args[1], pm, args[2])
				if err != nil {
					logError(err)
					return
				}
				logJSON(l)
				return
			}
			if args[0] == "parents" {
				if len(args) > 3 {
					logUsage(cmd.Use)
					return
				}
				pm := mfxsdk.PageMetadata{
					Offset: Offset,
					Limit:  Limit,
				}
				l, err := sdk.Parents(args[1], pm, args[2])
				if err != nil {
					logError(err)
					return
				}
				logJSON(l)
				return
			}
			if len(args) > 2 {
				logUsage(cmd.Use)
				return
			}
			t, err := sdk.Group(args[0], args[1])
			if err != nil {
				logError(err)
				return
			}
			logJSON(t)
		},
	},
	{
		Use:   "assign <allowed_actions> <member_id> <group_id> <user_auth_token>",
		Short: "Assign member",
		Long: "Assign members to a group\n" +
			"Usage:\n" +
			"\tmainflux-cli groups assign '[\"<allowed_action>\", \"<allowed_action>\"]' <member_id> <group_id> $USERTOKEN\n",
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 4 {
				logUsage(cmd.Use)
				return
			}
			var actions []string
			if err := json.Unmarshal([]byte(args[0]), &actions); err != nil {
				logError(err)
				return
			}
			if err := sdk.Assign(actions, args[1], args[2], args[3]); err != nil {
				logError(err)
				return
			}
			logOK()
		},
	},
	{
		Use:   "unassign <member_id> <group_id> <user_auth_token>",
		Short: "Unassign member",
		Long: "Unassign member from a group\n" +
			"Usage:\n" +
			"\tmainflux-cli groups unassign <member_id> <group_id> $USERTOKEN\n",
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 3 {
				logUsage(cmd.Use)
				return
			}
			if err := sdk.Unassign(args[0], args[1], args[2]); err != nil {
				logError(err)
				return
			}
			logOK()
		},
	},
	{
		Use:   "members <group_id> <user_auth_token>",
		Short: "Members list",
		Long: "List group's members\n" +
			"Usage:\n" +
			"\tmainflux-cli groups members <group_id> $USERTOKEN",
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 2 {
				logUsage(cmd.Use)
				return
			}
			pm := mfxsdk.PageMetadata{
				Offset: Offset,
				Limit:  Limit,
				Status: Status,
			}
			up, err := sdk.Members(args[0], pm, args[1])
			if err != nil {
				logError(err)
				return
			}
			logJSON(up)
		},
	},
	{
		Use:   "membership <member_id> <user_auth_token>",
		Short: "Membership list",
		Long: "List memberships of a member\n" +
			"Usage:\n" +
			"\tmainflux-cli groups membership <member_id> $USERTOKEN",
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 2 {
				logUsage(cmd.Use)
				return
			}
			pm := mfxsdk.PageMetadata{
				Offset: Offset,
				Limit:  Limit,
			}
			up, err := sdk.Memberships(args[0], pm, args[1])
			if err != nil {
				logError(err)
				return
			}
			logJSON(up)
		},
	},
	{
		Use:   "enable <group_id> <user_auth_token>",
		Short: "Change group status to enabled",
		Long: "Change group status to enabled\n" +
			"Usage:\n" +
			"\tmainflux-cli groups enable <group_id> $USERTOKEN\n",
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 2 {
				logUsage(cmd.Use)
				return
			}

			group, err := sdk.EnableGroup(args[0], args[1])
			if err != nil {
				logError(err)
				return
			}

			logJSON(group)
		},
	},
	{
		Use:   "disable <group_id> <user_auth_token>",
		Short: "Change group status to disabled",
		Long: "Change group status to disabled\n" +
			"Usage:\n" +
			"\tmainflux-cli groups disable <group_id> $USERTOKEN\n",
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 2 {
				logUsage(cmd.Use)
				return
			}

			group, err := sdk.DisableGroup(args[0], args[1])
			if err != nil {
				logError(err)
				return
			}

			logJSON(group)
		},
	},
}

// NewGroupsCmd returns users command.
func NewGroupsCmd() *cobra.Command {
	cmd := cobra.Command{
		Use:   "groups [create | get | update | delete | assign | unassign | members | membership]",
		Short: "Groups management",
		Long:  `Groups management: create, update, delete group and assign and unassign member to groups"`,
	}

	for i := range cmdGroups {
		cmd.AddCommand(&cmdGroups[i])
	}

	return &cmd
}
