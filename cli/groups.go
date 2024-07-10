// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"encoding/json"

	mgclients "github.com/absmach/magistrala/pkg/clients"
	mgxsdk "github.com/absmach/magistrala/pkg/sdk/go"
	"github.com/spf13/cobra"
)

var cmdGroups = []cobra.Command{
	{
		Use:   "create <JSON_group> <user_auth_token>",
		Short: "Create group",
		Long: "Creates new group\n" +
			"Usage:\n" +
			"\tmagistrala-cli groups create '{\"name\":\"new group\", \"description\":\"new group description\", \"metadata\":{\"key\": \"value\"}}' $USERTOKEN\n",
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 2 {
				logUsageCmd(*cmd, cmd.Use)
				return
			}
			var group mgxsdk.Group
			if err := json.Unmarshal([]byte(args[0]), &group); err != nil {
				logErrorCmd(*cmd, err)
				return
			}
			group.Status = mgclients.EnabledStatus.String()
			group, err := sdk.CreateGroup(group, args[1])
			if err != nil {
				logErrorCmd(*cmd, err)
				return
			}
			logJSONCmd(*cmd, group)
		},
	},
	{
		Use:   "update <JSON_group> <user_auth_token>",
		Short: "Update group",
		Long: "Updates group\n" +
			"Usage:\n" +
			"\tmagistrala-cli groups update '{\"id\":\"<group_id>\", \"name\":\"new group\", \"description\":\"new group description\", \"metadata\":{\"key\": \"value\"}}' $USERTOKEN\n",
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 2 {
				logUsageCmd(*cmd, cmd.Use)
				return
			}

			var group mgxsdk.Group
			if err := json.Unmarshal([]byte(args[0]), &group); err != nil {
				logErrorCmd(*cmd, err)
				return
			}

			group, err := sdk.UpdateGroup(group, args[1])
			if err != nil {
				logErrorCmd(*cmd, err)
				return
			}

			logJSONCmd(*cmd, group)
		},
	},
	{
		Use:   "get [all | children <group_id> | parents <group_id> | members <group_id> | <group_id>] <user_auth_token>",
		Short: "Get group",
		Long: "Get all users groups, group children or group by id.\n" +
			"Usage:\n" +
			"\tmagistrala-cli groups get all $USERTOKEN - lists all groups\n" +
			"\tmagistrala-cli groups get children <group_id> $USERTOKEN - lists all children groups of <group_id>\n" +
			"\tmagistrala-cli groups get parents <group_id> $USERTOKEN - lists all parent groups of <group_id>\n" +
			"\tmagistrala-cli groups get <group_id> $USERTOKEN - shows group with provided group ID\n",
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) < 2 {
				logUsageCmd(*cmd, cmd.Use)
				return
			}
			if args[0] == all {
				if len(args) > 2 {
					logUsageCmd(*cmd, cmd.Use)
					return
				}
				pm := mgxsdk.PageMetadata{
					Offset: Offset,
					Limit:  Limit,
				}
				l, err := sdk.Groups(pm, args[1])
				if err != nil {
					logErrorCmd(*cmd, err)
					return
				}
				logJSONCmd(*cmd, l)
				return
			}
			if args[0] == "children" {
				if len(args) > 3 {
					logUsageCmd(*cmd, cmd.Use)
					return
				}
				pm := mgxsdk.PageMetadata{
					Offset: Offset,
					Limit:  Limit,
				}
				l, err := sdk.Children(args[1], pm, args[2])
				if err != nil {
					logErrorCmd(*cmd, err)
					return
				}
				logJSONCmd(*cmd, l)
				return
			}
			if args[0] == "parents" {
				if len(args) > 3 {
					logUsageCmd(*cmd, cmd.Use)
					return
				}
				pm := mgxsdk.PageMetadata{
					Offset: Offset,
					Limit:  Limit,
				}
				l, err := sdk.Parents(args[1], pm, args[2])
				if err != nil {
					logErrorCmd(*cmd, err)
					return
				}
				logJSONCmd(*cmd, l)
				return
			}
			if len(args) > 2 {
				logUsageCmd(*cmd, cmd.Use)
				return
			}
			t, err := sdk.Group(args[0], args[1])
			if err != nil {
				logErrorCmd(*cmd, err)
				return
			}
			logJSONCmd(*cmd, t)
		},
	},
	{
		Use:   "delete <group_id> <user_auth_token>",
		Short: "Delete group",
		Long: "Delete group by id.\n" +
			"Usage:\n" +
			"\tmagistrala-cli groups delete <group_id> $USERTOKEN - delete the given group ID\n",
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 2 {
				logUsageCmd(*cmd, cmd.Use)
				return
			}
			if err := sdk.DeleteGroup(args[0], args[1]); err != nil {
				logErrorCmd(*cmd, err)
				return
			}
			logOKCmd(*cmd)
		},
	},
	{
		Use:   "users <group_id> <user_auth_token>",
		Short: "List users",
		Long: "List users in a group\n" +
			"Usage:\n" +
			"\tmagistrala-cli groups users <group_id> $USERTOKEN",
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 2 {
				logUsageCmd(*cmd, cmd.Use)
				return
			}
			pm := mgxsdk.PageMetadata{
				Offset: Offset,
				Limit:  Limit,
				Status: Status,
				Domain: args[0],
			}
			users, err := sdk.ListGroupUsers(pm, args[1])
			if err != nil {
				logErrorCmd(*cmd, err)
				return
			}
			logJSONCmd(*cmd, users)
		},
	},
	{
		Use:   "channels <group_id> <user_auth_token>",
		Short: "List channels",
		Long: "List channels in a group\n" +
			"Usage:\n" +
			"\tmagistrala-cli groups channels <group_id> $USERTOKEN",
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 2 {
				logUsageCmd(*cmd, cmd.Use)
				return
			}
			pm := mgxsdk.PageMetadata{
				Offset: Offset,
				Limit:  Limit,
				Status: Status,
				Group:  args[0],
			}
			channels, err := sdk.ListGroupChannels(pm, args[1])
			if err != nil {
				logErrorCmd(*cmd, err)
				return
			}
			logJSONCmd(*cmd, channels)
		},
	},
	{
		Use:   "enable <group_id> <user_auth_token>",
		Short: "Change group status to enabled",
		Long: "Change group status to enabled\n" +
			"Usage:\n" +
			"\tmagistrala-cli groups enable <group_id> $USERTOKEN\n",
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 2 {
				logUsageCmd(*cmd, cmd.Use)
				return
			}

			group, err := sdk.EnableGroup(args[0], args[1])
			if err != nil {
				logErrorCmd(*cmd, err)
				return
			}

			logJSONCmd(*cmd, group)
		},
	},
	{
		Use:   "disable <group_id> <user_auth_token>",
		Short: "Change group status to disabled",
		Long: "Change group status to disabled\n" +
			"Usage:\n" +
			"\tmagistrala-cli groups disable <group_id> $USERTOKEN\n",
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 2 {
				logUsageCmd(*cmd, cmd.Use)
				return
			}

			group, err := sdk.DisableGroup(args[0], args[1])
			if err != nil {
				logErrorCmd(*cmd, err)
				return
			}

			logJSONCmd(*cmd, group)
		},
	},
}

