// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"encoding/json"
	"fmt"

	"github.com/absmach/magistrala/groups"
	smqsdk "github.com/absmach/magistrala/pkg/sdk"
	"github.com/spf13/cobra"
)

const (
	tags             = "tags"
	add              = "add"
	list             = "list"
	availableActions = "available-actions"

	// Usage strings for group operations.
	usageGroupCreate     = "cli groups create <JSON_group> <domain_id> <user_auth_token>"
	usageGroupGet        = "cli groups <group_id|all> get <domain_id> <user_auth_token>"
	usageGroupUpdate     = "cli groups <group_id> update <JSON_string> <domain_id> <user_auth_token>"
	usageGroupUpdateTags = "cli groups <group_id> update tags <tags> <domain_id> <user_auth_token>"
	usageGroupDelete     = "cli groups <group_id> delete <domain_id> <user_auth_token>"
	usageGroupEnable     = "cli groups <group_id> enable <domain_id> <user_auth_token>"
	usageGroupDisable    = "cli groups <group_id> disable <domain_id> <user_auth_token>"

	// Usage strings for group roles operations.
	usageGroupRolesCreate = "cli groups <group_id> roles create <JSON_role> <domain_id> <user_auth_token>"
	usageGroupRolesGet    = "cli groups <group_id> roles get <role_id|all> <domain_id> <user_auth_token>"
	usageGroupRolesUpdate = "cli groups <group_id> roles update <role_id> <new_name> <domain_id> <user_auth_token>"
	usageGroupRolesDelete = "cli groups <group_id> roles delete <role_id> <domain_id> <user_auth_token>"

	// Usage strings for group role actions operations.
	usageGroupRoleActionsAdd       = "cli groups <group_id> roles actions add <role_id> <JSON_actions> <domain_id> <user_auth_token>"
	usageGroupRoleActionsList      = "cli groups <group_id> roles actions list <role_id> <domain_id> <user_auth_token>"
	usageGroupRoleActionsDelete    = "cli groups <group_id> roles actions delete <role_id> <JSON_actions|all> <domain_id> <user_auth_token>"
	usageGroupRoleActionsAvailable = "cli groups roles actions available-actions <domain_id> <user_auth_token>"

	// Usage strings for group role members operations.
	usageGroupRoleMembersAdd    = "cli groups <group_id> roles members add <role_id> <JSON_members> <domain_id> <user_auth_token>"
	usageGroupRoleMembersList   = "cli groups <group_id> roles members list <role_id> <domain_id> <user_auth_token>"
	usageGroupRoleMembersDelete = "cli groups <group_id> roles members delete <role_id> <JSON_members|all> <domain_id> <user_auth_token>"
)

func NewGroupsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "groups <group_id|all|create> [operation] [args...]",
		Short: "Groups management",
		Long: `Format: 
  groups create [args...]
  groups <group_id|all> <operation> [args...]

Operations (require group_id/all): get, update, delete, enable, disable, roles

Examples:
  groups create <JSON_group> <domain_id> <user_auth_token>
  groups all get <domain_id> <user_auth_token>
  groups <group_id> get <domain_id> <user_auth_token>
  groups <group_id> update <JSON_string> <domain_id> <user_auth_token>
  groups <group_id> update tags <tags> <domain_id> <user_auth_token>
  groups <group_id> delete <domain_id> <user_auth_token>
  groups <group_id> enable <domain_id> <user_auth_token>
  groups <group_id> disable <domain_id> <user_auth_token>`,

		Run: func(cmd *cobra.Command, args []string) {
			if len(args) == 0 {
				logUsageCmd(*cmd, cmd.Use)
				return
			}

			if args[0] == create {
				handleGroupCreate(cmd, args[1:])
				return
			}

			if len(args) < 2 {
				logUsageCmd(*cmd, "groups <group_id|all> <get|update|delete|enable|disable|roles> [args...]")
				return
			}

			groupParams := args[0]
			operation := args[1]
			opArgs := args[2:]

			switch operation {
			case get:
				handleGroupGet(cmd, groupParams, opArgs)
			case update:
				handleGroupUpdate(cmd, groupParams, opArgs)
			case delete:
				handleGroupDelete(cmd, groupParams, opArgs)
			case enable:
				handleGroupEnable(cmd, groupParams, opArgs)
			case disable:
				handleGroupDisable(cmd, groupParams, opArgs)
			case roles:
				handleGroupRoles(cmd, groupParams, opArgs)
			default:
				logErrorCmd(*cmd, fmt.Errorf("unknown operation: %s", operation))
			}
		},
	}

	return cmd
}

