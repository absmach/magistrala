// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"encoding/json"

	"github.com/absmach/supermq/groups"
	smqsdk "github.com/absmach/supermq/pkg/sdk"
	"github.com/spf13/cobra"
)

var cmdGroups = []cobra.Command{
	{
		Use:   "create <JSON_group> <domain_id> <user_auth_token>",
		Short: "Create group",
		Long: "Creates new group\n" +
			"Usage:\n" +
			"\tsupermq-cli groups create '{\"name\":\"new group\", \"description\":\"new group description\", \"metadata\":{\"key\": \"value\"}}' $DOMAINID $USERTOKEN\n",
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 3 {
				logUsageCmd(*cmd, cmd.Use)
				return
			}
			var group smqsdk.Group
			if err := json.Unmarshal([]byte(args[0]), &group); err != nil {
				logErrorCmd(*cmd, err)
				return
			}
			group.Status = groups.EnabledStatus.String()
			group, err := sdk.CreateGroup(group, args[1], args[2])
			if err != nil {
				logErrorCmd(*cmd, err)
				return
			}
			logJSONCmd(*cmd, group)
		},
	},
	{
		Use:   "update <JSON_group> <domain_id> <user_auth_token>",
		Short: "Update group",
		Long: "Updates group\n" +
			"Usage:\n" +
			"\tsupermq-cli groups update '{\"id\":\"<group_id>\", \"name\":\"new group\", \"description\":\"new group description\", \"metadata\":{\"key\": \"value\"}}' $DOMAINID $USERTOKEN\n",
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 3 {
				logUsageCmd(*cmd, cmd.Use)
				return
			}

			var group smqsdk.Group
			if err := json.Unmarshal([]byte(args[0]), &group); err != nil {
				logErrorCmd(*cmd, err)
				return
			}

			group, err := sdk.UpdateGroup(group, args[1], args[2])
			if err != nil {
				logErrorCmd(*cmd, err)
				return
			}

			logJSONCmd(*cmd, group)
		},
	},
	{
		Use:   "delete <group_id> <domain_id> <user_auth_token>",
		Short: "Delete group",
		Long: "Delete group by id.\n" +
			"Usage:\n" +
			"\tsupermq-cli groups delete <group_id> $DOMAINID $USERTOKEN - delete the given group ID\n",
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 3 {
				logUsageCmd(*cmd, cmd.Use)
				return
			}
			if err := sdk.DeleteGroup(args[0], args[1], args[2]); err != nil {
				logErrorCmd(*cmd, err)
				return
			}
			logOKCmd(*cmd)
		},
	},
	{
		Use:   "enable <group_id> <domain_id> <user_auth_token>",
		Short: "Change group status to enabled",
		Long: "Change group status to enabled\n" +
			"Usage:\n" +
			"\tsupermq-cli groups enable <group_id> $DOMAINID $USERTOKEN\n",
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 3 {
				logUsageCmd(*cmd, cmd.Use)
				return
			}

			group, err := sdk.EnableGroup(args[0], args[1], args[2])
			if err != nil {
				logErrorCmd(*cmd, err)
				return
			}

			logJSONCmd(*cmd, group)
		},
	},
	{
		Use:   "disable <group_id> <domain_id> <user_auth_token>",
		Short: "Change group status to disabled",
		Long: "Change group status to disabled\n" +
			"Usage:\n" +
			"\tsupermq-cli groups disable <group_id> $DOMAINID $USERTOKEN\n",
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 3 {
				logUsageCmd(*cmd, cmd.Use)
				return
			}

			group, err := sdk.DisableGroup(args[0], args[1], args[2])
			if err != nil {
				logErrorCmd(*cmd, err)
				return
			}

			logJSONCmd(*cmd, group)
		},
	},
}

