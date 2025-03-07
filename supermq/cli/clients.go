// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"encoding/json"

	"github.com/absmach/supermq/clients"
	smqsdk "github.com/absmach/supermq/pkg/sdk"
	"github.com/spf13/cobra"
)

var cmdClients = []cobra.Command{
	{
		Use:   "create <JSON_client> <domain_id> <user_auth_token>",
		Short: "Create client",
		Long: "Creates new client with provided name and metadata\n" +
			"Usage:\n" +
			"\tsupermq-cli clients create '{\"name\":\"new client\", \"metadata\":{\"key\": \"value\"}}' $DOMAINID $USERTOKEN\n",
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 3 {
				logUsageCmd(*cmd, cmd.Use)
				return
			}

			var client smqsdk.Client
			if err := json.Unmarshal([]byte(args[0]), &client); err != nil {
				logErrorCmd(*cmd, err)
				return
			}
			client.Status = clients.EnabledStatus.String()
			client, err := sdk.CreateClient(client, args[1], args[2])
			if err != nil {
				logErrorCmd(*cmd, err)
				return
			}

			logJSONCmd(*cmd, client)
		},
	},
	{
		Use:   "get [all | <client_id>] <domain_id> <user_auth_token>",
		Short: "Get clients",
		Long: "Get all clients or get client by id. Clients can be filtered by name or metadata\n" +
			"Usage:\n" +
			"\tsupermq-cli clients get all $DOMAINID $USERTOKEN - lists all clients\n" +
			"\tsupermq-cli clients get all $DOMAINID $USERTOKEN --offset=10 --limit=10 - lists all clients with offset and limit\n" +
			"\tsupermq-cli clients get <client_id> $DOMAINID $USERTOKEN - shows client with provided <client_id>\n",
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 3 {
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
			}
			if args[0] == all {
				l, err := sdk.Clients(pageMetadata, args[1], args[2])
				if err != nil {
					logErrorCmd(*cmd, err)
					return
				}
				logJSONCmd(*cmd, l)
				return
			}
			t, err := sdk.Client(args[0], args[1], args[2])
			if err != nil {
				logErrorCmd(*cmd, err)
				return
			}

			logJSONCmd(*cmd, t)
		},
	},
	{
		Use:   "delete <client_id> <domain_id> <user_auth_token>",
		Short: "Delete client",
		Long: "Delete client by id\n" +
			"Usage:\n" +
			"\tsupermq-cli clients delete <client_id> $DOMAINID $USERTOKEN - delete client with <client_id>\n",
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 3 {
				logUsageCmd(*cmd, cmd.Use)
				return
			}
			if err := sdk.DeleteClient(args[0], args[1], args[2]); err != nil {
				logErrorCmd(*cmd, err)
				return
			}
			logOKCmd(*cmd)
		},
	},
	{
		Use:   "update [<client_id> <JSON_string> | tags <client_id> <tags> | secret <client_id> <secret> ] <domain_id> <user_auth_token>",
		Short: "Update client",
		Long: "Updates client with provided id, name and metadata, or updates client's tags, secret\n" +
			"Usage:\n" +
			"\tsupermq-cli client update <client_id> '{\"name\":\"new name\", \"metadata\":{\"key\": \"value\"}}' $DOMAINID $USERTOKEN\n" +
			"\tsupermq-cli client update tags <client_id> '{\"tag1\":\"value1\", \"tag2\":\"value2\"}' $DOMAINID $USERTOKEN\n" +
			"\tsupermq-cli client update secret <client_id> <newsecret> $DOMAINID $USERTOKEN\n",
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 5 && len(args) != 4 {
				logUsageCmd(*cmd, cmd.Use)
				return
			}

			var client smqsdk.Client
			if args[0] == "tags" {
				if err := json.Unmarshal([]byte(args[2]), &client.Tags); err != nil {
					logErrorCmd(*cmd, err)
					return
				}
				client.ID = args[1]
				client, err := sdk.UpdateClientTags(client, args[3], args[4])
				if err != nil {
					logErrorCmd(*cmd, err)
					return
				}

				logJSONCmd(*cmd, client)
				return
			}

			if args[0] == "secret" {
				client, err := sdk.UpdateClientSecret(args[1], args[2], args[3], args[4])
				if err != nil {
					logErrorCmd(*cmd, err)
					return
				}

				logJSONCmd(*cmd, client)
				return
			}

			if err := json.Unmarshal([]byte(args[1]), &client); err != nil {
				logErrorCmd(*cmd, err)
				return
			}
			client.ID = args[0]
			client, err := sdk.UpdateClient(client, args[2], args[3])
			if err != nil {
				logErrorCmd(*cmd, err)
				return
			}

			logJSONCmd(*cmd, client)
		},
	},
	{
		Use:   "enable <client_id> <domain_id> <user_auth_token>",
		Short: "Change client status to enabled",
		Long: "Change client status to enabled\n" +
			"Usage:\n" +
			"\tsupermq-cli clients enable <client_id> $DOMAINID $USERTOKEN\n",
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 3 {
				logUsageCmd(*cmd, cmd.Use)
				return
			}

			client, err := sdk.EnableClient(args[0], args[1], args[2])
			if err != nil {
				logErrorCmd(*cmd, err)
				return
			}

			logJSONCmd(*cmd, client)
		},
	},
	{
		Use:   "disable <client_id> <domain_id> <user_auth_token>",
		Short: "Change client status to disabled",
		Long: "Change client status to disabled\n" +
			"Usage:\n" +
			"\tsupermq-cli clients disable <client_id> $DOMAINID $USERTOKEN\n",
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 3 {
				logUsageCmd(*cmd, cmd.Use)
				return
			}

			client, err := sdk.DisableClient(args[0], args[1], args[2])
			if err != nil {
				logErrorCmd(*cmd, err)
				return
			}

			logJSONCmd(*cmd, client)
		},
	},
	{
		Use:   "connect <client_id> <channel_id> <conn_types_json_list> <domain_id> <user_auth_token>",
		Short: "Connect client",
		Long: "Connect client to the channel\n" +
			"Usage:\n" +
			"\tsupermq-cli clients connect <client_id> <channel_id> <conn_types_json_list> $DOMAINID $USERTOKEN\n",
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 5 {
				logUsageCmd(*cmd, cmd.Use)
				return
			}
			var conn_types []string
			err := json.Unmarshal([]byte(args[2]), &conn_types)
			if err != nil {
				logErrorCmd(*cmd, err)
				return
			}

			connIDs := smqsdk.Connection{
				ChannelIDs: []string{args[1]},
				ClientIDs:  []string{args[0]},
				Types:      conn_types,
			}
			if err := sdk.Connect(connIDs, args[3], args[4]); err != nil {
				logErrorCmd(*cmd, err)
				return
			}

			logOKCmd(*cmd)
		},
	},
	{
		Use:   "disconnect <client_id> <channel_id> <conn_types_json_list> <domain_id> <user_auth_token>",
		Short: "Disconnect client",
		Long: "Disconnect client to the channel\n" +
			"Usage:\n" +
			"\tsupermq-cli clients disconnect <client_id> <channel_id> <conn_types_json_list> $DOMAINID $USERTOKEN\n",
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 5 {
				logUsageCmd(*cmd, cmd.Use)
				return
			}

			var conn_types []string
			err := json.Unmarshal([]byte(args[2]), &conn_types)
			if err != nil {
				logErrorCmd(*cmd, err)
				return
			}

			connIDs := smqsdk.Connection{
				ClientIDs:  []string{args[0]},
				ChannelIDs: []string{args[1]},
				Types:      conn_types,
			}
			if err := sdk.Disconnect(connIDs, args[3], args[4]); err != nil {
				logErrorCmd(*cmd, err)
				return
			}

			logOKCmd(*cmd)
		},
	},
	{
		Use:   "users <client_id> <domain_id> <user_auth_token>",
		Short: "List users",
		Long: "List users of a client\n" +
			"Usage:\n" +
			"\tsupermq-cli clients users <client_id> $DOMAINID $USERTOKEN\n",
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 3 {
				logUsageCmd(*cmd, cmd.Use)
				return
			}
			pm := smqsdk.PageMetadata{
				Offset: Offset,
				Limit:  Limit,
			}
			ul, err := sdk.ListClientMembers(args[0], args[1], pm, args[2])
			if err != nil {
				logErrorCmd(*cmd, err)
				return
			}

			logJSONCmd(*cmd, ul)
		},
	},
}