func handleGroupCreate(cmd *cobra.Command, args []string) {
	if len(args) != 3 {
		logUsageCmd(*cmd, usageGroupCreate)
		return
	}

	var group smqsdk.Group
	if err := json.Unmarshal([]byte(args[0]), &group); err != nil {
		logErrorCmd(*cmd, err)
		return
	}
	group.Status = groups.EnabledStatus.String()
	group, err := sdk.CreateGroup(cmd.Context(), group, args[1], args[2])
	if err != nil {
		logErrorCmd(*cmd, err)
		return
	}
	logJSONCmd(*cmd, group)
}

func handleGroupGet(cmd *cobra.Command, groupParams string, args []string) {
	if len(args) != 2 {
		logUsageCmd(*cmd, usageGroupGet)
		return
	}

	if groupParams == all {
		metadata, err := convertMetadata(Metadata)
		if err != nil {
			logErrorCmd(*cmd, err)
			return
		}

		pageMetadata := smqsdk.PageMetadata{
			Name:     Name,
			Offset:   Offset,
			Limit:    Limit,
			Metadata: metadata,
		}

		l, err := sdk.Groups(cmd.Context(), pageMetadata, args[0], args[1])
		if err != nil {
			logErrorCmd(*cmd, err)
			return
		}
		logJSONCmd(*cmd, l)
		return
	}

	g, err := sdk.Group(cmd.Context(), groupParams, args[0], args[1])
	if err != nil {
		logErrorCmd(*cmd, err)
		return
	}

	logJSONCmd(*cmd, g)
}

func handleGroupUpdate(cmd *cobra.Command, groupID string, args []string) {
	if len(args) < 3 || len(args) > 4 {
		if args[0] == tags {
			logUsageCmd(*cmd, usageGroupUpdateTags)
			return
		}
		logUsageCmd(*cmd, usageGroupUpdate)
		return
	}

	if len(args) == 4 && args[0] == tags {
		var group smqsdk.Group
		if err := json.Unmarshal([]byte(args[1]), &group.Tags); err != nil {
			logErrorCmd(*cmd, err)
			return
		}
		group.ID = groupID
		group, err := sdk.UpdateGroupTags(cmd.Context(), group, args[2], args[3])
		if err != nil {
			logErrorCmd(*cmd, err)
			return
		}
		logJSONCmd(*cmd, group)
		return
	}

	if len(args) != 3 {
		logUsageCmd(*cmd, usageGroupUpdate)
		return
	}

	var group smqsdk.Group
	if err := json.Unmarshal([]byte(args[0]), &group); err != nil {
		logErrorCmd(*cmd, err)
		return
	}

	group.ID = groupID
	group, err := sdk.UpdateGroup(cmd.Context(), group, args[1], args[2])
	if err != nil {
		logErrorCmd(*cmd, err)
		return
	}

	logJSONCmd(*cmd, group)
}

func handleGroupDelete(cmd *cobra.Command, groupID string, args []string) {
	if len(args) != 2 {
		logUsageCmd(*cmd, usageGroupDelete)
		return
	}

	if err := sdk.DeleteGroup(cmd.Context(), groupID, args[0], args[1]); err != nil {
		logErrorCmd(*cmd, err)
		return
	}
	logOKCmd(*cmd)
}

func handleGroupEnable(cmd *cobra.Command, groupID string, args []string) {
	if len(args) != 2 {
		logUsageCmd(*cmd, usageGroupEnable)
		return
	}

	group, err := sdk.EnableGroup(cmd.Context(), groupID, args[0], args[1])
	if err != nil {
		logErrorCmd(*cmd, err)
		return
	}

	logJSONCmd(*cmd, group)
}

func handleGroupDisable(cmd *cobra.Command, groupID string, args []string) {
	if len(args) != 2 {
		logUsageCmd(*cmd, usageGroupDisable)
		return
	}

	group, err := sdk.DisableGroup(cmd.Context(), groupID, args[0], args[1])
	if err != nil {
		logErrorCmd(*cmd, err)
		return
	}

	logJSONCmd(*cmd, group)
}

func handleGroupRoles(cmd *cobra.Command, groupID string, args []string) {
	if len(args) < 1 {
		logUsageCmd(*cmd, "cli groups <group_id> roles <operation> [args...]")
		return
	}

	operation := args[0]
	opArgs := args[1:]

	switch operation {
	case create:
		handleGroupRoleCreate(cmd, groupID, opArgs)
	case get:
		handleGroupRoleGet(cmd, groupID, opArgs)
	case update:
		handleGroupRoleUpdate(cmd, groupID, opArgs)
	case delete:
		handleGroupRoleDelete(cmd, groupID, opArgs)
	case actions:
		handleGroupRoleActions(cmd, groupID, opArgs)
	case members:
		handleGroupRoleMembers(cmd, groupID, opArgs)
	default:
		logErrorCmd(*cmd, fmt.Errorf("unknown roles operation: %s", operation))
	}
}

