// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	mgxsdk "github.com/absmach/magistrala/pkg/sdk/go"
	"github.com/spf13/cobra"
)

var cmdJournal = cobra.Command{
	Use:   "get <entity_type> <entity_id> <domain_id> <user_auth_token>",
	Short: "Get journal",
	Long: "Get journal\n" +
		"Usage:\n" +
		"\tmagistrala-cli journal get user <user_id> <user_auth_token> - lists user journal logs\n" +
		"\tmagistrala-cli journal get <entity_type> <entity_id> <domain_id> <user_auth_token> - lists entity journal logs\n" +
		"\tmagistrala-cli journal get <entity_type> <entity_id> <domain_id> <user_auth_token> --offset <offset> --limit <limit> - lists user journal logs with provided offset and limit\n",
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) < 3 || len(args) > 4 {
			logUsageCmd(*cmd, cmd.Use)
			return
		}
		pageMetadata := mgxsdk.PageMetadata{
			Offset: Offset,
			Limit:  Limit,
		}

		entityType, entityID, token := args[0], args[1], args[2]
		domainID := ""
		if len(args) == 4 {
			entityType, entityID, domainID, token = args[0], args[1], args[2], args[3]
		}

		journal, err := sdk.Journal(entityType, entityID, domainID, pageMetadata, token)
		if err != nil {
			logErrorCmd(*cmd, err)
			return
		}

		logJSONCmd(*cmd, journal)
	},
}

// NewJournalCmd returns journal log command.
func NewJournalCmd() *cobra.Command {
	cmd := cobra.Command{
		Use:   "journal get",
		Short: "journal log",
		Long:  `journal to read journal log`,
	}

	cmd.AddCommand(&cmdJournal)

	return &cmd
}
