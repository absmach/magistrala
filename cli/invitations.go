// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	mgxsdk "github.com/absmach/magistrala/pkg/sdk/go"
	"github.com/spf13/cobra"
)

var cmdInvitations = []cobra.Command{
	{
		Use:   "send <user_id> <domain_id> <relation> <user_auth_token>",
		Short: "Send invitation",
		Long: "Send invitation to user\n" +
			"For example:\n" +
			"\tmagistrala-cli invitations send 39f97daf-d6b6-40f4-b229-2697be8006ef 4ef09eff-d500-4d56-b04f-d23a512d6f2a administrator $USER_AUTH_TOKEN\n",
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 4 {
				logUsage(cmd.Use)
				return
			}
			inv := mgxsdk.Invitation{
				UserID:   args[0],
				DomainID: args[1],
				Relation: args[2],
			}
			if err := sdk.SendInvitation(inv, args[3]); err != nil {
				logError(err)
				return
			}

			logOK()
		},
	},
	{
		Use:   "get [all | <user_id> <domain_id> ] <user_auth_token>",
		Short: "Get invitations",
		Long: "Get invitations\n" +
			"Usage:\n" +
			"\tmagistrala-cli invitations get all <user_auth_token> - lists all invitations\n" +
			"\tmagistrala-cli invitations get all <user_auth_token> --offset <offset> --limit <limit> - lists all invitations with provided offset and limit\n" +
			"\tmagistrala-cli invitations get <user_id> <domain_id> <user_auth_token> - shows invitation by user id and domain id\n",
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 2 && len(args) != 3 {
				logUsage(cmd.Use)
				return
			}

			pageMetadata := mgxsdk.PageMetadata{
				Identity: Identity,
				Offset:   Offset,
				Limit:    Limit,
			}
			if args[0] == all {
				l, err := sdk.Invitations(pageMetadata, args[1])
				if err != nil {
					logError(err)
					return
				}
				logJSON(l)
				return
			}
			u, err := sdk.Invitation(args[0], args[1], args[2])
			if err != nil {
				logError(err)
				return
			}

			logJSON(u)
		},
	},
	{
		Use:   "accept <domain_id> <user_auth_token>",
		Short: "Accept invitation",
		Long: "Accept invitation to domain\n" +
			"Usage:\n" +
			"\tmagistrala-cli invitations accept 39f97daf-d6b6-40f4-b229-2697be8006ef $USERTOKEN\n",
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 2 {
				logUsage(cmd.Use)
				return
			}

			if err := sdk.AcceptInvitation(args[0], args[1]); err != nil {
				logError(err)
				return
			}

			logOK()
		},
	},
	{
		Use:   "delete <user_id> <domain_id> <user_auth_token>",
		Short: "Delete invitation",
		Long: "Delete invitation\n" +
			"Usage:\n" +
			"\tmagistrala-cli invitations delete 39f97daf-d6b6-40f4-b229-2697be8006ef 4ef09eff-d500-4d56-b04f-d23a512d6f2a $USERTOKEN\n",
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 3 {
				logUsage(cmd.Use)
				return
			}

			if err := sdk.DeleteInvitation(args[0], args[1], args[2]); err != nil {
				logError(err)
				return
			}

			logOK()
		},
	},
}

// NewInvitationsCmd returns invitations command.
func NewInvitationsCmd() *cobra.Command {
	cmd := cobra.Command{
		Use:   "invitations [send | get | accept | delete]",
		Short: "Invitations management",
		Long:  `Invitations management to send, get, accept and delete invitations`,
	}

	for i := range cmdInvitations {
		cmd.AddCommand(&cmdInvitations[i])
	}

	return &cmd
}
