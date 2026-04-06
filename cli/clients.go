// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"encoding/json"
	"fmt"

	"github.com/absmach/magistrala/clients"
	smqsdk "github.com/absmach/magistrala/pkg/sdk"
	"github.com/spf13/cobra"
)

const (
	connect    = "connect"
	disconnect = "disconnect"
	roles      = "roles"
	actions    = "actions"
	members    = "members"
	secret     = "secret"

	// Usage strings for client operations.
	usageClientCreate       = "cli clients create <JSON_client> <domain_id> <user_auth_token>"
	usageClientGet          = "cli clients <client_id|all> get <domain_id> <user_auth_token>"
	usageClientDelete       = "cli clients <client_id> delete <domain_id> <user_auth_token>"
	usageClientUpdate       = "cli clients <client_id> update <JSON_string> <domain_id> <user_auth_token>"
	usageClientUpdateTags   = "cli clients <client_id> update tags <tags> <domain_id> <user_auth_token>"
	usageClientUpdateSecret = "cli clients <client_id> update secret <secret> <domain_id> <user_auth_token>"
	usageClientEnable       = "cli clients <client_id> enable <domain_id> <user_auth_token>"
	usageClientDisable      = "cli clients <client_id> disable <domain_id> <user_auth_token>"
	usageClientConnect      = "cli clients <client_id> connect <channel_id> <conn_types_json_list> <domain_id> <user_auth_token>"
	usageClientDisconnect   = "cli clients <client_id> disconnect <channel_id> <conn_types_json_list> <domain_id> <user_auth_token>"
	usageClientUsers        = "cli clients <client_id> users <domain_id> <user_auth_token>"

	// Usage strings for client roles operations.
	usageClientRolesCreate = "cli clients <client_id> roles create <JSON_role> <domain_id> <user_auth_token>"
	usageClientRolesGet    = "cli clients <client_id> roles get <role_id|all> <domain_id> <user_auth_token>"
	usageClientRolesUpdate = "cli clients <client_id> roles update <role_id> <new_name> <domain_id> <user_auth_token>"
	usageClientRolesDelete = "cli clients <client_id> roles delete <role_id> <domain_id> <user_auth_token>"

	// Usage strings for client role actions operations.
	usageClientRoleActionsAdd       = "cli clients <client_id> roles actions add <role_id> <JSON_actions> <domain_id> <user_auth_token>"
	usageClientRoleActionsList      = "cli clients <client_id> roles actions list <role_id> <domain_id> <user_auth_token>"
	usageClientRoleActionsDelete    = "cli clients <client_id> roles actions delete <role_id> <JSON_actions|all> <domain_id> <user_auth_token>"
	usageClientRoleActionsAvailable = "cli clients roles actions available-actions <domain_id> <user_auth_token>"

	// Usage strings for client role members operations.
	usageClientRoleMembersAdd    = "cli clients <client_id> roles members add <role_id> <JSON_members> <domain_id> <user_auth_token>"
	usageClientRoleMembersList   = "cli clients <client_id> roles members list <role_id> <domain_id> <user_auth_token>"
	usageClientRoleMembersDelete = "cli clients <client_id> roles members delete <role_id> <JSON_members|all> <domain_id> <user_auth_token>"
)

func NewClientsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "clients <client_id|all|create> [operation] [args...]",
		Short: "Clients management",
		Long: `Format: 
  clients create [args...]
  clients <client_id|all> <operation> [args...]

Operations (require client_id/all): get, update, delete, enable, disable, connect, disconnect, users, roles

Examples:
  clients create <JSON_client> <domain_id> <user_auth_token>
  clients all get <domain_id> <user_auth_token>
  clients <client_id> get <domain_id> <user_auth_token>
  clients <client_id> update <JSON_string> <domain_id> <user_auth_token>
  clients <client_id> delete <domain_id> <user_auth_token>
  clients <client_id> enable <domain_id> <user_auth_token>
  clients <client_id> disable <domain_id> <user_auth_token>
  clients <client_id> connect <channel_id> <conn_types_json_list> <domain_id> <user_auth_token>
  clients <client_id> users <domain_id> <user_auth_token>`,

		Run: func(cmd *cobra.Command, args []string) {
			if len(args) == 0 {
				logUsageCmd(*cmd, cmd.Use)
				return
			}

			if args[0] == create {
				handleClientCreate(cmd, args[1:])
				return
			}

			if len(args) < 2 {
				logUsageCmd(*cmd, "clients <client_id|all> <get|update|delete|enable|disable|connect|disconnect|users|roles> [args...]")
				return
			}

			clientParams := args[0]
			operation := args[1]
			opArgs := args[2:]

			switch operation {
			case get:
				handleClientGet(cmd, clientParams, opArgs)
			case update:
				handleClientUpdate(cmd, clientParams, opArgs)
			case delete:
				handleClientDelete(cmd, clientParams, opArgs)
			case enable:
				handleClientEnable(cmd, clientParams, opArgs)
			case disable:
				handleClientDisable(cmd, clientParams, opArgs)
			case connect:
				handleClientConnect(cmd, clientParams, opArgs)
			case disconnect:
				handleClientDisconnect(cmd, clientParams, opArgs)
			case users:
				handleClientUsers(cmd, clientParams, opArgs)
			case roles:
				handleClientRoles(cmd, clientParams, opArgs)
			default:
				logErrorCmd(*cmd, fmt.Errorf("unknown operation: %s", operation))
			}
		},
	}

	return cmd
}

func handleClientCreate(cmd *cobra.Command, args []string) {
	if len(args) != 3 {
		logUsageCmd(*cmd, usageClientCreate)
		return
	}

	var client smqsdk.Client
	if err := json.Unmarshal([]byte(args[0]), &client); err != nil {
		logErrorCmd(*cmd, err)
		return
	}

	client.Status = clients.EnabledStatus.String()
	client, err := sdk.CreateClient(cmd.Context(), client, args[1], args[2])
	if err != nil {
		logErrorCmd(*cmd, err)
		return
	}

	logJSONCmd(*cmd, client)
}

func handleClientGet(cmd *cobra.Command, clientParams string, args []string) {
	if len(args) != 2 {
		logUsageCmd(*cmd, usageClientGet)
		return
	}

	if clientParams == all {
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

		l, err := sdk.Clients(cmd.Context(), pageMetadata, args[0], args[1])
		if err != nil {
			logErrorCmd(*cmd, err)
			return
		}
		logJSONCmd(*cmd, l)
		return
	}

	t, err := sdk.Client(cmd.Context(), clientParams, args[0], args[1])
	if err != nil {
		logErrorCmd(*cmd, err)
		return
	}

	logJSONCmd(*cmd, t)
}

func handleClientUpdate(cmd *cobra.Command, clientID string, args []string) {
	if len(args) < 3 || len(args) > 4 {
		if args[0] == tags {
			logUsageCmd(*cmd, usageClientUpdateTags)
			return
		}
		if args[0] == secret {
			logUsageCmd(*cmd, usageClientUpdateSecret)
			return
		}
		logUsageCmd(*cmd, usageClientUpdate)
		return
	}

	if len(args) == 4 && args[0] == "tags" {
		var client smqsdk.Client
		if err := json.Unmarshal([]byte(args[1]), &client.Tags); err != nil {
			logErrorCmd(*cmd, err)
			return
		}
		client.ID = clientID
		client, err := sdk.UpdateClientTags(cmd.Context(), client, args[2], args[3])
		if err != nil {
			logErrorCmd(*cmd, err)
			return
		}
		logJSONCmd(*cmd, client)
		return
	}

	if len(args) == 4 && args[0] == "secret" {
		client, err := sdk.UpdateClientSecret(cmd.Context(), clientID, args[1], args[2], args[3])
		if err != nil {
			logErrorCmd(*cmd, err)
			return
		}
		logJSONCmd(*cmd, client)
		return
	}

	if len(args) != 3 {
		logUsageCmd(*cmd, usageClientUpdate)
		return
	}

	var client smqsdk.Client
	if err := json.Unmarshal([]byte(args[0]), &client); err != nil {
		logErrorCmd(*cmd, err)
		return
	}

	client.ID = clientID
	client, err := sdk.UpdateClient(cmd.Context(), client, args[1], args[2])
	if err != nil {
		logErrorCmd(*cmd, err)
		return
	}

	logJSONCmd(*cmd, client)
}

