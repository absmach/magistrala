// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"encoding/json"

	smqsdk "github.com/absmach/supermq/pkg/sdk"
	"github.com/spf13/cobra"
)

var cmdDomains = []cobra.Command{
	{
		Use:   "create <name> <alias> <token>",
		Short: "Create Domain",
		Long: "Create Domain with provided name and alias. \n" +
			"For example:\n" +
			"\tsupermq-cli domains create domain_1 domain_1_alias $TOKEN\n",
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 3 {
				logUsageCmd(*cmd, cmd.Use)
				return
			}

			dom := smqsdk.Domain{
				Name:  args[0],
				Alias: args[1],
			}
			d, err := sdk.CreateDomain(dom, args[2])
			if err != nil {
				logErrorCmd(*cmd, err)
				return
			}
			logJSONCmd(*cmd, d)
		},
	},
	{
		Use:   "get [all | <domain_id> ] <token>",
		Short: "Get Domains",
		Long:  "Get all domains. Users can be filtered by name or metadata or status",
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
			pageMetadata := smqsdk.PageMetadata{
				Name:     Name,
				Offset:   Offset,
				Limit:    Limit,
				Metadata: metadata,
				Status:   Status,
			}
			if args[0] == all {
				l, err := sdk.Domains(pageMetadata, args[1])
				if err != nil {
					logErrorCmd(*cmd, err)
					return
				}
				logJSONCmd(*cmd, l)
				return
			}
			d, err := sdk.Domain(args[0], args[1])
			if err != nil {
				logErrorCmd(*cmd, err)
				return
			}

			logJSONCmd(*cmd, d)
		},
	},

	{
		Use:   "users <domain_id>  <token>",
		Short: "List Domain users",
		Long:  "List Domain users",
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
			pageMetadata := smqsdk.PageMetadata{
				Offset:   Offset,
				Limit:    Limit,
				Metadata: metadata,
				Status:   Status,
			}

			l, err := sdk.ListDomainMembers(args[0], pageMetadata, args[1])
			if err != nil {
				logErrorCmd(*cmd, err)
				return
			}
			logJSONCmd(*cmd, l)
		},
	},

	{
		Use:   "update <domain_id> <JSON_string> <user_auth_token>",
		Short: "Update domains",
		Long: "Updates domains name, alias and metadata \n" +
			"Usage:\n" +
			"\tsupermq-cli domains update <domain_id> '{\"name\":\"new name\", \"alias\":\"new_alias\", \"metadata\":{\"key\": \"value\"}}' $TOKEN \n",
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 4 && len(args) != 3 {
				logUsageCmd(*cmd, cmd.Use)
				return
			}

			var d smqsdk.Domain

			if err := json.Unmarshal([]byte(args[1]), &d); err != nil {
				logErrorCmd(*cmd, err)
				return
			}
			d.ID = args[0]
			d, err := sdk.UpdateDomain(d, args[2])
			if err != nil {
				logErrorCmd(*cmd, err)
				return
			}
			logJSONCmd(*cmd, d)
		},
	},

	{
		Use:   "enable <domain_id> <token>",
		Short: "Change domain status to enabled",
		Long: "Change domain status to enabled\n" +
			"Usage:\n" +
			"\tsupermq-cli domains enable <domain_id> <token>\n",
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 2 {
				logUsageCmd(*cmd, cmd.Use)
				return
			}

			if err := sdk.EnableDomain(args[0], args[1]); err != nil {
				logErrorCmd(*cmd, err)
				return
			}
			logOKCmd(*cmd)
		},
	},

	{
		Use:   "disable <domain_id> <token>",
		Short: "Change domain status to disabled",
		Long: "Change domain status to disabled\n" +
			"Usage:\n" +
			"\tsupermq-cli domains disable <domain_id> <token>\n",
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 2 {
				logUsageCmd(*cmd, cmd.Use)
				return
			}

			if err := sdk.DisableDomain(args[0], args[1]); err != nil {
				logErrorCmd(*cmd, err)
				return
			}
			logOKCmd(*cmd)
		},
	},

	{
		Use:   "freeze <domain_id> <token>",
		Short: "Change domain status to frozen",
		Long: "Change domain status to frozen\n" +
			"Usage:\n" +
			"\tsupermq-cli domains freeze <domain_id> <token>\n",
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 2 {
				logUsageCmd(*cmd, cmd.Use)
				return
			}

			if err := sdk.FreezeDomain(args[0], args[1]); err != nil {
				logErrorCmd(*cmd, err)
				return
			}
			logOKCmd(*cmd)
		},
	},
}