var cmdClientsRoles = []cobra.Command{
	{
		Use:   "create <JSON_role> <client_id> <domain_id> <user_auth_token>",
		Short: "Create client role",
		Long: "Create role\n" +
			"Usage:\n" +
			"\tsupermq-cli clients roles create <JSON_role> <client_id> <domain_id> <user_auth_token>\n" +
			"For example:\n" +
			"\tsupermq-cli clients roles create '{\"role_name\":\"admin\",\"optional_actions\":[\"read\",\"update\"]}' 4ef09eff-d500-4d56-b04f-d23a512d6f2a 39f97daf-d6b6-40f4-b229-2697be8006ef $USER_AUTH_TOKEN\n",
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
			r, err := sdk.CreateClientRole(args[1], args[2], roleReq, args[3])
			if err != nil {
				logErrorCmd(*cmd, err)
				return
			}

			logJSONCmd(*cmd, r)
		},
	},

	{
		Use:   "get [all | <role_id>] <client_id>, <domain_id> <user_auth_token>",
		Short: "Get client roles",
		Long: "Get client roles\n" +
			"Usage:\n" +
			"\tsupermq-cli clients roles get all <client_id> <domain_id> <user_auth_token> - lists all roles\n" +
			"\tsupermq-cli clients roles get all <client_id> <domain_id> <user_auth_token> --offset <offset> --limit <limit> - lists all roles with provided offset and limit\n" +
			"\tsupermq-cli clients roles get <role_id> <client_id> <domain_id> <user_auth_token> - shows role by role id and domain id\n",
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
				rs, err := sdk.ClientRoles(args[1], args[2], pageMetadata, args[3])
				if err != nil {
					logErrorCmd(*cmd, err)
					return
				}
				logJSONCmd(*cmd, rs)
				return
			}
			r, err := sdk.ClientRole(args[1], args[0], args[2], args[3])
			if err != nil {
				logErrorCmd(*cmd, err)
				return
			}
			logJSONCmd(*cmd, r)
		},
	},

	{
		Use:   "update <new_name> <role_id> <client_id> <domain_id> <user_auth_token>",
		Short: "Update client role name",
		Long: "Update client role name\n" +
			"Usage:\n" +
			"\tsupermq-cli clients roles update <new_name> <role_id> <client_id> <domain_id> <user_auth_token>\n" +
			"For example:\n" +
			"\tsupermq-cli clients roles update new_name 39f97daf-d6b6-40f4-b229-2697be8006ef 4ef09eff-d500-4d56-b04f-d23a512d6f2a 4ef09eff-d500-4d56-b04f-d23a512d6f2a $USER_AUTH_TOKEN\n",
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 5 {
				logUsageCmd(*cmd, cmd.Use)
				return
			}

			r, err := sdk.UpdateClientRole(args[2], args[1], args[0], args[3], args[4])
			if err != nil {
				logErrorCmd(*cmd, err)
				return
			}
			logJSONCmd(*cmd, r)
		},
	},

	{
		Use:   "delete <role_id> <client_id> <domain_id> <user_auth_token>",
		Short: "Delete client role",
		Long: "Delete client role\n" +
			"Usage:\n" +
			"\tsupermq-cli clients roles delete <role_id> <client_id> <domain_id> <user_auth_token>\n" +
			"For example:\n" +
			"\tsupermq-cli clients roles delete 39f97daf-d6b6-40f4-b229-2697be8006ef 4ef09eff-d500-4d56-b04f-d23a512d6f2a 4ef09eff-d500-4d56-b04f-d23a512d6f2a $USER_AUTH_TOKEN\n",
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 4 {
				logUsageCmd(*cmd, cmd.Use)
				return
			}

			if err := sdk.DeleteClientRole(args[1], args[0], args[2], args[3]); err != nil {
				logErrorCmd(*cmd, err)
				return
			}
			logOKCmd(*cmd)
		},
	},
}

