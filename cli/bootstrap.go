// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"encoding/json"

	mgxsdk "github.com/absmach/magistrala/pkg/sdk/go"
	"github.com/spf13/cobra"
)

var cmdBootstrap = []cobra.Command{
	{
		Use:   "create <JSON_config> <domain_id> <user_auth_token>",
		Short: "Create config",
		Long:  `Create new Client Bootstrap Config to the user identified by the provided key`,
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 3 {
				logUsageCmd(*cmd, cmd.Use)
				return
			}

			var cfg mgxsdk.BootstrapConfig
			if err := json.Unmarshal([]byte(args[0]), &cfg); err != nil {
				logErrorCmd(*cmd, err)
				return
			}

			id, err := sdk.AddBootstrap(cfg, args[1], args[2])
			if err != nil {
				logErrorCmd(*cmd, err)
				return
			}

			logCreatedCmd(*cmd, id)
		},
	},
	{
		Use:   "get [all | <client_id>] <domain_id> <user_auth_token>",
		Short: "Get config",
		Long: `Get Client Config with given ID belonging to the user identified by the given key.
				all - lists all config
				<client_id> - view config of <client_id>`,
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 3 {
				logUsageCmd(*cmd, cmd.Use)
				return
			}
			pageMetadata := mgxsdk.PageMetadata{
				Offset: Offset,
				Limit:  Limit,
				State:  State,
				Name:   Name,
			}
			if args[0] == "all" {
				l, err := sdk.Bootstraps(pageMetadata, args[1], args[2])
				if err != nil {
					logErrorCmd(*cmd, err)
					return
				}
				logJSONCmd(*cmd, l)
				return
			}

			c, err := sdk.ViewBootstrap(args[0], args[1], args[2])
			if err != nil {
				logErrorCmd(*cmd, err)
				return
			}

			logJSONCmd(*cmd, c)
		},
	},
	{
		Use:   "update [config <JSON_config> | connection <id> <channel_ids> | certs  <id> <client_cert> <client_key> <ca> ] <domain_id> <user_auth_token>",
		Short: "Update config",
		Long: `Updates editable fields of the provided Config.
				config <JSON_config> - Updates editable fields of the provided Config.
				connection <id> <channel_ids> - Updates connections performs update of the channel list corresponding Client is connected to.
				channel_ids - '["channel_id1", ...]'
				certs  <id> <client_cert> <client_key> <ca> - Update bootstrap config certificates.`,
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) < 4 {
				logUsageCmd(*cmd, cmd.Use)
				return
			}
			if args[0] == "config" {
				var cfg mgxsdk.BootstrapConfig
				if err := json.Unmarshal([]byte(args[1]), &cfg); err != nil {
					logErrorCmd(*cmd, err)
					return
				}

				if err := sdk.UpdateBootstrap(cfg, args[1], args[2]); err != nil {
					logErrorCmd(*cmd, err)
					return
				}

				logOKCmd(*cmd)
				return
			}
			if args[0] == "connection" {
				var ids []string
				if err := json.Unmarshal([]byte(args[2]), &ids); err != nil {
					logErrorCmd(*cmd, err)
					return
				}
				if err := sdk.UpdateBootstrapConnection(args[1], ids, args[3], args[4]); err != nil {
					logErrorCmd(*cmd, err)
					return
				}

				logOKCmd(*cmd)
				return
			}
			if args[0] == "certs" {
				cfg, err := sdk.UpdateBootstrapCerts(args[0], args[1], args[2], args[3], args[4], args[5])
				if err != nil {
					logErrorCmd(*cmd, err)
					return
				}

				logJSONCmd(*cmd, cfg)
				return
			}
			logUsageCmd(*cmd, cmd.Use)
		},
	},
	{
		Use:   "remove <client_id> <domain_id> <user_auth_token>",
		Short: "Remove config",
		Long:  `Removes Config with specified key that belongs to the user identified by the given key`,
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 3 {
				logUsageCmd(*cmd, cmd.Use)
				return
			}

			if err := sdk.RemoveBootstrap(args[0], args[1], args[2]); err != nil {
				logErrorCmd(*cmd, err)
				return
			}

			logOKCmd(*cmd)
		},
	},
	{
		Use:   "bootstrap [<external_id> <external_key> | secure <external_id> <external_key> <crypto_key> ]",
		Short: "Bootstrap config",
		Long: `Returns Config to the Client with provided external ID using external key.
				secure - Retrieves a configuration with given external ID and encrypted external key.`,
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) < 2 {
				logUsageCmd(*cmd, cmd.Use)
				return
			}
			if args[0] == "secure" {
				c, err := sdk.BootstrapSecure(args[1], args[2], args[3])
				if err != nil {
					logErrorCmd(*cmd, err)
					return
				}

				logJSONCmd(*cmd, c)
				return
			}
			c, err := sdk.Bootstrap(args[0], args[1])
			if err != nil {
				logErrorCmd(*cmd, err)
				return
			}

			logJSONCmd(*cmd, c)
		},
	},
	{
		Use:   "whitelist <JSON_config> <domain_id> <user_auth_token>",
		Short: "Whitelist config",
		Long:  `Whitelist updates client state config with given id from the authenticated user`,
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 3 {
				logUsageCmd(*cmd, cmd.Use)
				return
			}

			var cfg mgxsdk.BootstrapConfig
			if err := json.Unmarshal([]byte(args[0]), &cfg); err != nil {
				logErrorCmd(*cmd, err)
				return
			}

			if err := sdk.Whitelist(cfg.ClientID, cfg.State, args[1], args[2]); err != nil {
				logErrorCmd(*cmd, err)
				return
			}

			logOKCmd(*cmd)
		},
	},
}

// NewBootstrapCmd returns bootstrap command.
func NewBootstrapCmd() *cobra.Command {
	cmd := cobra.Command{
		Use:   "bootstrap [create | get | update | remove | bootstrap | whitelist]",
		Short: "Bootstrap management",
		Long:  `Bootstrap management: create, get, update, delete or whitelist Bootstrap config`,
	}

	for i := range cmdBootstrap {
		cmd.AddCommand(&cmdBootstrap[i])
	}

	return &cmd
}
