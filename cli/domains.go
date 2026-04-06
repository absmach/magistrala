// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"encoding/json"
	"fmt"

	smqsdk "github.com/absmach/magistrala/pkg/sdk"
	"github.com/spf13/cobra"
)

const (
	freeze = "freeze"

	// Usage strings for domain operations.
	usageDomainCreate  = "cli domains create <domain_name> <route> <user_auth_token>"
	usageDomainGet     = "cli domains <domain_id|all> get <user_auth_token>"
	usageDomainUpdate  = "cli domains <domain_id> update <JSON_string> <user_auth_token>"
	usageDomainEnable  = "cli domains <domain_id> enable <user_auth_token>"
	usageDomainDisable = "cli domains <domain_id> disable <user_auth_token>"
	usageDomainFreeze  = "cli domains <domain_id> freeze <user_auth_token>"
	usageDomainUsers   = "cli domains <domain_id> users <user_auth_token>"

	// Usage strings for domain roles operations.
	usageDomainRolesCreate = "cli domains <domain_id> roles create <JSON_role> <user_auth_token>"
	usageDomainRolesGet    = "cli domains <domain_id> roles get <role_id|all> <user_auth_token>"
	usageDomainRolesUpdate = "cli domains <domain_id> roles update <role_id> <new_name> <user_auth_token>"
	usageDomainRolesDelete = "cli domains <domain_id> roles delete <role_id> <user_auth_token>"

	// Usage strings for domain role actions operations.
	usageDomainRoleActionsAdd       = "cli domains <domain_id> roles actions add <role_id> <JSON_actions> <user_auth_token>"
	usageDomainRoleActionsList      = "cli domains <domain_id> roles actions list <role_id> <user_auth_token>"
	usageDomainRoleActionsDelete    = "cli domains <domain_id> roles actions delete <role_id> <JSON_actions|all> <user_auth_token>"
	usageDomainRoleActionsAvailable = "cli domains roles actions available-actions <user_auth_token>"

	// Usage strings for domain role members operations.
	usageDomainRoleMembersAdd    = "cli domains <domain_id> roles members add <role_id> <JSON_members> <user_auth_token>"
	usageDomainRoleMembersList   = "cli domains <domain_id> roles members list <role_id> <user_auth_token>"
	usageDomainRoleMembersDelete = "cli domains <domain_id> roles members delete <role_id> <JSON_members|all> <user_auth_token>"
)

func NewDomainsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "domains <domain_id|all|create> [operation] [args...]",
		Short: "Domains management",
		Long: `Format: 
  domains create [args...]
  domains <domain_id|all> <operation> [args...]

Operations (require domain_id/all): get, update, enable, disable, freeze, users, roles

Examples:
  domains create <domain_name> <route> <user_auth_token>
  domains all get <user_auth_token>
  domains <domain_id> get <user_auth_token>
  domains <domain_id> update <JSON_string> <user_auth_token>
  domains <domain_id> enable <user_auth_token>
  domains <domain_id> disable <user_auth_token>
  domains <domain_id> freeze <user_auth_token>
  domains <domain_id> users <user_auth_token>`,

		Run: func(cmd *cobra.Command, args []string) {
			if len(args) == 0 {
				logUsageCmd(*cmd, cmd.Use)
				return
			}

			if args[0] == create {
				handleDomainCreate(cmd, args[1:])
				return
			}

			if len(args) < 2 {
				logUsageCmd(*cmd, "domains <domain_id|all> <get|update|enable|disable|freeze|users|roles> [args...]")
				return
			}

			domainParams := args[0]
			operation := args[1]
			opArgs := args[2:]

			switch operation {
			case get:
				handleDomainGet(cmd, domainParams, opArgs)
			case update:
				handleDomainUpdate(cmd, domainParams, opArgs)
			case enable:
				handleDomainEnable(cmd, domainParams, opArgs)
			case disable:
				handleDomainDisable(cmd, domainParams, opArgs)
			case freeze:
				handleDomainFreeze(cmd, domainParams, opArgs)
			case users:
				handleDomainUsers(cmd, domainParams, opArgs)
			case roles:
				handleDomainRoles(cmd, domainParams, opArgs)
			default:
				logErrorCmd(*cmd, fmt.Errorf("unknown operation: %s", operation))
			}
		},
	}

	return cmd
}