var cmdGroupsRoles = []cobra.Command{
	{
		Use:   "create <JSON_role> <group_id> <domain_id> <user_auth_token>",
		Short: "Create group role",
		Long: "Create role\n" +
			"Usage:\n" +
			"\tsupermq-cli groups roles create <JSON_role> <group_id> <domain_id> <user_auth_token>\n" +
			"For example:\n" +
			"\tsupermq-cli groups roles create '{\"role_name\":\"admin\",\"optional_actions\":[\"read\",\"update\"]}' 4ef09eff-d500-4d56-b04f-d23a512d6f2a 39f97daf-d6b6-40f4-b229-2697be8006ef $USER_AUTH_TOKEN\n",
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 4 {
				logUsageCmd(*cmd, cmd.Use)
				return
			}
			var roleReq smqsdk.RoleReq
			if err := json.Unmarshal([]byte(args[0]), &roleReq); err != nil {
				logErrorCmd(*cmd, err)
				return
			}
			r, err := sdk.CreateGroupRole(args[1], args[2], roleReq, args[3])
			if err != nil {
				logErrorCmd(*cmd, err)
				return
			}

			logJSONCmd(*cmd, r)
		},
	},

	{
		Use:   "get [all | <role_id>] <group_id>, <domain_id> <user_auth_token>",
		Short: "Get group roles",
		Long: "Get group roles\n" +
			"Usage:\n" +
			"\tsupermq-cli groups roles get all <group_id> <domain_id> <user_auth_token> - lists all roles\n" +
			"\tsupermq-cli groups roles get all <group_id> <domain_id> <user_auth_token> --offset <offset> --limit <limit> - lists all roles with provided offset and limit\n" +
			"\tsupermq-cli groups roles get <role_id> <group_id> <domain_id> <user_auth_token> - shows role by role id and domain id\n",
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 4 {
				logUsageCmd(*cmd, cmd.Use)
				return
			}
			pageMetadata := smqsdk.PageMetadata{
				Offset: Offset,
				Limit:  Limit,
			}
			if args[0] == all {
				rs, err := sdk.GroupRoles(args[1], args[2], pageMetadata, args[3])
				if err != nil {
					logErrorCmd(*cmd, err)
					return
				}
				logJSONCmd(*cmd, rs)
				return
			}
			r, err := sdk.GroupRole(args[1], args[0], args[2], args[3])
			if err != nil {
				logErrorCmd(*cmd, err)
				return
			}
			logJSONCmd(*cmd, r)
		},
	},

	{
		Use:   "update <new_name> <role_id> <group_id> <domain_id> <user_auth_token>",
		Short: "Update group role name",
		Long: "Update group role name\n" +
			"Usage:\n" +
			"\tsupermq-cli groups roles update <new_name> <role_id> <group_id> <domain_id> <user_auth_token>\n" +
			"For example:\n" +
			"\tsupermq-cli groups roles update new_name 39f97daf-d6b6-40f4-b229-2697be8006ef 4ef09eff-d500-4d56-b04f-d23a512d6f2a 4ef09eff-d500-4d56-b04f-d23a512d6f2a $USER_AUTH_TOKEN\n",
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 5 {
				logUsageCmd(*cmd, cmd.Use)
				return
			}

			r, err := sdk.UpdateGroupRole(args[2], args[1], args[0], args[3], args[4])
			if err != nil {
				logErrorCmd(*cmd, err)
				return
			}
			logJSONCmd(*cmd, r)
		},
	},

	{
		Use:   "delete <role_id> <group_id> <domain_id> <user_auth_token>",
		Short: "Delete group role",
		Long: "Delete group role\n" +
			"Usage:\n" +
			"\tsupermq-cli groups roles delete <role_id> <group_id> <domain_id> <user_auth_token>\n" +
			"For example:\n" +
			"\tsupermq-cli groups roles delete 39f97daf-d6b6-40f4-b229-2697be8006ef 4ef09eff-d500-4d56-b04f-d23a512d6f2a 4ef09eff-d500-4d56-b04f-d23a512d6f2a $USER_AUTH_TOKEN\n",
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 4 {
				logUsageCmd(*cmd, cmd.Use)
				return
			}

			if err := sdk.DeleteGroupRole(args[1], args[0], args[2], args[3]); err != nil {
				logErrorCmd(*cmd, err)
				return
			}
			logOKCmd(*cmd)
		},
	},
}