var cmdClientsActions = []cobra.Command{
	{
		Use:   "add <JSON_actions> <role_id> <client_id> <domain_id> <user_auth_token>",
		Short: "Add actions to role",
		Long: "Add actions to role\n" +
			"Usage:\n" +
			"\tsupermq-cli clients roles actions add <JSON_actions> <role_id> <client_id> <domain_id> <user_auth_token>\n" +
			"For example:\n" +
			"\tsupermq-cli clients roles actions add '{\"actions\":[\"read\",\"write\"]}' 39f97daf-d6b6-40f4-b229-2697be8006ef 4ef09eff-d500-4d56-b04f-d23a512d6f2a 4ef09eff-d500-4d56-b04f-d23a512d6f2a $USER_AUTH_TOKEN\n",
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

			acts, err := sdk.AddClientRoleActions(args[2], args[1], args[3], actions.Actions, args[4])
			if err != nil {
				logErrorCmd(*cmd, err)
				return
			}
			logJSONCmd(*cmd, acts)
		},
	},

	{
		Use:   "list <role_id> <client_id> <domain_id> <user_auth_token>",
		Short: "List actions of role",
		Long: "List actions of role\n" +
			"Usage:\n" +
			"\tsupermq-cli clients roles actions list <role_id> <client_id> <domain_id> <user_auth_token>\n" +
			"For example:\n" +
			"\tsupermq-cli clients roles actions list 39f97daf-d6b6-40f4-b229-2697be8006ef 4ef09eff-d500-4d56-b04f-d23a512d6f2a 4ef09eff-d500-4d56-b04f-d23a512d6f2a $USER_AUTH_TOKEN\n",
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 4 {
				logUsageCmd(*cmd, cmd.Use)
				return
			}

			l, err := sdk.ClientRoleActions(args[1], args[0], args[2], args[3])
			if err != nil {
				logErrorCmd(*cmd, err)
				return
			}
			logJSONCmd(*cmd, l)
		},
	},

	{
		Use:   "delete [all | <JSON_actions>] <role_id> <client_id> <domain_id> <user_auth_token>",
		Short: "Delete actions from role",
		Long: "Delete actions from role\n" +
			"Usage:\n" +
			"\tsupermq-cli clients roles actions delete <JSON_actions> <role_id> <client_id> <domain_id> <user_auth_token>\n" +
			"\tsupermq-cli clients roles actions delete all <role_id> <client_id> <domain_id> <user_auth_token>\n" +
			"For example:\n" +
			"\tsupermq-cli clients roles actions delete '{\"actions\":[\"read\",\"write\"]}' 39f97daf-d6b6-40f4-b229-2697be8006ef 4ef09eff-d500-4d56-b04f-d23a512d6f2a 4ef09eff-d500-4d56-b04f-d23a512d6f2a $USER_AUTH_TOKEN\n",
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 5 {
				logUsageCmd(*cmd, cmd.Use)
				return
			}
			if args[0] == all {
				if err := sdk.RemoveAllClientRoleActions(args[2], args[1], args[3], args[4]); err != nil {
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
			if err := sdk.RemoveClientRoleActions(args[2], args[1], args[3], actions.Actions, args[4]); err != nil {
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
			"\tsupermq-cli clients roles actions available-actions <domain_id> <user_auth_token>\n" +
			"For example:\n" +
			"\tsupermq-cli clients roles actions available-actions 39f97daf-d6b6-40f4-b229-2697be8006ef $USER_AUTH_TOKEN\n",
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 2 {
				logUsageCmd(*cmd, cmd.Use)
				return
			}
			acts, err := sdk.AvailableClientRoleActions(args[0], args[1])
			if err != nil {
				logErrorCmd(*cmd, err)
				return
			}
			logJSONCmd(*cmd, acts)
		},
	},
}

var cmdClientMembers = []cobra.Command{
	{
		Use:   "add <JSON_members> <role_id> <client_id> <domain_id> <user_auth_token>",
		Short: "Add members to role",
		Long: "Add members to role\n" +
			"Usage:\n" +
			"\tsupermq-cli clients roles members add <JSON_members> <role_id> <client_id> <domain_id> <user_auth_token>\n" +
			"For example:\n" +
			"\tsupermq-cli clients roles members add '{\"members\":[\"5dc1ce4b-7cc9-4f12-98a6-9d74cc4980bb\", \"5dc1ce4b-7cc9-4f12-98a6-9d74cc4980bb\"]}' 39f97daf-d6b6-40f4-b229-2697be8006ef 4ef09eff-d500-4d56-b04f-d23a512d6f2a 4ef09eff-d500-4d56-b04f-d23a512d6f2a $USER_AUTH_TOKEN\n",
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

			memb, err := sdk.AddClientRoleMembers(args[2], args[1], args[3], members.Members, args[4])
			if err != nil {
				logErrorCmd(*cmd, err)
				return
			}
			logJSONCmd(*cmd, memb)
		},
	},

	{
		Use:   "list <role_id> <client_id> <domain_id> <user_auth_token>",
		Short: "List members of role",
		Long: "List members of role\n" +
			"Usage:\n" +
			"\tsupermq-cli clients roles members list <role_id> <domain_id> <user_auth_token>\n" +
			"For example:\n" +
			"\tsupermq-cli clients roles members list 39f97daf-d6b6-40f4-b229-2697be8006ef 4ef09eff-d500-4d56-b04f-d23a512d6f2a 4ef09eff-d500-4d56-b04f-d23a512d6f2a $USER_AUTH_TOKEN\n",
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 4 {
				logUsageCmd(*cmd, cmd.Use)
				return
			}
			pageMetadata := smqsdk.PageMetadata{
				Offset: Offset,
				Limit:  Limit,
			}

			l, err := sdk.ClientRoleMembers(args[1], args[0], args[2], pageMetadata, args[3])
			if err != nil {
				logErrorCmd(*cmd, err)
				return
			}
			logJSONCmd(*cmd, l)
		},
	},

	{
		Use:   "delete [all | <JSON_members>] <role_id> <client_id> <domain_id> <user_auth_token>",
		Short: "Delete members from role",
		Long: "Delete members from role\n" +
			"Usage:\n" +
			"\tsupermq-cli clients roles members delete <JSON_members> <role_id> <client_id> <domain_id> <user_auth_token>\n" +
			"\tsupermq-cli clients roles members delete all <role_id> <client_id> <domain_id> <user_auth_token>\n" +
			"For example:\n" +
			"\tsupermq-cli clients roles members delete all 39f97daf-d6b6-40f4-b229-2697be8006ef 4ef09eff-d500-4d56-b04f-d23a512d6f2a 4ef09eff-d500-4d56-b04f-d23a512d6f2a $USER_AUTH_TOKEN\n" +
			"\tsupermq-cli clients roles members delete '{\"members\":[\"5dc1ce4b-7cc9-4f12-98a6-9d74cc4980bb\", \"5dc1ce4b-7cc9-4f12-98a6-9d74cc4980bb\"]}' 39f97daf-d6b6-40f4-b229-2697be8006ef 4ef09eff-d500-4d56-b04f-d23a512d6f2a 4ef09eff-d500-4d56-b04f-d23a512d6f2a $USER_AUTH_TOKEN\n",
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 5 {
				logUsageCmd(*cmd, cmd.Use)
				return
			}
			if args[0] == all {
				if err := sdk.RemoveAllClientRoleMembers(args[2], args[1], args[3], args[4]); err != nil {
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

			if err := sdk.RemoveClientRoleMembers(args[2], args[1], args[3], members.Members, args[4]); err != nil {
				logErrorCmd(*cmd, err)
				return
			}
			logOKCmd(*cmd)
		},
	},
}

