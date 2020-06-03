// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"log"

	"github.com/mainflux/mainflux/cli"
	sdk "github.com/mainflux/mainflux/pkg/sdk/go"
	"github.com/spf13/cobra"
)

func main() {
	msgContentType := string(sdk.CTJSONSenML)
	sdkConf := sdk.Config{
		BaseURL:           "http://localhost",
		ReaderURL:         "http://localhost:8905",
		BootstrapURL:      "http://localhost:8202",
		ReaderPrefix:      "",
		UsersPrefix:       "",
		ThingsPrefix:      "",
		HTTPAdapterPrefix: "http",
		BootstrapPrefix:   "things",
		MsgContentType:    sdk.ContentType(msgContentType),
		TLSVerification:   false,
	}

	// Root
	var rootCmd = &cobra.Command{
		Use: "mainflux-cli",
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			sdkConf.MsgContentType = sdk.ContentType(msgContentType)
			s := sdk.NewSDK(sdkConf)
			cli.SetSDK(s)
		},
	}

	// API commands
	versionCmd := cli.NewVersionCmd()
	usersCmd := cli.NewUsersCmd()
	thingsCmd := cli.NewThingsCmd()
	channelsCmd := cli.NewChannelsCmd()
	messagesCmd := cli.NewMessagesCmd()
	provisionCmd := cli.NewProvisionCmd()
	bootstrapCmd := cli.NewBootstrapCmd()

	// Root Commands
	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(usersCmd)
	rootCmd.AddCommand(thingsCmd)
	rootCmd.AddCommand(channelsCmd)
	rootCmd.AddCommand(messagesCmd)
	rootCmd.AddCommand(provisionCmd)
	rootCmd.AddCommand(bootstrapCmd)

	// Root Flags
	rootCmd.PersistentFlags().StringVarP(
		&sdkConf.BaseURL,
		"mainflux-url",
		"m",
		sdkConf.BaseURL,
		"Mainflux host URL",
	)

	rootCmd.PersistentFlags().StringVarP(
		&sdkConf.UsersPrefix,
		"users-prefix",
		"u",
		sdkConf.UsersPrefix,
		"Mainflux users service prefix",
	)

	rootCmd.PersistentFlags().StringVarP(
		&sdkConf.ThingsPrefix,
		"things-prefix",
		"t",
		sdkConf.ThingsPrefix,
		"Mainflux things service prefix",
	)

	rootCmd.PersistentFlags().StringVarP(
		&sdkConf.HTTPAdapterPrefix,
		"http-prefix",
		"a",
		sdkConf.HTTPAdapterPrefix,
		"Mainflux http adapter prefix",
	)

	rootCmd.PersistentFlags().StringVarP(
		&msgContentType,
		"content-type",
		"c",
		msgContentType,
		"Mainflux message content type",
	)

	rootCmd.PersistentFlags().BoolVarP(
		&sdkConf.TLSVerification,
		"insecure",
		"i",
		sdkConf.TLSVerification,
		"Do not check for TLS cert",
	)

	// Client and Channels Flags
	rootCmd.PersistentFlags().UintVarP(
		&cli.Limit,
		"limit",
		"l",
		100,
		"limit query parameter",
	)

	rootCmd.PersistentFlags().UintVarP(
		&cli.Offset,
		"offset",
		"o",
		0,
		"offset query parameter",
	)

	rootCmd.PersistentFlags().StringVarP(
		&cli.Name,
		"name",
		"n",
		"",
		"name query parameter",
	)

	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err)
	}
}