func handleDomainCreate(cmd *cobra.Command, args []string) {
	if len(args) != 3 {
		logUsageCmd(*cmd, usageDomainCreate)
		return
	}

	dom := smqsdk.Domain{
		Name:  args[0],
		Route: args[1],
	}
	d, err := sdk.CreateDomain(cmd.Context(), dom, args[2])
	if err != nil {
		logErrorCmd(*cmd, err)
		return
	}
	logJSONCmd(*cmd, d)
}

func handleDomainGet(cmd *cobra.Command, domainParams string, args []string) {
	if len(args) != 1 {
		logUsageCmd(*cmd, usageDomainGet)
		return
	}

	if domainParams == all {
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
			Status:   Status,
		}

		l, err := sdk.Domains(cmd.Context(), pageMetadata, args[0])
		if err != nil {
			logErrorCmd(*cmd, err)
			return
		}
		logJSONCmd(*cmd, l)
		return
	}

	d, err := sdk.Domain(cmd.Context(), domainParams, args[0])
	if err != nil {
		logErrorCmd(*cmd, err)
		return
	}

	logJSONCmd(*cmd, d)
}

func handleDomainUpdate(cmd *cobra.Command, domainID string, args []string) {
	if len(args) != 2 {
		logUsageCmd(*cmd, usageDomainUpdate)
		return
	}

	var d smqsdk.Domain
	if err := json.Unmarshal([]byte(args[0]), &d); err != nil {
		logErrorCmd(*cmd, err)
		return
	}
	d.ID = domainID
	d, err := sdk.UpdateDomain(cmd.Context(), d, args[1])
	if err != nil {
		logErrorCmd(*cmd, err)
		return
	}
	logJSONCmd(*cmd, d)
}

func handleDomainEnable(cmd *cobra.Command, domainID string, args []string) {
	if len(args) != 1 {
		logUsageCmd(*cmd, usageDomainEnable)
		return
	}

	if err := sdk.EnableDomain(cmd.Context(), domainID, args[0]); err != nil {
		logErrorCmd(*cmd, err)
		return
	}
	logOKCmd(*cmd)
}

func handleDomainDisable(cmd *cobra.Command, domainID string, args []string) {
	if len(args) != 1 {
		logUsageCmd(*cmd, usageDomainDisable)
		return
	}

	if err := sdk.DisableDomain(cmd.Context(), domainID, args[0]); err != nil {
		logErrorCmd(*cmd, err)
		return
	}
	logOKCmd(*cmd)
}

func handleDomainFreeze(cmd *cobra.Command, domainID string, args []string) {
	if len(args) != 1 {
		logUsageCmd(*cmd, usageDomainFreeze)
		return
	}

	if err := sdk.FreezeDomain(cmd.Context(), domainID, args[0]); err != nil {
		logErrorCmd(*cmd, err)
		return
	}
	logOKCmd(*cmd)
}

func handleDomainUsers(cmd *cobra.Command, domainID string, args []string) {
	if len(args) != 1 {
		logUsageCmd(*cmd, usageDomainUsers)
		return
	}

	metadata, err := convertMetadata(Metadata)
	if err != nil {
		logErrorCmd(*cmd, err)
		return
	}
	pageMetadata := smqsdk.PageMetadata{
		Offset:   Offset,
		Limit:    Limit,
		Metadata: metadata,
		Status:   Status,
	}

	l, err := sdk.ListDomainMembers(cmd.Context(), domainID, pageMetadata, args[0])
	if err != nil {
		logErrorCmd(*cmd, err)
		return
	}
	logJSONCmd(*cmd, l)
}

func handleDomainRoles(cmd *cobra.Command, domainID string, args []string) {
	if len(args) < 1 {
		logUsageCmd(*cmd, "cli domains <domain_id> roles <operation> [args...]")
		return
	}

	operation := args[0]
	opArgs := args[1:]

	switch operation {
	case create:
		handleDomainRoleCreate(cmd, domainID, opArgs)
	case get:
		handleDomainRoleGet(cmd, domainID, opArgs)
	case update:
		handleDomainRoleUpdate(cmd, domainID, opArgs)
	case delete:
		handleDomainRoleDelete(cmd, domainID, opArgs)
	case actions:
		handleDomainRoleActions(cmd, domainID, opArgs)
	case members:
		handleDomainRoleMembers(cmd, domainID, opArgs)
	default:
		logErrorCmd(*cmd, fmt.Errorf("unknown roles operation: %s", operation))
	}
}

