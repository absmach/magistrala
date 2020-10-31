// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"encoding/json"

	mfxsdk "github.com/mainflux/mainflux/pkg/sdk/go"
	"github.com/spf13/cobra"
)

var cmdGroups = []cobra.Command{
	cobra.Command{
		Use:   "create",
		Short: "create <JSON_group> <user_auth_token>",
		Long: `Creates new group
		JSON_group: 
		{
			"Name":<group_name>,
			"Description":<description>,
			"ParentID":<parent_id>,
			"Metadata":<metadata>,
		}
		Name - is unique group name
		ParentID - ID of a group that is a parent to the creating group
		Metadata - JSON structured string`,
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 2 {
				logUsage(cmd.Short)
				return
			}
			var group mfxsdk.Group
			if err := json.Unmarshal([]byte(args[0]), &group); err != nil {
				logError(err)
				return
			}
			id, err := sdk.CreateGroup(group, args[1])
			if err != nil {
				logError(err)
				return
			}
			logCreated(id)
		},
	},
	cobra.Command{
		Use:   "get",
		Short: "get [all | children <group_id> | group_id] <user_auth_token>",
		Long: `Get all users groups, group children or group by id.
		all - lists all groups
		children <group_id> - lists all children groups of <group_id>
		<group_id> - shows group with provided group ID`,
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) < 2 {
				logUsage(cmd.Short)
				return
			}
			if args[0] == "all" {
				l, err := sdk.Groups(args[1], uint64(Offset), uint64(Limit), "")
				if err != nil {
					logError(err)
					return
				}
				logJSON(l)
				return
			}
			if args[0] == "children" {
				l, err := sdk.Groups(args[2], uint64(Offset), uint64(Limit), args[1])
				if err != nil {
					logError(err)
					return
				}
				logJSON(l)
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
	cobra.Command{
		Use:   "assign",
		Short: "assign <user_id> <group_id> <user_auth_token>",
		Long:  `Assign user to a group.`,
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 3 {
				logUsage(cmd.Short)
				return
			}
			if err := sdk.Assign(args[0], args[1], args[2]); err != nil {
				logError(err)
				return
			}
			logOK()
		},
	},
	cobra.Command{
		Use:   "unassign",
		Short: "unassign <user_id> <group_id> <user_auth_token>",
		Long:  `Unassign user from a group.`,
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 3 {
				logUsage(cmd.Short)
				return
			}
			if err := sdk.Unassign(args[0], args[1], args[2]); err != nil {
				logError(err)
				return
			}
			logOK()
		},
	},
	cobra.Command{
		Use:   "delete",
		Short: "delete <group_id> <user_auth_token>",
		Long:  `Delete users group.`,
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 2 {
				logUsage(cmd.Short)
				return
			}
			if err := sdk.DeleteGroup(args[0], args[1]); err != nil {
				logError(err)
				return
			}
			logOK()
		},
	},
	cobra.Command{
		Use:   "members",
		Short: "members <group_id> <user_auth_token>",
		Long:  `Lists all user members of a group.`,
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 2 {
				logUsage(cmd.Short)
				return
			}
			up, err := sdk.Members(args[0], args[1], uint64(Offset), uint64(Limit))
			if err != nil {
				logError(err)
				return
			}
			logJSON(up)
		},
	},
	cobra.Command{
		Use:   "membership",
		Short: "membership <user_id> <user_auth_token>",
		Long:  `List user groups membership`,
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 2 {
				logUsage(cmd.Short)
				return
			}
			up, err := sdk.Memberships(args[0], args[1], uint64(Offset), uint64(Limit))
			if err != nil {
				logError(err)
				return
			}
			logJSON(up)
		},
	},
}

// NewGroupsCmd returns users command.
func NewGroupsCmd() *cobra.Command {
	cmd := cobra.Command{
		Use:   "groups",
		Short: "Groups management",
		Long:  `Groups management: create groups and assigns user to groups"`,
		Run: func(cmd *cobra.Command, args []string) {
			logUsage("groups [create | get | delete | assign | unassign | members | membership]")
		},
	}
	for i := range cmdGroups {
		cmd.AddCommand(&cmdGroups[i])
	}
	return &cmd
}
