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
		Use:   "accept <workspace_id> <user_auth_token>",
		Short: "Accept invitation",
		Long: "Accept invitation to workspace\n" +
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
		Use:   "reject <workspace_id> <user_auth_token>",
		Short: "Reject invitation",
		Long: "Reject invitation to workspace\n" +
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

var cmdWorkspaceInvitations = []cobra.Command{
	{
		Use:   "send <user_id> <workspace_id> <role_id> <user_auth_token>",
		Short: "Send workspace invitation",
		Long: "Send invitation to user for a workspace\n" +
			"For example:\n" +
			"\tmagistrala-cli invitations workspace send 39f97daf-d6b6-40f4-b229-2697be8006ef 4ef09eff-d500-4d56-b04f-d23a512d6f2a ba4c904c-e6d4-4978-9417-1694aac6793e $USER_AUTH_TOKEN\n",
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 4 {
				logUsageCmd(*cmd, cmd.Use)
				return
			}
			inv := smqsdk.Invitation{
				InviteeUserID: args[0],
				WorkspaceID:   args[1],
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
		Use:   "get <workspace_id> <user_auth_token>",
		Short: "Get workspace invitations",
		Long: "Get all invitations for a specific workspace\n" +
			"Usage:\n" +
			"\tmagistrala-cli invitations workspace get <workspace_id> <user_auth_token> - shows invitations for workspace\n" +
			"\tmagistrala-cli invitations workspace get <workspace_id> <user_auth_token> --offset <offset> --limit <limit> - shows invitations with provided offset and limit\n",
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 2 {
				logUsageCmd(*cmd, cmd.Use)
				return
			}

			pageMetadata := smqsdk.PageMetadata{
				Offset: Offset,
				Limit:  Limit,
			}

			u, err := sdk.WorkspaceInvitations(cmd.Context(), pageMetadata, args[1], args[0])
			if err != nil {
				logErrorCmd(*cmd, err)
				return
			}

			logJSONCmd(*cmd, u)
		},
	},
	{
		Use:   "delete <user_id> <workspace_id> <user_auth_token>",
		Short: "Delete workspace invitation",
		Long: "Delete invitation for a specific user and workspace\n" +
			"Usage:\n" +
			"\tmagistrala-cli invitations workspace delete 39f97daf-d6b6-40f4-b229-2697be8006ef 4ef09eff-d500-4d56-b04f-d23a512d6f2a $USER_TOKEN\n",
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

// NewWorkspaceInvitationsCmd returns workspace invitations command.
func NewWorkspaceInvitationsCmd() *cobra.Command {
	cmd := cobra.Command{
		Use:   "workspace [send | get | delete]",
		Short: "Workspace invitations management",
		Long:  `Workspace invitations management to send, get and delete invitations for workspaces`,
	}

	for i := range cmdWorkspaceInvitations {
		cmd.AddCommand(&cmdWorkspaceInvitations[i])
	}

	return &cmd
}

// NewInvitationsCmd returns invitations command.
func NewInvitationsCmd() *cobra.Command {
	cmd := cobra.Command{
		Use:   "invitations [user | workspace]",
		Short: "Invitations management",
		Long:  `Invitations management with separate commands for user and workspace invitations`,
	}

	cmd.AddCommand(NewUserInvitationsCmd())
	cmd.AddCommand(NewWorkspaceInvitationsCmd())

	return &cmd
}
