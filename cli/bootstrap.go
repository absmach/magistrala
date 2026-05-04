// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"encoding/json"

	mgsdk "github.com/absmach/magistrala/pkg/sdk"
	"github.com/spf13/cobra"
)

var cmdBootstrap = []cobra.Command{
	{
		Use:   "create <JSON_config> <domain_id> <user_auth_token>",
		Short: "Create config",
		Long:  `Create a new bootstrap enrollment in the given domain`,
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 3 {
				logUsageCmd(*cmd, cmd.Use)
				return
			}

			var cfg mgsdk.BootstrapConfig
			if err := json.Unmarshal([]byte(args[0]), &cfg); err != nil {
				logErrorCmd(*cmd, err)
				return
			}

			id, err := sdk.AddBootstrap(cmd.Context(), cfg, args[1], args[2])
			if err != nil {
				logErrorCmd(*cmd, err)
				return
			}

			logCreatedCmd(*cmd, id)
		},
	},
	{
		Use:   "get [all | <config_id>] <domain_id> <user_auth_token>",
		Short: "Get config",
		Long: `Get bootstrap enrollment with given ID belonging to the user identified by the given key.
				all - lists all config
				<config_id> - view config of <config_id>`,
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 3 {
				logUsageCmd(*cmd, cmd.Use)
				return
			}
			pageMetadata := mgsdk.PageMetadata{
				Offset: Offset,
				Limit:  Limit,
				Status: Status,
				Name:   Name,
			}
			if args[0] == all {
				l, err := sdk.Bootstraps(cmd.Context(), pageMetadata, args[1], args[2])
				if err != nil {
					logErrorCmd(*cmd, err)
					return
				}
				logJSONCmd(*cmd, l)
				return
			}

			c, err := sdk.ViewBootstrap(cmd.Context(), args[0], args[1], args[2])
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
				connection <id> <channel_ids> - Unsupported legacy operation kept for compatibility.
				certs  <id> <client_cert> <client_key> <ca> - Update bootstrap config certificates.`,
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) < 4 {
				logUsageCmd(*cmd, cmd.Use)
				return
			}
			if args[0] == "config" {
				var cfg mgsdk.BootstrapConfig
				if err := json.Unmarshal([]byte(args[1]), &cfg); err != nil {
					logErrorCmd(*cmd, err)
					return
				}

				if err := sdk.UpdateBootstrap(cmd.Context(), cfg, args[2], args[3]); err != nil {
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
				if err := sdk.UpdateBootstrapConnection(cmd.Context(), args[1], ids, args[3], args[4]); err != nil {
					logErrorCmd(*cmd, err)
					return
				}

				logOKCmd(*cmd)
				return
			}
			if args[0] == "certs" {
				cfg, err := sdk.UpdateBootstrapCerts(cmd.Context(), args[1], args[2], args[3], args[4], args[5], args[6])
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
		Use:   "remove <config_id> <domain_id> <user_auth_token>",
		Short: "Remove config",
		Long:  `Removes Config with specified key that belongs to the user identified by the given key`,
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 3 {
				logUsageCmd(*cmd, cmd.Use)
				return
			}

			if err := sdk.RemoveBootstrap(cmd.Context(), args[0], args[1], args[2]); err != nil {
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
				c, err := sdk.BootstrapSecure(cmd.Context(), args[1], args[2], args[3])
				if err != nil {
					logErrorCmd(*cmd, err)
					return
				}

				logJSONCmd(*cmd, c)
				return
			}
			c, err := sdk.Bootstrap(cmd.Context(), args[0], args[1])
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
		Long:  `Whitelist updates bootstrap status for the given enrollment`,
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 3 {
				logUsageCmd(*cmd, cmd.Use)
				return
			}

			var cfg mgsdk.BootstrapConfig
			if err := json.Unmarshal([]byte(args[0]), &cfg); err != nil {
				logErrorCmd(*cmd, err)
				return
			}

			if err := sdk.Whitelist(cmd.Context(), cfg.ID, cfg.Status, args[1], args[2]); err != nil {
				logErrorCmd(*cmd, err)
				return
			}

			logOKCmd(*cmd)
		},
	},
	{
		Use:   "profiles [create <JSON_profile> <domain_id> <user_auth_token> | get [all | <profile_id>] <domain_id> <user_auth_token> | update <JSON_profile> <domain_id> <user_auth_token> | remove <profile_id> <domain_id> <user_auth_token>]",
		Short: "Manage bootstrap profiles",
		Long: `Manage bootstrap profiles.
				create <JSON_profile> - Create a bootstrap profile.
				get all - List bootstrap profiles.
				get <profile_id> - View bootstrap profile.
				update <JSON_profile> - Update bootstrap profile.
				remove <profile_id> - Remove bootstrap profile.`,
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) == 0 {
				logUsageCmd(*cmd, cmd.Use)
				return
			}

			switch args[0] {
			case create:
				if len(args) != 4 {
					logUsageCmd(*cmd, cmd.Use)
					return
				}

				var profile mgsdk.BootstrapProfile
				if err := json.Unmarshal([]byte(args[1]), &profile); err != nil {
					logErrorCmd(*cmd, err)
					return
				}

				profile, err := sdk.CreateBootstrapProfile(cmd.Context(), profile, args[2], args[3])
				if err != nil {
					logErrorCmd(*cmd, err)
					return
				}

				logJSONCmd(*cmd, profile)
			case get:
				if len(args) != 4 {
					logUsageCmd(*cmd, cmd.Use)
					return
				}

				if args[1] == all {
					pageMetadata := mgsdk.PageMetadata{
						Offset: Offset,
						Limit:  Limit,
					}
					profiles, err := sdk.BootstrapProfiles(cmd.Context(), pageMetadata, args[2], args[3])
					if err != nil {
						logErrorCmd(*cmd, err)
						return
					}

					logJSONCmd(*cmd, profiles)
					return
				}

				profile, err := sdk.ViewBootstrapProfile(cmd.Context(), args[1], args[2], args[3])
				if err != nil {
					logErrorCmd(*cmd, err)
					return
				}

				logJSONCmd(*cmd, profile)
			case update:
				if len(args) != 4 {
					logUsageCmd(*cmd, cmd.Use)
					return
				}

				var profile mgsdk.BootstrapProfile
				if err := json.Unmarshal([]byte(args[1]), &profile); err != nil {
					logErrorCmd(*cmd, err)
					return
				}

				if err := sdk.UpdateBootstrapProfile(cmd.Context(), profile, args[2], args[3]); err != nil {
					logErrorCmd(*cmd, err)
					return
				}

				logOKCmd(*cmd)
			case "remove":
				if len(args) != 4 {
					logUsageCmd(*cmd, cmd.Use)
					return
				}

				if err := sdk.RemoveBootstrapProfile(cmd.Context(), args[1], args[2], args[3]); err != nil {
					logErrorCmd(*cmd, err)
					return
				}

				logOKCmd(*cmd)
			default:
				logUsageCmd(*cmd, cmd.Use)
			}
		},
	},
	{
		Use:   "enrollments [assign-profile <config_id> <profile_id> <domain_id> <user_auth_token> | bind <config_id> <JSON_bindings> <domain_id> <user_auth_token> | get-bindings <config_id> <domain_id> <user_auth_token> | refresh-bindings <config_id> <domain_id> <user_auth_token>]",
		Short: "Manage bootstrap enrollment bindings",
		Long: `Manage bootstrap enrollment profile assignments and bindings.
				assign-profile <config_id> <profile_id> - Assign a profile to an enrollment.
				bind <config_id> <JSON_bindings> - Bind concrete resources to an enrollment.
				get-bindings <config_id> - List stored binding snapshots for an enrollment.
				refresh-bindings <config_id> - Refresh stored binding snapshots for an enrollment.`,
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) == 0 {
				logUsageCmd(*cmd, cmd.Use)
				return
			}

			switch args[0] {
			case "assign-profile":
				if len(args) != 5 {
					logUsageCmd(*cmd, cmd.Use)
					return
				}

				if err := sdk.AssignBootstrapProfile(cmd.Context(), args[1], args[2], args[3], args[4]); err != nil {
					logErrorCmd(*cmd, err)
					return
				}

				logOKCmd(*cmd)
			case "bind":
				if len(args) != 5 {
					logUsageCmd(*cmd, cmd.Use)
					return
				}

				var bindings []mgsdk.BootstrapBindingRequest
				if err := json.Unmarshal([]byte(args[2]), &bindings); err != nil {
					logErrorCmd(*cmd, err)
					return
				}

				if err := sdk.BindBootstrapResources(cmd.Context(), args[1], bindings, args[3], args[4]); err != nil {
					logErrorCmd(*cmd, err)
					return
				}

				logOKCmd(*cmd)
			case "get-bindings":
				if len(args) != 4 {
					logUsageCmd(*cmd, cmd.Use)
					return
				}

				bindings, err := sdk.BootstrapBindings(cmd.Context(), args[1], args[2], args[3])
				if err != nil {
					logErrorCmd(*cmd, err)
					return
				}

				logJSONCmd(*cmd, bindings)
			case "refresh-bindings":
				if len(args) != 4 {
					logUsageCmd(*cmd, cmd.Use)
					return
				}

				if err := sdk.RefreshBootstrapBindings(cmd.Context(), args[1], args[2], args[3]); err != nil {
					logErrorCmd(*cmd, err)
					return
				}

				logOKCmd(*cmd)
			default:
				logUsageCmd(*cmd, cmd.Use)
			}
		},
	},
}

// NewBootstrapCmd returns bootstrap command.
func NewBootstrapCmd() *cobra.Command {
	cmd := cobra.Command{
		Use:   "bootstrap [create | get | update | remove | bootstrap | whitelist | profiles | enrollments]",
		Short: "Bootstrap management",
		Long:  `Bootstrap management: create, get, update, delete, whitelist, profiles, and enrollment bindings`,
	}

	for i := range cmdBootstrap {
		cmd.AddCommand(&cmdBootstrap[i])
	}

	return &cmd
}