func handleClientDelete(cmd *cobra.Command, clientID string, args []string) {
	if len(args) != 2 {
		logUsageCmd(*cmd, usageClientDelete)
		return
	}

	if err := sdk.DeleteClient(cmd.Context(), clientID, args[0], args[1]); err != nil {
		logErrorCmd(*cmd, err)
		return
	}
	logOKCmd(*cmd)
}

func handleClientEnable(cmd *cobra.Command, clientID string, args []string) {
	if len(args) != 2 {
		logUsageCmd(*cmd, usageClientEnable)
		return
	}

	client, err := sdk.EnableClient(cmd.Context(), clientID, args[0], args[1])
	if err != nil {
		logErrorCmd(*cmd, err)
		return
	}

	logJSONCmd(*cmd, client)
}

func handleClientDisable(cmd *cobra.Command, clientID string, args []string) {
	if len(args) != 2 {
		logUsageCmd(*cmd, usageClientDisable)
		return
	}

	client, err := sdk.DisableClient(cmd.Context(), clientID, args[0], args[1])
	if err != nil {
		logErrorCmd(*cmd, err)
		return
	}

	logJSONCmd(*cmd, client)
}

func handleClientConnect(cmd *cobra.Command, clientID string, args []string) {
	if len(args) != 4 {
		logUsageCmd(*cmd, usageClientConnect)
		return
	}

	var conn_types []string
	err := json.Unmarshal([]byte(args[1]), &conn_types)
	if err != nil {
		logErrorCmd(*cmd, err)
		return
	}

	connIDs := smqsdk.Connection{
		ChannelIDs: []string{args[0]},
		ClientIDs:  []string{clientID},
		Types:      conn_types,
	}
	if err := sdk.Connect(cmd.Context(), connIDs, args[2], args[3]); err != nil {
		logErrorCmd(*cmd, err)
		return
	}

	logOKCmd(*cmd)
}

func handleClientDisconnect(cmd *cobra.Command, clientID string, args []string) {
	if len(args) != 4 {
		logUsageCmd(*cmd, usageClientDisconnect)
		return
	}

	var conn_types []string
	err := json.Unmarshal([]byte(args[1]), &conn_types)
	if err != nil {
		logErrorCmd(*cmd, err)
		return
	}

	connIDs := smqsdk.Connection{
		ClientIDs:  []string{clientID},
		ChannelIDs: []string{args[0]},
		Types:      conn_types,
	}
	if err := sdk.Disconnect(cmd.Context(), connIDs, args[2], args[3]); err != nil {
		logErrorCmd(*cmd, err)
		return
	}

	logOKCmd(*cmd)
}

func handleClientUsers(cmd *cobra.Command, clientID string, args []string) {
	if len(args) != 2 {
		logUsageCmd(*cmd, usageClientUsers)
		return
	}

	pm := smqsdk.PageMetadata{
		Offset: Offset,
		Limit:  Limit,
	}
	ul, err := sdk.ListClientMembers(cmd.Context(), clientID, args[0], pm, args[1])
	if err != nil {
		logErrorCmd(*cmd, err)
		return
	}

	logJSONCmd(*cmd, ul)
}

func handleClientRoles(cmd *cobra.Command, clientID string, args []string) {
	if len(args) < 1 {
		logUsageCmd(*cmd, "cli clients <client_id> roles <operation> [args...]")
		return
	}

	operation := args[0]
	opArgs := args[1:]

	switch operation {
	case create:
		handleClientRoleCreate(cmd, clientID, opArgs)
	case get:
		handleClientRoleGet(cmd, clientID, opArgs)
	case update:
		handleClientRoleUpdate(cmd, clientID, opArgs)
	case delete:
		handleClientRoleDelete(cmd, clientID, opArgs)
	case actions:
		handleClientRoleActions(cmd, clientID, opArgs)
	case members:
		handleClientRoleMembers(cmd, clientID, opArgs)
	default:
		logErrorCmd(*cmd, fmt.Errorf("unknown roles operation: %s", operation))
	}
}

