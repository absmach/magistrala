// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"encoding/json"

	mfxsdk "github.com/mainflux/mainflux/pkg/sdk/go"
	"github.com/spf13/cobra"
)

const (
	users  = "users"
	things = "things"
)

var cmdPolicies = []cobra.Command{
	{
		Use:   "create [ users | things ] <subject_id> <object_id> <actions> <user_auth_token>",
		Short: "Create policy",
		Long: "Create a new policy\n" +
			"Usage:\n" +
			"\tmainflux-cli policies create users <user_id> <group_id> '[\"c_list\"]' $USERTOKEN\n" +
			"\tmainflux-cli policies create things <thing_id> <channel_id> '[\"m_write\"]' $USERTOKEN\n",
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 5 {
				logUsage(cmd.Use)
				return
			}

			var actions []string
			if err := json.Unmarshal([]byte(args[3]), &actions); err != nil {
				logError(err)
				return
			}

			var policy = mfxsdk.Policy{
				Subject: args[1],
				Object:  args[2],
				Actions: actions,
			}

			switch args[0] {
			case things:
				if err := sdk.CreateThingPolicy(policy, args[4]); err != nil {
					logError(err)
					return
				}
			case users:
				if err := sdk.CreateUserPolicy(policy, args[4]); err != nil {
					logError(err)
					return
				}
			default:
				logUsage(cmd.Use)
			}
		},
	},
	{
		Use:   "update [ users | things ] <subject_id> <object_id> <actions> <user_auth_token>",
		Short: "Update policy",
		Long: "Update policy\n" +
			"Usage:\n" +
			"\tmainflux-cli policies update users <user_id> <group_id> '[\"c_list\"]' $USERTOKEN\n" +
			"\tmainflux-cli policies update things <thing_id> <channel_id> '[\"m_write\"]' $USERTOKEN\n",
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 5 {
				logUsage(cmd.Use)
				return
			}

			var actions []string
			if err := json.Unmarshal([]byte(args[3]), &actions); err != nil {
				logError(err)
				return
			}

			var policy = mfxsdk.Policy{
				Subject: args[1],
				Object:  args[2],
				Actions: actions,
			}

			switch args[0] {
			case things:
				if err := sdk.UpdateThingPolicy(policy, args[4]); err != nil {
					logError(err)
					return
				}
			case users:
				if err := sdk.UpdateUserPolicy(policy, args[4]); err != nil {
					logError(err)
					return
				}
			default:
				logUsage(cmd.Use)
			}
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
				Offset: Offset,
				Limit:  Limit,
			}
			switch args[0] {
			case things:
				policies, err := sdk.ListThingPolicies(pm, args[1])
				if err != nil {
					logError(err)
					return
				}
				logJSON(policies)
				return
			case users:
				policies, err := sdk.ListUserPolicies(pm, args[1])
				if err != nil {
					logError(err)
					return
				}

				logJSON(policies)
				return
			default:
				logUsage(cmd.Use)
			}
		},
	},
	{
		Use:   "remove [ users | things ] <subject_id> <object_id> <user_auth_token>",
		Short: "Remove policy",
		Long: "Removes a policy with the provided object and subject\n" +
			"Usage:\n" +
			"\tmainflux-cli policies remove users <user_id> <group_id> $USERTOKEN\n" +
			"\tmainflux-cli policies remove things <thing_id> <channel_id> $USERTOKEN\n",
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 4 {
				logUsage(cmd.Use)
				return
			}

			var policy = mfxsdk.Policy{
				Subject: args[1],
				Object:  args[2],
			}
			switch args[0] {
			case things:
				if err := sdk.DeleteThingPolicy(policy, args[3]); err != nil {
					logError(err)
					return
				}
			case users:
				if err := sdk.DeleteUserPolicy(policy, args[3]); err != nil {
					logError(err)
					return
				}
			default:
				logUsage(cmd.Use)
			}
		},
	},
	{
		Use:   "authorize [ users | things ] <subject_id> <object_id> <action> <entity_type> <user_auth_token>",
		Short: "Authorize access request",
		Long: "Authorize subject over object with provided actions\n" +
			"Usage:\n" +
			"\tmainflux-cli policies authorize users <user_id> <group_id> \"c_list\" <entity_type> $USERTOKEN\n" +
			"\tmainflux-cli policies authorize things <thing_id> <channel_id> \"m_read\" <entity_type> $USERTOKEN\n",
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 6 {
				logUsage(cmd.Use)
				return
			}

			var areq = mfxsdk.AccessRequest{
				Subject:    args[1],
				Object:     args[2],
				Action:     args[3],
				EntityType: args[4],
			}

			switch args[0] {
			case users:
				ok, err := sdk.AuthorizeUser(areq, args[5])
				if err != nil {
					logError(err)
					return
				}
				logJSON(ok)
			case things:
				ok, _, err := sdk.AuthorizeThing(areq, args[5])
				if err != nil {
					logError(err)
					return
				}
				logJSON(ok)
			default:
				logUsage(cmd.Use)
			}
		},
	},
}

// NewPolicyCmd returns policies command.
func NewPolicyCmd() *cobra.Command {
	cmd := cobra.Command{
		Use:   "policies [create | update | list | remove | authorize ]",
		Short: "Policies management",
		Long:  `Policies management: create or update or list or delete or check policies`,
	}

	for i := range cmdPolicies {
		cmd.AddCommand(&cmdPolicies[i])
	}

	return &cmd
}
