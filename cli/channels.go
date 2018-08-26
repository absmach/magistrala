//
// Copyright (c) 2018
// Mainflux
//
// SPDX-License-Identifier: Apache-2.0
//

package cli

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
)

const channelsEP = "channels"

var cmdChannels = []cobra.Command{
	cobra.Command{
		Use:   "create",
		Short: "create <JSON_channel> <user_auth_token>",
		Long:  `Creates new channel and generates it's UUID`,
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 2 {
				LogUsage(cmd.Short)
				return
			}
			CreateChannel(args[0], args[1])
		},
	},
	cobra.Command{
		Use:   "get",
		Short: "get all/<channel_id> <user_auth_token>",
		Long:  `Gets list of all channels or gets channel by id`,
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 2 {
				LogUsage(cmd.Short)
				return
			}
			if args[0] == "all" {
				GetChannels(args[1])
				return
			}
			GetChannel(args[0], args[1])
		},
	},
	cobra.Command{
		Use:   "update",
		Short: "update <channel_id> <JSON_string> <user_auth_token>",
		Long:  `Updates channel record`,
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 3 {
				LogUsage(cmd.Short)
				return
			}
			UpdateChannel(args[0], args[1], args[2])
		},
	},
	cobra.Command{
		Use:   "delete",
		Short: "delete <channel_id> <user_auth_token>",
		Long:  `Delete channel by ID`,
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 2 {
				LogUsage(cmd.Short)
				return
			}
			DeleteChannel(args[0], args[1])
		},
	},
}

func NewChannelsCmd() *cobra.Command {
	cmd := cobra.Command{
		Use:   "channels",
		Short: "Manipulation with channels",
		Long:  `Manipulation with channels: create, delete or update channels`,
		Run: func(cmd *cobra.Command, args []string) {
			LogUsage(cmd.Short)
		},
	}

	for i, _ := range cmdChannels {
		cmd.AddCommand(&cmdChannels[i])
	}

	return &cmd
}

// CreateChannel - creates new channel and generates UUID
func CreateChannel(data, token string) {
	url := fmt.Sprintf("%s/%s", serverAddr, channelsEP)
	req, err := http.NewRequest("POST", url, strings.NewReader(data))
	SendRequest(req, token, err)
}

// GetChannels - gets all channels
func GetChannels(token string) {
	url := fmt.Sprintf("%s/%s?offset=%s&limit=%s",
		serverAddr, channelsEP, strconv.Itoa(Offset), strconv.Itoa(Limit))
	req, err := http.NewRequest("GET", url, nil)
	SendRequest(req, token, err)
}

// GetChannel - gets channel by ID
func GetChannel(id, token string) {
	url := fmt.Sprintf("%s/%s/%s", serverAddr, channelsEP, id)
	req, err := http.NewRequest("GET", url, nil)
	SendRequest(req, token, err)
}

// UpdateChannel - update a channel
func UpdateChannel(id, data, token string) {
	url := fmt.Sprintf("%s/%s/%s", serverAddr, channelsEP, id)
	req, err := http.NewRequest("PUT", url, strings.NewReader(data))
	SendRequest(req, token, err)
}

// DeleteChannel - removes channel
func DeleteChannel(id, token string) {
	url := fmt.Sprintf("%s/%s/%s", serverAddr, channelsEP, id)
	req, err := http.NewRequest("DELETE", url, nil)
	SendRequest(req, token, err)
}