func handleGroupRoleCreate(cmd *cobra.Command, groupID string, args []string) {
	if len(args) != 3 {
		logUsageCmd(*cmd, usageGroupRolesCreate)
		return
	}

	var roleReq smqsdk.RoleReq
	if err := json.Unmarshal([]byte(args[0]), &roleReq); err != nil {
		logErrorCmd(*cmd, err)
		return
	}

	r, err := sdk.CreateGroupRole(cmd.Context(), groupID, args[1], roleReq, args[2])
	if err != nil {
		logErrorCmd(*cmd, err)
		return
	}

	logJSONCmd(*cmd, r)
}

func handleGroupRoleGet(cmd *cobra.Command, groupID string, args []string) {
	if len(args) != 3 {
		logUsageCmd(*cmd, usageGroupRolesGet)
		return
	}

	roleID := args[0]
	domainID := args[1]
	token := args[2]

	if roleID == all {
		pageMetadata := smqsdk.PageMetadata{
			Offset: Offset,
			Limit:  Limit,
		}
		rs, err := sdk.GroupRoles(cmd.Context(), groupID, domainID, pageMetadata, token)
		if err != nil {
			logErrorCmd(*cmd, err)
			return
		}
		logJSONCmd(*cmd, rs)
		return
	}

	r, err := sdk.GroupRole(cmd.Context(), groupID, roleID, domainID, token)
	if err != nil {
		logErrorCmd(*cmd, err)
		return
	}
	logJSONCmd(*cmd, r)
}

func handleGroupRoleUpdate(cmd *cobra.Command, groupID string, args []string) {
	if len(args) != 4 {
		logUsageCmd(*cmd, usageGroupRolesUpdate)
		return
	}

	roleID := args[0]
	newName := args[1]
	domainID := args[2]
	token := args[3]

	r, err := sdk.UpdateGroupRole(cmd.Context(), groupID, roleID, newName, domainID, token)
	if err != nil {
		logErrorCmd(*cmd, err)
		return
	}
	logJSONCmd(*cmd, r)
}

func handleGroupRoleDelete(cmd *cobra.Command, groupID string, args []string) {
	if len(args) != 3 {
		logUsageCmd(*cmd, usageGroupRolesDelete)
		return
	}

	roleID := args[0]
	domainID := args[1]
	token := args[2]

	if err := sdk.DeleteGroupRole(cmd.Context(), groupID, roleID, domainID, token); err != nil {
		logErrorCmd(*cmd, err)
		return
	}
	logOKCmd(*cmd)
}

func handleGroupRoleActions(cmd *cobra.Command, groupID string, args []string) {
	if len(args) < 1 {
		logUsageCmd(*cmd, "cli groups <group_id> roles actions <operation> [args...]")
		return
	}

	operation := args[0]
	opArgs := args[1:]

	switch operation {
	case add:
		handleGroupRoleActionsAdd(cmd, groupID, opArgs)
	case list:
		handleGroupRoleActionsList(cmd, groupID, opArgs)
	case delete:
		handleGroupRoleActionsDelete(cmd, groupID, opArgs)
	case availableActions:
		handleGroupRoleActionsAvailable(cmd, opArgs)
	default:
		logErrorCmd(*cmd, fmt.Errorf("unknown actions operation: %s", operation))
	}
}

func handleGroupRoleActionsAdd(cmd *cobra.Command, groupID string, args []string) {
	if len(args) != 4 {
		logUsageCmd(*cmd, usageGroupRoleActionsAdd)
		return
	}

	roleID := args[0]
	actionsJSON := args[1]
	domainID := args[2]
	token := args[3]

	actions := struct {
		Actions []string `json:"actions"`
	}{}
	if err := json.Unmarshal([]byte(actionsJSON), &actions); err != nil {
		logErrorCmd(*cmd, err)
		return
	}

	acts, err := sdk.AddGroupRoleActions(cmd.Context(), groupID, roleID, domainID, actions.Actions, token)
	if err != nil {
		logErrorCmd(*cmd, err)
		return
	}
	logJSONCmd(*cmd, acts)
}

func handleGroupRoleActionsList(cmd *cobra.Command, groupID string, args []string) {
	if len(args) != 3 {
		logUsageCmd(*cmd, usageGroupRoleActionsList)
		return
	}

	roleID := args[0]
	domainID := args[1]
	token := args[2]

	l, err := sdk.GroupRoleActions(cmd.Context(), groupID, roleID, domainID, token)
	if err != nil {
		logErrorCmd(*cmd, err)
		return
	}
	logJSONCmd(*cmd, l)
}