var cmdDomainRoles = []cobra.Command{
	{
		Use:   "create <JSON_role> <domain_id> <user_auth_token>",
		Short: "Create domain role",
		Long: "Create role\n" +
			"Usage:\n" +
			"\tsupermq-cli domains roles create <JSON_role> <domain_id> <user_auth_token>\n" +
			"For example:\n" +
			"\tsupermq-cli domains roles create '{\"role_name\":\"admin\",\"optional_actions\":[\"read\",\"update\"]}' 4ef09eff-d500-4d56-b04f-d23a512d6f2a $USER_AUTH_TOKEN\n",
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 3 {
				logUsageCmd(*cmd, cmd.Use)
				return
			}
			var roleReq smqsdk.RoleReq
			if err := json.Unmarshal([]byte(args[0]), &roleReq); err != nil {
				logErrorCmd(*cmd, err)
				return
			}
			r, err := sdk.CreateDomainRole(args[1], roleReq, args[2])
			if err != nil {
				logErrorCmd(*cmd, err)
				return
			}

			logJSONCmd(*cmd, r)
		},
	},

	{
		Use:   "get [all | <role_id>] <domain_id> <user_auth_token>",
		Short: "Get roles",
		Long: "Get roles\n" +
			"Usage:\n" +
			"\tsupermq-cli domains roles get all <domain_id> <user_auth_token> - lists all roles\n" +
			"\tsupermq-cli domains roles get all <domain_id> <user_auth_token> --offset <offset> --limit <limit> - lists all roles with provided offset and limit\n" +
			"\tsupermq-cli domains roles get <role_id> <domain_id> <user_auth_token> - shows role by role id and domain id\n",
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 3 {
				logUsageCmd(*cmd, cmd.Use)
				return
			}
			pageMetadata := smqsdk.PageMetadata{
				Offset: Offset,
				Limit:  Limit,
			}
			if args[0] == all {
				rs, err := sdk.DomainRoles(args[1], pageMetadata, args[2])
				if err != nil {
					logErrorCmd(*cmd, err)
					return
				}
				logJSONCmd(*cmd, rs)
				return
			}
			r, err := sdk.DomainRole(args[0], args[1], args[2])
			if err != nil {
				logErrorCmd(*cmd, err)
				return
			}
			logJSONCmd(*cmd, r)
		},
	},

	{
		Use:   "update <new_name> <role_id> <domain_id> <user_auth_token>",
		Short: "Update role name",
		Long: "Update role name\n" +
			"Usage:\n" +
			"\tsupermq-cli domains roles update <new_name> <role_id> <domain_id> <user_auth_token>\n" +
			"For example:\n" +
			"\tsupermq-cli domains roles update 'new_name' 39f97daf-d6b6-40f4-b229-2697be8006ef 4ef09eff-d500-4d56-b04f-d23a512d6f2a $USER_AUTH_TOKEN\n",
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 4 {
				logUsageCmd(*cmd, cmd.Use)
				return
			}

			r, err := sdk.UpdateDomainRole(args[2], args[1], args[0], args[3])
			if err != nil {
				logErrorCmd(*cmd, err)
				return
			}
			logJSONCmd(*cmd, r)
		},
	},

	{
		Use:   "delete <role_id> <domain_id> <user_auth_token>",
		Short: "Delete role",
		Long: "Delete role\n" +
			"Usage:\n" +
			"\tsupermq-cli domains roles delete <role_id> <domain_id> <user_auth_token>\n" +
			"For example:\n" +
			"\tsupermq-cli domains roles delete 39f97daf-d6b6-40f4-b229-2697be8006ef 4ef09eff-d500-4d56-b04f-d23a512d6f2a $USER_AUTH_TOKEN\n",
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 3 {
				logUsageCmd(*cmd, cmd.Use)
				return
			}

			if err := sdk.DeleteDomainRole(args[1], args[0], args[2]); err != nil {
				logErrorCmd(*cmd, err)
				return
			}
			logOKCmd(*cmd)
		},
	},
}