var cmdGroupsActions = []cobra.Command{
	{
		Use:   "add <JSON_actions> <role_id> <group_id> <domain_id> <user_auth_token>",
		Short: "Add actions to role",
		Long: "Add actions to role\n" +
			"Usage:\n" +
			"\tsupermq-cli groups roles actions add <JSON_actions> <role_id> <group_id> <domain_id> <user_auth_token>\n" +
			"For example:\n" +
			"\tsupermq-cli groups roles actions add '{\"actions\":[\"read\",\"write\"]}' 39f97daf-d6b6-40f4-b229-2697be8006ef 4ef09eff-d500-4d56-b04f-d23a512d6f2a 4ef09eff-d500-4d56-b04f-d23a512d6f2a $USER_AUTH_TOKEN\n",
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 5 {
				logUsageCmd(*cmd, cmd.Use)
				return
			}
			actions := struct {
				Actions []string `json:"actions"`
			}{}
			if err := json.Unmarshal([]byte(args[0]), &actions); err != nil {
				logErrorCmd(*cmd, err)
				return
			}

			acts, err := sdk.AddGroupRoleActions(args[2], args[1], args[3], actions.Actions, args[4])
			if err != nil {
				logErrorCmd(*cmd, err)
				return
			}
			logJSONCmd(*cmd, acts)
		},
	},

	{
		Use:   "list <role_id> <group_id> <domain_id> <user_auth_token>",
		Short: "List actions of role",
		Long: "List actions of role\n" +
			"Usage:\n" +
			"\tsupermq-cli groups roles actions list <role_id> <group_id> <domain_id> <user_auth_token>\n" +
			"For example:\n" +
			"\tsupermq-cli groups roles actions list 39f97daf-d6b6-40f4-b229-2697be8006ef 4ef09eff-d500-4d56-b04f-d23a512d6f2a 4ef09eff-d500-4d56-b04f-d23a512d6f2a $USER_AUTH_TOKEN\n",
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 4 {
				logUsageCmd(*cmd, cmd.Use)
				return
			}

			l, err := sdk.GroupRoleActions(args[1], args[0], args[2], args[3])
			if err != nil {
				logErrorCmd(*cmd, err)
				return
			}
			logJSONCmd(*cmd, l)
		},
	},

	{
		Use:   "delete [all | <JSON_actions>] <role_id> <group_id> <domain_id> <user_auth_token>",
		Short: "Delete actions from role",
		Long: "Delete actions from role\n" +
			"Usage:\n" +
			"\tsupermq-cli groups roles actions delete <JSON_actions> <role_id> <group_id> <domain_id> <user_auth_token>\n" +
			"\tsupermq-cli groups roles actions delete all <role_id> <group_id> <domain_id> <user_auth_token>\n" +
			"For example:\n" +
			"\tsupermq-cli groups roles actions delete '{\"actions\":[\"read\",\"write\"]}' 39f97daf-d6b6-40f4-b229-2697be8006ef 4ef09eff-d500-4d56-b04f-d23a512d6f2a 4ef09eff-d500-4d56-b04f-d23a512d6f2a $USER_AUTH_TOKEN\n",
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 5 {
				logUsageCmd(*cmd, cmd.Use)
				return
			}
			if args[0] == all {
				if err := sdk.RemoveAllGroupRoleActions(args[2], args[1], args[3], args[4]); err != nil {
					logErrorCmd(*cmd, err)
					return
				}
				logOKCmd(*cmd)
				return
			}
			actions := struct {
				Actions []string `json:"actions"`
			}{}
			if err := json.Unmarshal([]byte(args[0]), &actions); err != nil {
				logErrorCmd(*cmd, err)
				return
			}
			if err := sdk.RemoveGroupRoleActions(args[2], args[1], args[3], actions.Actions, args[4]); err != nil {
				logErrorCmd(*cmd, err)
				return
			}
			logOKCmd(*cmd)
		},
	},

	{
		Use:   "available-actions <domain_id> <user_auth_token>",
		Short: "List available actions",
		Long: "List available actions\n" +
			"Usage:\n" +
			"\tsupermq-cli groups roles actions available-actions <domain_id> <user_auth_token>\n" +
			"For example:\n" +
			"\tsupermq-cli groups roles actions available-actions 39f97daf-d6b6-40f4-b229-2697be8006ef $USER_AUTH_TOKEN\n",
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 2 {
				logUsageCmd(*cmd, cmd.Use)
				return
			}
			acts, err := sdk.AvailableGroupRoleActions(args[0], args[1])
			if err != nil {
				logErrorCmd(*cmd, err)
				return
			}
			logJSONCmd(*cmd, acts)
		},
	},
}