func handleClientRoleCreate(cmd *cobra.Command, clientID string, args []string) {
	if len(args) != 3 {
		logUsageCmd(*cmd, usageClientRolesCreate)
		return
	}

	var roleReq smqsdk.RoleReq
	if err := json.Unmarshal([]byte(args[0]), &roleReq); err != nil {
		logErrorCmd(*cmd, err)
		return
	}

	r, err := sdk.CreateClientRole(cmd.Context(), clientID, args[1], roleReq, args[2])
	if err != nil {
		logErrorCmd(*cmd, err)
		return
	}

	logJSONCmd(*cmd, r)
}

func handleClientRoleGet(cmd *cobra.Command, clientID string, args []string) {
	if len(args) != 3 {
		logUsageCmd(*cmd, usageClientRolesGet)
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
		rs, err := sdk.ClientRoles(cmd.Context(), clientID, domainID, pageMetadata, token)
		if err != nil {
			logErrorCmd(*cmd, err)
			return
		}
		logJSONCmd(*cmd, rs)
		return
	}

	r, err := sdk.ClientRole(cmd.Context(), clientID, roleID, domainID, token)
	if err != nil {
		logErrorCmd(*cmd, err)
		return
	}
	logJSONCmd(*cmd, r)
}

func handleClientRoleUpdate(cmd *cobra.Command, clientID string, args []string) {
	if len(args) != 4 {
		logUsageCmd(*cmd, usageClientRolesUpdate)
		return
	}

	roleID := args[0]
	newName := args[1]
	domainID := args[2]
	token := args[3]

	r, err := sdk.UpdateClientRole(cmd.Context(), clientID, roleID, newName, domainID, token)
	if err != nil {
		logErrorCmd(*cmd, err)
		return
	}
	logJSONCmd(*cmd, r)
}

func handleClientRoleDelete(cmd *cobra.Command, clientID string, args []string) {
	if len(args) != 3 {
		logUsageCmd(*cmd, usageClientRolesDelete)
		return
	}

	roleID := args[0]
	domainID := args[1]
	token := args[2]

	if err := sdk.DeleteClientRole(cmd.Context(), clientID, roleID, domainID, token); err != nil {
		logErrorCmd(*cmd, err)
		return
	}
	logOKCmd(*cmd)
}

func handleClientRoleActions(cmd *cobra.Command, clientID string, args []string) {
	if len(args) < 1 {
		logUsageCmd(*cmd, "cli clients <client_id> roles actions <operation> [args...]")
		return
	}

	operation := args[0]
	opArgs := args[1:]

	switch operation {
	case add:
		handleClientRoleActionsAdd(cmd, clientID, opArgs)
	case list:
		handleClientRoleActionsList(cmd, clientID, opArgs)
	case delete:
		handleClientRoleActionsDelete(cmd, clientID, opArgs)
	case availableActions:
		handleClientRoleActionsAvailable(cmd, opArgs)
	default:
		logErrorCmd(*cmd, fmt.Errorf("unknown actions operation: %s", operation))
	}
}

