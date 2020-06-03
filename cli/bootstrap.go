// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"encoding/json"

	mfxsdk "github.com/mainflux/mainflux/pkg/sdk/go"
	"github.com/spf13/cobra"
)

var cmdBootstrap = []cobra.Command{
	cobra.Command{
		Use:   "add",
		Short: "add <JSON_config> <user_auth_token>",
		Long:  `Adds new Thing Bootstrap Config to the user identified by the provided key`,
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 2 {
				logUsage(cmd.Short)
				return
			}

			var cfg mfxsdk.BootstrapConfig
			if err := json.Unmarshal([]byte(args[0]), &cfg); err != nil {
				logError(err)
				return
			}

			id, err := sdk.AddBootstrap(args[1], cfg)
			if err != nil {
				logError(err)
				return
			}

			logCreated(id)
		},
	},
	cobra.Command{
		Use:   "view",
		Short: "view <thing_id> <user_auth_token>",
		Long:  `Returns Thing Config with given ID belonging to the user identified by the given key`,
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 2 {
				logUsage(cmd.Short)
				return
			}

			c, err := sdk.ViewBootstrap(args[1], args[0])
			if err != nil {
				logError(err)
				return
			}

			logJSON(c)
		},
	},
	cobra.Command{
		Use:   "update",
		Short: "update <JSON_config> <user_auth_token>",
		Long:  `Updates editable fields of the provided Config`,
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 2 {
				logUsage(cmd.Short)
				return
			}

			var cfg mfxsdk.BootstrapConfig
			if err := json.Unmarshal([]byte(args[0]), &cfg); err != nil {
				logError(err)
				return
			}

			if err := sdk.UpdateBootstrap(args[1], cfg); err != nil {
				logError(err)
				return
			}

			logOK()
		},
	},
	cobra.Command{
		Use:   "remove",
		Short: "remove <thing_id> <user_auth_token>",
		Long:  `Removes Config with specified key that belongs to the user identified by the given key`,
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 2 {
				logUsage(cmd.Short)
				return
			}

			if err := sdk.RemoveBootstrap(args[1], args[0]); err != nil {
				logError(err)
				return
			}

			logOK()
		},
	},
	cobra.Command{
		Use:   "bootstrap",
		Short: "bootstrap <external_id> <external_key>",
		Long:  `Returns Config to the Thing with provided external ID using external key`,
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 2 {
				logUsage(cmd.Short)
				return
			}

			c, err := sdk.Bootstrap(args[1], args[0])
			if err != nil {
				logError(err)
				return
			}

			logJSON(c)
		},
	},
}

// NewBootstrapCmd returns bootstrap command.
func NewBootstrapCmd() *cobra.Command {
	cmd := cobra.Command{
		Use:   "bootstrap",
		Short: "Bootstrap management",
		Long:  `Bootstrap management: create, get, update or delete Bootstrap config`,
		Run: func(cmd *cobra.Command, args []string) {
			logUsage("bootstrap [add | view | update | remove | bootstrap]")
		},
	}

	for i := range cmdBootstrap {
		cmd.AddCommand(&cmdBootstrap[i])
	}

	return &cmd
}