var cmdGroupMembers = []cobra.Command{
	{
		Use:   "add <JSON_members> <role_id> <group_id> <domain_id> <user_auth_token>",
		Short: "Add members to role",
		Long: "Add members to role\n" +
			"Usage:\n" +
			"\tsupermq-cli groups roles members add <JSON_members> <role_id> <group_id> <domain_id> <user_auth_token>\n" +
			"For example:\n" +
			"\tsupermq-cli groups roles members add '{\"members\":[\"5dc1ce4b-7cc9-4f12-98a6-9d74cc4980bb\", \"5dc1ce4b-7cc9-4f12-98a6-9d74cc4980bb\"]}' 39f97daf-d6b6-40f4-b229-2697be8006ef 4ef09eff-d500-4d56-b04f-d23a512d6f2a 4ef09eff-d500-4d56-b04f-d23a512d6f2a $USER_AUTH_TOKEN\n",
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 5 {
				logUsageCmd(*cmd, cmd.Use)
				return
			}
			members := struct {
				Members []string `json:"members"`
			}{}
			if err := json.Unmarshal([]byte(args[0]), &members); err != nil {
				logErrorCmd(*cmd, err)
				return
			}

			memb, err := sdk.AddGroupRoleMembers(args[2], args[1], args[3], members.Members, args[4])
			if err != nil {
				logErrorCmd(*cmd, err)
				return
			}
			logJSONCmd(*cmd, memb)
		},
	},

	{
		Use:   "list <role_id> <group_id> <domain_id> <user_auth_token>",
		Short: "List members of role",
		Long: "List members of role\n" +
			"Usage:\n" +
			"\tsupermq-cli groups roles members list <role_id> <domain_id> <user_auth_token>\n" +
			"For example:\n" +
			"\tsupermq-cli groups roles members list 39f97daf-d6b6-40f4-b229-2697be8006ef 4ef09eff-d500-4d56-b04f-d23a512d6f2a 4ef09eff-d500-4d56-b04f-d23a512d6f2a $USER_AUTH_TOKEN\n",
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 4 {
				logUsageCmd(*cmd, cmd.Use)
				return
			}
			pageMetadata := smqsdk.PageMetadata{
				Offset: Offset,
				Limit:  Limit,
			}

			l, err := sdk.GroupRoleMembers(args[1], args[0], args[2], pageMetadata, args[3])
			if err != nil {
				logErrorCmd(*cmd, err)
				return
			}
			logJSONCmd(*cmd, l)
		},
	},

	{
		Use:   "delete [all | <JSON_members>] <role_id> <group_id> <domain_id> <user_auth_token>",
		Short: "Delete members from role",
		Long: "Delete members from role\n" +
			"Usage:\n" +
			"\tsupermq-cli groups roles members delete <JSON_members> <role_id> <group_id> <domain_id> <user_auth_token>\n" +
			"\tsupermq-cli groups roles members delete all <role_id> <group_id> <domain_id> <user_auth_token>\n" +
			"For example:\n" +
			"\tsupermq-cli groups roles members delete all 39f97daf-d6b6-40f4-b229-2697be8006ef 4ef09eff-d500-4d56-b04f-d23a512d6f2a 4ef09eff-d500-4d56-b04f-d23a512d6f2a $USER_AUTH_TOKEN\n" +
			"\tsupermq-cli groups roles members delete '{\"members\":[\"5dc1ce4b-7cc9-4f12-98a6-9d74cc4980bb\", \"5dc1ce4b-7cc9-4f12-98a6-9d74cc4980bb\"]}' 39f97daf-d6b6-40f4-b229-2697be8006ef 4ef09eff-d500-4d56-b04f-d23a512d6f2a 4ef09eff-d500-4d56-b04f-d23a512d6f2a $USER_AUTH_TOKEN\n",
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 5 {
				logUsageCmd(*cmd, cmd.Use)
				return
			}
			if args[0] == all {
				if err := sdk.RemoveAllGroupRoleMembers(args[2], args[1], args[3], args[4]); err != nil {
					logErrorCmd(*cmd, err)
					return
				}
				logOKCmd(*cmd)
				return
			}

			members := struct {
				Members []string `json:"members"`
			}{}
			if err := json.Unmarshal([]byte(args[0]), &members); err != nil {
				logErrorCmd(*cmd, err)
				return
			}

			if err := sdk.RemoveGroupRoleMembers(args[2], args[1], args[3], members.Members, args[4]); err != nil {
				logErrorCmd(*cmd, err)
				return
			}
			logOKCmd(*cmd)
		},
	},
}

// NewGroupsCmd returns users command.
func NewGroupsCmd() *cobra.Command {
	actionsCmd := cobra.Command{
		Use:   "actions [add | list | delete | available-actions]",
		Short: "Actions management",
		Long:  "Actions management: add, list, delete actions and list available actions",
	}
	for i := range cmdGroupsActions {
		actionsCmd.AddCommand(&cmdGroupsActions[i])
	}

	membersCmd := cobra.Command{
		Use:   "members [add | list | delete]",
		Short: "Members management",
		Long:  "Members management: add, list, delete members",
	}
	for i := range cmdGroupMembers {
		membersCmd.AddCommand(&cmdGroupMembers[i])
	}

	rolesCmd := cobra.Command{
		Use:   "roles [create | get | update | delete | actions | members]",
		Short: "Roles management",
		Long:  "Roles management: create, update, retrieve roles and assign/unassign members to roles",
	}

	rolesCmd.AddCommand(&actionsCmd)
	rolesCmd.AddCommand(&membersCmd)

	for i := range cmdGroupsRoles {
		rolesCmd.AddCommand(&cmdGroupsRoles[i])
	}

	cmd := cobra.Command{
		Use:   "groups [create | get | update | delete | assign | unassign | users | channels ]",
		Short: "Groups management",
		Long:  `Groups management: create, update, delete group and assign and unassign member to groups"`,
	}

	cmd.AddCommand(&rolesCmd)

	for i := range cmdGroups {
		cmd.AddCommand(&cmdGroups[i])
	}

	return &cmd
}