// NewClientsCmd returns clients command.
func NewClientsCmd() *cobra.Command {
	actionsCmd := cobra.Command{
		Use:   "actions [add | list | delete | available-actions]",
		Short: "Actions management",
		Long:  "Actions management: add, list, delete actions and list available actions",
	}
	for i := range cmdClientsActions {
		actionsCmd.AddCommand(&cmdClientsActions[i])
	}

	membersCmd := cobra.Command{
		Use:   "members [add | list | delete]",
		Short: "Members management",
		Long:  "Members management: add, list, delete members",
	}
	for i := range cmdClientMembers {
		membersCmd.AddCommand(&cmdClientMembers[i])
	}

	rolesCmd := cobra.Command{
		Use:   "roles [create | get | update | delete | actions | members]",
		Short: "Roles management",
		Long:  "Roles management: create, update, retrieve roles and assign/unassign members to roles",
	}

	rolesCmd.AddCommand(&actionsCmd)
	rolesCmd.AddCommand(&membersCmd)

	for i := range cmdClientsRoles {
		rolesCmd.AddCommand(&cmdClientsRoles[i])
	}

	cmd := cobra.Command{
		Use:   "clients [create | get | update | delete | share | connect | disconnect | connections | not-connected | users ]",
		Short: "Clients management",
		Long:  `Clients management: create, get, update, delete or share Client, connect or disconnect Client from Channel and get the list of Channels connected or disconnected from a Client`,
	}
	cmd.AddCommand(&rolesCmd)

	for i := range cmdClients {
		cmd.AddCommand(&cmdClients[i])
	}

	return &cmd
}
