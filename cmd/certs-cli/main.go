// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

// Package main contains cli main function to run the cli.
package main

import (
	"log"

	"github.com/absmach/supermq/certs/cli"
	"github.com/absmach/supermq/certs/sdk"
	"github.com/spf13/cobra"
)

func main() {
	msgContentType := string(sdk.CTJSONSenML)
	sdkConf := sdk.Config{
		MsgContentType: sdk.ContentType(msgContentType),
	}

	// Root
	rootCmd := &cobra.Command{
		Use: "certs-cli",
		PersistentPreRun: func(_ *cobra.Command, _ []string) {
			cliConf, err := cli.ParseConfig(sdkConf)
			if err != nil {
				log.Fatalf("Failed to parse config: %s", err)
			}
			if cliConf.MsgContentType == "" {
				cliConf.MsgContentType = sdk.ContentType(msgContentType)
			}
			s := sdk.NewSDK(cliConf)
			cli.SetSDK(s)
		},
	}
	// API commands
	certsCmd := cli.NewCertsCmd()

	// Root Commands
	rootCmd.AddCommand(certsCmd)

	rootCmd.PersistentFlags().StringVarP(
		&sdkConf.CertsURL,
		"certs-url",
		"s",
		sdkConf.CertsURL,
		"Certs service URL",
	)

	rootCmd.PersistentFlags().StringVarP(
		&sdkConf.HostURL,
		"host-url",
		"H",
		sdkConf.HostURL,
		"Host URL",
	)

	rootCmd.PersistentFlags().StringVarP(
		&msgContentType,
		"content-type",
		"y",
		msgContentType,
		"Message content type",
	)

	rootCmd.PersistentFlags().BoolVarP(
		&sdkConf.TLSVerification,
		"insecure",
		"i",
		sdkConf.TLSVerification,
		"Do not check for TLS cert",
	)

	rootCmd.PersistentFlags().StringVarP(
		&cli.ConfigPath,
		"config",
		"c",
		cli.ConfigPath,
		"Config path",
	)

	rootCmd.PersistentFlags().BoolVarP(
		&cli.RawOutput,
		"raw",
		"r",
		cli.RawOutput,
		"Enables raw output mode for easier parsing of output",
	)
	rootCmd.PersistentFlags().BoolVarP(
		&sdkConf.CurlFlag,
		"curl",
		"x",
		false,
		"Convert HTTP request to cURL command",
	)

	// Client and Channels Flags
	rootCmd.PersistentFlags().Uint64VarP(
		&cli.Limit,
		"limit",
		"l",
		10,
		"Limit query parameter",
	)

	rootCmd.PersistentFlags().Uint64VarP(
		&cli.Offset,
		"offset",
		"o",
		0,
		"Offset query parameter",
	)

	rootCmd.PersistentFlags().StringVarP(
		&cli.Name,
		"name",
		"n",
		"",
		"Name query parameter",
	)

	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err)
	}
}
