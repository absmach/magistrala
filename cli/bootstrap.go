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
		Use:   "create <JSON_config> <user_auth_token>",
		Short: "Create config",
		Long:  `Create new Thing Bootstrap Config to the user identified by the provided key`,
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 2 {
				logUsage(cmd.Use)
				return
			}

			var cfg mgxsdk.BootstrapConfig
			if err := json.Unmarshal([]byte(args[0]), &cfg); err != nil {
				logError(err)
				return
			}

			id, err := sdk.AddBootstrap(cfg, args[1])
			if err != nil {
				logError(err)
				return
			}

			logCreated(id)
		},
	},
	{
		Use:   "get [all | <thing_id>] <user_auth_token>",
		Short: "Get config",
		Long: `Get Thing Config with given ID belonging to the user identified by the given key.
				all - lists all config
				<thing_id> - view config of <thing_id>`,
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 2 {
				logUsage(cmd.Use)
				return
			}
			pageMetadata := mgxsdk.PageMetadata{
				Offset: Offset,
				Limit:  Limit,
				State:  State,
				Name:   Name,
			}
			if args[0] == "all" {
				l, err := sdk.Bootstraps(pageMetadata, args[1])
				if err != nil {
					logError(err)
					return
				}
				logJSON(l)
				return
			}

			c, err := sdk.ViewBootstrap(args[0], args[1])
			if err != nil {
				logError(err)
				return
			}

			logJSON(c)
		},
	},
	{
		Use:   "update [config <JSON_config> | connection <id> <channel_ids> | certs  <id> <client_cert> <client_key> <ca> ] <user_auth_token>",
		Short: "Update config",
		Long: `Updates editable fields of the provided Config.
				config <JSON_config> - Updates editable fields of the provided Config.
				connection <id> <channel_ids> - Updates connections performs update of the channel list corresponding Thing is connected to.
				channel_ids - '["channel_id1", ...]'
				certs  <id> <client_cert> <client_key> <ca> - Update boostrap config certificates.`,
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) < 3 {
				logUsage(cmd.Use)
				return
			}
			if args[0] == "config" {
				var cfg mgxsdk.BootstrapConfig
				if err := json.Unmarshal([]byte(args[0]), &cfg); err != nil {
					logError(err)
					return
				}

				if err := sdk.UpdateBootstrap(cfg, args[1]); err != nil {
					logError(err)
					return
				}

				logOK()
				return
			}
			if args[0] == "connection" {
				var ids []string
				if err := json.Unmarshal([]byte(args[1]), &ids); err != nil {
					logError(err)
					return
				}
				if err := sdk.UpdateBootstrapConnection(args[0], ids, args[2]); err != nil {
					logError(err)
					return
				}

				logOK()
				return
			}
			if args[0] == "certs" {
				cfg, err := sdk.UpdateBootstrapCerts(args[0], args[1], args[2], args[3], args[4])
				if err != nil {
					logError(err)
					return
				}

				logJSON(cfg)
				return
			}
			logUsage(cmd.Use)
		},
	},
	{
		Use:   "remove <thing_id> <user_auth_token>",
		Short: "Remove config",
		Long:  `Removes Config with specified key that belongs to the user identified by the given key`,
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 2 {
				logUsage(cmd.Use)
				return
			}

			if err := sdk.RemoveBootstrap(args[0], args[1]); err != nil {
				logError(err)
				return
			}

			logOK()
		},
	},
	{
		Use:   "bootstrap [<external_id> <external_key> | secure <external_id> <external_key> ]",
		Short: "Bootstrap config",
		Long: `Returns Config to the Thing with provided external ID using external key.
				secure - Retrieves a configuration with given external ID and encrypted external key.`,
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) < 2 {
				logUsage(cmd.Use)
				return
			}
			if args[0] == "secure" {
				c, err := sdk.BootstrapSecure(args[1], args[2])
				if err != nil {
					logError(err)
					return
				}

				logJSON(c)
				return
			}
			c, err := sdk.Bootstrap(args[0], args[1])
			if err != nil {
				logError(err)
				return
			}

			logJSON(c)
		},
	},
	{
		Use:   "whitelist <JSON_config> <user_auth_token>",
		Short: "Whitelist config",
		Long:  `Whitelist updates thing state config with given id from the authenticated user`,
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 2 {
				logUsage(cmd.Use)
				return
			}

			var cfg mgxsdk.BootstrapConfig
			if err := json.Unmarshal([]byte(args[0]), &cfg); err != nil {
				logError(err)
				return
			}

			if err := sdk.Whitelist(cfg, args[1]); err != nil {
				logError(err)
				return
			}

			logOK()
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
