// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

// Package main contains cli main function to run the cli.
package main

import (
	"log"

	certscli "github.com/absmach/certs/cli"
	"github.com/absmach/magistrala/cli"
	mgcli "github.com/absmach/magistrala/cli"
	mgsdk "github.com/absmach/magistrala/pkg/sdk"
	smqcli "github.com/absmach/supermq/cli"
	smqsdk "github.com/absmach/supermq/pkg/sdk"
	"github.com/spf13/cobra"
)

func main() {
	msgContentType := string(smqsdk.CTJSONSenML)
	smqsdkConf := smqsdk.Config{
		MsgContentType: smqsdk.ContentType(msgContentType),
	}
	mgsdkConf := mgsdk.Config{
		MsgContentType: smqsdk.ContentType(msgContentType),
	}

	// Root
	rootCmd := &cobra.Command{
		Use: "magistrala-cli",
		PersistentPreRun: func(_ *cobra.Command, _ []string) {
			smqcliConf, err := smqcli.ParseConfig(smqsdkConf)
			if err != nil {
				log.Fatalf("Failed to parse config: %s", err)
			}
			if smqcliConf.MsgContentType == "" {
				smqcliConf.MsgContentType = smqsdk.ContentType(msgContentType)
			}
			ss := smqsdk.NewSDK(smqcliConf)
			smqcli.SetSDK(ss)

			mgcliConf, err := mgcli.ParseConfig(mgsdkConf)
			if err != nil {
				log.Fatalf("Failed to parse config: %s", err)
			}
			if mgcliConf.MsgContentType == "" {
				mgcliConf.MsgContentType = smqsdk.ContentType(msgContentType)
			}
			ms := mgsdk.NewSDK(mgcliConf)
			mgcli.SetSDK(ms)
		},
	}
	// SuperMQ API commands
	healthCmd := smqcli.NewHealthCmd()
	usersCmd := smqcli.NewUsersCmd()
	domainsCmd := smqcli.NewDomainsCmd()
	clientsCmd := smqcli.NewClientsCmd()
	groupsCmd := smqcli.NewGroupsCmd()
	channelsCmd := smqcli.NewChannelsCmd()
	messagesCmd := smqcli.NewMessagesCmd()
	certsCmd := certscli.NewCertsCmd()
	configCmd := smqcli.NewConfigCmd()
	invitationsCmd := smqcli.NewInvitationsCmd()
	journalCmd := smqcli.NewJournalCmd()

	// Magistrala API commands
	provisionCmd := mgcli.NewProvisionCmd()
	bootstrapCmd := mgcli.NewBootstrapCmd()
	subscriptionsCmd := mgcli.NewSubscriptionCmd()

	// Root Commands
	rootCmd.AddCommand(healthCmd)
	rootCmd.AddCommand(usersCmd)
	rootCmd.AddCommand(domainsCmd)
	rootCmd.AddCommand(groupsCmd)
	rootCmd.AddCommand(clientsCmd)
	rootCmd.AddCommand(channelsCmd)
	rootCmd.AddCommand(messagesCmd)
	rootCmd.AddCommand(provisionCmd)
	rootCmd.AddCommand(bootstrapCmd)
	rootCmd.AddCommand(certsCmd)
	rootCmd.AddCommand(subscriptionsCmd)
	rootCmd.AddCommand(configCmd)
	rootCmd.AddCommand(invitationsCmd)
	rootCmd.AddCommand(journalCmd)

	// Root Flags
	rootCmd.PersistentFlags().StringVarP(
		&mgsdkConf.BootstrapURL,
		"bootstrap-url",
		"b",
		mgsdkConf.BootstrapURL,
		"Bootstrap service URL",
	)

	rootCmd.PersistentFlags().StringVarP(
		&mgsdkConf.CertsURL,
		"certs-url",
		"s",
		mgsdkConf.CertsURL,
		"Certs service URL",
	)

	rootCmd.PersistentFlags().StringVarP(
		&mgsdkConf.ClientsURL,
		"clients-url",
		"t",
		mgsdkConf.ClientsURL,
		"Clients service URL",
	)

	rootCmd.PersistentFlags().StringVarP(
		&mgsdkConf.UsersURL,
		"users-url",
		"u",
		mgsdkConf.UsersURL,
		"Users service URL",
	)

	rootCmd.PersistentFlags().StringVarP(
		&mgsdkConf.DomainsURL,
		"domains-url",
		"d",
		mgsdkConf.DomainsURL,
		"Domains service URL",
	)

	rootCmd.PersistentFlags().StringVarP(
		&mgsdkConf.HTTPAdapterURL,
		"http-url",
		"p",
		mgsdkConf.HTTPAdapterURL,
		"HTTP adapter URL",
	)

	rootCmd.PersistentFlags().StringVarP(
		&mgsdkConf.ReaderURL,
		"reader-url",
		"R",
		mgsdkConf.ReaderURL,
		"Reader URL",
	)

	rootCmd.PersistentFlags().StringVarP(
		&mgsdkConf.JournalURL,
		"journal-url",
		"a",
		mgsdkConf.JournalURL,
		"Journal Log URL",
	)

	rootCmd.PersistentFlags().StringVarP(
		&mgsdkConf.HostURL,
		"host-url",
		"H",
		mgsdkConf.HostURL,
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
		&mgsdkConf.TLSVerification,
		"insecure",
		"i",
		mgsdkConf.TLSVerification,
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
		&mgsdkConf.CurlFlag,
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

	rootCmd.PersistentFlags().StringVarP(
		&cli.Identity,
		"identity",
		"I",
		"",
		"User identity query parameter",
	)

	rootCmd.PersistentFlags().StringVarP(
		&cli.Metadata,
		"metadata",
		"m",
		"",
		"Metadata query parameter",
	)

	rootCmd.PersistentFlags().StringVarP(
		&cli.Status,
		"status",
		"S",
		"",
		"User status query parameter",
	)

	rootCmd.PersistentFlags().StringVarP(
		&cli.State,
		"state",
		"z",
		"",
		"Bootstrap state query parameter",
	)

	rootCmd.PersistentFlags().StringVarP(
		&cli.Topic,
		"topic",
		"T",
		"",
		"Subscription topic query parameter",
	)

	rootCmd.PersistentFlags().StringVarP(
		&cli.Contact,
		"contact",
		"C",
		"",
		"Subscription contact query parameter",
	)
	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err)
	}
}
