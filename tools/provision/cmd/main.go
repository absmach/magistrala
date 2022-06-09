// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"log"

	"github.com/mainflux/mainflux/tools/provision"
	"github.com/spf13/cobra"
)

func main() {
	pconf := provision.Config{}

	var rootCmd = &cobra.Command{
		Use:   "provision",
		Short: "provision is provisioning tool for Mainflux",
		Long: `Tool for provisioning series of Mainflux channels and things and connecting them together.
Complete documentation is available at https://docs.mainflux.io`,
		Run: func(cmd *cobra.Command, args []string) {
			provision.Provision(pconf)
		},
	}

	// Root Flags
	rootCmd.PersistentFlags().StringVarP(&pconf.Host, "host", "", "https://localhost", "address for mainflux instance")
	rootCmd.PersistentFlags().StringVarP(&pconf.Prefix, "prefix", "", "", "name prefix for things and channels")
	rootCmd.PersistentFlags().StringVarP(&pconf.Username, "username", "u", "", "mainflux user")
	rootCmd.PersistentFlags().StringVarP(&pconf.Password, "password", "p", "", "mainflux users password")
	rootCmd.PersistentFlags().IntVarP(&pconf.Num, "num", "", 10, "number of channels and things to create and connect")
	rootCmd.PersistentFlags().BoolVarP(&pconf.SSL, "ssl", "", false, "create certificates for mTLS access")
	rootCmd.PersistentFlags().StringVarP(&pconf.CAKey, "cakey", "", "ca.key", "ca.key for creating and signing things certificate")
	rootCmd.PersistentFlags().StringVarP(&pconf.CA, "ca", "", "ca.crt", "CA for creating and signing things certificate")

	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err)
	}
}
