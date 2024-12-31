// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

// Package main contains entry point for provisioning tool.
package main

import (
	"log"

	"github.com/absmach/supermq/tools/provision"
	"github.com/spf13/cobra"
)

func main() {
	pconf := provision.Config{}

	rootCmd := &cobra.Command{
		Use:   "provision",
		Short: "provision is provisioning tool for SuperMQ",
		Long: `Tool for provisioning series of SuperMQ channels and clients and connecting them together.
Complete documentation is available at https://docs.supermq.abstractmachines.fr`,
		Run: func(_ *cobra.Command, _ []string) {
			if err := provision.Provision(pconf); err != nil {
				log.Fatal(err)
			}
		},
	}

	// Root Flags
	rootCmd.PersistentFlags().StringVarP(&pconf.Host, "host", "", "https://localhost", "address for supermq instance")
	rootCmd.PersistentFlags().StringVarP(&pconf.Prefix, "prefix", "", "", "name prefix for clients and channels")
	rootCmd.PersistentFlags().StringVarP(&pconf.Username, "username", "u", "", "supermq user")
	rootCmd.PersistentFlags().StringVarP(&pconf.Password, "password", "p", "", "supermq users password")
	rootCmd.PersistentFlags().IntVarP(&pconf.Num, "num", "", 10, "number of channels and clients to create and connect")
	rootCmd.PersistentFlags().BoolVarP(&pconf.SSL, "ssl", "", false, "create certificates for mTLS access")
	rootCmd.PersistentFlags().StringVarP(&pconf.CAKey, "cakey", "", "ca.key", "ca.key for creating and signing clients certificate")
	rootCmd.PersistentFlags().StringVarP(&pconf.CA, "ca", "", "ca.crt", "CA for creating and signing clients certificate")

	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err)
	}
}