var cmdDomainsActions = []cobra.Command{
	{
		Use:   "add <JSON_actions> <role_id> <domain_id> <user_auth_token>",
		Short: "Add actions to role",
		Long: "Add actions to role\n" +
			"Usage:\n" +
			"\tsupermq-cli domains roles actions add <JSON_actions> <role_id> <domain_id> <user_auth_token>\n" +
			"For example:\n" +
			"\tsupermq-cli domains roles actions add '{\"actions\":[\"read\",\"write\"]}' 39f97daf-d6b6-40f4-b229-2697be8006ef 4ef09eff-d500-4d56-b04f-d23a512d6f2a $USER_AUTH_TOKEN\n",
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 4 {
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

			acts, err := sdk.AddDomainRoleActions(args[2], args[1], actions.Actions, args[3])
			if err != nil {
				logErrorCmd(*cmd, err)
				return
			}
			logJSONCmd(*cmd, acts)
		},
	},

	{
		Use:   "list <role_id> <domain_id> <user_auth_token>",
		Short: "List actions of role",
		Long: "List actions of role\n" +
			"Usage:\n" +
			"\tsupermq-cli domains roles actions list <role_id> <domain_id> <user_auth_token>\n" +
			"For example:\n" +
			"\tsupermq-cli domains roles actions list 39f97daf-d6b6-40f4-b229-2697be8006ef 4ef09eff-d500-4d56-b04f-d23a512d6f2a $USER_AUTH_TOKEN\n",
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 3 {
				logUsageCmd(*cmd, cmd.Use)
				return
			}

			l, err := sdk.DomainRoleActions(args[1], args[0], args[2])
			if err != nil {
				logErrorCmd(*cmd, err)
				return
			}
			logJSONCmd(*cmd, l)
		},
	},

	{
		Use:   "delete [all | <JSON_actions>] <role_id> <domain_id> <user_auth_token>",
		Short: "Delete actions from role",
		Long: "Delete actions from role\n" +
			"Usage:\n" +
			"\tsupermq-cli domains roles actions delete <JSON_actions> <role_id> <domain_id> <user_auth_token>\n" +
			"\tsupermq-cli domains roles actions delete all <role_id> <domain_id> <user_auth_token>\n" +
			"For example:\n" +
			"\tsupermq-cli domains roles actions delete '{\"actions\":[\"read\",\"write\"]}' 39f97daf-d6b6-40f4-b229-2697be8006ef 4ef09eff-d500-4d56-b04f-d23a512d6f2a $USER_AUTH_TOKEN\n",
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 4 {
				logUsageCmd(*cmd, cmd.Use)
				return
			}
			if args[0] == all {
				if err := sdk.RemoveAllDomainRoleActions(args[2], args[1], args[3]); err != nil {
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
			if err := sdk.RemoveDomainRoleActions(args[2], args[1], actions.Actions, args[3]); err != nil {
				logErrorCmd(*cmd, err)
				return
			}
			logOKCmd(*cmd)
		},
	},

	{
		Use:   "available-actions <user_auth_token>",
		Short: "List available actions",
		Long: "List available actions\n" +
			"Usage:\n" +
			"\tsupermq-cli domains roles actions available-actions <user_auth_token>\n" +
			"For example:\n" +
			"\tsupermq-cli domains roles actions available-actions $USER_AUTH_TOKEN\n",
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 1 {
				logUsageCmd(*cmd, cmd.Use)
				return
			}
			acts, err := sdk.AvailableDomainRoleActions(args[0])
			if err != nil {
				logErrorCmd(*cmd, err)
				return
			}
			logJSONCmd(*cmd, acts)
		},
	},
}

