// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"encoding/json"
	"fmt"

	smqsdk "github.com/absmach/magistrala/pkg/sdk"
	"github.com/spf13/cobra"
)

const (
	all              = "all"
	create           = "create"
	get              = "get"
	update           = "update"
	delete           = "delete"
	enable           = "enable"
	disable          = "disable"
	users            = "users"
	sendVerification = "send-verification"
	verifyEmail      = "verify-email"

	usageCreate  = "cli channels create <JSON_channel> <domain_id> <user_auth_token>"
	usageGet     = "cli channels <channel_id|all> get <domain_id> <user_auth_token>"
	usageUpdate  = "cli channels <channel_id> update <JSON_string> <domain_id> <user_auth_token>"
	usageDelete  = "cli channels <channel_id> delete <domain_id> <user_auth_token>"
	usageEnable  = "cli channels <channel_id> enable <domain_id> <user_auth_token>"
	usageDisable = "cli channels <channel_id> disable <domain_id> <user_auth_token>"
	usageUsers   = "cli channels <channel_id> users <domain_id> <user_auth_token>"
)

func NewChannelsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "channels <channel_id|all|create> [operation] [args...]",
		Short: "Channels management",
		Long: `Format: 
  channels create [args...]
  channels <channel_id|all> <operation> [args...]

Operations (require channel_id/all): get, update, delete, enable, disable, users

Examples:
  channels create <JSON_channel> <domain_id> <user_auth_token>
  channels all get <domain_id> <user_auth_token>
  channels <channel_id> get <domain_id> <user_auth_token>
  channels <channel_id> update <JSON_string> <domain_id> <user_auth_token>
  channels <channel_id> delete <domain_id> <user_auth_token>
  channels <channel_id> enable <domain_id> <user_auth_token>
  channels <channel_id> disable <domain_id> <user_auth_token>
  channels <channel_id> users <domain_id> <user_auth_token>`,

		Run: func(cmd *cobra.Command, args []string) {
			if len(args) == 0 {
				logUsageCmd(*cmd, cmd.Use)
				return
			}

			if args[0] == create {
				handleCreate(cmd, args[1:])
				return
			}

			if len(args) < 2 {
				logUsageCmd(*cmd, "channels <channel_id|all> <get|update|delete|enable|disable|users> [args...]")
				return
			}

			channelParams := args[0]
			operation := args[1]
			opArgs := args[2:]

			switch operation {
			case get:
				handleGet(cmd, channelParams, opArgs)
			case update:
				handleUpdate(cmd, channelParams, opArgs)
			case delete:
				handleDelete(cmd, channelParams, opArgs)
			case enable:
				handleEnable(cmd, channelParams, opArgs)
			case disable:
				handleDisable(cmd, channelParams, opArgs)
			case users:
				handleUsers(cmd, channelParams, opArgs)
			default:
				logErrorCmd(*cmd, fmt.Errorf("unknown operation: %s", operation))
			}
		},
	}

	return cmd
}

func handleCreate(cmd *cobra.Command, args []string) {
	if len(args) != 3 {
		logUsageCmd(*cmd, usageCreate)
		return
	}

	var channel smqsdk.Channel
	if err := json.Unmarshal([]byte(args[0]), &channel); err != nil {
		logErrorCmd(*cmd, err)
		return
	}

	channel, err := sdk.CreateChannel(cmd.Context(), channel, args[1], args[2])
	if err != nil {
		logErrorCmd(*cmd, err)
		return
	}

	logJSONCmd(*cmd, channel)
}

func handleGet(cmd *cobra.Command, channelID string, args []string) {
	if len(args) != 2 {
		logUsageCmd(*cmd, usageGet)
		return
	}

	if channelID == all {
		metadata, err := convertMetadata(Metadata)
		if err != nil {
			logErrorCmd(*cmd, err)
			return
		}

		pageMetadata := smqsdk.PageMetadata{
			Name:     "",
			Offset:   Offset,
			Limit:    Limit,
			Metadata: metadata,
		}

		l, err := sdk.Channels(cmd.Context(), pageMetadata, args[0], args[1])
		if err != nil {
			logErrorCmd(*cmd, err)
			return
		}

		logJSONCmd(*cmd, l)
		return
	}

	c, err := sdk.Channel(cmd.Context(), channelID, args[0], args[1])
	if err != nil {
		logErrorCmd(*cmd, err)
		return
	}

	logJSONCmd(*cmd, c)
}

func handleUpdate(cmd *cobra.Command, channelID string, args []string) {
	if len(args) != 3 {
		logUsageCmd(*cmd, usageUpdate)
		return
	}

	var channel smqsdk.Channel
	if err := json.Unmarshal([]byte(args[0]), &channel); err != nil {
		logErrorCmd(*cmd, err)
		return
	}

	channel.ID = channelID
	channel, err := sdk.UpdateChannel(cmd.Context(), channel, args[1], args[2])
	if err != nil {
		logErrorCmd(*cmd, err)
		return
	}

	logJSONCmd(*cmd, channel)
}

func handleDelete(cmd *cobra.Command, channelID string, args []string) {
	if len(args) != 2 {
		logUsageCmd(*cmd, usageDelete)
		return
	}

	if err := sdk.DeleteChannel(cmd.Context(), channelID, args[0], args[1]); err != nil {
		logErrorCmd(*cmd, err)
		return
	}
	logOKCmd(*cmd)
}

func handleEnable(cmd *cobra.Command, channelID string, args []string) {
	if len(args) != 2 {
		logUsageCmd(*cmd, usageEnable)
		return
	}

	channel, err := sdk.EnableChannel(cmd.Context(), channelID, args[0], args[1])
	if err != nil {
		logErrorCmd(*cmd, err)
		return
	}

	logJSONCmd(*cmd, channel)
}

func handleDisable(cmd *cobra.Command, channelID string, args []string) {
	if len(args) != 2 {
		logUsageCmd(*cmd, usageDisable)
		return
	}

	channel, err := sdk.DisableChannel(cmd.Context(), channelID, args[0], args[1])
	if err != nil {
		logErrorCmd(*cmd, err)
		return
	}

	logJSONCmd(*cmd, channel)
}

func handleUsers(cmd *cobra.Command, channelID string, args []string) {
	if len(args) != 2 {
		logUsageCmd(*cmd, usageUsers)
		return
	}

	pm := smqsdk.PageMetadata{
		Offset: Offset,
		Limit:  Limit,
	}
	ul, err := sdk.ListChannelMembers(cmd.Context(), channelID, args[0], pm, args[1])
	if err != nil {
		logErrorCmd(*cmd, err)
		return
	}

	logJSONCmd(*cmd, ul)
}