func handleClientRoleActionsAdd(cmd *cobra.Command, clientID string, args []string) {
	if len(args) != 4 {
		logUsageCmd(*cmd, usageClientRoleActionsAdd)
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

	acts, err := sdk.AddClientRoleActions(cmd.Context(), clientID, roleID, domainID, actions.Actions, token)
	if err != nil {
		logErrorCmd(*cmd, err)
		return
	}
	logJSONCmd(*cmd, acts)
}

func handleClientRoleActionsList(cmd *cobra.Command, clientID string, args []string) {
	if len(args) != 3 {
		logUsageCmd(*cmd, usageClientRoleActionsList)
		return
	}

	roleID := args[0]
	domainID := args[1]
	token := args[2]

	l, err := sdk.ClientRoleActions(cmd.Context(), clientID, roleID, domainID, token)
	if err != nil {
		logErrorCmd(*cmd, err)
		return
	}
	logJSONCmd(*cmd, l)
}

func handleClientRoleActionsDelete(cmd *cobra.Command, clientID string, args []string) {
	if len(args) != 4 {
		logUsageCmd(*cmd, usageClientRoleActionsDelete)
		return
	}

	roleID := args[0]
	actionsJSON := args[1]
	domainID := args[2]
	token := args[3]

	if actionsJSON == all {
		if err := sdk.RemoveAllClientRoleActions(cmd.Context(), clientID, roleID, domainID, token); err != nil {
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

	if err := sdk.RemoveClientRoleActions(cmd.Context(), clientID, roleID, domainID, actions.Actions, token); err != nil {
		logErrorCmd(*cmd, err)
		return
	}
	logOKCmd(*cmd)
}

func handleClientRoleActionsAvailable(cmd *cobra.Command, args []string) {
	if len(args) != 2 {
		logUsageCmd(*cmd, usageClientRoleActionsAvailable)
		return
	}

	domainID := args[0]
	token := args[1]

	acts, err := sdk.AvailableClientRoleActions(cmd.Context(), domainID, token)
	if err != nil {
		logErrorCmd(*cmd, err)
		return
	}
	logJSONCmd(*cmd, acts)
}

func handleClientRoleMembers(cmd *cobra.Command, clientID string, args []string) {
	if len(args) < 1 {
		logUsageCmd(*cmd, "cli clients <client_id> roles members <operation> [args...]")
		return
	}

	operation := args[0]
	opArgs := args[1:]

	switch operation {
	case add:
		handleClientRoleMembersAdd(cmd, clientID, opArgs)
	case list:
		handleClientRoleMembersList(cmd, clientID, opArgs)
	case delete:
		handleClientRoleMembersDelete(cmd, clientID, opArgs)
	default:
		logErrorCmd(*cmd, fmt.Errorf("unknown members operation: %s", operation))
	}
}

func handleClientRoleMembersAdd(cmd *cobra.Command, clientID string, args []string) {
	if len(args) != 4 {
		logUsageCmd(*cmd, usageClientRoleMembersAdd)
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

	memb, err := sdk.AddClientRoleMembers(cmd.Context(), clientID, roleID, domainID, members.Members, token)
	if err != nil {
		logErrorCmd(*cmd, err)
		return
	}
	logJSONCmd(*cmd, memb)
}

func handleClientRoleMembersList(cmd *cobra.Command, clientID string, args []string) {
	if len(args) != 3 {
		logUsageCmd(*cmd, usageClientRoleMembersList)
		return
	}

	roleID := args[0]
	domainID := args[1]
	token := args[2]

	pageMetadata := smqsdk.PageMetadata{
		Offset: Offset,
		Limit:  Limit,
	}

	l, err := sdk.ClientRoleMembers(cmd.Context(), clientID, roleID, domainID, pageMetadata, token)
	if err != nil {
		logErrorCmd(*cmd, err)
		return
	}
	logJSONCmd(*cmd, l)
}

func handleClientRoleMembersDelete(cmd *cobra.Command, clientID string, args []string) {
	if len(args) != 4 {
		logUsageCmd(*cmd, usageClientRoleMembersDelete)
		return
	}

	roleID := args[0]
	membersJSON := args[1]
	domainID := args[2]
	token := args[3]

	if membersJSON == all {
		if err := sdk.RemoveAllClientRoleMembers(cmd.Context(), clientID, roleID, domainID, token); err != nil {
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

	if err := sdk.RemoveClientRoleMembers(cmd.Context(), clientID, roleID, domainID, members.Members, token); err != nil {
		logErrorCmd(*cmd, err)
		return
	}
	logOKCmd(*cmd)
}
