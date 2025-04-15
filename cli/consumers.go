// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	mgsdk "github.com/absmach/magistrala/pkg/sdk"
	"github.com/spf13/cobra"
)

var cmdSubscription = []cobra.Command{
	{
		Use:   "create <topic> <contact> <user_auth_token>",
		Short: "Create subscription",
		Long:  `Create new subscription`,
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 3 {
				logUsageCmd(*cmd, cmd.Use)
				return
			}

			id, err := sdk.CreateSubscription(cmd.Context(), args[0], args[1], args[2])
			if err != nil {
				logErrorCmd(*cmd, err)
				return
			}

			logCreatedCmd(*cmd, id)
		},
	},
	{
		Use:   "get [all | <sub_id>] <user_auth_token>",
		Short: "Get subscription",
		Long: `Get subscription.
				all - lists all subscriptions
				<sub_id> - view subscription of <sub_id>`,
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 2 {
				logUsageCmd(*cmd, cmd.Use)
				return
			}
			pageMetadata := mgsdk.PageMetadata{
				Offset:  Offset,
				Limit:   Limit,
				Topic:   Topic,
				Contact: Contact,
			}
			if args[0] == "all" {
				sub, err := sdk.ListSubscriptions(cmd.Context(), pageMetadata, args[1])
				if err != nil {
					logErrorCmd(*cmd, err)
					return
				}
				logJSONCmd(*cmd, sub)
				return
			}

			c, err := sdk.ViewSubscription(cmd.Context(), args[0], args[1])
			if err != nil {
				logErrorCmd(*cmd, err)
				return
			}

			logJSONCmd(*cmd, c)
		},
	},
	{
		Use:   "remove <sub_id> <user_auth_token>",
		Short: "Remove subscription",
		Long:  `Removes removes a subscription with the provided id`,
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 2 {
				logUsageCmd(*cmd, cmd.Use)
				return
			}

			if err := sdk.DeleteSubscription(cmd.Context(), args[0], args[1]); err != nil {
				logErrorCmd(*cmd, err)
				return
			}

			logOKCmd(*cmd)
		},
	},
}

// NewSubscriptionCmd returns subscription command.
func NewSubscriptionCmd() *cobra.Command {
	cmd := cobra.Command{
		Use:   "subscription [create | get | remove ]",
		Short: "Subscription management",
		Long:  `Subscription management: create, get, or delete subscription`,
	}

	for i := range cmdSubscription {
		cmd.AddCommand(&cmdSubscription[i])
	}

	return &cmd
}