var groupAssignCmds = []cobra.Command{
	{
		Use:   "users <relation> <user_ids> <group_id> <user_auth_token>",
		Short: "Assign users",
		Long: "Assign users to a group\n" +
			"Usage:\n" +
			"\tmagistrala-cli groups assign users <relation> '[\"<user_id_1>\", \"<user_id_2>\"]' <group_id> $USERTOKEN\n",
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
			if err := sdk.AddUserToGroup(args[2], mgxsdk.UsersRelationRequest{Relation: args[0], UserIDs: userIDs}, args[3]); err != nil {
				logErrorCmd(*cmd, err)
				return
			}
			logOKCmd(*cmd)
		},
	},
}

var groupUnassignCmds = []cobra.Command{
	{
		Use:   "users <relation> <user_ids> <group_id> <user_auth_token>",
		Short: "Unassign users",
		Long: "Unassign users from a group\n" +
			"Usage:\n" +
			"\tmagistrala-cli groups unassign users <relation> '[\"<user_id_1>\", \"<user_id_2>\"]' <group_id> $USERTOKEN\n",
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
			if err := sdk.RemoveUserFromGroup(args[2], mgxsdk.UsersRelationRequest{Relation: args[0], UserIDs: userIDs}, args[3]); err != nil {
				logErrorCmd(*cmd, err)
				return
			}
			logOKCmd(*cmd)
		},
	},
}

func NewGroupAssignCmds() *cobra.Command {
	cmd := cobra.Command{
		Use:   "assign [users]",
		Short: "Assign users to a group",
		Long:  "Assign users to a group",
	}

	for i := range groupAssignCmds {
		cmd.AddCommand(&groupAssignCmds[i])
	}
	return &cmd
}

func NewGroupUnassignCmds() *cobra.Command {
	cmd := cobra.Command{
		Use:   "unassign [users]",
		Short: "Unassign users from a group",
		Long:  "Unassign users from a group",
	}

	for i := range groupUnassignCmds {
		cmd.AddCommand(&groupUnassignCmds[i])
	}
	return &cmd
}

// NewGroupsCmd returns users command.
func NewGroupsCmd() *cobra.Command {
	cmd := cobra.Command{
		Use:   "groups [create | get | update | delete | assign | unassign | users | channels ]",
		Short: "Groups management",
		Long:  `Groups management: create, update, delete group and assign and unassign member to groups"`,
	}

	for i := range cmdGroups {
		cmd.AddCommand(&cmdGroups[i])
	}

	cmd.AddCommand(NewGroupAssignCmds())
	cmd.AddCommand(NewGroupUnassignCmds())
	return &cmd
}