func handleDomainRoleCreate(cmd *cobra.Command, domainID string, args []string) {
	if len(args) != 2 {
		logUsageCmd(*cmd, usageDomainRolesCreate)
		return
	}

	var roleReq smqsdk.RoleReq
	if err := json.Unmarshal([]byte(args[0]), &roleReq); err != nil {
		logErrorCmd(*cmd, err)
		return
	}

	r, err := sdk.CreateDomainRole(cmd.Context(), domainID, roleReq, args[1])
	if err != nil {
		logErrorCmd(*cmd, err)
		return
	}

	logJSONCmd(*cmd, r)
}

func handleDomainRoleGet(cmd *cobra.Command, domainID string, args []string) {
	if len(args) != 2 {
		logUsageCmd(*cmd, usageDomainRolesGet)
		return
	}

	roleID := args[0]
	token := args[1]

	if roleID == all {
		pageMetadata := smqsdk.PageMetadata{
			Offset: Offset,
			Limit:  Limit,
		}
		rs, err := sdk.DomainRoles(cmd.Context(), domainID, pageMetadata, token)
		if err != nil {
			logErrorCmd(*cmd, err)
			return
		}
		logJSONCmd(*cmd, rs)
		return
	}

	r, err := sdk.DomainRole(cmd.Context(), domainID, roleID, token)
	if err != nil {
		logErrorCmd(*cmd, err)
		return
	}
	logJSONCmd(*cmd, r)
}

func handleDomainRoleUpdate(cmd *cobra.Command, domainID string, args []string) {
	if len(args) != 3 {
		logUsageCmd(*cmd, usageDomainRolesUpdate)
		return
	}

	roleID := args[0]
	newName := args[1]
	token := args[2]

	r, err := sdk.UpdateDomainRole(cmd.Context(), domainID, roleID, newName, token)
	if err != nil {
		logErrorCmd(*cmd, err)
		return
	}
	logJSONCmd(*cmd, r)
}

func handleDomainRoleDelete(cmd *cobra.Command, domainID string, args []string) {
	if len(args) != 2 {
		logUsageCmd(*cmd, usageDomainRolesDelete)
		return
	}

	roleID := args[0]
	token := args[1]

	if err := sdk.DeleteDomainRole(cmd.Context(), domainID, roleID, token); err != nil {
		logErrorCmd(*cmd, err)
		return
	}
	logOKCmd(*cmd)
}

func handleDomainRoleActions(cmd *cobra.Command, domainID string, args []string) {
	if len(args) < 1 {
		logUsageCmd(*cmd, "cli domains <domain_id> roles actions <operation> [args...]")
		return
	}

	operation := args[0]
	opArgs := args[1:]

	switch operation {
	case add:
		handleDomainRoleActionsAdd(cmd, domainID, opArgs)
	case list:
		handleDomainRoleActionsList(cmd, domainID, opArgs)
	case delete:
		handleDomainRoleActionsDelete(cmd, domainID, opArgs)
	case availableActions:
		handleDomainRoleActionsAvailable(cmd, opArgs)
	default:
		logErrorCmd(*cmd, fmt.Errorf("unknown actions operation: %s", operation))
	}
}

func handleDomainRoleActionsAdd(cmd *cobra.Command, domainID string, args []string) {
	if len(args) != 3 {
		logUsageCmd(*cmd, usageDomainRoleActionsAdd)
		return
	}

	roleID := args[0]
	actionsJSON := args[1]
	token := args[2]

	actions := struct {
		Actions []string `json:"actions"`
	}{}
	if err := json.Unmarshal([]byte(actionsJSON), &actions); err != nil {
		logErrorCmd(*cmd, err)
		return
	}

	acts, err := sdk.AddDomainRoleActions(cmd.Context(), domainID, roleID, actions.Actions, token)
	if err != nil {
		logErrorCmd(*cmd, err)
		return
	}
	logJSONCmd(*cmd, acts)
}