func handleGroupRoleActionsDelete(cmd *cobra.Command, groupID string, args []string) {
	if len(args) != 4 {
		logUsageCmd(*cmd, usageGroupRoleActionsDelete)
		return
	}

	roleID := args[0]
	actionsJSON := args[1]
	domainID := args[2]
	token := args[3]

	if actionsJSON == all {
		if err := sdk.RemoveAllGroupRoleActions(cmd.Context(), groupID, roleID, domainID, token); err != nil {
			logErrorCmd(*cmd, err)
			return
		}
		logOKCmd(*cmd)
		return
	}

	actions := struct {
		Actions []string `json:"actions"`
	}{}
	if err := json.Unmarshal([]byte(actionsJSON), &actions); err != nil {
		logErrorCmd(*cmd, err)
		return
	}

	if err := sdk.RemoveGroupRoleActions(cmd.Context(), groupID, roleID, domainID, actions.Actions, token); err != nil {
		logErrorCmd(*cmd, err)
		return
	}
	logOKCmd(*cmd)
}

func handleGroupRoleActionsAvailable(cmd *cobra.Command, args []string) {
	if len(args) != 2 {
		logUsageCmd(*cmd, usageGroupRoleActionsAvailable)
		return
	}

	domainID := args[0]
	token := args[1]

	acts, err := sdk.AvailableGroupRoleActions(cmd.Context(), domainID, token)
	if err != nil {
		logErrorCmd(*cmd, err)
		return
	}
	logJSONCmd(*cmd, acts)
}

func handleGroupRoleMembers(cmd *cobra.Command, groupID string, args []string) {
	if len(args) < 1 {
		logUsageCmd(*cmd, "cli groups <group_id> roles members <operation> [args...]")
		return
	}

	operation := args[0]
	opArgs := args[1:]

	switch operation {
	case add:
		handleGroupRoleMembersAdd(cmd, groupID, opArgs)
	case list:
		handleGroupRoleMembersList(cmd, groupID, opArgs)
	case delete:
		handleGroupRoleMembersDelete(cmd, groupID, opArgs)
	default:
		logErrorCmd(*cmd, fmt.Errorf("unknown members operation: %s", operation))
	}
}

func handleGroupRoleMembersAdd(cmd *cobra.Command, groupID string, args []string) {
	if len(args) != 4 {
		logUsageCmd(*cmd, usageGroupRoleMembersAdd)
		return
	}

	roleID := args[0]
	membersJSON := args[1]
	domainID := args[2]
	token := args[3]

	members := struct {
		Members []string `json:"members"`
	}{}
	if err := json.Unmarshal([]byte(membersJSON), &members); err != nil {
		logErrorCmd(*cmd, err)
		return
	}

	memb, err := sdk.AddGroupRoleMembers(cmd.Context(), groupID, roleID, domainID, members.Members, token)
	if err != nil {
		logErrorCmd(*cmd, err)
		return
	}
	logJSONCmd(*cmd, memb)
}

func handleGroupRoleMembersList(cmd *cobra.Command, groupID string, args []string) {
	if len(args) != 3 {
		logUsageCmd(*cmd, usageGroupRoleMembersList)
		return
	}

	roleID := args[0]
	domainID := args[1]
	token := args[2]

	pageMetadata := smqsdk.PageMetadata{
		Offset: Offset,
		Limit:  Limit,
	}

	l, err := sdk.GroupRoleMembers(cmd.Context(), groupID, roleID, domainID, pageMetadata, token)
	if err != nil {
		logErrorCmd(*cmd, err)
		return
	}
	logJSONCmd(*cmd, l)
}

func handleGroupRoleMembersDelete(cmd *cobra.Command, groupID string, args []string) {
	if len(args) != 4 {
		logUsageCmd(*cmd, usageGroupRoleMembersDelete)
		return
	}

	roleID := args[0]
	membersJSON := args[1]
	domainID := args[2]
	token := args[3]

	if membersJSON == all {
		if err := sdk.RemoveAllGroupRoleMembers(cmd.Context(), groupID, roleID, domainID, token); err != nil {
			logErrorCmd(*cmd, err)
			return
		}
		logOKCmd(*cmd)
		return
	}

	members := struct {
		Members []string `json:"members"`
	}{}
	if err := json.Unmarshal([]byte(membersJSON), &members); err != nil {
		logErrorCmd(*cmd, err)
		return
	}

	if err := sdk.RemoveGroupRoleMembers(cmd.Context(), groupID, roleID, domainID, members.Members, token); err != nil {
		logErrorCmd(*cmd, err)
		return
	}
	logOKCmd(*cmd)
}
