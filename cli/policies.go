// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"encoding/json"

	mfxsdk "github.com/mainflux/mainflux/pkg/sdk/go"
	"github.com/spf13/cobra"
)

var cmdPolicies = []cobra.Command{
	{
		Use:   "create <JSON_policy> <user_auth_token>",
		Short: "Create policy",
		Long: "Create a new policy\n" +
			"Usage:\n" +
			"\tmainflux-cli policies create '{\"object\":\"<group_id>\", \"subject\":\"<user_id>\",\"actions\":[\"c_list\"]}' $USERTOKEN\n",
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 2 {
				logUsage(cmd.Use)
				return
			}

			var policy mfxsdk.Policy
			if err := json.Unmarshal([]byte(args[0]), &policy); err != nil {
				logError(err)
				return
			}
			if err := sdk.CreatePolicy(policy, args[1]); err != nil {
				logError(err)
				return
			}

			logOK()
		},
	},
	{
		Use:   "update [ <JSON_policy> | things <JSON_policy> ] <user_auth_token>",
		Short: "Update policy",
		Long: "Update policy\n" +
			"Usage:\n" +
			"\tmainflux-cli policies update '{\"object\":\"<group_id>\", \"subject\":\"<user_id>\",\"actions\":[\"c_list\"]}' $USERTOKEN\n" +
			"\tmainflux-cli policies update things '{\"object\":\"<channel_id>\", \"subject\":\"<thing_id>\",\"actions\":[\"m_write\"]}' $USERTOKEN\n",
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 2 && len(args) != 3 {
				logUsage(cmd.Use)
				return
			}

			var policy mfxsdk.Policy
			if err := json.Unmarshal([]byte(args[0]), &policy); err != nil {
				logError(err)
				return
			}
			if args[0] == "things" {
				if err := sdk.UpdateThingsPolicy(policy, args[2]); err != nil {
					logError(err)
					return
				}
			}
			if err := sdk.UpdatePolicy(policy, args[1]); err != nil {
				logError(err)
				return
			}

			logOK()
		},
	},
	{
		Use:   "list [ users | things ] <user_auth_token>",
		Short: "List policies",
		Long: "List policies\n" +
			"Usage:\n" +
			"\tmainflux-cli policies list users $USERTOKEN\n" +
			"\tmainflux-cli policies list things $USERTOKEN\n",
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 2 {
				logUsage(cmd.Use)
				return
			}
			pm := mfxsdk.PageMetadata{
				Offset: uint64(Offset),
				Limit:  uint64(Limit),
			}
			if args[0] == "things" {
				policies, err := sdk.ListThingsPolicies(pm, args[1])
				if err != nil {
					logError(err)
					return
				}
				logJSON(policies)
				return
			}
			if args[0] == "users" {
				policies, err := sdk.ListPolicies(pm, args[0])
				if err != nil {
					logError(err)
					return
				}

				logJSON(policies)
				return
			}
		},
	},
	{
		Use:   "remove <JSON_policy> <user_auth_token>",
		Short: "Remove policy",
		Long: "Removes a policy with the provided object and subject\n" +
			"Usage:\n" +
			"\tmainflux-cli policies remove '{\"object\":\"<group_id>\", \"subject\":\"<user_id>\"}' $USERTOKEN\n",
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 2 {
				logUsage(cmd.Use)
				return
			}

			var policy mfxsdk.Policy
			if err := json.Unmarshal([]byte(args[0]), &policy); err != nil {
				logError(err)
				return
			}
			if err := sdk.DeletePolicy(policy, args[1]); err != nil {
				logError(err)
				return
			}

			logOK()
		},
	},
}

// NewPolicyCmd returns policies command.
func NewPolicyCmd() *cobra.Command {
	cmd := cobra.Command{
		Use:   "policies [create | update | list | remove ]",
		Short: "Policies management",
		Long:  `Policies management: create or update or list or delete policies`,
	}

	for i := range cmdPolicies {
		cmd.AddCommand(&cmdPolicies[i])
	}

	return &cmd
}
