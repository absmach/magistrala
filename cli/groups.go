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
		Short: "get [all | children <group_id> | parents <group_id> | group_id] <user_auth_token>",
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
				if len(args) > 2 {
					logUsage(cmd.Short)
					return
				}
				l, err := sdk.Groups(uint64(Offset), uint64(Limit), args[1])
				if err != nil {
					logError(err)
					return
				}
				logJSON(l)
				return
			}
			if args[0] == "children" {
				if len(args) > 3 {
					logUsage(cmd.Short)
					return
				}
				l, err := sdk.Children(args[1], uint64(Offset), uint64(Limit), args[2])
				if err != nil {
					logError(err)
					return
				}
				logJSON(l)
				return
			}
			if args[0] == "parents" {
				if len(args) > 3 {
					logUsage(cmd.Short)
					return
				}
				l, err := sdk.Parents(args[1], uint64(Offset), uint64(Limit), args[2])
				if err != nil {
					logError(err)
					return
				}
				logJSON(l)
				return
			}
			if len(args) > 2 {
				logUsage(cmd.Short)
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
		Short: "assign <member_ids> <member_type> <group_id> <user_auth_token>",
		Long: `Assign members to a group.
				member_ids - '["member_id",...]`,
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 4 {
				logUsage(cmd.Short)
				return
			}
			var ids []string
			if err := json.Unmarshal([]byte(args[0]), &ids); err != nil {
				logError(err)
				return
			}
			if err := sdk.Assign(ids, args[1], args[2], args[3]); err != nil {
				logError(err)
				return
			}
			logOK()
		},
	},
	cobra.Command{
		Use:   "unassign",
		Short: "unassign <member_ids> <group_id> <user_auth_token>",
		Long: `Unassign members from a group
				member_ids - '["member_id",...]`,
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 3 {
				logUsage(cmd.Short)
				return
			}
			var ids []string
			if err := json.Unmarshal([]byte(args[0]), &ids); err != nil {
				logError(err)
				return
			}
			if err := sdk.Unassign(args[2], args[1], ids...); err != nil {
				logError(err)
				return
			}
			logOK()
		},
	},
	cobra.Command{
		Use:   "delete",
		Short: "delete <group_id> <user_auth_token>",
		Long:  `Delete group.`,
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
		Long:  `Lists all members of a group.`,
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
		Short: "membership <member_id> <user_auth_token>",
		Long:  `List member group's membership`,
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
		Long:  `Groups management: create groups and assigns member to groups"`,
		Run: func(cmd *cobra.Command, args []string) {
			logUsage("groups [create | get | delete | assign | unassign | members | membership]")
		},
	}
	for i := range cmdGroups {
		cmd.AddCommand(&cmdGroups[i])
	}
	return &cmd
}
