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
		Long: `Create new policy:
				{
					"Object":<object>,
					"Subjects":[<subject1>, ...],
					"Policies":[<policy1>, ...],
				}`,
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
		Use:   "remove <JSON_policy> <user_auth_token>",
		Short: "Remove policy",
		Long: `Removes removes a policy with the provided object and subject
				{
					"Object":<object>,
					"Subjects":[<subject1>, ...],
					"Policies":[<policy1>, ...],
				}`,
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
		Use:   "policies [create | remove ]",
		Short: "Policies management",
		Long:  `Policies management: create or delete policies`,
	}

	for i := range cmdPolicies {
		cmd.AddCommand(&cmdPolicies[i])
	}

	return &cmd
}
