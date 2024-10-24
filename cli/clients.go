// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"encoding/json"

	"github.com/absmach/magistrala/clients"
	mgxsdk "github.com/absmach/magistrala/pkg/sdk/go"
	"github.com/spf13/cobra"
)

var cmdClients = []cobra.Command{
	{
		Use:   "create <JSON_client> <domain_id> <user_auth_token>",
		Short: "Create client",
		Long: "Creates new client with provided name and metadata\n" +
			"Usage:\n" +
			"\tmagistrala-cli clients create '{\"name\":\"new client\", \"metadata\":{\"key\": \"value\"}}' $DOMAINID $USERTOKEN\n",
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 3 {
				logUsageCmd(*cmd, cmd.Use)
				return
			}

			var client mgxsdk.Client
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
			"\tmagistrala-cli clients get all $DOMAINID $USERTOKEN - lists all clients\n" +
			"\tmagistrala-cli clients get all $DOMAINID $USERTOKEN --offset=10 --limit=10 - lists all clients with offset and limit\n" +
			"\tmagistrala-cli clients get <client_id> $DOMAINID $USERTOKEN - shows client with provided <client_id>\n",
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
			pageMetadata := mgxsdk.PageMetadata{
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
			"\tmagistrala-cli clients delete <client_id> $DOMAINID $USERTOKEN - delete client with <client_id>\n",
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
			"\tmagistrala-cli client update <client_id> '{\"name\":\"new name\", \"metadata\":{\"key\": \"value\"}}' $DOMAINID $USERTOKEN\n" +
			"\tmagistrala-cli client update tags <client_id> '{\"tag1\":\"value1\", \"tag2\":\"value2\"}' $DOMAINID $USERTOKEN\n" +
			"\tmagistrala-cli client update secret <client_id> <newsecret> $DOMAINID $USERTOKEN\n",
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 5 && len(args) != 4 {
				logUsageCmd(*cmd, cmd.Use)
				return
			}

			var client mgxsdk.Client
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
			"\tmagistrala-cli clients enable <client_id> $DOMAINID $USERTOKEN\n",
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
			"\tmagistrala-cli clients disable <client_id> $DOMAINID $USERTOKEN\n",
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
		Use:   "share <client_id> <user_id> <relation> <domain_id> <user_auth_token>",
		Short: "Share client with a user",
		Long: "Share client with a user\n" +
			"Usage:\n" +
			"\tmagistrala-cli clients share <client_id> <user_id> <relation> $DOMAINID $USERTOKEN\n",
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 5 {
				logUsageCmd(*cmd, cmd.Use)
				return
			}
			req := mgxsdk.UsersRelationRequest{
				Relation: args[2],
				UserIDs:  []string{args[1]},
			}
			err := sdk.ShareClient(args[0], req, args[3], args[4])
			if err != nil {
				logErrorCmd(*cmd, err)
				return
			}

			logOKCmd(*cmd)
		},
	},
	{
		Use:   "unshare <client_id> <user_id> <relation> <domain_id> <user_auth_token>",
		Short: "Unshare client with a user",
		Long: "Unshare client with a user\n" +
			"Usage:\n" +
			"\tmagistrala-cli clients share  <client_id> <user_id> <relation> $DOMAINID $USERTOKEN\n",
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 5 {
				logUsageCmd(*cmd, cmd.Use)
				return
			}
			req := mgxsdk.UsersRelationRequest{
				Relation: args[2],
				UserIDs:  []string{args[1]},
			}
			err := sdk.UnshareClient(args[0], req, args[3], args[4])
			if err != nil {
				logErrorCmd(*cmd, err)
				return
			}

			logOKCmd(*cmd)
		},
	},
	{
		Use:   "connect <client_id> <channel_id> <domain_id> <user_auth_token>",
		Short: "Connect client",
		Long: "Connect client to the channel\n" +
			"Usage:\n" +
			"\tmagistrala-cli clients connect <client_id> <channel_id> $DOMAINID $USERTOKEN\n",
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 4 {
				logUsageCmd(*cmd, cmd.Use)
				return
			}

			connIDs := mgxsdk.Connection{
				ChannelID: args[1],
				ClientID:  args[0],
			}
			if err := sdk.Connect(connIDs, args[2], args[3]); err != nil {
				logErrorCmd(*cmd, err)
				return
			}

			logOKCmd(*cmd)
		},
	},
	{
		Use:   "disconnect <client_id> <channel_id> <domain_id> <user_auth_token>",
		Short: "Disconnect client",
		Long: "Disconnect client to the channel\n" +
			"Usage:\n" +
			"\tmagistrala-cli clients disconnect <client_id> <channel_id> $DOMAINID $USERTOKEN\n",
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 4 {
				logUsageCmd(*cmd, cmd.Use)
				return
			}

			connIDs := mgxsdk.Connection{
				ClientID:  args[0],
				ChannelID: args[1],
			}
			if err := sdk.Disconnect(connIDs, args[2], args[3]); err != nil {
				logErrorCmd(*cmd, err)
				return
			}

			logOKCmd(*cmd)
		},
	},
	{
		Use:   "connections <client_id> <domain_id> <user_auth_token>",
		Short: "Connected list",
		Long: "List of Channels connected to Client\n" +
			"Usage:\n" +
			"\tmagistrala-cli connections <client_id> $DOMAINID $USERTOKEN\n",
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 3 {
				logUsageCmd(*cmd, cmd.Use)
				return
			}
			pm := mgxsdk.PageMetadata{
				Offset: Offset,
				Limit:  Limit,
			}
			cl, err := sdk.ChannelsByClient(args[0], pm, args[1], args[2])
			if err != nil {
				logErrorCmd(*cmd, err)
				return
			}

			logJSONCmd(*cmd, cl)
		},
	},
	{
		Use:   "users <client_id> <domain_id> <user_auth_token>",
		Short: "List users",
		Long: "List users of a client\n" +
			"Usage:\n" +
			"\tmagistrala-cli clients users <client_id> $DOMAINID $USERTOKEN\n",
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 3 {
				logUsageCmd(*cmd, cmd.Use)
				return
			}
			pm := mgxsdk.PageMetadata{
				Offset: Offset,
				Limit:  Limit,
			}
			ul, err := sdk.ListClientUsers(args[0], pm, args[1], args[2])
			if err != nil {
				logErrorCmd(*cmd, err)
				return
			}

			logJSONCmd(*cmd, ul)
		},
	},
}

// NewClientsCmd returns clients command.
func NewClientsCmd() *cobra.Command {
	cmd := cobra.Command{
		Use:   "clients [create | get | update | delete | share | connect | disconnect | connections | not-connected | users ]",
		Short: "Clients management",
		Long:  `Clients management: create, get, update, delete or share Client, connect or disconnect Client from Channel and get the list of Channels connected or disconnected from a Client`,
	}

	for i := range cmdClients {
		cmd.AddCommand(&cmdClients[i])
	}

	return &cmd
}