var cmdDomainsMembers = []cobra.Command{
	{
		Use:   "add <JSON_members> <role_id> <domain_id> <user_auth_token>",
		Short: "Add members to role",
		Long: "Add members to role\n" +
			"Usage:\n" +
			"\tsupermq-cli domains roles members add <JSON_members> <role_id> <domain_id> <user_auth_token>\n" +
			"For example:\n" +
			"\tsupermq-cli domains roles members add '{\"members\":[\"5dc1ce4b-7cc9-4f12-98a6-9d74cc4980bb\", \"5dc1ce4b-7cc9-4f12-98a6-9d74cc4980bb\"]}' 39f97daf-d6b6-40f4-b229-2697be8006ef 4ef09eff-d500-4d56-b04f-d23a512d6f2a $USER_AUTH_TOKEN\n",
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 4 {
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

			memb, err := sdk.AddDomainRoleMembers(args[2], args[1], members.Members, args[3])
			if err != nil {
				logErrorCmd(*cmd, err)
				return
			}
			logJSONCmd(*cmd, memb)
		},
	},

	{
		Use:   "list <role_id> <domain_id> <user_auth_token>",
		Short: "List members of role",
		Long: "List members of role\n" +
			"Usage:\n" +
			"\tsupermq-cli domains roles members list <role_id> <domain_id> <user_auth_token>\n" +
			"For example:\n" +
			"\tsupermq-cli domains roles members list 39f97daf-d6b6-40f4-b229-2697be8006ef 4ef09eff-d500-4d56-b04f-d23a512d6f2a $USER_AUTH_TOKEN\n",
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 3 {
				logUsageCmd(*cmd, cmd.Use)
				return
			}
			pageMetadata := smqsdk.PageMetadata{
				Offset: Offset,
				Limit:  Limit,
			}

			l, err := sdk.DomainRoleMembers(args[1], args[0], pageMetadata, args[2])
			if err != nil {
				logErrorCmd(*cmd, err)
				return
			}
			logJSONCmd(*cmd, l)
		},
	},

	{
		Use:   "delete [all | <JSON_members>] <role_id> <domain_id> <user_auth_token>",
		Short: "Delete members from role",
		Long: "Delete members from role\n" +
			"Usage:\n" +
			"\tsupermq-cli domains roles members delete <JSON_members> <role_id> <domain_id> <user_auth_token>\n" +
			"\tsupermq-cli domains roles members delete all <role_id> <domain_id> <user_auth_token>\n" +
			"For example:\n" +
			"\tsupermq-cli domains roles members delete all 39f97daf-d6b6-40f4-b229-2697be8006ef 4ef09eff-d500-4d56-b04f-d23a512d6f2a $USER_AUTH_TOKEN\n" +
			"\tsupermq-cli domains roles members delete '{\"members\":[\"5dc1ce4b-7cc9-4f12-98a6-9d74cc4980bb\", \"5dc1ce4b-7cc9-4f12-98a6-9d74cc4980bb\"]}' 39f97daf-d6b6-40f4-b229-2697be8006ef 4ef09eff-d500-4d56-b04f-d23a512d6f2a $USER_AUTH_TOKEN\n",
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 4 {
				logUsageCmd(*cmd, cmd.Use)
				return
			}
			if args[0] == all {
				if err := sdk.RemoveAllDomainRoleMembers(args[2], args[1], args[3]); err != nil {
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

			if err := sdk.RemoveDomainRoleMembers(args[2], args[1], members.Members, args[3]); err != nil {
				logErrorCmd(*cmd, err)
				return
			}
			logOKCmd(*cmd)
		},
	},
}

// NewDomainsCmd returns domains command.
func NewDomainsCmd() *cobra.Command {
	actionsCmd := cobra.Command{
		Use:   "actions [add | list | delete | available-actions]",
		Short: "Actions management",
		Long:  "Actions management: add, list, delete actions and list available actions",
	}
	for i := range cmdDomainsActions {
		actionsCmd.AddCommand(&cmdDomainsActions[i])
	}

	membersCmd := cobra.Command{
		Use:   "members [add | list | delete]",
		Short: "Members management",
		Long:  "Members management: add, list, delete members",
	}
	for i := range cmdDomainsMembers {
		membersCmd.AddCommand(&cmdDomainsMembers[i])
	}

	rolesCmd := cobra.Command{
		Use:   "roles [create | get | update | delete | actions | members]",
		Short: "Roles management",
		Long:  "Roles management: create, update, retrieve roles and assign/unassign members to roles",
	}

	rolesCmd.AddCommand(&actionsCmd)
	rolesCmd.AddCommand(&membersCmd)

	for i := range cmdDomainRoles {
		rolesCmd.AddCommand(&cmdDomainRoles[i])
	}
	cmd := cobra.Command{
		Use:   "domains [create | get | update | enable | disable | enable | users | assign | unassign]",
		Short: "Domains management",
		Long:  `Domains management: create, update, retrieve domains , assign/unassign users to domains and list users of domain"`,
	}
	cmd.AddCommand(&rolesCmd)

	for i := range cmdDomains {
		cmd.AddCommand(&cmdDomains[i])
	}

	return &cmd
}
