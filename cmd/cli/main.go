//
// Copyright (c) 2018
// Mainflux
//
// SPDX-License-Identifier: Apache-2.0
//

package main

import (
	"log"

	"github.com/mainflux/mainflux/cli"
	"github.com/mainflux/mainflux/sdk/go"
	"github.com/spf13/cobra"
)

func main() {
	conf := struct {
		host     string
		port     string
		insecure bool
	}{
		"localhost",
		"",
		false,
	}

	// Root
	var rootCmd = &cobra.Command{
		Use: "mainflux-cli",
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			s := sdk.NewMfxSDK(conf.host, conf.port, !conf.insecure)
			cli.SetSDK(s)
		},
	}

	// API commands
	versionCmd := cli.NewVersionCmd()
	usersCmd := cli.NewUsersCmd()
	thingsCmd := cli.NewThingsCmd()
	channelsCmd := cli.NewChannelsCmd()
	messagesCmd := cli.NewMessagesCmd()

	// Root Commands
	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(usersCmd)
	rootCmd.AddCommand(thingsCmd)
	rootCmd.AddCommand(channelsCmd)
	rootCmd.AddCommand(messagesCmd)

	// Root Flags
	rootCmd.PersistentFlags().StringVarP(
		&conf.host, "host", "m", conf.host, "HTTP Host address")
	rootCmd.PersistentFlags().StringVarP(
		&conf.port, "port", "p", conf.port, "HTTP Host Port")
	rootCmd.PersistentFlags().BoolVarP(
		&conf.insecure, "insecure", "i", false, "do not use TLS")

	// Client and Channels Flags
	rootCmd.PersistentFlags().UintVarP(
		&cli.Limit, "limit", "l", 100, "limit query parameter")
	rootCmd.PersistentFlags().UintVarP(
		&cli.Offset, "offset", "o", 0, "offset query parameter")

	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err)
	}
}
