// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"encoding/json"

	mgxsdk "github.com/absmach/magistrala/pkg/sdk/go"
	"github.com/spf13/cobra"
)

var cmdDomains = []cobra.Command{
	{
		Use:   "create <name> <alias> <token>",
		Short: "Create Domain",
		Long: "Create Domain with provided name and alias. \n" +
			"For example:\n" +
			"\tmagistrala-cli domains create domain_1 domain_1_alias $TOKEN\n",
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 3 {
				logUsageCmd(*cmd, cmd.Use)
				return
			}

			dom := mgxsdk.Domain{
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
			pageMetadata := mgxsdk.PageMetadata{
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
			pageMetadata := mgxsdk.PageMetadata{
				Offset:   Offset,
				Limit:    Limit,
				Metadata: metadata,
				Status:   Status,
				Domain:   args[0],
			}

			l, err := sdk.ListDomainUsers(pageMetadata, args[1])
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
			"\tmagistrala-cli domains update <domain_id> '{\"name\":\"new name\", \"alias\":\"new_alias\", \"metadata\":{\"key\": \"value\"}}' $TOKEN \n",
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 4 && len(args) != 3 {
				logUsageCmd(*cmd, cmd.Use)
				return
			}

			var d mgxsdk.Domain

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
			"\tmagistrala-cli domains enable <domain_id> <token>\n",
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
			"\tmagistrala-cli domains disable <domain_id> <token>\n",
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
}

var domainAssignCmds = []cobra.Command{
	{
		Use:   "users <relation> <user_ids> <domain_id> <token>",
		Short: "Assign users",
		Long: "Assign users to a domain\n" +
			"Usage:\n" +
			"\tmagistrala-cli domains assign users <relation> '[\"<user_id_1>\", \"<user_id_2>\"]' <domain_id> $TOKEN\n",
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
			if err := sdk.AddUserToDomain(args[2], mgxsdk.UsersRelationRequest{Relation: args[0], UserIDs: userIDs}, args[3]); err != nil {
				logErrorCmd(*cmd, err)
				return
			}
			logOKCmd(*cmd)
		},
	},
}

var domainUnassignCmds = []cobra.Command{
	{
		Use:   "users <user_id> <domain_id> <token>",
		Short: "Unassign users",
		Long: "Unassign users from a domain\n" +
			"Usage:\n" +
			"\tmagistrala-cli domains unassign users <user_id> <domain_id> $TOKEN\n",
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 3 {
				logUsageCmd(*cmd, cmd.Use)
				return
			}

			if err := sdk.RemoveUserFromDomain(args[1], args[0], args[2]); err != nil {
				logErrorCmd(*cmd, err)
				return
			}
			logOKCmd(*cmd)
		},
	},
}

func NewDomainAssignCmds() *cobra.Command {
	cmd := cobra.Command{
		Use:   "assign [users]",
		Short: "Assign users to a domain",
		Long:  "Assign users to a domain",
	}
	for i := range domainAssignCmds {
		cmd.AddCommand(&domainAssignCmds[i])
	}
	return &cmd
}

func NewDomainUnassignCmds() *cobra.Command {
	cmd := cobra.Command{
		Use:   "unassign [users]",
		Short: "Unassign users from a domain",
		Long:  "Unassign users from a domain",
	}
	for i := range domainUnassignCmds {
		cmd.AddCommand(&domainUnassignCmds[i])
	}
	return &cmd
}

// NewDomainsCmd returns domains command.
func NewDomainsCmd() *cobra.Command {
	cmd := cobra.Command{
		Use:   "domains [create | get | update | enable | disable | enable | users | assign | unassign]",
		Short: "Domains management",
		Long:  `Domains management: create, update, retrieve domains , assign/unassign users to domains and list users of domain"`,
	}

	for i := range cmdDomains {
		cmd.AddCommand(&cmdDomains[i])
	}

	cmd.AddCommand(NewDomainAssignCmds())
	cmd.AddCommand(NewDomainUnassignCmds())
	return &cmd
}