func handleDomainRoleActionsList(cmd *cobra.Command, domainID string, args []string) {
	if len(args) != 2 {
		logUsageCmd(*cmd, usageDomainRoleActionsList)
		return
	}

	roleID := args[0]
	token := args[1]

	l, err := sdk.DomainRoleActions(cmd.Context(), domainID, roleID, token)
	if err != nil {
		logErrorCmd(*cmd, err)
		return
	}
	logJSONCmd(*cmd, l)
}

func handleDomainRoleActionsDelete(cmd *cobra.Command, domainID string, args []string) {
	if len(args) != 3 {
		logUsageCmd(*cmd, usageDomainRoleActionsDelete)
		return
	}

	roleID := args[0]
	actionsJSON := args[1]
	token := args[2]

	if actionsJSON == all {
		if err := sdk.RemoveAllDomainRoleActions(cmd.Context(), domainID, roleID, token); err != nil {
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

	if err := sdk.RemoveDomainRoleActions(cmd.Context(), domainID, roleID, actions.Actions, token); err != nil {
		logErrorCmd(*cmd, err)
		return
	}
	logOKCmd(*cmd)
}

func handleDomainRoleActionsAvailable(cmd *cobra.Command, args []string) {
	if len(args) != 1 {
		logUsageCmd(*cmd, usageDomainRoleActionsAvailable)
		return
	}

	token := args[0]

	acts, err := sdk.AvailableDomainRoleActions(cmd.Context(), token)
	if err != nil {
		logErrorCmd(*cmd, err)
		return
	}
	logJSONCmd(*cmd, acts)
}

func handleDomainRoleMembers(cmd *cobra.Command, domainID string, args []string) {
	if len(args) < 1 {
		logUsageCmd(*cmd, "cli domains <domain_id> roles members <operation> [args...]")
		return
	}

	operation := args[0]
	opArgs := args[1:]

	switch operation {
	case add:
		handleDomainRoleMembersAdd(cmd, domainID, opArgs)
	case list:
		handleDomainRoleMembersList(cmd, domainID, opArgs)
	case delete:
		handleDomainRoleMembersDelete(cmd, domainID, opArgs)
	default:
		logErrorCmd(*cmd, fmt.Errorf("unknown members operation: %s", operation))
	}
}

func handleDomainRoleMembersAdd(cmd *cobra.Command, domainID string, args []string) {
	if len(args) != 3 {
		logUsageCmd(*cmd, usageDomainRoleMembersAdd)
		return
	}

	roleID := args[0]
	membersJSON := args[1]
	token := args[2]

	members := struct {
		Members []string `json:"members"`
	}{}
	if err := json.Unmarshal([]byte(membersJSON), &members); err != nil {
		logErrorCmd(*cmd, err)
		return
	}

	memb, err := sdk.AddDomainRoleMembers(cmd.Context(), domainID, roleID, members.Members, token)
	if err != nil {
		logErrorCmd(*cmd, err)
		return
	}
	logJSONCmd(*cmd, memb)
}

func handleDomainRoleMembersList(cmd *cobra.Command, domainID string, args []string) {
	if len(args) != 2 {
		logUsageCmd(*cmd, usageDomainRoleMembersList)
		return
	}

	roleID := args[0]
	token := args[1]

	pageMetadata := smqsdk.PageMetadata{
		Offset: Offset,
		Limit:  Limit,
	}

	l, err := sdk.DomainRoleMembers(cmd.Context(), domainID, roleID, pageMetadata, token)
	if err != nil {
		logErrorCmd(*cmd, err)
		return
	}
	logJSONCmd(*cmd, l)
}

func handleDomainRoleMembersDelete(cmd *cobra.Command, domainID string, args []string) {
	if len(args) != 3 {
		logUsageCmd(*cmd, usageDomainRoleMembersDelete)
		return
	}

	roleID := args[0]
	membersJSON := args[1]
	token := args[2]

	if membersJSON == all {
		if err := sdk.RemoveAllDomainRoleMembers(cmd.Context(), domainID, roleID, token); err != nil {
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

	if err := sdk.RemoveDomainRoleMembers(cmd.Context(), domainID, roleID, members.Members, token); err != nil {
		logErrorCmd(*cmd, err)
		return
	}
	logOKCmd(*cmd)
}
