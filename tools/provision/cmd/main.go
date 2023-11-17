// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

// Package main contains entry point for provisioning tool.
package main

import (
	"log"

	"github.com/absmach/magistrala/tools/provision"
	"github.com/spf13/cobra"
)

func main() {
	pconf := provision.Config{}

	rootCmd := &cobra.Command{
		Use:   "provision",
		Short: "provision is provisioning tool for Magistrala",
		Long: `Tool for provisioning series of Magistrala channels and things and connecting them together.
Complete documentation is available at https://docs.mainflux.io`,
		Run: func(_ *cobra.Command, _ []string) {
			if err := provision.Provision(pconf); err != nil {
				log.Fatal(err)
			}
		},
	}

	// Root Flags
	rootCmd.PersistentFlags().StringVarP(&pconf.Host, "host", "", "https://localhost", "address for magistrala instance")
	rootCmd.PersistentFlags().StringVarP(&pconf.Prefix, "prefix", "", "", "name prefix for things and channels")
	rootCmd.PersistentFlags().StringVarP(&pconf.Username, "username", "u", "", "magistrala user")
	rootCmd.PersistentFlags().StringVarP(&pconf.Password, "password", "p", "", "magistrala users password")
	rootCmd.PersistentFlags().IntVarP(&pconf.Num, "num", "", 10, "number of channels and things to create and connect")
	rootCmd.PersistentFlags().BoolVarP(&pconf.SSL, "ssl", "", false, "create certificates for mTLS access")
	rootCmd.PersistentFlags().StringVarP(&pconf.CAKey, "cakey", "", "ca.key", "ca.key for creating and signing things certificate")
	rootCmd.PersistentFlags().StringVarP(&pconf.CA, "ca", "", "ca.crt", "CA for creating and signing things certificate")

	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err)
	}
}
