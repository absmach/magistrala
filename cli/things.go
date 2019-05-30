//
// Copyright (c) 2018
// Mainflux
//
// SPDX-License-Identifier: Apache-2.0
//

package cli

import (
	"encoding/json"

	mfxsdk "github.com/mainflux/mainflux/sdk/go"
	"github.com/spf13/cobra"
)

const thingsEP = "things"

var cmdThings = []cobra.Command{
	cobra.Command{
		Use:   "create",
		Short: "create <JSON_thing> <user_auth_token>",
		Long:  `Create new thing, generate his UUID and store it`,
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 2 {
				logUsage(cmd.Short)
				return
			}

			var thing mfxsdk.Thing
			if err := json.Unmarshal([]byte(args[0]), &thing); err != nil {
				logError(err)
				return
			}

			id, err := sdk.CreateThing(thing, args[1])
			if err != nil {
				logError(err)
				return
			}

			logCreated(id)
		},
	},
	cobra.Command{
		Use:   "get",
		Short: "get [all | <thing_id>] <user_auth_token>",
		Long:  `Get all things or thing by id`,
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 2 {
				logUsage(cmd.Short)
				return
			}

			if args[0] == "all" {
				l, err := sdk.Things(args[1], uint64(Offset), uint64(Limit), Name)
				if err != nil {
					logError(err)
					return
				}
				logJSON(l)
				return
			}

			t, err := sdk.Thing(args[0], args[1])
			if err != nil {
				logError(err)
				return
			}

			logJSON(t)
		},
	},
	cobra.Command{
		Use:   "delete",
		Short: "delete <thing_id> <user_auth_token>",
		Long:  `Removes thing from database`,
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 2 {
				logUsage(cmd.Short)
				return
			}

			if err := sdk.DeleteThing(args[0], args[1]); err != nil {
				logError(err)
				return
			}

			logOK()
		},
	},
	cobra.Command{
		Use:   "update",
		Short: "update <JSON_string> <user_auth_token>",
		Long:  `Update thing record`,
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 2 {
				logUsage(cmd.Short)
				return
			}

			var thing mfxsdk.Thing
			if err := json.Unmarshal([]byte(args[0]), &thing); err != nil {
				logError(err)
				return
			}

			if err := sdk.UpdateThing(thing, args[1]); err != nil {
				logError(err)
				return
			}

			logOK()
		},
	},
	cobra.Command{
		Use:   "connect",
		Short: "connect <thing_id> <channel_id> <user_auth_token>",
		Long:  `Connect thing to the channel`,
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 3 {
				logUsage(cmd.Short)
				return
			}

			if err := sdk.ConnectThing(args[0], args[1], args[2]); err != nil {
				logError(err)
				return
			}

			logOK()
		},
	},
	cobra.Command{
		Use:   "disconnect",
		Short: "disconnect <thing_id> <channel_id> <user_auth_token>",
		Long:  `Disconnect thing to the channel`,
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 3 {
				logUsage(cmd.Short)
				return
			}

			if err := sdk.DisconnectThing(args[0], args[1], args[2]); err != nil {
				logError(err)
				return
			}

			logOK()
		},
	},
	cobra.Command{
		Use:   "connections",
		Short: "connections <thing_id> <user_auth_token>",
		Long:  `List of Channels connected to Thing`,
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 2 {
				logUsage(cmd.Short)
				return
			}

			cl, err := sdk.ChannelsByThing(args[1], args[0], uint64(Offset), uint64(Limit))
			if err != nil {
				logError(err)
				return
			}

			logJSON(cl)
		},
	},
}

// NewThingsCmd returns things command.
func NewThingsCmd() *cobra.Command {
	cmd := cobra.Command{
		Use:   "things",
		Short: "Things management",
		Long:  `Things management: create, get, update or delete Thing, connect or disconnect Thing from Channel and get the list of Channels connected to Thing`,
		Run: func(cmd *cobra.Command, args []string) {
			logUsage("things [create | get | update | delete | connect | disconnect | connections]")
		},
	}

	for i := range cmdThings {
		cmd.AddCommand(&cmdThings[i])
	}

	return &cmd
}
