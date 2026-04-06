// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	smqsdk "github.com/absmach/magistrala/pkg/sdk"
	"github.com/spf13/cobra"
)

var cmdUserInvitations = []cobra.Command{
	{
		Use:   "get <user_auth_token>",
		Short: "Get user invitations",
		Long: "Get all invitations for the authenticated user\n" +
			"Usage:\n" +
			"\tmagistrala-cli invitations user get <user_auth_token> - lists all invitations for the user\n" +
			"\tmagistrala-cli invitations user get <user_auth_token> --offset <offset> --limit <limit> - lists all invitations with provided offset and limit\n",
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 1 {
				logUsageCmd(*cmd, cmd.Use)
				return
			}

			pageMetadata := smqsdk.PageMetadata{
				Offset: Offset,
				Limit:  Limit,
			}

			l, err := sdk.Invitations(cmd.Context(), pageMetadata, args[0])
			if err != nil {
				logErrorCmd(*cmd, err)
				return
			}
			logJSONCmd(*cmd, l)
		},
	},
	{
		Use:   "accept <domain_id> <user_auth_token>",
		Short: "Accept invitation",
		Long: "Accept invitation to domain\n" +
			"Usage:\n" +
			"\tmagistrala-cli invitations user accept 39f97daf-d6b6-40f4-b229-2697be8006ef $USER_TOKEN\n",
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 2 {
				logUsageCmd(*cmd, cmd.Use)
				return
			}

			if err := sdk.AcceptInvitation(cmd.Context(), args[0], args[1]); err != nil {
				logErrorCmd(*cmd, err)
				return
			}

			logOKCmd(*cmd)
		},
	},
	{
		Use:   "reject <domain_id> <user_auth_token>",
		Short: "Reject invitation",
		Long: "Reject invitation to domain\n" +
			"Usage:\n" +
			"\tmagistrala-cli invitations user reject 39f97daf-d6b6-40f4-b229-2697be8006ef $USER_AUTH_TOKEN\n",
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 2 {
				logUsageCmd(*cmd, cmd.Use)
				return
			}

			if err := sdk.RejectInvitation(cmd.Context(), args[0], args[1]); err != nil {
				logErrorCmd(*cmd, err)
				return
			}

			logOKCmd(*cmd)
		},
	},
}

var cmdDomainInvitations = []cobra.Command{
	{
		Use:   "send <user_id> <domain_id> <role_id> <user_auth_token>",
		Short: "Send domain invitation",
		Long: "Send invitation to user for a domain\n" +
			"For example:\n" +
			"\tmagistrala-cli invitations domain send 39f97daf-d6b6-40f4-b229-2697be8006ef 4ef09eff-d500-4d56-b04f-d23a512d6f2a ba4c904c-e6d4-4978-9417-1694aac6793e $USER_AUTH_TOKEN\n",
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 4 {
				logUsageCmd(*cmd, cmd.Use)
				return
			}
			inv := smqsdk.Invitation{
				InviteeUserID: args[0],
				DomainID:      args[1],
				RoleID:        args[2],
			}
			if err := sdk.SendInvitation(cmd.Context(), inv, args[3]); err != nil {
				logErrorCmd(*cmd, err)
				return
			}

			logOKCmd(*cmd)
		},
	},
	{
		Use:   "get <domain_id> <user_auth_token>",
		Short: "Get domain invitations",
		Long: "Get all invitations for a specific domain\n" +
			"Usage:\n" +
			"\tmagistrala-cli invitations domain get <domain_id> <user_auth_token> - shows invitations for domain\n" +
			"\tmagistrala-cli invitations domain get <domain_id> <user_auth_token> --offset <offset> --limit <limit> - shows invitations with provided offset and limit\n",
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 2 {
				logUsageCmd(*cmd, cmd.Use)
				return
			}

			pageMetadata := smqsdk.PageMetadata{
				Offset: Offset,
				Limit:  Limit,
			}

			u, err := sdk.DomainInvitations(cmd.Context(), pageMetadata, args[1], args[0])
			if err != nil {
				logErrorCmd(*cmd, err)
				return
			}

			logJSONCmd(*cmd, u)
		},
	},
	{
		Use:   "delete <user_id> <domain_id> <user_auth_token>",
		Short: "Delete domain invitation",
		Long: "Delete invitation for a specific user and domain\n" +
			"Usage:\n" +
			"\tmagistrala-cli invitations domain delete 39f97daf-d6b6-40f4-b229-2697be8006ef 4ef09eff-d500-4d56-b04f-d23a512d6f2a $USER_TOKEN\n",
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 3 {
				logUsageCmd(*cmd, cmd.Use)
				return
			}

			if err := sdk.DeleteInvitation(cmd.Context(), args[0], args[1], args[2]); err != nil {
				logErrorCmd(*cmd, err)
				return
			}

			logOKCmd(*cmd)
		},
	},
}

// NewUserInvitationsCmd returns user invitations command.
func NewUserInvitationsCmd() *cobra.Command {
	cmd := cobra.Command{
		Use:   "user [get | accept | reject]",
		Short: "User invitations management",
		Long:  `User invitations management to get, accept and reject invitations received by the user`,
	}

	for i := range cmdUserInvitations {
		cmd.AddCommand(&cmdUserInvitations[i])
	}

	return &cmd
}

// NewDomainInvitationsCmd returns domain invitations command.
func NewDomainInvitationsCmd() *cobra.Command {
	cmd := cobra.Command{
		Use:   "domain [send | get | delete]",
		Short: "Domain invitations management",
		Long:  `Domain invitations management to send, get and delete invitations for domains`,
	}

	for i := range cmdDomainInvitations {
		cmd.AddCommand(&cmdDomainInvitations[i])
	}

	return &cmd
}

// NewInvitationsCmd returns invitations command.
func NewInvitationsCmd() *cobra.Command {
	cmd := cobra.Command{
		Use:   "invitations [user | domain]",
		Short: "Invitations management",
		Long:  `Invitations management with separate commands for user and domain invitations`,
	}

	cmd.AddCommand(NewUserInvitationsCmd())
	cmd.AddCommand(NewDomainInvitationsCmd())

	return &cmd
}
